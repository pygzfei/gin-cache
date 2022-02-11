package startup

import (
	"errors"
	"github.com/gin-gonic/gin"
	gincache "github.com/pygzfei/gin-cache/internal"
	"github.com/pygzfei/gin-cache/internal/drivers/memcache"
	"time"
)

// MemCache NewMemoryCache init memory support
func MemCache(cacheTime time.Duration, onCacheHit ...func(c *gin.Context, cacheValue string)) (*gincache.CacheHandler, error) {
	if cacheTime <= 0 {
		return nil, errors.New("CacheTime greater than 0")
	}
	return gincache.New(memcache.NewMemoryHandler(cacheTime), onCacheHit...), nil
}
