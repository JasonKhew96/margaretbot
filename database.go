package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/JasonKhew96/margaretbot/models"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	_ "modernc.org/sqlite"
)

/*

CREATE TABLE IF NOT EXISTS subscription (
	id INTEGER PRIMARY KEY,
	channel_id TEXT UNIQUE NOT NULL,
	thread_id INTEGER,
	regex TEXT,
	expired_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS subscription_channel_id_idx ON subscription (channel_id, thread_id);

CREATE TABLE IF NOT EXISTS cache (
    video_id TEXT PRIMARY KEY NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

*/

type Database struct {
	db  *sql.DB
	ctx context.Context
}

func NewDatabaseHelper() (*Database, error) {
	db, err := sql.Open("sqlite", "data.db?cache=shared")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	return &Database{
		db:  db,
		ctx: context.Background(),
	}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

type SubscriptionOpts struct {
	ExpiredAt time.Time
	ThreadID  int64
	Regex     string
}

func (d *Database) UpsertSubscription(channelID string, opts *SubscriptionOpts) error {
	c := models.Subscription{
		ChannelID: channelID,
	}
	whitelist := []string{}
	if opts != nil {
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
	}
	return c.Upsert(d.ctx, d.db, true, []string{"channel_id"}, boil.Whitelist(whitelist...), boil.Infer())
}

func (d *Database) DeleteSubscription(channelID string) error {
	_, err := models.Subscriptions(models.SubscriptionWhere.ChannelID.EQ(channelID)).DeleteAll(d.ctx, d.db)
	return err
}

func (d *Database) GetSubscription(channelID string) (*models.Subscription, error) {
	return models.Subscriptions(models.SubscriptionWhere.ChannelID.EQ(channelID)).One(d.ctx, d.db)
}

func (d *Database) GetSubscriptions() (models.SubscriptionSlice, error) {
	return models.Subscriptions().All(d.ctx, d.db)
}

func (d *Database) UpsertCache(videoId string) error {
	c := models.Cache{
		VideoID: videoId,
	}
	return c.Upsert(d.ctx, d.db, false, []string{"video_id"}, boil.Infer(), boil.Infer())
}

func (d *Database) IsCached(videoId string) (bool, error) {
	count, err := models.Caches(models.CacheWhere.VideoID.EQ(videoId)).Count(d.ctx, d.db)
	return count > 0, err
}

func (d *Database) DeleteCache() error {
	_, err := models.Caches(models.CacheWhere.CreatedAt.LT(time.Now().Add(-time.Hour*24*7))).DeleteAll(d.ctx, d.db)
	return err
}
