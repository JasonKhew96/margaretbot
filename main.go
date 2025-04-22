package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"github.com/JasonKhew96/margaretbot/websub"
)

type MargaretBot struct {
	c  *Config
	db *Database
	b  *Bot
	y  *YoutubeHelper
	wh *WebhookHandler
	ws *websub.WebSub
}

func main() {
	var err error
	m := &MargaretBot{}

	m.c, err = parseConfig()
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

	m.y, err = NewYoutubeHelper()
	if err != nil {
		fmt.Printf("failed to create youtube helper: %v\n", err)
		return
	}
	defer m.y.Close()

	m.b, err = NewBot(m)
	if err != nil {
		fmt.Printf("failed to create bot: %v\n", err)
		return
	}

	m.wh, err = NewWebhookHandler(m)
	if err != nil {
		fmt.Printf("failed to create webhook server: %v\n", err)
		return
	}

	subscribeLink := "https://pubsubhubbub.appspot.com/subscribe"
	addr := fmt.Sprintf(":%d", m.c.Port)
	m.ws = websub.NewWebSub(subscribeLink, addr, m.wh.handleWebhook, &websub.WebSubOpts{
		Pattern:       "/webhook/{secret}/{thread_id}/{channel_id}",
		ClientTimeout: 10 * time.Second,
	})

	time.AfterFunc(5*time.Minute, func() {
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
		time.Sleep(5 * time.Minute)
		return
	}

	subs, err := margaret.db.GetSubscriptions()
	if err != nil {
		fmt.Printf("failed to get subscriptions: %v\n", err)
		time.Sleep(5 * time.Minute)
		return
	}

	for _, sub := range subs {
		if time.Now().Add(24 * time.Hour).After(sub.ExpiredAt) {
			log.Println("renewing subscription for channel", sub.ChannelID)
			time.Sleep(5 * time.Second)
			newSecret := sha256.Sum256([]byte(margaret.c.Secret))
			callbackUrl := fmt.Sprintf("https://%s/webhook/%s/%d/%s", margaret.c.ServerDomain, fmt.Sprintf("%x", newSecret), sub.ThreadID.Int64, sub.ChannelID)
			topicUrl := fmt.Sprintf("https://www.youtube.com/xml/feeds/videos.xml?channel_id=%s", sub.ChannelID)

			if err := margaret.ws.Subscribe(websub.ModeSubscribe, callbackUrl, topicUrl, nil); err != nil {
				fmt.Printf("failed to renew subscription: %v\n", err)
				continue
			}
		}
	}

	time.AfterFunc(1*time.Hour, func() {
		loop(margaret)
	})
}
