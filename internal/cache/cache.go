package cache

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/thebenkogan/ufc/internal/events"
)

type EventCacheRepository interface {
	GetEvent(ctx context.Context, id string) (*events.Event, error)
	SetEvent(ctx context.Context, id string, event *events.Event) error
}

type RedisEventCache struct {
	client *redis.Client
}

func NewRedisEventCache(client *redis.Client) *RedisEventCache {
	return &RedisEventCache{
		client: client,
	}
}

func (r *RedisEventCache) GetEvent(ctx context.Context, id string) (*events.Event, error) {
	eventJSON, err := r.client.Get(ctx, id).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var event events.Event
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *RedisEventCache) SetEvent(ctx context.Context, id string, event *events.Event) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := r.client.Set(ctx, id, string(eventJSON), event.FreshTime()).Err(); err != nil {
		return err
	}
	return nil
}
