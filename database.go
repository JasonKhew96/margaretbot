package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/JasonKhew96/margaretbot/models"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/im"
	_ "modernc.org/sqlite"
)

/*
CREATE TABLE IF NOT EXISTS subscription (
	id INTEGER PRIMARY KEY,
	channel_id TEXT UNIQUE NOT NULL,
	channel_title TEXT,
	thread_id INTEGER NOT NULL,
	regex TEXT,
	regex_ban TEXT,
	expired_at TIMESTAMP,
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

CREATE TABLE IF NOT EXISTS forward (
	chat_id INTEGER PRIMARY KEY,
	regex TEXT,
	regex_ban TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS forward_no (
	id INTEGER PRIMARY KEY,
	chat_id INTEGER NOT NULL,
	channel_id TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS forward_no_idx ON forward_no (chat_id, channel_id);

*/

type DbHelper struct {
	oriDb *sql.DB
	db    bob.Executor
	ctx   context.Context
}

func NewDatabaseHelper() (*DbHelper, error) {
	db, err := sql.Open("sqlite", "data.db?cache=shared")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	bobDb := bob.NewDB(db)
	// bobDb := bob.Debug(bob.NewDB(db))

	return &DbHelper{
		oriDb: db,
		db:    bobDb,
		ctx:   context.Background(),
	}, nil
}

func (d *DbHelper) Close() error {
	return d.oriDb.Close()
}

type SubscriptionOpts struct {
	ChannelTitle string
	ExpiredAt    time.Time
	ThreadID     int64
	Regex        string
	RegexBan     string
}

func (d *DbHelper) UpsertSubscription(channelID string, opts *SubscriptionOpts) error {
	s := &models.SubscriptionSetter{
		ChannelID: omit.From(channelID),
	}
	s.UpdatedAt = omit.From(time.Now())
	whitelist := []string{"updated_at"}
	if opts != nil {
		if opts.ChannelTitle != "" {
			s.ChannelTitle = omitnull.From(opts.ChannelTitle)
			whitelist = append(whitelist, "channel_title")
		}
		if !opts.ExpiredAt.IsZero() {
			s.ExpiredAt = omitnull.From(opts.ExpiredAt)
			whitelist = append(whitelist, "expired_at")
		}
		if opts.ThreadID != 0 {
			s.ThreadID = omit.From(opts.ThreadID)
			whitelist = append(whitelist, "thread_id")
		}
		if opts.Regex != "" {
			s.Regex = omitnull.From(opts.Regex)
			whitelist = append(whitelist, "regex")
		}
		if opts.RegexBan != "" {
			s.RegexBan = omitnull.From(opts.RegexBan)
			whitelist = append(whitelist, "regex_ban")
		}
	}
	_, err := models.Subscriptions.Insert(
		s, im.OnConflict("channel_id").DoUpdate(im.SetExcluded(whitelist...)),
	).Exec(d.ctx, d.db)
	return err
}

func (d *DbHelper) DeleteSubscription(channelID string) error {
	_, err := models.Subscriptions.Delete(models.DeleteWhere.Subscriptions.ChannelID.EQ(channelID)).All(d.ctx, d.db)
	return err
}

func (d *DbHelper) GetSubscription(channelID string) (*models.Subscription, error) {
	return models.Subscriptions.Query(models.SelectWhere.Subscriptions.ChannelID.EQ(channelID)).One(d.ctx, d.db)
}

func (d *DbHelper) GetSubscriptionsByThreadID(threadID int64) (models.SubscriptionSlice, error) {
	return models.Subscriptions.Query(models.SelectWhere.Subscriptions.ThreadID.EQ(threadID)).All(d.ctx, d.db)
}

func (d *DbHelper) GetSubscriptions() (models.SubscriptionSlice, error) {
	return models.Subscriptions.Query().All(d.ctx, d.db)
}

