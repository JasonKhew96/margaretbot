package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JasonKhew96/margaretbot/entityhelper"
	"github.com/JasonKhew96/margaretbot/websub"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/google/shlex"
)

type BotHelper struct {
	mb *MargaretBot

	bot *gotgbot.Bot

	msgChannel chan MultiMessage

	msgForwards map[int64]chan MultiMessage

	ctx context.Context
}

func NewCommand(command string, handler func(*gotgbot.Bot, *ext.Context) error) handlers.Command {
	return handlers.NewCommand(command, handler).SetTriggers([]rune("/!"))
}

func NewBot(mb *MargaretBot) error {
	botApiUrl := gotgbot.DefaultAPIURL
	if mb.config.BotApiUrl != "" {
		botApiUrl = mb.config.BotApiUrl
	}

	bot, err := gotgbot.NewBot(mb.config.BotToken, &gotgbot.BotOpts{
		BotClient: &gotgbot.BaseBotClient{
			Client: http.Client{},
			DefaultRequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 30,
				APIURL:  botApiUrl,
			},
		},
		RequestOpts: &gotgbot.RequestOpts{
			APIURL: botApiUrl,
		},
	})
	if err != nil {
		return err
	}

	msgForwards := make(map[int64]chan MultiMessage)

	b := &BotHelper{
		mb:          mb,
		bot:         bot,
		msgChannel:  make(chan MultiMessage),
		msgForwards: msgForwards,
		ctx:         context.Background(),
	}
	mb.bot = b

	forwards, err := mb.db.GetForwards()
	if err == nil && forwards != nil {
		for _, forward := range forwards {
			msgForwards[forward.ChatID] = make(chan MultiMessage)
			go b.telegramWorker(forward.ChatID, msgForwards[forward.ChatID])
		}
	}

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("an error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
	})
	updater := ext.NewUpdater(dispatcher, nil)

	dispatcher.AddHandler(NewCommand("sub", b.handleSubCommand))
	dispatcher.AddHandler(NewCommand("unsub", b.handleUnsubCommand))
	dispatcher.AddHandler(NewCommand("r", b.handleRegexCommand))
	dispatcher.AddHandler(NewCommand("rb", b.handleRegexBanCommand))
	dispatcher.AddHandler(NewCommand("debug", b.handleDebugCommand))
	dispatcher.AddHandler(NewCommand("l", b.handleListCommand))
	dispatcher.AddHandler(NewCommand("reload", b.handleReloadCommand))
	dispatcher.AddHandler(NewCommand("lf", b.handleListForwardCommand))
	dispatcher.AddHandler(NewCommand("f", b.handleForwardCommand))
	dispatcher.AddHandler(NewCommand("rf", b.handleRemoveForwardCommand))
	// dispatcher.AddHandler(NewCommand("sfr", b.handleSetForwardRegexCommand))
	// dispatcher.AddHandler(NewCommand("sfrb", b.handleSetForwardRegexBanCommand))
	dispatcher.AddHandler(NewCommand("lfn", b.handleListForwardNoCommand))
	// dispatcher.AddHandler(NewCommand("sfn", b.handleSetForwardNoCommand))
	// dispatcher.AddHandler(NewCommand("rfn", b.handleSetForwardNoCommand))

	err = updater.StartPolling(bot, &ext.PollingOpts{
		DropPendingUpdates: false,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout:        59,
			AllowedUpdates: []string{"message"},
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Minute,
				APIURL:  botApiUrl,
			},
		},
	})
	if err != nil {
		return err
	}

	go b.telegramWorker(mb.config.ChatId, b.msgChannel)

	log.Printf("Bot %s started", bot.Username)

	return nil
}

