package cache

import (
	"context"
	"fmt"
	"playstore-api/internal/metrics"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(ctx context.Context, addr string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &RedisCache{client: client}, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	metrics.SetCacheSize(float64(r.client.DBSize(ctx).Val()))

	res, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		metrics.IncCacheMiss()
		return "", err
	}
	if err != nil {
		return "", err
	}

	metrics.IncCacheHit()
	return res, nil
}

func (r *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
