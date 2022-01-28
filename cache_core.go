package gincache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-redis/redis/v8"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ICacheAction : memoryHandler and redisHandler implement
type ICacheAction interface {
	LoadCache(ctx context.Context, key string) string
	SetCache(ctx context.Context, key string, data string)
	DoCacheEvict(ctx context.Context, keys []string)
}

// CacheHitHook cache on hit hook
type CacheHitHook []func(c *gin.Context, cacheValue string)

// Cacheable do caching
type Cacheable struct {
	CacheName  string
	Key        string
	onCacheHit CacheHitHook // 命中缓存钩子 优先级最高, 可覆盖Caching的OnCacheHitting
}

// CacheEvict do Evict
type CacheEvict struct {
	CacheName []string
	Key       string
}

// Caching mixins Cacheable and CacheEvict
type Caching struct {
	Cacheable []Cacheable
	Evict     []CacheEvict
}

// Cache handler
type Cache struct {
	CacheHandler ICacheAction
	OnCacheHit   CacheHitHook // 命中缓存钩子 优先级低
}

// NewRedisCache init redis support
func NewRedisCache(cacheTime time.Duration, options *redis.Options, onCacheHit ...func(c *gin.Context, cacheValue string)) (*Cache, error) {
	if options == nil || cacheTime <= 0 {
		return nil, errors.New("option can not be nil or CacheTime greater than 0")
	}
	return &Cache{NewRedisHandler(redis.NewClient(options), cacheTime), onCacheHit}, nil
}

// NewMemoryCache init memory support
func NewMemoryCache(cacheTime time.Duration, onCacheHit ...func(c *gin.Context, cacheValue string)) (*Cache, error) {
	if cacheTime <= 0 {
		return nil, errors.New("CacheTime greater than 0")
	}
	return &Cache{NewMemoryHandler(cacheTime), onCacheHit}, nil
}

// Handler for cache
func (cache *Cache) Handler(caching Caching, next gin.HandlerFunc) gin.HandlerFunc {

	return func(c *gin.Context) {

		doCache := len(caching.Cacheable) > 0
		doEvict := len(caching.Evict) > 0
		ctx := context.Background()

		var key string
		var cacheString string

		if doCache {
			// pointer 指向 writer, 重写 c.writer
			c.Writer = &ResponseBodyWriter{
				body:           bytes.NewBufferString(""),
				ResponseWriter: c.Writer,
			}

			key = cache.getCacheKey(caching.Cacheable[0], c)
			cacheString = cache.loadCache(ctx, key)
		}

		if cacheString == "" {
			next(c)
		} else {
			cache.doCacheHit(c, caching, cacheString)
		}
		if doCache && cacheString == "" {
			s := c.Writer.(*ResponseBodyWriter).body.String()
			cache.setCache(ctx, key, s)
		}
		if doEvict {
			cache.doCacheEvict(ctx, c, caching.Evict...)
		}
	}
}

func (cache *Cache) getCacheKey(cacheable Cacheable, c *gin.Context) string {
	compile, _ := regexp.Compile(`#(.*?)#`)
	subMatch := compile.FindAllStringSubmatch(cacheable.Key, -1)
	result := make([]interface{}, len(subMatch))
	for i, item := range subMatch {
		s := item[1]
		if s != "" {
			if query, b := c.GetQuery(s); b {
				result[i] = query
			}
		}
	}
	replaceAllString := compile.ReplaceAllString(strings.ToLower(cacheable.Key), "%v")
	return strings.ToLower(fmt.Sprintf(cacheable.CacheName+":"+replaceAllString, result...))
}

func (cache *Cache) loadCache(ctx context.Context, key string) string {
	return cache.CacheHandler.LoadCache(ctx, key)
}

func (cache *Cache) setCache(ctx context.Context, key string, data string) {
	cache.CacheHandler.SetCache(ctx, key, data)
}

func (cache *Cache) doCacheEvict(ctx context.Context, c *gin.Context, cacheEvicts ...CacheEvict) {
	keys := []string{}
	json := make(map[string]interface{})
	c.ShouldBindBodyWith(&json, binding.JSON)

	compile, _ := regexp.Compile(`#(.*?)#`)
	for _, evict := range cacheEvicts {
		subMatch := compile.FindAllStringSubmatch(evict.Key, -1)
		result := make([]interface{}, len(subMatch))
		for i, item := range subMatch {
			s := item[1]
			if s != "" {
				param := json[s]
				if param == nil {
					break
				}
				result[i] = param
			}
		}

		for _, prefix := range evict.CacheName {
			replaceAllString := compile.ReplaceAllString(strings.ToLower(evict.Key), "%v")
			keys = append(keys, strings.ToLower(fmt.Sprintf(prefix+":"+replaceAllString, result...)))
		}
	}
	if len(keys) > 0 {
		cache.CacheHandler.DoCacheEvict(ctx, keys)
	}
}

func (cache *Cache) doCacheHit(ctx *gin.Context, caching Caching, cacheValue string) {

	if len(caching.Cacheable[0].onCacheHit) > 0 {
		caching.Cacheable[0].onCacheHit[0](ctx, cacheValue)
		ctx.Abort()
		return
	}

	if len(cache.OnCacheHit) > 0 {
		cache.OnCacheHit[0](ctx, cacheValue)
		ctx.Abort()
		return
	}

	// default hit cache
	ctx.Writer.Header().Set("Content-Type", "application/json; Charset=utf-8")
	ctx.String(http.StatusOK, cacheValue)
	ctx.Abort()
}
