package startup

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/pygzfei/gin-cache/internal"
	"github.com/pygzfei/gin-cache/internal/drivers/memcache"
	"github.com/pygzfei/gin-cache/pkg/define"
	"time"
)

// MemCache NewMemoryCache init memory support
func MemCache(cacheTime time.Duration, onCacheHit ...func(c *gin.Context, cacheValue *define.CacheItem)) (*internal.CacheHandler, error) {
	if cacheTime <= 0 {
		return nil, errors.New("CacheTime greater than 0")
	}
	return internal.New(memcache.NewMemoryHandler(cacheTime), onCacheHit...), nil
}
