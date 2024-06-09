package picks

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/thebenkogan/ufc/internal/auth"
)

type EventPicksRepository interface {
	SavePicks(ctx context.Context, user auth.User, eventId string, picks []string) error
}

type RedisEventPicks struct {
	client *redis.Client
}

func NewRedisEventPicks(client *redis.Client) *RedisEventPicks {
	return &RedisEventPicks{
		client: client,
	}
}

func (_ *RedisEventPicks) key(userId string, eventId string) string {
	return fmt.Sprintf("%s#%s#%s", userId, "picks", eventId)
}

func (r *RedisEventPicks) SavePicks(ctx context.Context, user auth.User, eventId string, picks []string) error {
	key := r.key(user.Id, eventId)
	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.RPush(ctx, key, picks)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	return nil
}
