package cache

import (
	"context"
	"fmt"
	"playstore-api/internal/metrics"
	"time"

	"github.com/redis/go-redis/v9"
)

// How often the cache-size gauge is refreshed in the background.
const cacheSizeInterval = 15 * time.Second

type RedisCache struct {
	client *redis.Client
	stop   chan struct{}
}

func NewRedisCache(ctx context.Context, addr string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	r := &RedisCache{client: client, stop: make(chan struct{})}
	go r.pollCacheSize()
	return r, nil
}

// pollCacheSize keeps the cache-size gauge current without putting an extra
// round trip on every read. It is only ever scraped on the metrics interval,
// so sampling it per request bought nothing.
func (r *RedisCache) pollCacheSize() {
	ticker := time.NewTicker(cacheSizeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), cacheSizeInterval)
			metrics.SetCacheSize(float64(r.client.DBSize(ctx).Val()))
			cancel()
		}
	}
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
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
	close(r.stop)
	return r.client.Close()
}
