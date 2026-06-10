package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(redisURL string) (*RedisCache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisCache{client: client}, nil
}

func (r *RedisCache) Close() error { return r.client.Close() }

// CheckIdempotency returns true if key was already seen (duplicate publish).
func (r *RedisCache) CheckIdempotency(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if key == "" {
		return false, nil
	}
	ok, err := r.client.SetNX(ctx, "idempotency:"+key, "1", ttl).Result()
	if err != nil {
		return false, err
	}
	return !ok, nil
}

func (r *RedisCache) SetQueueDepth(ctx context.Context, topic string, depth int64) error {
	return r.client.Set(ctx, "queue_depth:"+topic, depth, 30*time.Second).Err()
}

func (r *RedisCache) GetQueueDepth(ctx context.Context, topic string) (int64, error) {
	return r.client.Get(ctx, "queue_depth:"+topic).Int64()
}

func (r *RedisCache) LockWorkflow(ctx context.Context, workflowID string, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, "workflow_lock:"+workflowID, "1", ttl).Result()
}

func (r *RedisCache) UnlockWorkflow(ctx context.Context, workflowID string) error {
	return r.client.Del(ctx, "workflow_lock:"+workflowID).Err()
}
