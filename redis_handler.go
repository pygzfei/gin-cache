package gin_cache

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

func NewRedisHandler(client *redis.Client, cacheTime time.Duration) *redisHandler {
	return &redisHandler{cacheStore: client, cacheTime: cacheTime}
}

func (this *redisHandler) LoadCache(ctx context.Context, key string) string {
	return this.cacheStore.Get(ctx, key).Val()
}

func (this *redisHandler) SetCache(ctx context.Context, key string, data string) {
	this.cacheStore.Set(ctx, key, data, this.cacheTime)
}

func (this *redisHandler) DoCacheEvict(ctx context.Context, keys []string) {
	for _, key := range keys {
		var cursor uint64
		deleteKeys, _, err := this.cacheStore.Scan(ctx, cursor, key, math.MaxUint16).Result()

		if err != nil {
			log.Println(err)
			return
		}

		if len(deleteKeys) > 0 && err == nil {
			this.cacheStore.Del(ctx, deleteKeys...)
		}
	}
}
