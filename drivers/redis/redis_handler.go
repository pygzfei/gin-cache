package redis

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	gincache "github.com/pygzfei/gin-cache"
	"math"
	"time"
)

type redisCache struct {
	cacheStore *redis.Client
	cacheTime  time.Duration
}

// NewRedisHandler do new Redis cache object
func NewRedisHandler(client *redis.Client, cacheTime time.Duration) *redisCache {
	return &redisCache{cacheStore: client, cacheTime: cacheTime}
}

func (r *redisCache) Load(ctx context.Context, key string) string {
	return r.cacheStore.Get(ctx, key).Val()
}

func (r *redisCache) Set(ctx context.Context, key string, data string) {
	r.cacheStore.Set(ctx, key, data, r.cacheTime)
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

// NewCacheHandler NewMemoryCache init memory support
func NewCacheHandler(cacheTime time.Duration, options *redis.Options, onCacheHit ...func(c *gin.Context, cacheValue string)) (*gincache.CacheHandler, error) {
	if cacheTime <= 0 {
		return nil, errors.New("CacheTime greater than 0")
	}
	return gincache.New(NewRedisHandler(redis.NewClient(options), cacheTime), onCacheHit...), nil
}
