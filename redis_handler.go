package gincache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"math"
	"time"
)

type redisHandler struct {
	cacheStore *redis.Client
	cacheTime  time.Duration
}

// NewRedisHandler do new Redis cache object
func NewRedisHandler(client *redis.Client, cacheTime time.Duration) *redisHandler {
	return &redisHandler{cacheStore: client, cacheTime: cacheTime}
}

func (r *redisHandler) LoadCache(ctx context.Context, key string) string {
	return r.cacheStore.Get(ctx, key).Val()
}

func (r *redisHandler) SetCache(ctx context.Context, key string, data string) {
	r.cacheStore.Set(ctx, key, data, r.cacheTime)
}

func (r *redisHandler) DoCacheEvict(ctx context.Context, keys []string) []string {
	evictKeys := []string{}
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
	return evictKeys
}
