package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"math"
	"time"
)

type redisCache struct {
	cacheStore *redis.Client
	cacheTime  time.Duration
}

// NewRedisHandler do new Redis startup object
func NewRedisHandler(client *redis.Client, cacheTime time.Duration) *redisCache {
	return &redisCache{cacheStore: client, cacheTime: cacheTime}
}

func (r *redisCache) Load(ctx context.Context, key string) string {
	return r.cacheStore.Get(ctx, key).Val()
}

func (r *redisCache) Set(ctx context.Context, key string, data string, timeout time.Duration) {
	if timeout > 0 {
		r.cacheStore.Set(ctx, key, data, timeout)
	} else {
		r.cacheStore.Set(ctx, key, data, r.cacheTime)
	}
}

func (r *redisCache) DoEvict(ctx context.Context, keys []string) {
	var evictKeys []string
	for _, key := range keys {
		var cursor uint64
		deleteKeys, _, err := r.cacheStore.Scan(ctx, cursor, key, math.MaxUint16).Result()

		if err == nil {
			evictKeys = append(evictKeys, deleteKeys...)
		}
	}

	if len(evictKeys) > 0 {
		r.cacheStore.Del(ctx, evictKeys...)
	}
}
