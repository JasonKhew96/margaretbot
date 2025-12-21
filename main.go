package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"github.com/JasonKhew96/margaretbot/websub"
)

type MargaretBot struct {
	config *Config
	db     *DbHelper
	bot    *BotHelper
	yt     *YoutubeHelper
	wh     *WebhookHandler
	ws     *websub.WebSub
}

func main() {
	var err error
	m := &MargaretBot{}

	m.config, err = parseConfig()
	if err != nil {
		fmt.Printf("failed to parse config: %v\n", err)
		return
	}

	m.db, err = NewDatabaseHelper()
	if err != nil {
		fmt.Printf("failed to create database: %v\n", err)
		return
	}
	defer m.db.Close()

	m.yt, err = NewYoutubeHelper()
	if err != nil {
		fmt.Printf("failed to create youtube helper: %v\n", err)
		return
	}
	defer m.yt.Close()

	err = NewBot(m)
	if err != nil {
		fmt.Printf("failed to create bot: %v\n", err)
		return
	}

	NewWebhookHandler(m)

	subscribeLink := "https://pubsubhubbub.appspot.com/subscribe"
	addr := fmt.Sprintf(":%d", m.config.Port)
	m.ws = websub.NewWebSub(subscribeLink, addr, m.wh.handleWebhook, &websub.WebSubOpts{
		Pattern:       "/webhook/{secret}/{thread_id}/{channel_id}",
		ClientTimeout: 10 * time.Second,
	})

	time.AfterFunc(time.Minute, func() {
		loop(m)
	})

	err = m.ws.Run()
	if err != nil {
		fmt.Printf("failed to run webhook server: %v\n", err)
		return
	}
}

func loop(margaret *MargaretBot) {
	if err := margaret.db.DeleteCache(); err != nil {
		fmt.Printf("failed to delete cache: %v\n", err)
		time.AfterFunc(5*time.Minute, func() {
			loop(margaret)
		})
		return
	}

	subs, err := margaret.db.GetExpiringSubscriptions()
	if err != nil {
		fmt.Printf("failed to get subscriptions: %v\n", err)
		time.AfterFunc(5*time.Minute, func() {
			loop(margaret)
		})
		return
	}

	for _, sub := range subs {
		log.Println("renewing subscription for channel", sub.ChannelID)
		time.Sleep(5 * time.Second)
		newSecret := sha256.Sum256([]byte(margaret.config.Secret))
		callbackUrl := fmt.Sprintf("https://%s/webhook/%s/%d/%s", margaret.config.ServerDomain, fmt.Sprintf("%x", newSecret), sub.ThreadID, sub.ChannelID)
		topicUrl := fmt.Sprintf("https://www.youtube.com/xml/feeds/videos.xml?channel_id=%s", sub.ChannelID)

		if err := margaret.ws.Subscribe(websub.ModeSubscribe, callbackUrl, topicUrl, &websub.SubscribeOpts{LeaseSeconds: 604800}); err != nil {
			fmt.Printf("failed to renew subscription: %v\n", err)
			continue
		}
	}

	time.AfterFunc(1*time.Hour, func() {
		loop(margaret)
	})
}
