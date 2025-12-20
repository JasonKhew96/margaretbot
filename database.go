package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/JasonKhew96/margaretbot/models"
	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	_ "modernc.org/sqlite"
)

/*
CREATE TABLE IF NOT EXISTS subscription (
	id INTEGER PRIMARY KEY,
	channel_id TEXT UNIQUE NOT NULL,
	channel_title TEXT,
	thread_id INTEGER,
	regex TEXT,
	regex_ban TEXT,
	expired_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS subscription_channel_id_idx ON subscription (channel_id, thread_id);

CREATE TABLE IF NOT EXISTS cache (
    video_id TEXT PRIMARY KEY NOT NULL,
	is_scheduled BOOLEAN DEFAULT FALSE NOT NULL,
	is_published BOOLEAN DEFAULT FALSE NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

*/

type DbHelper struct {
	db  *sql.DB
	ctx context.Context
}

func NewDatabaseHelper() (*DbHelper, error) {
	db, err := sql.Open("sqlite", "data.db?cache=shared")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	return &DbHelper{
		db:  db,
		ctx: context.Background(),
	}, nil
}

func (d *DbHelper) Close() error {
	return d.db.Close()
}

type SubscriptionOpts struct {
	ChannelTitle string
	ExpiredAt    time.Time
	ThreadID     int64
	Regex        string
	RegexBan     string
}

func (d *DbHelper) UpsertSubscription(channelID string, opts *SubscriptionOpts) error {
	c := models.Subscription{
		ChannelID: channelID,
	}
	whitelist := []string{}
	if opts != nil {
		if opts.ChannelTitle != "" {
			c.ChannelTitle = null.StringFrom(opts.ChannelTitle)
			whitelist = append(whitelist, "channel_title")
		}
		if !opts.ExpiredAt.IsZero() {
			c.ExpiredAt = opts.ExpiredAt
			whitelist = append(whitelist, "expired_at")
		}
		if opts.ThreadID != 0 {
			c.ThreadID = null.Int64From(opts.ThreadID)
			whitelist = append(whitelist, "thread_id")
		}
		if opts.Regex != "" {
			c.Regex = null.StringFrom(opts.Regex)
			whitelist = append(whitelist, "regex")
		}
		if opts.RegexBan != "" {
			c.RegexBan = null.StringFrom(opts.RegexBan)
			whitelist = append(whitelist, "regex_ban")
		}
	}
	return c.Upsert(d.ctx, d.db, true, []string{"channel_id"}, boil.Whitelist(whitelist...), boil.Infer())
}

func (d *DbHelper) DeleteSubscription(channelID string) error {
	_, err := models.Subscriptions(models.SubscriptionWhere.ChannelID.EQ(channelID)).DeleteAll(d.ctx, d.db)
	return err
}

func (d *DbHelper) GetSubscription(channelID string) (*models.Subscription, error) {
	return models.Subscriptions(models.SubscriptionWhere.ChannelID.EQ(channelID)).One(d.ctx, d.db)
}

func (d *DbHelper) GetSubscriptionsByThreadID(threadID int64) (models.SubscriptionSlice, error) {
	return models.Subscriptions(models.SubscriptionWhere.ThreadID.EQ(null.Int64From(threadID))).All(d.ctx, d.db)
}

func (d *DbHelper) GetSubscriptions() (models.SubscriptionSlice, error) {
	return models.Subscriptions().All(d.ctx, d.db)
}

func (d *DbHelper) GetExpiringSubscriptions() (models.SubscriptionSlice, error) {
	return models.Subscriptions(models.SubscriptionWhere.ExpiredAt.LT(time.Now())).All(d.ctx, d.db)
}

func (d *DbHelper) UpsertCache(videoId string, isScheduled, isPublished bool) error {
	c := models.Cache{
		VideoID:     videoId,
		IsScheduled: isScheduled,
		IsPublished: isPublished,
	}
	return c.Upsert(d.ctx, d.db, true, []string{"video_id"}, boil.Whitelist("is_scheduled", "is_published"), boil.Infer())
}

func (d *DbHelper) GetCache(videoId string) (*models.Cache, error) {
	return models.Caches(models.CacheWhere.VideoID.EQ(videoId)).One(d.ctx, d.db)
}

func (d *DbHelper) DeleteCache() error {
	_, err := models.Caches(models.CacheWhere.CreatedAt.LT(time.Now().Add(-time.Hour*24*7))).DeleteAll(d.ctx, d.db)
	return err
}
