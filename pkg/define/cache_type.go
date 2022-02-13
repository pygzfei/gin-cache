package define

import (
	"github.com/gin-gonic/gin"
	"time"
)

// CacheHitHook startup on hit hook
type CacheHitHook []func(c *gin.Context, cacheValue string)

// GenKeyFunc startup on hit hook
type GenKeyFunc func(params map[string]interface{}) string

// CacheEvict do Evict
type CacheEvict GenKeyFunc

// Cacheable do caching
type Cacheable struct {
	GenKey     GenKeyFunc
	CacheTime  time.Duration
	OnCacheHit CacheHitHook // 命中缓存钩子 优先级最高, 可覆盖Caching的OnCacheHitting
}

// CacheEvict do Evict
//type CacheEvict struct {
//	CacheName []string
//	Key       string
//}

// Caching mixins Cacheable and CacheEvict
type Caching struct {
	Cacheable []Cacheable
	Evict     []CacheEvict
}