func (b *BotHelper) handleSubCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.mb.config.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}
	messageThreadId := ctx.EffectiveMessage.MessageThreadId
	text := ctx.EffectiveMessage.Text
	channelId := text[5:]

	if !strings.HasPrefix(channelId, "UC") || strings.Contains(channelId, " ") {
		_, err := ctx.EffectiveMessage.Reply(bot, "Invalid channel ID", nil)
		if err != nil {
			log.Println(err)
			return err
		}
		return nil
	}

	newSecret := sha256.Sum256([]byte(b.mb.config.Secret))

	callbackUrl := fmt.Sprintf("https://%s/webhook/%s/%d/%s", b.mb.config.ServerDomain, fmt.Sprintf("%x", newSecret), messageThreadId, channelId)
	topicUrl := fmt.Sprintf("https://www.youtube.com/xml/feeds/videos.xml?channel_id=%s", channelId)

	// subs, err := b.db.GetSubscription(channelId)
	// if err != nil && !errors.Is(err, sql.ErrNoRows) {
	// 	return err
	// }
	// if subs != nil {
	// TODO
	// ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("already subscribed at https://"))
	// }

	if err := b.mb.ws.Subscribe(websub.ModeSubscribe, callbackUrl, topicUrl, &websub.SubscribeOpts{
		LeaseSeconds: 86400,
	}); err != nil {
		log.Println(err)
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	log.Printf("subscribing to %s...", channelId)

	// swap prefix UC to UU for default playlist
	// UCuAXFkgsw1L7xaCfnd5JJOw channel id
	// UUuAXFkgsw1L7xaCfnd5JJOw default playlist id a.k.a. "uploads"

	playlistId := "UU" + channelId[2:]

	playlistItems, err := b.mb.yt.service.PlaylistItems.List([]string{"snippet", "contentDetails"}).PlaylistId(playlistId).MaxResults(8).Do()
	if err != nil {
		return err
	}

	var videoIdList []string

	for i := len(playlistItems.Items) - 1; i >= 0; i-- {
		videoId := playlistItems.Items[i].ContentDetails.VideoId
		isShort, err := b.mb.yt.IsShort(videoId, playlistItems.Items[i].Snippet.Title)
		if err != nil {
			log.Printf("failed to check if video is short: %v", err)
		}
		if isShort {
			log.Printf("video is a short: %s", videoId)
			continue
		}
		videoIdList = append(videoIdList, videoId)

		if err := b.mb.db.UpsertCache(videoId, true, true); err != nil {
			log.Printf("unable to insert cache: %v", err)
			continue
		}
	}

	videoList, err := b.mb.yt.service.Videos.List([]string{"snippet", "contentDetails", "liveStreamingDetails"}).Id(videoIdList...).Do()
	if err != nil {
		log.Printf("failed to get video list: %v", err)
		return err
	}
	for _, video := range videoList.Items {
		videoId := video.Id

		videoTitle := video.Snippet.Title
		videoUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoId)
		// thumbnailUrl := fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", videoId)
		videoDescription := video.Snippet.Description
		if utf8.RuneCountInString(videoDescription) > 4096 {
			videoDescription = truncateByRunes(videoDescription, 4095) + "…"
		}
		var allowedRegion string
		var blockedRegion string
		if video.ContentDetails.RegionRestriction != nil {
			if len(video.ContentDetails.RegionRestriction.Blocked) >= 249 {
				continue
			}

			if video.ContentDetails.RegionRestriction.Allowed != nil {
				allowedRegion = strings.Join(video.ContentDetails.RegionRestriction.Allowed, ", ")
			}
			if video.ContentDetails.RegionRestriction.Blocked != nil {
				blockedRegion = strings.Join(video.ContentDetails.RegionRestriction.Blocked, ", ")
			}
		}
		var scheduledStartTime string
		if video.LiveStreamingDetails != nil && video.LiveStreamingDetails.ScheduledStartTime != "" {
			scheduledStartTime = video.LiveStreamingDetails.ScheduledStartTime
		}
		channelName := video.Snippet.ChannelTitle
		publishedTime := video.Snippet.PublishedAt

		var thumbnailUrl string
		if video.Snippet.Thumbnails != nil && video.Snippet.Thumbnails.Maxres != nil {
			thumbnailUrl = video.Snippet.Thumbnails.Maxres.Url
		}

		var timezone string
		if video.Snippet.DefaultLanguage != "" {
			timezone = GetTimeZone(video.Snippet.DefaultLanguage)
		} else if video.Snippet.DefaultAudioLanguage != "" {
			timezone = GetTimeZone(video.Snippet.DefaultAudioLanguage)
		} else {
			timezone = "UTC"
		}

		// duration

		caption, entities := BuildCaption(&Caption{
			VideoTitle:         videoTitle,
			VideoUrl:           videoUrl,
			VideoDescription:   videoDescription,
			ChannelName:        channelName,
			ChannelUrl:         fmt.Sprintf("https://www.youtube.com/channel/%s", video.Snippet.ChannelId),
			AllowedRegion:      allowedRegion,
			BlockedRegion:      blockedRegion,
			ScheduledStartTime: scheduledStartTime,
			PublishedTime:      publishedTime,
			TimeZone:           timezone,
		})

		if len(caption) < 1024 {
			msg := Message{
				text:            caption,
				videoUrl:        videoUrl,
				messageThreadId: messageThreadId,
				entities:        entities,
				linkPreviewOptions: &gotgbot.LinkPreviewOptions{
					Url:              videoUrl,
					PreferLargeMedia: true,
					ShowAboveText:    true,
				},
			}
			if thumbnailUrl != "" {
				msg.imageUrl = thumbnailUrl
			}
			b.mb.bot.msgChannel <- MultiMessage{
				First: &msg,
			}
		} else {
			caption, entities := BuildCaption(&Caption{
				VideoTitle: videoTitle,
				VideoUrl:   videoUrl,
				// VideoDescription:   videoDescription,
				ChannelName:        channelName,
				ChannelUrl:         fmt.Sprintf("https://www.youtube.com/channel/%s", video.Snippet.ChannelId),
				AllowedRegion:      allowedRegion,
				BlockedRegion:      blockedRegion,
				ScheduledStartTime: scheduledStartTime,
				PublishedTime:      publishedTime,
				TimeZone:           timezone,
			})
			msg := Message{
				text:            caption,
				videoUrl:        videoUrl,
				messageThreadId: messageThreadId,
				entities:        entities,
				linkPreviewOptions: &gotgbot.LinkPreviewOptions{
					Url:              videoUrl,
					PreferLargeMedia: true,
					ShowAboveText:    true,
				},
			}
			if thumbnailUrl != "" {
				msg.imageUrl = thumbnailUrl
			}
			b.mb.bot.msgChannel <- MultiMessage{
				First: &msg,
				Last: []Message{
					{
						text:            videoDescription,
						messageThreadId: messageThreadId,
						entities: []gotgbot.MessageEntity{
							{
								Type:   "expandable_blockquote",
								Offset: 0,
								Length: getUtf16Len(videoDescription),
							},
						},
						linkPreviewOptions: &gotgbot.LinkPreviewOptions{
							IsDisabled: true,
						},
					},
				},
			}
		}
	}

	return nil
}

