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
	m *MargaretBot

	// config        *Config
	// db            *Database
	// bot           *Bot
	// youtubeHelper *YoutubeHelper

	queue map[string]Queue

	debounced func(f func())
}

func NewWebhookHandler(m *MargaretBot) (*WebhookHandler, error) {
	return &WebhookHandler{
		m:         m,
		queue:     make(map[string]Queue),
		debounced: debounce.New(2500 * time.Millisecond),
	}, nil
}

func (s *WebhookHandler) GetTimeZone(language string) string {
	switch language {
	case "zh-Hant":
		return "Asia/Taipei"
	case "zh-Hans":
		return "Asia/Shanghai"
	case "ja":
		return "Asia/Tokyo"
	case "ko":
		return "Asia/Seoul"
	default:
		return "UTC"
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
	videoList, err := s.m.y.service.Videos.List([]string{"snippet", "contentDetails", "liveStreamingDetails"}).Id(videoIdList...).MaxResults(50).Do()
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
			s.m.b.msgChannel <- Message{
				text:     caption,
				videoUrl: q.videoUrl,
				// imageUrl:        fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", videoId),
				messageThreadId: q.threadId,
				entities:        entities,
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

		if publishedTime != "" {
			parsedTime, err := time.Parse("2006-01-02T15:04:05Z", publishedTime)
			if err != nil {
				log.Printf("failed to parse scheduled start time: %v", err)
			} else if time.Since(parsedTime) > 10*time.Minute {
				log.Printf("%s publishedTime is in the past: %s", video.Id, publishedTime)
				continue
			}
		}
		if scheduledStartTime != "" {
			parsedTime, err := time.Parse("2006-01-02T15:04:05Z", scheduledStartTime)
			if err != nil {
				log.Printf("failed to parse scheduled start time: %v", err)
			}
			if time.Since(parsedTime) > 10*time.Minute {
				log.Printf("%s scheduledStartTime is in the past: %s", video.Id, scheduledStartTime)
				continue
			}
		}

		var thumbnailUrl string
		if video.Snippet.Thumbnails != nil && video.Snippet.Thumbnails.Maxres != nil {
			thumbnailUrl = video.Snippet.Thumbnails.Maxres.Url
		}

		var timezone string
		if video.Snippet.DefaultLanguage != "" {
			timezone = s.GetTimeZone(video.Snippet.DefaultLanguage)
		} else if video.Snippet.DefaultAudioLanguage != "" {
			timezone = s.GetTimeZone(video.Snippet.DefaultAudioLanguage)
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
			s.m.b.msgChannel <- msg
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
			s.m.b.msgChannel <- msg
			s.m.b.msgChannel <- Message{
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
	h.Write([]byte(s.m.c.Secret))
	expectedSecret := h.Sum(nil)
	if secret != fmt.Sprintf("%x", expectedSecret) {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		query := r.URL.Query()
		mode := query.Get("hub.mode")
		// topic := query.Get("hub.topic")
		challenge := query.Get("hub.challenge")
		leaseSeconds := query.Get("hub.lease_seconds")

		if mode == "denied" {
			reason := query.Get("hub.reason")
			s.m.b.b.SendMessage(s.m.c.ChatId, fmt.Sprintf("failed to subscribe to channel %s: %s", channelId, reason), &gotgbot.SendMessageOpts{
				MessageThreadId: threadIdInt,
			})
			w.WriteHeader(http.StatusOK)
			return
		} else if mode == "unsubscribe" {
			err := s.m.db.DeleteSubscription(channelId)
			if err != nil {
				http.Error(w, "failed to delete subscription", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))

			s.m.b.msgChannel <- Message{
				text:            fmt.Sprintf("unsubscribed from channel %s", channelId),
				messageThreadId: threadIdInt,
			}
			return
		}

		leaseSecondsInt, err := strconv.Atoi(leaseSeconds)
		if err != nil {
			http.Error(w, "failed to parse lease_seconds", http.StatusInternalServerError)
			log.Printf("failed to parse lease_seconds: %v", err)
			return
		}

		_, err = s.m.db.GetSubscription(channelId)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "failed to get subscription", http.StatusInternalServerError)
			log.Printf("failed to get subscription: %v", err)
			return
		}

		if errors.Is(err, sql.ErrNoRows) {
			if err := s.m.db.UpsertSubscription(channelId, &SubscriptionOpts{
				ExpiredAt: time.Now().Add(10 * 24 * 60 * time.Minute),
				ThreadID:  threadIdInt,
			}); err != nil {
				http.Error(w, "failed to upsert subscription", http.StatusInternalServerError)
				log.Printf("failed to upsert subscription: %v", err)
				return
			}
			s.m.b.msgChannel <- Message{
				text:            fmt.Sprintf("subscribed to channel %s", channelId),
				messageThreadId: threadIdInt,
			}
		}

		expiredAt := time.Now().Add(time.Duration(leaseSecondsInt) * time.Second)
		err = s.m.db.UpsertSubscription(channelId, &SubscriptionOpts{
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
	} else if r.Method == http.MethodPost {
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

		channel, err := s.m.db.GetSubscription(channelId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("channel not found: %s", channelId)
				return
			}
			log.Printf("failed to get channel: %v", err)
			return
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

		isCached, err := s.m.db.IsCached(videoId)
		if err != nil {
			log.Printf("get isCached failed: %v", err)
			return
		}
		if isCached {
			return
		}
		if err := s.m.db.UpsertCache(videoId); err != nil {
			log.Printf("unable to insert cache: %v", err)
			return
		}

		isShort, err := s.m.y.IsShort(videoId)
		if err != nil {
			log.Printf("failed to check if video is short: %v", err)
		}
		if isShort {
			log.Printf("video is a short: %s", videoId)
			return
		}

		videoTitle := feed.Entry.Title
		videoUrl := feed.Entry.Link.Href

		if channel.RegexBan.Valid && channel.RegexBan.String != "" {
			re, err := regexp.Compile(channel.RegexBan.String)
			if err != nil {
				log.Printf("failed to compile regex: %v", err)
				return
			}
			if re.MatchString(videoTitle) {
				log.Printf("feed title matches regex: %s", videoTitle)
				return
			}
		}

		if channel.Regex.Valid && channel.Regex.String != "" {
			re, err := regexp.Compile(channel.Regex.String)
			if err != nil {
				log.Printf("failed to compile regex: %v", err)
				return
			}
			if !re.MatchString(videoTitle) {
				log.Printf("feed title does not match regex: %s", videoTitle)
				return
			}
		}

		channelName := feed.Entry.Author.Name
		channelUri := feed.Entry.Author.URI

		// thumbnailUrl := fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", videoId)

		log.Println("new video:", videoId, videoTitle)

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
