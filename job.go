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

type MultiMessage struct {
	First *Message
	Last  *Message
}

func (b *Bot) work(chatId int64, multiMsg MultiMessage) {
	if multiMsg.First == nil {
		return
	}
	fallback := func() {
		b.limiter.Wait(b.ctx)
		first := multiMsg.First
		msg, err := b.m.b.b.SendMessage(chatId, first.text, &gotgbot.SendMessageOpts{
			MessageThreadId:    first.messageThreadId,
			Entities:           first.entities,
			LinkPreviewOptions: first.linkPreviewOptions,
		})
		if err != nil {
			log.Println("failed to send message:", first, err)
		}
		last := multiMsg.Last
		if last == nil {
			return
		}
		b.limiter.Wait(b.ctx)
		if _, err := b.m.b.b.SendMessage(chatId, last.text, &gotgbot.SendMessageOpts{
			MessageThreadId:    last.messageThreadId,
			Entities:           last.entities,
			LinkPreviewOptions: last.linkPreviewOptions,
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: msg.MessageId,
			},
		}); err != nil {
			log.Println("failed to send message:", last, err)
		}
	}
	if multiMsg.First.imageUrl == "" {
		fallback()
		return
	}

	b.limiter.Wait(b.ctx)
	first := multiMsg.First
	msg, err := b.m.b.b.SendPhoto(chatId, gotgbot.InputFileByURL(first.imageUrl), &gotgbot.SendPhotoOpts{
		MessageThreadId: first.messageThreadId,
		Caption:         first.text,
		CaptionEntities: first.entities,
	})
	if err != nil {
		log.Println("failed to send message:", first, err)
		fallback()
		return
	}
	last := multiMsg.Last
	if last == nil {
		return
	}
	b.limiter.Wait(b.ctx)
	if _, err := b.m.b.b.SendMessage(chatId, last.text, &gotgbot.SendMessageOpts{
		MessageThreadId:    last.messageThreadId,
		Entities:           last.entities,
		LinkPreviewOptions: last.linkPreviewOptions,
		ReplyParameters: &gotgbot.ReplyParameters{
			MessageId: msg.MessageId,
		},
	}); err != nil {
		log.Println("failed to send message:", last, err)
	}
}

func (b *Bot) telegramWorker(chatId int64, multiMsgs <-chan MultiMessage) {
	for multiMsg := range multiMsgs {
		b.work(chatId, multiMsg)
	}
}
