package internal

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/pygzfei/gin-cache/internal/utils"
	. "github.com/pygzfei/gin-cache/pkg/define"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var bodyBytesKey = "bodyIO"

type Cache interface {
	Load(ctx context.Context, key string) string
	Set(ctx context.Context, key string, data string, timeout time.Duration)
	DoEvict(ctx context.Context, keys []string)
}

type CacheHandler struct {
	Cache      Cache
	OnCacheHit CacheHitHook // 命中缓存钩子 优先级低
}

func (cache *CacheHandler) Load(ctx context.Context, key string) string {
	return cache.Cache.Load(ctx, key)
}

func (cache *CacheHandler) Set(ctx context.Context, key string, data string, timeout time.Duration) {
	cache.Cache.Set(ctx, key, data, timeout)
}

func (cache *CacheHandler) DoEvict(ctx context.Context, keys []string) {
	cache.Cache.DoEvict(ctx, keys)
}

func New(c Cache, onCacheHit ...func(c *gin.Context, cacheValue string)) *CacheHandler {
	return &CacheHandler{c, onCacheHit}
}

// Handler for startup
func (cache *CacheHandler) Handler(caching Caching, next gin.HandlerFunc) gin.HandlerFunc {

	return func(c *gin.Context) {

		doCache := len(caching.Cacheable) > 0
		doEvict := len(caching.Evict) > 0
		ctx := context.Background()

		var key = ""
		var cacheString = ""

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
			if key != "" {
				cacheString = cache.loadCache(ctx, key)
			}
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
				cache.setCache(ctx, key, s, caching.Cacheable[0].CacheTime)
			}
		}

	}
}

func (cache *CacheHandler) getCacheKey(cacheable Cacheable, c *gin.Context) string {
	params := utils.ParameterParser(c)
	return strings.ToLower(cacheable.GenKey(params))
}

func (cache *CacheHandler) loadCache(ctx context.Context, key string) string {
	return cache.Cache.Load(ctx, key)
}

func (cache *CacheHandler) setCache(ctx context.Context, key string, data string, timeout time.Duration) {
	cache.Cache.Set(ctx, key, data, timeout)
}

func (cache *CacheHandler) doCacheEvict(ctx context.Context, c *gin.Context, cacheEvicts ...CacheEvict) {
	keys := make([]string, 0)
	params := utils.ParameterParser(c)
	for _, evict := range cacheEvicts {
		s := evict(params)
		if s != "" {
			keys = append(keys, strings.ToLower(s))
		}
	}

	if len(keys) > 0 {
		cache.Cache.DoEvict(ctx, keys)
	}
}

func (cache *CacheHandler) doCacheHit(ctx *gin.Context, caching Caching, cacheValue string) {

	if len(caching.Cacheable[0].OnCacheHit) > 0 {
		caching.Cacheable[0].OnCacheHit[0](ctx, cacheValue)
		ctx.Abort()
		return
	}

	if len(cache.OnCacheHit) > 0 {
		cache.OnCacheHit[0](ctx, cacheValue)
		ctx.Abort()
		return
	}

	// default hit startup
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
