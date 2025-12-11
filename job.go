package main

import (
	"log"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"golang.org/x/time/rate"
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

func (b *BotHelper) work(chatId int64, limiter *rate.Limiter, multiMsg MultiMessage) {
	if multiMsg.First == nil {
		return
	}
	fallback := func() {
		limiter.Wait(b.ctx)
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
			limiter.Wait(b.ctx)
			opts = gotgbot.SendMessageOpts{
				Entities:           m.entities,
				LinkPreviewOptions: m.linkPreviewOptions,
				ReplyParameters: &gotgbot.ReplyParameters{
					MessageId: msg.MessageId,
				},
			}
			if !multiMsg.IgnoreThreadId {
				opts.MessageThreadId = m.messageThreadId
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

	limiter.Wait(b.ctx)
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
		limiter.Wait(b.ctx)
		opts := gotgbot.SendMessageOpts{
			Entities:           m.entities,
			LinkPreviewOptions: m.linkPreviewOptions,
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: msg.MessageId,
			},
		}
		if !multiMsg.IgnoreThreadId {
			opts.MessageThreadId = m.messageThreadId
		}
		if _, err := b.mb.bot.bot.SendMessage(chatId, m.text, &opts); err != nil {
			log.Printf("failed to send message: %+v\n%+v", last, err)
		}
	}
}

func (b *BotHelper) telegramWorker(chatId int64, limiter *rate.Limiter, multiMsgs <-chan MultiMessage) {
	for multiMsg := range multiMsgs {
		b.work(chatId, limiter, multiMsg)
	}
}
