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
	IgnoreThreadId bool
	First          *Message
	Last           []Message
}

func (b *BotHelper) work(chatId int64, multiMsg MultiMessage) {
	if multiMsg.First == nil {
		return
	}
	fallback := func() {
		b.limiter.Wait(b.ctx)
		first := multiMsg.First
		opts := gotgbot.SendMessageOpts{
			Entities:           first.entities,
			LinkPreviewOptions: first.linkPreviewOptions,
		}
		if !multiMsg.IgnoreThreadId {
			opts.MessageThreadId = first.messageThreadId
		}
		msg, err := b.mb.bot.bot.SendMessage(chatId, first.text, &opts)
		if err != nil {
			log.Printf("failed to send message: %+v\n%+v", first, err)
		}
		last := multiMsg.Last
		if last == nil {
			return
		}
		for _, m := range multiMsg.Last {
			b.limiter.Wait(b.ctx)
			opts = gotgbot.SendMessageOpts{
				Entities:           m.entities,
				LinkPreviewOptions: m.linkPreviewOptions,
				ReplyParameters: &gotgbot.ReplyParameters{
					MessageId: msg.MessageId,
				},
			}
			if !multiMsg.IgnoreThreadId {
				opts.MessageThreadId = first.messageThreadId
			}
			if _, err := b.mb.bot.bot.SendMessage(chatId, m.text, &opts); err != nil {
				log.Printf("failed to send message: %+v\n%+v", last, err)
			}
		}
	}
	if multiMsg.First.imageUrl == "" {
		fallback()
		return
	}

	b.limiter.Wait(b.ctx)
	first := multiMsg.First

	inputFile, err := downloadToBuffer(first.imageUrl)
	if err != nil {
		inputFile = gotgbot.InputFileByURL(first.imageUrl)
	}

	photoOpts := gotgbot.SendPhotoOpts{
		Caption:         first.text,
		CaptionEntities: first.entities,
	}
	if !multiMsg.IgnoreThreadId {
		photoOpts.MessageThreadId = first.messageThreadId
	}
	msg, err := b.mb.bot.bot.SendPhoto(chatId, inputFile, &photoOpts)
	if err != nil {
		log.Printf("failed to send message: %+v\n%+v", first, err)
		fallback()
		return
	}
	last := multiMsg.Last
	if last == nil {
		return
	}
	for _, m := range last {
		b.limiter.Wait(b.ctx)
		opts := gotgbot.SendMessageOpts{
			MessageThreadId:    m.messageThreadId,
			Entities:           m.entities,
			LinkPreviewOptions: m.linkPreviewOptions,
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: msg.MessageId,
			},
		}
		if !multiMsg.IgnoreThreadId {
			opts.MessageThreadId = first.messageThreadId
		}
		if _, err := b.mb.bot.bot.SendMessage(chatId, m.text, &opts); err != nil {
			log.Printf("failed to send message: %+v\n%+v", last, err)
		}
	}
}

func (b *BotHelper) telegramWorker(chatId int64, multiMsgs <-chan MultiMessage) {
	for multiMsg := range multiMsgs {
		b.work(chatId, multiMsg)
	}
}
