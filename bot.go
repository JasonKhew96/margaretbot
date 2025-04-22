package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

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

	msgChannel chan Message

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
				Timeout: time.Second * 15,
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
		msgChannel: make(chan Message),
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
	dispatcher.AddHandler(NewCommand("re", b.handleReCommand))
	dispatcher.AddHandler(NewCommand("debug", b.handleDebugCommand))

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

func (b *Bot) handleReCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
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
		_, err := ctx.EffectiveMessage.Reply(bot, "Usage: /re <channel_id> <regex>", nil)
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

	_, err = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("subscribed to %s with regex %s", s[0], s[1]), nil)

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
