package redis

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/pygzfei/gin-cache/pkg/define"
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

func (r *redisCache) Load(ctx context.Context, key string) *define.CacheItem {
	item := new(define.CacheItem)
	if err := json.Unmarshal([]byte(r.cacheStore.Get(ctx, key).Val()), item); err != nil {
		return nil
	}
	return item
}

func (r *redisCache) Set(ctx context.Context, key string, data *define.CacheItem, timeout time.Duration) {
	d, _ := json.Marshal(data)
	if timeout > 0 {
		r.cacheStore.Set(ctx, key, d, timeout)
	} else {
		r.cacheStore.Set(ctx, key, d, r.cacheTime)
	}
}

func (r *redisCache) DoEvict(ctx context.Context, keys []string) {
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
}
