package main

import (
	"log"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"golang.org/x/time/rate"
)

type Message struct {
	text            string
	caption         *RichMessageOpt
	messageThreadId int64
}

func (b *BotHelper) work(chatId int64, limiter *rate.Limiter, msg Message) {
	if msg.caption != nil {
		sendRichMessageOpts := &gotgbot.SendRichMessageOpts{
			MessageThreadId: msg.messageThreadId,
		}

		fallback := func() {
			limiter.Wait(b.ctx)
			msg.caption.ThumbnailUrl = ""
			inputRichMessage := BuildRichMessage(*msg.caption)

			_, err := b.mb.bot.bot.SendRichMessage(chatId, inputRichMessage, sendRichMessageOpts)
			if err != nil {
				log.Printf("failed to send message: %+v\n%+v", inputRichMessage, err)
			}
		}
		if msg.caption.ThumbnailUrl == "" {
			fallback()
			return
		}

		limiter.Wait(b.ctx)

		inputRichMessage := BuildRichMessage(*msg.caption)
		_, err := b.mb.bot.bot.SendRichMessage(chatId, inputRichMessage, sendRichMessageOpts)
		if err != nil {
			log.Printf("failed to send message: %+v\n%+v", inputRichMessage, err)
			fallback()
			return
		}
		return
	}

	if len(msg.text) <= 0 {
		return
	}

	_, err := b.mb.bot.bot.SendMessage(chatId, msg.text, &gotgbot.SendMessageOpts{
		MessageThreadId: msg.messageThreadId,
	})
	if err != nil {
		log.Printf("failed to send message: %+v\n%+v", msg.text, err)
		return
	}
}

func (b *BotHelper) telegramWorker(chatId int64, msgs chan Message) {
	log.Printf("%d worker started...", chatId)
	limiter := rate.NewLimiter(rate.Every(time.Minute/20), 1)
	for {
		msg, ok := <-msgs
		if !ok {
			log.Printf("%d worker removed...", chatId)
			return
		}
		b.work(chatId, limiter, msg)
	}
}
