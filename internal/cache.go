package internal

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/pygzfei/gin-cache/internal/utils"
	. "github.com/pygzfei/gin-cache/pkg/define"
	"io/ioutil"
	"strings"
	"time"
)

var bodyBytesKey = "bodyIO"

type Cache interface {
	Load(ctx context.Context, key string) *CacheItem
	Set(ctx context.Context, key string, data *CacheItem, timeout time.Duration)
	DoEvict(ctx context.Context, keys []string)
}

type CacheHandler struct {
	Cache      Cache
	OnCacheHit CacheHitHook // 命中缓存钩子 优先级低
}

func (cache *CacheHandler) Load(ctx context.Context, key string) *CacheItem {
	return cache.Cache.Load(ctx, key)
}

func (cache *CacheHandler) Set(ctx context.Context, key string, data *CacheItem, timeout time.Duration) {
	cache.Cache.Set(ctx, key, data, timeout)
}

func (cache *CacheHandler) DoEvict(ctx context.Context, keys []string) {
	cache.Cache.DoEvict(ctx, keys)
}

func New(c Cache, onCacheHit ...func(c *gin.Context, cacheValue *CacheItem)) *CacheHandler {
	return &CacheHandler{c, onCacheHit}
}

// Handler for startup
func (cache *CacheHandler) Handler(caching Caching, next gin.HandlerFunc) gin.HandlerFunc {

	return func(c *gin.Context) {
		doCache := len(caching.Cacheable) > 0
		doEvict := len(caching.Evict) > 0
		ctx := context.Background()

		var key = ""
		var cacheItem *CacheItem = nil

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
				cacheItem = cache.loadCache(ctx, key)
			}
		}

		if cacheItem == nil {

			refreshBodyData(c)

			next(c)

			refreshBodyData(c)

		} else {
			cache.doCacheHit(c, caching, cacheItem)
		}
		if doEvict {
			refreshBodyData(c)
			cache.doCacheEvict(ctx, c, caching.Evict...)
		}
		if doCache {
			if cacheItem = cache.loadCache(ctx, key); cacheItem == nil {
				//todo 添加缓存控制头，允许浏览器缓存控制
				//s := c.Writer.(*ResponseBodyWriter).body.String()
				//cloneHeader := c.Writer.Header().Clone()
				//// cache control
				//cloneHeader.Set("X-Cache", "HIT;")
				//cloneHeader.Set("Cache-Control", "private;")
				//cloneHeader.Set("Expires", time.Now().Add(caching.Cacheable[0].CacheTime).Format(time.RFC1123))
				cacheItem = &CacheItem{
					Header:     c.Writer.Header().Clone(),
					HeaderCode: c.Writer.Status(),
					Body:       c.Writer.(*ResponseBodyWriter).body.Bytes(),
				}
				cache.setCache(ctx, key, cacheItem, caching.Cacheable[0].CacheTime)
			}
		}

	}
}

func (cache *CacheHandler) getCacheKey(cacheable Cacheable, c *gin.Context) string {
	params := utils.ParameterParser(c)
	return strings.ToLower(cacheable.GenKey(params))
}

func (cache *CacheHandler) loadCache(ctx context.Context, key string) *CacheItem {
	return cache.Cache.Load(ctx, key)
}

func (cache *CacheHandler) setCache(ctx context.Context, key string, data *CacheItem, timeout time.Duration) {
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

func (cache *CacheHandler) doCacheHit(ctx *gin.Context, caching Caching, item *CacheItem) {
	if len(caching.Cacheable) > 0 && len(caching.Cacheable[0].OnCacheHit) > 0 {
		caching.Cacheable[0].OnCacheHit[0](ctx, item)
		ctx.Abort()
		return
	}

	if len(cache.OnCacheHit) > 0 {
		cache.OnCacheHit[0](ctx, item)
		ctx.Abort()
		return
	}

	// default hit startup
	for k, v := range item.Header {
		ctx.Writer.Header()[k] = v
	}

	_, _ = ctx.Writer.Write(item.Body)
	ctx.Writer.WriteHeader(item.HeaderCode)
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
