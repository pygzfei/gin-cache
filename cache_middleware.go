package gincache

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-redis/redis/v8"
	"log"
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

// Cacheable do caching
type Cacheable struct {
	CacheName string
	Key       string
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
}

// NewRedisCache init redis support
func NewRedisCache(cacheTime time.Duration, options *redis.Options) *Cache {
	if options == nil {
		log.Fatalln("Option can not be nil")
	}
	if cacheTime <= 0 {
		log.Fatalln("CacheTime greater than 0")
	}
	return &Cache{NewRedisHandler(redis.NewClient(options), cacheTime)}
}

// NewMemoryCache init memory support
func NewMemoryCache(cacheTime time.Duration) *Cache {
	return &Cache{NewMemoryHandler(cacheTime)}
}

// Handler for cache
func (cache *Cache) Handler(apiCache Caching, next gin.HandlerFunc) gin.HandlerFunc {

	return func(c *gin.Context) {

		doCache := len(apiCache.Cacheable) > 0
		doEvict := len(apiCache.Evict) > 0
		ctx := context.Background()

		var key string
		var cacheString string

		if doCache {
			// pointer 指向 writer, 重写 c.writer
			c.Writer = &ResponseBodyWriter{
				body:           bytes.NewBufferString(""),
				ResponseWriter: c.Writer,
			}

			key = cache.getCacheKey(apiCache.Cacheable[0], c)
			cacheString = cache.loadCache(ctx, key)
		}

		if cacheString == "" {
			next(c)
		} else {
			c.Writer.Header().Set("Content-Type", "application/json; Charset=utf-8")
			c.String(http.StatusOK, cacheString)
			c.Abort()
			return
		}
		if doCache {
			cache.setCache(ctx, key, c.Writer.(*ResponseBodyWriter).body.String())
		}
		if doEvict {
			cache.doCacheEvict(ctx, c, apiCache.Evict...)
		}
	}
}

func (cache *Cache) getCacheKey(cacheable Cacheable, c *gin.Context) string {
	compile, err := regexp.Compile(`#(.*?)#`)
	if err != nil {
		return ""
	}
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
	err := c.ShouldBindBodyWith(&json, binding.JSON)
	if err != nil {
		return
	}
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
