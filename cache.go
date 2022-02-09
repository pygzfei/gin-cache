package gincache

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

var bodyBytesKey = "bodyIO"

type Cache interface {
	Load(ctx context.Context, key string) string
	Set(ctx context.Context, key string, data string)
	DoEvict(ctx context.Context, keys []string)
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

type CacheHandler struct {
	Cache      Cache
	OnCacheHit CacheHitHook // 命中缓存钩子 优先级低
}

func (cache *CacheHandler) Load(ctx context.Context, key string) string {
	return cache.Cache.Load(ctx, key)
}

func (cache *CacheHandler) Set(ctx context.Context, key string, data string) {
	cache.Cache.Set(ctx, key, data)
}

func (cache *CacheHandler) DoEvict(ctx context.Context, keys []string) {
	cache.Cache.DoEvict(ctx, keys)
}

func New(c Cache, onCacheHit ...func(c *gin.Context, cacheValue string)) *CacheHandler {
	return &CacheHandler{c, onCacheHit}
}

// Handler for cache
func (cache *CacheHandler) Handler(caching Caching, next gin.HandlerFunc) gin.HandlerFunc {

	return func(c *gin.Context) {

		doCache := len(caching.Cacheable) > 0
		doEvict := len(caching.Evict) > 0
		ctx := context.Background()

		var key string
		var cacheString string

		if c.Request.Body != nil {
			body, err := ioutil.ReadAll(c.Request.Body)
			if err != nil {
				body = []byte("")
			}
			c.Set(bodyBytesKey, body)
			c.Request.Body = ioutil.NopCloser(bytes.NewReader(body))
		}

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

			refreshBodyData(c)

			next(c)

			refreshBodyData(c)

		} else {
			cache.doCacheHit(c, caching, cacheString)
		}
		if doEvict {
			refreshBodyData(c)
			cache.doCacheEvict(ctx, c, caching.Evict...)
		}
		if doCache {
			if cacheString = cache.loadCache(ctx, key); cacheString == "" {
				s := c.Writer.(*ResponseBodyWriter).body.String()
				cache.setCache(ctx, key, s)
			}
		}

	}
}

func (cache *CacheHandler) getCacheKey(cacheable Cacheable, c *gin.Context) string {
	compile, _ := regexp.Compile(`#(.*?)#`)
	subMatch := compile.FindAllStringSubmatch(cacheable.Key, -1)
	result := make([]interface{}, len(subMatch))
	for i, item := range subMatch {
		s := item[1]
		if s != "" {
			if c.Request.Method == http.MethodGet {
				if query, ok := c.GetQuery(s); ok {
					result[i] = query
				} else if strings.Contains(c.FullPath(), ":") {
					if param := c.Param(s); param != "" {
						result[i] = param
					}
				}
			}
			if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
				mapFromData := make(map[string]interface{})
				err := c.ShouldBindBodyWith(&mapFromData, binding.JSON)
				if err == nil {
					result[i] = mapFromData[s]
				} else {
					result[i] = ""
				}
			}
		}
	}
	replaceAllString := compile.ReplaceAllString(strings.ToLower(cacheable.Key), "%v")
	return strings.ToLower(fmt.Sprintf(cacheable.CacheName+":"+replaceAllString, result...))
}

func (cache *CacheHandler) loadCache(ctx context.Context, key string) string {
	return cache.Cache.Load(ctx, key)
}

func (cache *CacheHandler) setCache(ctx context.Context, key string, data string) {
	cache.Cache.Set(ctx, key, data)
}

func (cache *CacheHandler) doCacheEvict(ctx context.Context, c *gin.Context, cacheEvicts ...CacheEvict) {
	keys := make([]string, 0)
	json := make(map[string]interface{})

	_ = c.ShouldBindBodyWith(&json, binding.JSON)

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
		cache.Cache.DoEvict(ctx, keys)
	}
}

func (cache *CacheHandler) doCacheHit(ctx *gin.Context, caching Caching, cacheValue string) {

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

func refreshBodyData(c *gin.Context) {
	if c.Request.Body != nil {
		bodyStr, exists := c.Get(bodyBytesKey)
		if exists {
			c.Request.Body = ioutil.NopCloser(bytes.NewReader(bodyStr.([]byte)))
		}
	}
}
