package startup

import (
	"github.com/gin-gonic/gin"
	"github.com/pygzfei/gin-cache/internal"
	"github.com/pygzfei/gin-cache/internal/drivers/memcache"
)

// MemCache NewMemoryCache init memory support
func MemCache(onCacheHit ...func(c *gin.Context, cacheValue string)) (*internal.CacheHandler, error) {
	return internal.New(memcache.NewMemoryHandler(), onCacheHit...), nil
}
