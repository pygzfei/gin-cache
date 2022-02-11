package startup

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	gincache "github.com/pygzfei/gin-cache/internal"
	rediscache "github.com/pygzfei/gin-cache/internal/drivers/redis"
	"time"
)

// RedisCache NewMemoryCache init memory support
func RedisCache(cacheTime time.Duration, options *redis.Options, onCacheHit ...func(c *gin.Context, cacheValue string)) (*gincache.CacheHandler, error) {
	if cacheTime <= 0 {
		return nil, errors.New("CacheTime greater than 0")
	}
	return gincache.New(rediscache.NewRedisHandler(redis.NewClient(options), cacheTime), onCacheHit...), nil
}
