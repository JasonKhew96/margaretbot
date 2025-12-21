package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JasonKhew96/margaretbot/entity"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/bep/debounce"
)

type Queue struct {
	threadId      int64
	videoTitle    string
	videoUrl      string
	channelName   string
	channelUrl    string
	publishedTime string
}

type WebhookHandler struct {
	mb *MargaretBot

	// config        *Config
	// db            *Database
	// bot           *Bot
	// youtubeHelper *YoutubeHelper

	queue map[string]Queue

	debounced func(f func())
}

func NewWebhookHandler(mb *MargaretBot) {
	mb.wh = &WebhookHandler{
		mb:        mb,
		queue:     make(map[string]Queue),
		debounced: debounce.New(1500 * time.Millisecond),
	}
}

func (s *WebhookHandler) processAPI() {
	var videoIdList []string
	newQueue := make(map[string]Queue, 0)
	for videoId, q := range s.queue {
		newQueue[videoId] = q
		videoIdList = append(videoIdList, videoId)
		delete(s.queue, videoId)

		if len(videoIdList) >= 50 {
			break
		}
	}

	videoList, err := s.mb.yt.service.Videos.List([]string{"snippet", "contentDetails", "liveStreamingDetails"}).Id(videoIdList...).Do()
	if err != nil {
		log.Printf("failed to get video list: %v", err)

		for _, q := range newQueue {
			caption, entities := BuildCaption(&Caption{
				VideoTitle:    q.videoTitle,
				VideoUrl:      q.videoUrl,
				ChannelName:   q.channelName,
				ChannelUrl:    q.channelUrl,
				PublishedTime: q.publishedTime,
			})
			s.mb.bot.msgChannel <- MultiMessage{
				First: &Message{
					text:     caption,
					videoUrl: q.videoUrl,
					// imageUrl:        fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", videoId),
					messageThreadId: q.threadId,
					entities:        entities,
				},
			}
		}
		return
	}
	for _, video := range videoList.Items {
		videoId := video.Id

		q, ok := newQueue[videoId]
		if !ok {
			continue
		}

		videoTitle := video.Snippet.Title
		videoUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoId)
		// thumbnailUrl := fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", videoId)
		videoDescription := video.Snippet.Description
		if len(videoDescription) > 4096 {
			videoDescription = videoDescription[:4095] + "…"
		}
		var allowedRegion string
		var blockedRegion string
		if video.ContentDetails.RegionRestriction != nil {
			if len(video.ContentDetails.RegionRestriction.Blocked) >= 249 {
				log.Printf("%s is globally restricted: %s", video.Id, video.Snippet.Title)
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

		cache, _ := s.mb.db.GetCache(videoId)

		if scheduledStartTime != "" {
			if err := s.mb.db.UpsertCache(videoId, true, false); err != nil {
				log.Printf("failed to update cache: %v", err)
			}
			parsedTime, err := time.Parse("2006-01-02T15:04:05Z", scheduledStartTime)
			if err != nil {
				log.Printf("failed to parse scheduled start time: %v", err)
			}
			if time.Since(parsedTime) > 24*time.Hour*3 {
				log.Printf("%s scheduledStartTime is in the past 3 days %s: %s", video.Id, scheduledStartTime, video.Snippet.Title)
				continue
			}
			if cache != nil && time.Now().Before(parsedTime) && cache.IsScheduled {
				log.Printf("skip scheduled %s: %s", videoId, videoTitle)
				continue
			}
		}
		if publishedTime != "" {
			if cache != nil && cache.IsPublished {
				log.Printf("skip published %s: %s", videoId, videoTitle)
				continue
			}
			if err := s.mb.db.UpsertCache(videoId, true, true); err != nil {
				log.Printf("failed to update cache: %v", err)
			}
			parsedTime, err := time.Parse("2006-01-02T15:04:05Z", publishedTime)
			if err != nil {
				log.Printf("failed to parse published time: %v", err)
			} else if time.Since(parsedTime) > 24*time.Hour*3 {
				log.Printf("%s publishedTime is in the past 3 days %s: %s", video.Id, publishedTime, video.Snippet.Title)
				continue
			}
		}

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

		isForward := false
		if s.mb.config.ForwardChatId != 0 {
			if !slices.Contains(s.mb.config.NoForwardChannelIds, video.Snippet.ChannelId) {
				re, err := regexp.Compile(s.mb.config.ForwardRegex)
				if err == nil {
					if re.MatchString(videoTitle) {
						isForward = true
					} else {
						log.Printf("%s does not match forward regex: %s", video.Id, video.Snippet.Title)
					}
				} else {
					log.Printf("failed to compile regex: %v", err)
				}
			} else {
				log.Printf("%s %s is from no forward channel: %s", video.Snippet.ChannelId, video.Id, video.Snippet.Title)
			}
		}

		c := &Caption{
			VideoTitle:         videoTitle,
			VideoUrl:           videoUrl,
			VideoDescription:   videoDescription,
			VideoDuration:      video.ContentDetails.Duration,
			ChannelName:        channelName,
			ChannelUrl:         fmt.Sprintf("https://www.youtube.com/channel/%s", video.Snippet.ChannelId),
			AllowedRegion:      allowedRegion,
			BlockedRegion:      blockedRegion,
			ScheduledStartTime: scheduledStartTime,
			PublishedTime:      publishedTime,
			TimeZone:           timezone,
		}
		caption, entities := BuildCaption(c)

		if len(caption) < 1024 {
			msg := Message{
				text:            caption,
				videoUrl:        videoUrl,
				messageThreadId: q.threadId,
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
			mm := MultiMessage{
				First: &msg,
			}
			s.mb.bot.msgChannel <- mm
			if isForward {
				mm.IgnoreThreadId = true
				s.mb.bot.msgForward <- mm
			}
			continue
		}

		c.VideoDescription = ""
		caption, entities = BuildCaption(c)
		if len(caption) < 1024 {
			msg := Message{
				text:            caption,
				videoUrl:        videoUrl,
				messageThreadId: q.threadId,
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
			mm := MultiMessage{
				First: &msg,
				Last: []Message{
					{
						text:            videoDescription,
						messageThreadId: q.threadId,
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
			s.mb.bot.msgChannel <- mm
			if isForward {
				mm.IgnoreThreadId = true
				s.mb.bot.msgForward <- mm
			}
			continue
		}

		c.AllowedRegion = ""
		c.BlockedRegion = ""
		caption, entities = BuildCaption(c)
		if len(caption) < 1024 {
			msg := Message{
				text:            caption,
				videoUrl:        videoUrl,
				messageThreadId: q.threadId,
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
			regionMsg, regionEntities := BuildCaption(&Caption{
				AllowedRegion: allowedRegion,
				BlockedRegion: blockedRegion,
			})
			mm := MultiMessage{
				First: &msg,
				Last: []Message{
					{
						text:            regionMsg,
						messageThreadId: q.threadId,
						entities:        regionEntities,
						linkPreviewOptions: &gotgbot.LinkPreviewOptions{
							IsDisabled: true,
						},
					},
					{
						text:            videoDescription,
						messageThreadId: q.threadId,
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
			s.mb.bot.msgChannel <- mm
			if isForward {
				mm.IgnoreThreadId = true
				s.mb.bot.msgForward <- mm
			}
		}
	}
}

func (s *WebhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	secret := r.PathValue("secret")
	threadId := r.PathValue("thread_id")
	channelId := r.PathValue("channel_id")
	if secret == "" || threadId == "" || channelId == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	threadIdInt, err := strconv.ParseInt(threadId, 10, 64)
	if err != nil {
		http.Error(w, "failed to parse thread_id", http.StatusInternalServerError)
		return
	}

	h := sha256.New()
	h.Write([]byte(s.mb.config.Secret))
	expectedSecret := h.Sum(nil)
	if secret != fmt.Sprintf("%x", expectedSecret) {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query()
		mode := query.Get("hub.mode")
		// topic := query.Get("hub.topic")
		challenge := query.Get("hub.challenge")
		leaseSeconds := query.Get("hub.lease_seconds")

		log.Printf("%s %s", r.URL.Path, query)

		switch mode {
		case "denied":
			reason := query.Get("hub.reason")
			s.mb.bot.bot.SendMessage(s.mb.config.ChatId, fmt.Sprintf("failed to subscribe to channel %s: %s", channelId, reason), &gotgbot.SendMessageOpts{
				MessageThreadId: threadIdInt,
			})
			w.WriteHeader(http.StatusOK)
			return
		case "unsubscribe":
			err := s.mb.db.DeleteSubscription(channelId)
			if err != nil {
				http.Error(w, "failed to delete subscription", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))

			s.mb.bot.msgChannel <- MultiMessage{
				First: &Message{
					text:            fmt.Sprintf("unsubscribed from channel %s", channelId),
					messageThreadId: threadIdInt,
				},
			}
			return
		}

		leaseSecondsInt, err := strconv.Atoi(leaseSeconds)
		if err != nil {
			http.Error(w, "failed to parse lease_seconds", http.StatusInternalServerError)
			log.Printf("failed to parse lease_seconds: %v", err)
			return
		}
		expiredAt := time.Now().Add(time.Duration(leaseSecondsInt) * time.Second)

		_, err = s.mb.db.GetSubscription(channelId)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "failed to get subscription", http.StatusInternalServerError)
			log.Printf("failed to get subscription: %v", err)
			return
		}

		if errors.Is(err, sql.ErrNoRows) {
			if err := s.mb.db.UpsertSubscription(channelId, &SubscriptionOpts{
				ExpiredAt: expiredAt,
				ThreadID:  threadIdInt,
			}); err != nil {
				http.Error(w, "failed to upsert subscription", http.StatusInternalServerError)
				log.Printf("failed to upsert subscription: %v", err)
				return
			}
			s.mb.bot.msgChannel <- MultiMessage{
				First: &Message{
					text:            fmt.Sprintf("subscribed to channel %s", channelId),
					messageThreadId: threadIdInt,
				},
			}
		}

		log.Println("renewed subscription for channel", channelId)
		err = s.mb.db.UpsertSubscription(channelId, &SubscriptionOpts{
			ExpiredAt: expiredAt,
			ThreadID:  threadIdInt,
		})
		if err != nil {
			http.Error(w, "failed to upsert subscription", http.StatusInternalServerError)
			log.Printf("failed to upsert subscription: %v", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
	case http.MethodPost:
		w.WriteHeader(http.StatusOK)

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			log.Printf("content type is required")
			return
		}
		if contentType != "application/atom+xml" {
			return
		}

		channelId := r.PathValue("channel_id")
		if channelId == "" {
			log.Printf("channel_id is missing")
			return
		}

		feed := &entity.Feed{}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("failed to read body: %v", err)
			return
		}
		defer r.Body.Close()

		// fmt.Println("--------------------------------")
		// fmt.Println(string(body))
		// fmt.Println("--------------------------------")

		if err := xml.Unmarshal(body, feed); err != nil {
			log.Printf("failed to unmarshal xml feed: %v", err)
			return
		}

		channel, err := s.mb.db.GetSubscription(channelId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("channel not found: %s", channelId)
				return
			}
			log.Printf("failed to get channel: %v", err)
			return
		}

		subs, err := s.mb.db.GetSubscriptionsByThreadID(threadIdInt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("subscriptions not found: %s", channelId)
				return
			}
			log.Printf("failed to get subscriptions: %v", err)
			return
		}

		channelName := feed.Entry.Author.Name
		channelUri := feed.Entry.Author.URI

		if channel.ChannelTitle.GetOrZero() != channelName {
			err = s.mb.db.UpsertSubscription(channelId, &SubscriptionOpts{
				ChannelTitle: channelName,
			})
			if err != nil {
				log.Printf("failed to upsert subscription: %v", err)
			} else if len(subs) == 1 {
				log.Printf("updating channel title %s: %s", channelId, channelName)
				tid, ok := channel.ThreadID.Get()
				if ok {
					if _, err := s.mb.bot.bot.EditForumTopic(s.mb.config.ChatId, tid, &gotgbot.EditForumTopicOpts{
						Name: channelName,
					}); err != nil {
						log.Printf("failed to EditForumTopic: %v", err)
					}
				}
			}
		}

		if feed.DeletedEntry.Link.Href != "" {
			// ignore deleted entry
			log.Printf("deleted video: %s", feed.DeletedEntry.Link.Href)
			return
		}

		videoId := feed.Entry.VideoId
		if videoId == "" {
			log.Println("videoId is missing...")
			return
		}

		videoTitle := feed.Entry.Title
		videoUrl := feed.Entry.Link.Href

		cache, err := s.mb.db.GetCache(videoId)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("get cache failed: %v", err)
			return
		}
		if cache != nil && cache.IsScheduled && cache.IsPublished {
			log.Printf("already scheduled/published %s: %s", videoId, videoTitle)
			return
		}
		if cache == nil {
			if err := s.mb.db.UpsertCache(videoId, false, false); err != nil {
				log.Printf("unable to insert cache: %v", err)
				return
			}
		}

		isShort, err := s.mb.yt.IsShort(videoId, videoTitle)
		if err != nil {
			log.Printf("failed to check if video is short: %v", err)
		}
		if isShort {
			log.Printf("video is a short %s: %s", videoId, videoTitle)
			return
		}

		rb, ok := channel.RegexBan.Get()
		if ok && rb != "" {
			re, err := regexp.Compile(rb)
			if err != nil {
				log.Printf("failed to compile regex: %v", err)
				return
			}
			if re.MatchString(videoTitle) {
				log.Printf("feed title matches regex %s: %s", videoId, videoTitle)
				return
			}
		}

		r, ok := channel.Regex.Get()
		if ok && r != "" {
			re, err := regexp.Compile(r)
			if err != nil {
				log.Printf("failed to compile regex: %v", err)
				return
			}
			if !re.MatchString(videoTitle) {
				log.Printf("feed title does not match regex %s: %s", videoId, videoTitle)
				return
			}
		}

		// thumbnailUrl := fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", videoId)

		log.Printf("new video %s: %s", videoId, videoTitle)

		s.queue[videoId] = Queue{
			threadId:      threadIdInt,
			videoTitle:    videoTitle,
			videoUrl:      videoUrl,
			channelName:   channelName,
			channelUrl:    channelUri,
			publishedTime: feed.Entry.Published,
		}

		s.debounced(s.processAPI)

		// text := fmt.Sprintf("[%s](%s) \\[[封面](%s)\\]\n\n[%s](%s)", EscapeMarkdownV2(videoTitle), videoUrl, thumbnailUrl, EscapeMarkdownV2(channelName), channelUri)

		// message := Message{
		// 	text:            text,
		// 	videoUrl:        videoUrl,
		// 	imageUrl:        thumbnailUrl,
		// 	messageThreadId: threadIdInt,
		// 	parseMode:       "MarkdownV2",
		// }
		// s.m.bot.msgChannel <- message
	}

}