func (d *DbHelper) GetExpiringSubscriptions() (models.SubscriptionSlice, error) {
	return models.Subscriptions.Query(models.SelectWhere.Subscriptions.ExpiredAt.LT(time.Now())).All(d.ctx, d.db)
}

func (d *DbHelper) GetTitleNullSubscriptions() (models.SubscriptionSlice, error) {
	return models.Subscriptions.Query(models.SelectWhere.Subscriptions.ChannelTitle.IsNull()).All(d.ctx, d.db)
}

func (d *DbHelper) UpsertCache(videoId string, isScheduled, isPublished bool) error {
	_, err := models.Caches.Insert(&models.CacheSetter{
		VideoID:     omit.From(videoId),
		IsScheduled: omit.From(isScheduled),
		IsPublished: omit.From(isPublished),
		UpdatedAt:   omit.From(time.Now()),
	}, im.OnConflict("video_id").DoUpdate(im.SetExcluded("is_scheduled", "is_published"))).Exec(d.ctx, d.db)
	return err
}

func (d *DbHelper) GetCache(videoId string) (*models.Cache, error) {
	return models.Caches.Query(models.SelectWhere.Caches.VideoID.EQ(videoId)).One(d.ctx, d.db)
}

func (d *DbHelper) DeleteCache() error {
	_, err := models.Caches.Delete(models.DeleteWhere.Caches.CreatedAt.LT(time.Now().Add(-time.Hour*24*7))).All(d.ctx, d.db)
	return err
}

func (d *DbHelper) GetForwards() (models.ForwardSlice, error) {
	return models.Forwards.Query().All(d.ctx, d.db)
}

func (d *DbHelper) GetForward(chatId int64) (*models.Forward, error) {
	return models.Forwards.Query(models.SelectWhere.Forwards.ChatID.EQ(chatId)).One(d.ctx, d.db)
}

func (d *DbHelper) UpsertForward(chatId int64, regex string, banRegex string) error {
	_, err := models.Forwards.Insert(&models.ForwardSetter{
		ChatID:    omit.From(chatId),
		Regex:     omitnull.From(regex),
		RegexBan:  omitnull.From(banRegex),
		UpdatedAt: omit.From(time.Now()),
	}, im.OnConflict("chat_id").DoUpdate(im.SetExcluded("regex", "regex_ban"))).Exec(d.ctx, d.db)
	return err
}

func (d *DbHelper) DeleteForward(chatId int64) error {
	if _, err := models.Forwards.Delete(models.DeleteWhere.Forwards.ChatID.EQ(chatId)).All(d.ctx, d.db); err != nil {
		return err
	}
	_, err := models.ForwardNos.Delete(models.DeleteWhere.ForwardNos.ChatID.EQ(chatId)).All(d.ctx, d.db)
	return err
}

func (d *DbHelper) GetForwardNos() (models.ForwardNoSlice, error) {
	return models.ForwardNos.Query().All(d.ctx, d.db)
}

func (d *DbHelper) IsForwardNo(chatId int64, channelId string) (bool, error) {
	return models.ForwardNos.Query(models.SelectWhere.ForwardNos.ChatID.EQ(chatId), models.SelectWhere.ForwardNos.ChannelID.EQ(channelId)).Exists(d.ctx, d.db)
}

func (d *DbHelper) UpsertForwardNo(chatId int64, channelId string) error {
	_, err := models.ForwardNos.Insert(&models.ForwardNoSetter{
		ChatID:    omit.From(chatId),
		ChannelID: omit.From(channelId),
	}).Exec(d.ctx, d.db)
	return err
}

func (d *DbHelper) DeleteForwardNo(chatId int64, channelId string) error {
	_, err := models.ForwardNos.Delete(models.DeleteWhere.ForwardNos.ChatID.EQ(chatId), models.DeleteWhere.ForwardNos.ChannelID.EQ(channelId)).All(d.ctx, d.db)
	return err
}
