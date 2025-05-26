package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thebenkogan/ufc/internal/model"
)

type EventCacheRepository interface {
	GetEvent(ctx context.Context, id string) (*model.Event, error)
	SetEvent(ctx context.Context, id string, event *model.Event, ttl time.Duration) error
	GetSchedule(ctx context.Context) ([]*model.EventInfo, error)
	SetSchedule(ctx context.Context, events []*model.EventInfo, ttl time.Duration) error
}

type RedisEventCache struct {
	client *redis.Client
}

func NewRedisEventCache(client *redis.Client) *RedisEventCache {
	return &RedisEventCache{
		client: client,
	}
}

func (_ *RedisEventCache) key(id string) string {
	return "events#" + id
}

func (r *RedisEventCache) GetEvent(ctx context.Context, id string) (*model.Event, error) {
	eventJSON, err := r.client.Get(ctx, r.key(id)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var event model.Event
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *RedisEventCache) SetEvent(ctx context.Context, id string, event *model.Event, ttl time.Duration) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := r.client.Set(ctx, r.key(id), string(eventJSON), ttl).Err(); err != nil {
		return err
	}
	return nil
}

func (_ *RedisEventCache) upcomingEventsKey() string {
	return "upcoming_events"
}

func (r *RedisEventCache) GetSchedule(ctx context.Context) ([]*model.EventInfo, error) {
	eventJSON, err := r.client.Get(ctx, r.upcomingEventsKey()).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var events []*model.EventInfo
	if err := json.Unmarshal([]byte(eventJSON), &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *RedisEventCache) SetSchedule(ctx context.Context, events []*model.EventInfo, ttl time.Duration) error {
	eventJSON, err := json.Marshal(events)
	if err != nil {
		return err
	}
	if err := r.client.Set(ctx, r.upcomingEventsKey(), string(eventJSON), ttl).Err(); err != nil {
		return err
	}
	return nil
}