func (b *BotHelper) handleUnsubCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.mb.config.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}
	messageThreadId := ctx.EffectiveMessage.MessageThreadId
	text := ctx.EffectiveMessage.Text
	channelId := text[7:]

	newSecret := sha256.Sum256([]byte(b.mb.config.Secret))

	callbackUrl := fmt.Sprintf("https://%s/webhook/%s/%d/%s", b.mb.config.ServerDomain, fmt.Sprintf("%x", newSecret), messageThreadId, channelId)
	topicUrl := fmt.Sprintf("https://www.youtube.com/xml/feeds/videos.xml?channel_id=%s", channelId)

	if err := b.mb.ws.Subscribe(websub.ModeUnsubscribe, callbackUrl, topicUrl, nil); err != nil {
		log.Println(err)
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}

func (b *BotHelper) handleRegexCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.mb.config.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}
	text := ctx.EffectiveMessage.Text
	s, err := shlex.Split(text[3:])
	if err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	if len(s) != 2 {
		_, err := ctx.EffectiveMessage.Reply(bot, "Usage: /r <channel_id> <regex>", nil)
		return err
	}
	if _, err := b.mb.db.GetSubscription(s[0]); errors.Is(err, sql.ErrNoRows) {
		_, err := ctx.EffectiveMessage.Reply(bot, "channel_id does not exists", nil)
		return err
	}
	if _, err := regexp.Compile(s[1]); err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	if err := b.mb.db.UpsertSubscription(s[0], &SubscriptionOpts{
		Regex: s[1],
	}); err != nil {
		log.Println(err)
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}

	msg := entityhelper.NewMessage()
	msg.AddText("subscribed to ")
	msg.AddEntity(s[0], gotgbot.MessageEntity{
		Type: "code",
	})
	msg.AddText(" with regex ")
	msg.AddEntity(s[1], gotgbot.MessageEntity{
		Type: "code",
	})

	_, err = ctx.EffectiveMessage.Reply(bot, msg.GetText(), &gotgbot.SendMessageOpts{
		Entities: msg.GetEntities(),
	})

	return err
}

