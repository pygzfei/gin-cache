package gincache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
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

func (handler *redisHandler) LoadCache(ctx context.Context, key string) string {
	return handler.cacheStore.Get(ctx, key).Val()
}

func (handler *redisHandler) SetCache(ctx context.Context, key string, data string) {
	handler.cacheStore.Set(ctx, key, data, handler.cacheTime)
}

func (handler *redisHandler) DoCacheEvict(ctx context.Context, keys []string) {
	for _, key := range keys {
		var cursor uint64
		deleteKeys, _, err := handler.cacheStore.Scan(ctx, cursor, key, math.MaxUint16).Result()

		if err != nil {
			log.Println(err)
			return
		}

		if len(deleteKeys) > 0 && err == nil {
			handler.cacheStore.Del(ctx, deleteKeys...)
		}
	}
}
