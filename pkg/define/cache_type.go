package define

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// CacheHitHook startup on hit hook
type CacheHitHook []func(c *gin.Context, item *CacheItem)

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

type CacheItem struct {
	Header     http.Header `json:"header"`
	HeaderCode int         `json:"headerCode"`
	Body       []byte      `json:"body"`
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
