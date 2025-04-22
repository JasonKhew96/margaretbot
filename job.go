package main

import (
	"log"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Message struct {
	text               string
	videoUrl           string
	imageUrl           string
	messageThreadId    int64
	entities           []gotgbot.MessageEntity
	linkPreviewOptions *gotgbot.LinkPreviewOptions
}

func (b *Bot) work(chatId int64, message Message) {
	fallback := func() {
		b.limiter.Wait(b.ctx)
		if _, err := b.m.b.b.SendMessage(chatId, message.text, &gotgbot.SendMessageOpts{
			MessageThreadId:    message.messageThreadId,
			Entities:           message.entities,
			LinkPreviewOptions: message.linkPreviewOptions,
		}); err != nil {
			log.Println("failed to send message:", message, err)
		}
	}
	if message.imageUrl == "" {
		fallback()
		return
	}

	b.limiter.Wait(b.ctx)
	if _, err := b.m.b.b.SendPhoto(chatId, gotgbot.InputFileByURL(message.imageUrl), &gotgbot.SendPhotoOpts{
		MessageThreadId: message.messageThreadId,
		Caption:         message.text,
		CaptionEntities: message.entities,
	}); err != nil {
		log.Println("failed to send message:", message, err)
		fallback()
	}
}

func (b *Bot) telegramWorker(chatId int64, messages <-chan Message) {
	for message := range messages {
		b.work(chatId, message)
	}
}