func (b *BotHelper) handleRegexBanCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.mb.config.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}
	text := ctx.EffectiveMessage.Text
	s, err := shlex.Split(text[4:])
	if err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	if len(s) != 2 {
		_, err := ctx.EffectiveMessage.Reply(bot, "Usage: /rb <channel_id> <regex>", nil)
		return err
	}
	if _, err := b.mb.db.GetSubscription(s[0]); errors.Is(err, sql.ErrNoRows) {
		_, err := ctx.EffectiveMessage.Reply(bot, "channel_id does not exists", nil)
		return err
	}
	if _, err := regexp.Compile(s[1]); err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	if err := b.mb.db.UpsertSubscription(s[0], &SubscriptionOpts{
		RegexBan: s[1],
	}); err != nil {
		log.Println(err)
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}

	msg := entityhelper.NewMessage()
	msg.AddText("subscribed to ")
	msg.AddEntity(s[0], gotgbot.MessageEntity{
		Type: "code",
	})
	msg.AddText(" with regexban ")
	msg.AddEntity(s[1], gotgbot.MessageEntity{
		Type: "code",
	})

	_, err = ctx.EffectiveMessage.Reply(bot, msg.GetText(), &gotgbot.SendMessageOpts{
		Entities: msg.GetEntities(),
	})

	return err
}

func (b *BotHelper) handleListCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.mb.config.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}

	subscriptions, err := b.mb.db.GetSubscriptionsByThreadID(ctx.EffectiveMessage.MessageThreadId)
	if err != nil {
		log.Println(err)
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}

	if len(subscriptions) == 0 {
		_, err := ctx.EffectiveMessage.Reply(bot, "No subscriptions found", nil)
		return err
	}

	msg := entityhelper.NewMessage()
	for _, sub := range subscriptions {
		quotedMsg := entityhelper.NewMessage()
		quotedMsg.AddEntity(sub.ChannelID, gotgbot.MessageEntity{
			Type: "code",
		})
		if sub.ChannelTitle.IsValue() {
			quotedMsg.AddText(" - ")
			quotedMsg.AddEntity(sub.ChannelTitle.GetOrZero(), gotgbot.MessageEntity{
				Type: "code",
			})
		}
		quotedMsg.AddText("\n")
		r, ok := sub.Regex.Get()
		if ok && r != "" {
			quotedMsg.AddText("regex: ")
			quotedMsg.AddEntity(r, gotgbot.MessageEntity{
				Type: "code",
			})
			quotedMsg.AddText("\n")
		}
		rb, ok := sub.RegexBan.Get()
		if ok && rb != "" {
			quotedMsg.AddText("regexban: ")
			quotedMsg.AddEntity(rb, gotgbot.MessageEntity{
				Type: "code",
			})
			quotedMsg.AddText("\n")
		}
		msg.AddNestedEntity(quotedMsg, gotgbot.MessageEntity{
			Type: "expandable_blockquote",
		})
	}

	_, err = ctx.EffectiveMessage.Reply(bot, msg.GetText(), &gotgbot.SendMessageOpts{
		Entities: msg.GetEntities(),
	})

	return err
}

func (b *BotHelper) handleDebugCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.mb.config.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}
	messageThreadId := ctx.EffectiveMessage.MessageThreadId
	videoId := "dQw4w9WgXcQ"

	b.mb.wh.queue[videoId] = Queue{
		threadId:      messageThreadId,
		videoTitle:    "TEST TEST TEST",
		videoUrl:      "https://www.youtube.com/watch?v=" + videoId,
		channelName:   "TEST TEST TEST",
		channelUrl:    "https://www.youtube.com/channel/UCuAXFkgsw1L7xaCfnd5JJOw",
		publishedTime: time.Now().Format(time.RFC3339),
	}

	b.mb.wh.debounced(b.mb.wh.processAPI)

	return nil
}

func (b *BotHelper) handleReloadCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}

	var err error
	b.mb.config, err = parseConfig()

	return err
}

