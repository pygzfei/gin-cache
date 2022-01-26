package gin_cache

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

type ICacheAction interface {
	LoadCache(ctx context.Context, key string) string
	SetCache(ctx context.Context, key string, data string)
	DoCacheEvict(ctx context.Context, keys []string)
}

type Cacheable struct {
	CacheName string
	Key       string
}

type CacheEvict struct {
	CacheName []string
	Key       string
}

type ApiCacheable struct {
	Cacheable  []Cacheable
	CacheEvict []CacheEvict
}

type Cache struct {
	CacheHandler ICacheAction
}

func NewRedisCache(cacheTime time.Duration, options *redis.Options) *Cache {
	if options == nil {
		log.Fatalln("Option can not be nil")
	}
	if cacheTime <= 0 {
		log.Fatalln("CacheTime greater than 0")
	}
	redisClient := redis.NewClient(options)
	return &Cache{NewRedisHandler(redisClient, cacheTime)}
}

func NewMemoryCache(cacheTime time.Duration) *Cache {
	return &Cache{NewMemoryHandler(cacheTime)}
}

func (this *Cache) Handler(apiCache ApiCacheable, f gin.HandlerFunc) gin.HandlerFunc {

	return func(c *gin.Context) {

		doCache := len(apiCache.Cacheable) > 0
		doEvict := len(apiCache.CacheEvict) > 0
		var cacheString string
		ctx := context.Background()
		var key string
		if doCache {
			// pointer 指向 writer, 重写 c.writer
			c.Writer = &ResponseBodyWriter{
				body:           bytes.NewBufferString(""),
				ResponseWriter: c.Writer,
			}

			key = this.getCacheKey(apiCache.Cacheable[0], c)
			cacheString = this.loadCache(ctx, key)
		}

		if cacheString == "" {
			f(c)
		} else {
			c.Writer.Header().Set("Content-Type", "application/json; Charset=utf-8")
			c.String(http.StatusOK, cacheString)
			c.Abort()
			return
		}
		if doCache {
			this.setCache(ctx, key, c.Writer.(*ResponseBodyWriter).body.String())
		}
		if doEvict {
			this.doCacheEvict(c, ctx, apiCache.CacheEvict...)
		}
	}
}

func (this *Cache) getCacheKey(cacheable Cacheable, c *gin.Context) string {
	compile, err := regexp.Compile(`#(.*?)#`)
	if err != nil {
		return ""
	}
	submatch := compile.FindAllStringSubmatch(cacheable.Key, -1)
	result := make([]interface{}, len(submatch))
	for i, item := range submatch {
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

func (this *Cache) loadCache(ctx context.Context, key string) string {
	return this.CacheHandler.LoadCache(ctx, key)
}

func (this *Cache) setCache(ctx context.Context, key string, data string) {
	this.CacheHandler.SetCache(ctx, key, data)
}

func (this *Cache) doCacheEvict(c *gin.Context, ctx context.Context, cacheEvicts ...CacheEvict) {
	keys := []string{}
	json := make(map[string]interface{})
	err := c.ShouldBindBodyWith(&json, binding.JSON)
	if err != nil {
		fmt.Println(err)
		return
	}
	compile, _ := regexp.Compile(`#(.*?)#`)
	for _, evict := range cacheEvicts {
		submatch := compile.FindAllStringSubmatch(evict.Key, -1)
		result := make([]interface{}, len(submatch))
		for i, item := range submatch {
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
		this.CacheHandler.DoCacheEvict(ctx, keys)
	}
}
