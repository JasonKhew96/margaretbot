package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/JasonKhew96/margaretbot/entityhelper"
	"github.com/JasonKhew96/margaretbot/websub"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/google/shlex"
	"golang.org/x/time/rate"
)

type Bot struct {
	m *MargaretBot

	b *gotgbot.Bot

	msgChannel chan MultiMessage

	limiter *rate.Limiter

	ctx context.Context
}

func NewCommand(command string, handler func(*gotgbot.Bot, *ext.Context) error) handlers.Command {
	return handlers.NewCommand(command, handler).SetTriggers([]rune("/!"))
}

func NewBot(margaret *MargaretBot) (*Bot, error) {
	botApiUrl := gotgbot.DefaultAPIURL
	if margaret.c.BotApiUrl != "" {
		botApiUrl = margaret.c.BotApiUrl
	}

	bot, err := gotgbot.NewBot(margaret.c.BotToken, &gotgbot.BotOpts{
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
		return nil, err
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute/20), 1)

	b := &Bot{
		m:          margaret,
		b:          bot,
		msgChannel: make(chan MultiMessage),
		limiter:    limiter,
		ctx:        context.Background(),
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
		return nil, err
	}

	go b.telegramWorker(margaret.c.ChatId, b.msgChannel)

	log.Printf("Bot %s started", bot.Username)

	return b, nil
}

func (b *Bot) handleSubCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.m.c.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.m.c.OwnerId {
		return nil
	}
	messageThreadId := ctx.EffectiveMessage.MessageThreadId
	text := ctx.EffectiveMessage.Text
	channelId := text[5:]

	newSecret := sha256.Sum256([]byte(b.m.c.Secret))

	callbackUrl := fmt.Sprintf("https://%s/webhook/%s/%d/%s", b.m.c.ServerDomain, fmt.Sprintf("%x", newSecret), messageThreadId, channelId)
	topicUrl := fmt.Sprintf("https://www.youtube.com/xml/feeds/videos.xml?channel_id=%s", channelId)

	// subs, err := b.db.GetSubscription(channelId)
	// if err != nil && !errors.Is(err, sql.ErrNoRows) {
	// 	return err
	// }
	// if subs != nil {
	// TODO
	// ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("already subscribed at https://"))
	// }

	if err := b.m.ws.Subscribe(websub.ModeSubscribe, callbackUrl, topicUrl, &websub.SubscribeOpts{
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

	playlistItems, err := b.m.y.service.PlaylistItems.List([]string{"contentDetails"}).PlaylistId(playlistId).MaxResults(8).Do()
	if err != nil {
		return err
	}

	var videoIdList []string

	for i := len(playlistItems.Items) - 1; i >= 0; i-- {
		videoId := playlistItems.Items[i].ContentDetails.VideoId
		isShort, err := b.m.y.IsShort(videoId)
		if err != nil {
			log.Printf("failed to check if video is short: %v", err)
		}
		if isShort {
			log.Printf("video is a short: %s", videoId)
			continue
		}
		videoIdList = append(videoIdList, videoId)

		if err := b.m.db.UpsertCache(videoId); err != nil {
			log.Printf("unable to insert cache: %v", err)
			continue
		}
	}

	videoList, err := b.m.y.service.Videos.List([]string{"snippet", "contentDetails", "liveStreamingDetails"}).Id(videoIdList...).Do()
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
			b.m.b.msgChannel <- MultiMessage{
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
			b.m.b.msgChannel <- MultiMessage{
				First: &msg,
				Last: &Message{
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
			}
		}
	}

	return nil
}

func (b *Bot) handleUnsubCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.m.c.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.m.c.OwnerId {
		return nil
	}
	messageThreadId := ctx.EffectiveMessage.MessageThreadId
	text := ctx.EffectiveMessage.Text
	channelId := text[7:]

	newSecret := sha256.Sum256([]byte(b.m.c.Secret))

	callbackUrl := fmt.Sprintf("https://%s/webhook/%s/%d/%s", b.m.c.ServerDomain, fmt.Sprintf("%x", newSecret), messageThreadId, channelId)
	topicUrl := fmt.Sprintf("https://www.youtube.com/xml/feeds/videos.xml?channel_id=%s", channelId)

	if err := b.m.ws.Subscribe(websub.ModeUnsubscribe, callbackUrl, topicUrl, nil); err != nil {
		log.Println(err)
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}

func (b *Bot) handleRegexCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.m.c.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.m.c.OwnerId {
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
	if _, err := regexp.Compile(s[1]); err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	if err := b.m.db.UpsertSubscription(s[0], &SubscriptionOpts{
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

func (b *Bot) handleRegexBanCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.m.c.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.m.c.OwnerId {
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
	if _, err := regexp.Compile(s[1]); err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	if err := b.m.db.UpsertSubscription(s[0], &SubscriptionOpts{
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

func (b *Bot) handleListCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.m.c.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.m.c.OwnerId {
		return nil
	}

	subscriptions, err := b.m.db.GetSubscriptionsByThreadID(ctx.EffectiveMessage.MessageThreadId)
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
		quotedMsg.AddText("channel_id: ")
		quotedMsg.AddEntity(sub.ChannelID, gotgbot.MessageEntity{
			Type: "code",
		})
		quotedMsg.AddText("\n")
		if sub.Regex.Valid && sub.Regex.String != "" {
			quotedMsg.AddText("regex: ")
			quotedMsg.AddEntity(sub.Regex.String, gotgbot.MessageEntity{
				Type: "code",
			})
			quotedMsg.AddText("\n")
		}
		if sub.RegexBan.Valid && sub.RegexBan.String != "" {
			quotedMsg.AddText("regexban: ")
			quotedMsg.AddEntity(sub.RegexBan.String, gotgbot.MessageEntity{
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

func (b *Bot) handleDebugCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}
	if ctx.EffectiveChat.Id != b.m.c.ChatId {
		return nil
	}
	if !ctx.EffectiveSender.IsUser() {
		return nil
	}
	if ctx.EffectiveSender.User.Id != b.m.c.OwnerId {
		return nil
	}
	messageThreadId := ctx.EffectiveMessage.MessageThreadId
	videoId := "dQw4w9WgXcQ"

	b.m.wh.queue[videoId] = Queue{
		threadId:      messageThreadId,
		videoTitle:    "TEST TEST TEST",
		videoUrl:      "https://www.youtube.com/watch?v=" + videoId,
		channelName:   "TEST TEST TEST",
		channelUrl:    "https://www.youtube.com/channel/UCuAXFkgsw1L7xaCfnd5JJOw",
		publishedTime: time.Now().Format(time.RFC3339),
	}

	b.m.wh.debounced(b.m.wh.processAPI)

	return nil
}