func (b *BotHelper) handleListForwardCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}

	forwards, err := b.mb.db.GetForwards()
	if errors.Is(err, sql.ErrNoRows) {
		_, err := ctx.EffectiveMessage.Reply(bot, "There is no forward(s)", nil)
		return err
	}

	quotedMsg := entityhelper.NewMessage()
	for _, forward := range forwards {
		quotedMsg.AddEntity(strconv.FormatInt(forward.ChatID, 10), gotgbot.MessageEntity{
			Type: "code",
		})
		if forward.Regex.IsValue() {
			quotedMsg.AddText("\nregex: ")
			quotedMsg.AddEntity(forward.Regex.GetOrZero(), gotgbot.MessageEntity{
				Type: "code",
			})
		}
		if forward.RegexBan.IsValue() {
			quotedMsg.AddText("\nregexBan: ")
			quotedMsg.AddEntity(forward.RegexBan.GetOrZero(), gotgbot.MessageEntity{
				Type: "code",
			})
		}
		quotedMsg.AddText("\n")
	}
	msg := entityhelper.NewMessage()
	msg.AddNestedEntity(quotedMsg, gotgbot.MessageEntity{
		Type: "expandable_blockquote",
	})

	_, err = ctx.EffectiveMessage.Reply(bot, msg.GetText(), &gotgbot.SendMessageOpts{
		Entities: msg.GetEntities(),
	})

	return err
}

func (b *BotHelper) handleForwardCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}

	text := ctx.EffectiveMessage.Text
	if len(text) <= 3 {
		_, err := ctx.EffectiveMessage.Reply(bot, "Usage: /f <chat_id>", nil)
		return err
	}

	s, err := shlex.Split(text[3:])
	if err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, "Usage: /f <chat_id> <regex> <regex_ban>", nil)
		return err
	}

	chatId, err := strconv.ParseInt(s[0], 10, 64)
	if err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, "Usage: /f <chat_id>", nil)
		return err
	}

	forward, err := b.mb.db.GetForward(chatId)
	if !errors.Is(err, sql.ErrNoRows) {
		_, err := ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("%d is already subscribed to forward", forward.ChatID), nil)
		return err
	}

	regex := ""
	regexBan := ""
	if len(s) > 1 {
		regex = s[1]
	}
	if len(s) > 2 {
		regexBan = s[2]
	}
	if err := b.mb.db.UpsertForward(chatId, regex, regexBan); err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("%d failed to subscribe to forward", forward.ChatID), nil)
		return err
	}

	b.msgForwards[chatId] = make(chan MultiMessage)
	go b.telegramWorker(chatId, b.msgForwards[chatId])

	_, err = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("%d subscribe to forward successful", forward.ChatID), nil)

	return err
}

func (b *BotHelper) handleRemoveForwardCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}

	text := ctx.EffectiveMessage.Text
	if len(text) <= 4 {
		_, err := ctx.EffectiveMessage.Reply(bot, "Usage: /rf <chat_id>", nil)
		return err
	}

	chatId, err := strconv.ParseInt(text[4:], 10, 64)
	if err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, "Usage: /rf <chat_id>", nil)
		return err
	}

	if err := b.mb.db.DeleteForward(chatId); err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("%d failed unsubscribe forward", chatId), nil)
		return err
	}

	forwardChannel, ok := b.msgForwards[chatId]
	if ok {
		close(forwardChannel)
		delete(b.msgForwards, chatId)
	}

	_, err = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("%d unsubscribe to forward successful", chatId), nil)

	return err
}

func (b *BotHelper) handleListForwardNoCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.mb.config.OwnerId {
		return nil
	}

	forwards, err := b.mb.db.GetForwardNos()
	if errors.Is(err, sql.ErrNoRows) {
		_, err := ctx.EffectiveMessage.Reply(bot, "There is no noforward(s)", nil)
		return err
	}

	quotedMsg := entityhelper.NewMessage()
	for _, forward := range forwards {
		quotedMsg.AddEntity(strconv.FormatInt(forward.ChatID, 10), gotgbot.MessageEntity{
			Type: "code",
		})
		quotedMsg.AddText(" ")
		quotedMsg.AddEntity(forward.ChannelID, gotgbot.MessageEntity{
			Type: "code",
		})
		quotedMsg.AddText("\n")
	}
	msg := entityhelper.NewMessage()
	msg.AddNestedEntity(quotedMsg, gotgbot.MessageEntity{
		Type: "expandable_blockquote",
	})

	_, err = ctx.EffectiveMessage.Reply(bot, msg.GetText(), &gotgbot.SendMessageOpts{
		Entities: msg.GetEntities(),
	})

	return err
}
