package gincache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type RunFor uint8

const (
	MemoryCache RunFor = 0
	RedisCache  RunFor = 1
)

func givingCacheOfHttpServer(timeout time.Duration, runFor RunFor, onHit ...func(c *gin.Context, cacheValue string)) (*gin.Engine, *Cache) {
	var cache *Cache

	if runFor == MemoryCache {
		cache, _ = NewMemoryCache(timeout, onHit...)
	} else if runFor == RedisCache {
		cache, _ = NewRedisCache(
			timeout,
			&redis.Options{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			},
			onHit...,
		)
	}

	gin.ForceConsoleColor()
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	r.GET("/ping", cache.Handler(
		Caching{
			Cacheable: []Cacheable{
				{CacheName: "anson", Key: `userId:#id# hash:#hash#`},
			},
		},
		func(c *gin.Context) {
			id, _ := c.GetQuery("id")
			hash, _ := c.GetQuery("hash")
			c.JSON(200, gin.H{
				"id":   id,
				"hash": hash,
			})
		},
	))

	r.POST("/ping", cache.Handler(
		Caching{
			Evict: []CacheEvict{
				{CacheName: []string{"anson"}, Key: "userId:#id#*"},
			},
		},
		func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "delete cache",
			})
		},
	))

	return r, cache
}

func Test_Cache_CanStore(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for _, item := range map[string]struct {
			Id   string
			Hash string
		}{
			"key1": {
				Id: "1", Hash: "anson",
			},
			"key2": {
				Id: "2", Hash: "anson",
			},
		} {
			t.Run(fmt.Sprintf(`key: %s  %s`, item.Id, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor, func(c *gin.Context, cacheValue string) {
					assert.True(t, len(cacheValue) > 0)
				})

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ping?id=%s&hash=%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)

				cacheKey := fmt.Sprintf("anson:userid:%s hash:%s", item.Id, item.Hash)
				loadCache := cache.loadCache(context.Background(), cacheKey)
				assert.Equal(t, 200, w.Code)

				sprintf := fmt.Sprintf(`{"id": "%s", "hash": "%s"}`, item.Id, item.Hash)
				equalJSON, err := AreEqualJSON(sprintf, loadCache)
				assert.Equal(t, equalJSON && err == nil, true)

				//test for cache hit hook
				w = httptest.NewRecorder()
				req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/ping?id=%s&hash=%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)
			})
		}
	}

}

func Test_Cache_CanStore_Hit_Hook(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for _, item := range map[string]struct {
			Id   string
			Hash string
		}{
			"key1": {
				Id: "1", Hash: "anson",
			},
			"key2": {
				Id: "2", Hash: "anson",
			},
		} {
			t.Run(fmt.Sprintf(`key: %s  %s`, item.Id, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor, func(c *gin.Context, cacheValue string) {
					// 这里不会被触发
					err := errors.New("Should not trigger this func")
					assert.NoError(t, err)
				})

				r.GET("/pings", cache.Handler(
					Caching{
						Cacheable: []Cacheable{
							{CacheName: "anson", Key: `userId:#id# hash:#hash#`, onCacheHit: CacheHitHook{func(c *gin.Context, cacheValue string) {
								// 这里会覆盖cache 实例的方法
								assert.True(t, len(cacheValue) > 0)
							}}},
						},
					},
					func(c *gin.Context) {
						id, _ := c.GetQuery("id")
						hash, _ := c.GetQuery("hash")
						c.JSON(200, gin.H{
							"id":   id,
							"hash": hash,
						})
					},
				))

				r.POST("/pings", cache.Handler(
					Caching{
						Evict: []CacheEvict{
							{CacheName: []string{"anson"}, Key: "userId:#id#*"},
						},
					},
					func(c *gin.Context) {
						c.JSON(200, gin.H{
							"message": "delete cache",
						})
					},
				))

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/pings?id=%s&hash=%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)

				loadCache := cache.loadCache(context.Background(), fmt.Sprintf("anson:userid:%s hash:%s", item.Id, item.Hash))
				assert.Equal(t, 200, w.Code)

				equalJSON, err := AreEqualJSON(fmt.Sprintf(`{"id": "%s", "hash": "%s"}`, item.Id, item.Hash), loadCache)
				assert.Equal(t, equalJSON && err == nil, true)

				// test for cache hit hook
				w = httptest.NewRecorder()
				req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/pings?id=%s&hash=%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)
			})
		}
	}

}

func Test_Cache_Evict(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for _, item := range []struct {
			Id   string
			Hash string
		}{
			{Id: "10", Hash: "anson"},
			{Id: "2", Hash: "anson"},
			{Id: "1", Hash: "anson"},
		} {
			t.Run(fmt.Sprintf(`can cache %s  %s`, item.Id, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ping?id=%s&hash=%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)

				w = httptest.NewRecorder()
				req, _ = http.NewRequest(http.MethodPost, "/ping", strings.NewReader(fmt.Sprintf(`{"id": "%s", "hash": "%s"}`, item.Id, item.Hash)))
				r.ServeHTTP(w, req)

				loadCache := cache.loadCache(context.Background(), fmt.Sprintf("anson:userid:%s hash:%s", item.Id, item.Hash))

				assert.Equal(t, loadCache, "")
			})

		}
	}
}

func Test_Cache_Fuzzy_Evict(t *testing.T) {
	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for _, item := range []struct {
			Hash string
		}{
			{Hash: "hash111"},
			{Hash: "hash222"},
			{Hash: "hash333"},
		} {
			t.Run(fmt.Sprintf(`can like delete %s`, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.PUT("/ping", cache.Handler(
					Caching{
						Evict: []CacheEvict{
							{CacheName: []string{"anson"}, Key: "hash*"},
						},
					},
					func(c *gin.Context) {
						json := make(map[string]interface{})
						c.BindJSON(&json)
						c.JSON(200, gin.H{
							"message": "12123",
						})
					},
				))

				r.DELETE("/pings", cache.Handler(
					Caching{
						Evict: []CacheEvict{
							{
								CacheName: []string{"anson"},
								Key:       "name:#name#",
							},
						},
					},
					func(c *gin.Context) {
						json := make(map[string]interface{})
						c.BindJSON(&json)
						c.JSON(200, gin.H{
							"message": "12123",
						})
					},
				))

				r.GET("/pings", cache.Handler(
					Caching{
						Cacheable: []Cacheable{
							{CacheName: "anson", Key: `hash:#hash#`},
						},
					},
					func(c *gin.Context) {
						hash, _ := c.GetQuery("hash")
						c.JSON(200, gin.H{
							"hash": hash,
						})
					},
				))

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/pings?hash=%s", item.Hash), nil)
				r.ServeHTTP(w, req)

				w = httptest.NewRecorder()
				req, _ = http.NewRequest(http.MethodDelete, "/pings", strings.NewReader(fmt.Sprintf(`{"hash": "%s"}`, item.Hash)))
				r.ServeHTTP(w, req)

				equalJSON, _ := AreEqualJSON(fmt.Sprintf(`{"message": "12123"}`), w.Body.String())
				assert.True(t, equalJSON)

				w = httptest.NewRecorder()
				req, _ = http.NewRequest(http.MethodPut, "/ping", strings.NewReader(fmt.Sprintf(`{"hash": "%s"}`, item.Hash)))
				r.ServeHTTP(w, req)

				cacheValue := cache.loadCache(context.Background(), fmt.Sprintf("anson:hash:%s", item.Hash))
				assert.Equal(t, cacheValue, "")
			})
		}
	}
}

func Test_Cache_Timeout_Event(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for key, val := range map[string]string{
			"1": "anson",
			"2": "anson",
		} {
			t.Run("%s %s", func(t *testing.T) {
				var timeout time.Duration
				if runFor == MemoryCache {
					timeout = time.Second * 1
				} else {
					timeout = time.Second
				}
				r, cache := givingCacheOfHttpServer(timeout, runFor)
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ping?id=%s&hash=%s", key, val), nil)
				r.ServeHTTP(w, req)

				cacheKey := fmt.Sprintf("anson:userid:%s hash:%s", key, val)

				time.Sleep(time.Second * 2)
				loadCache := cache.loadCache(context.Background(), cacheKey)
				assert.Equal(t, loadCache, "")
			})
		}
	}
}

func Test_Post_Method_Should_Be_Cache(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for _, item := range []struct {
			Hash string
		}{
			{Hash: "hash111"},
			{Hash: "hash222"},
			{Hash: "hash333"},
		} {
			t.Run(fmt.Sprintf(`Not Error %s`, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.POST("/pings", cache.Handler(
					Caching{
						Cacheable: []Cacheable{
							{CacheName: "anson", Key: `hash:#hash#`, onCacheHit: CacheHitHook{func(c *gin.Context, cacheValue string) {
								// 这里会覆盖cache 实例的方法
								assert.True(t, len(cacheValue) > 0)
							}}},
						},
					},
					func(c *gin.Context) {
						json := make(map[string]interface{})
						c.BindJSON(&json)
						c.JSON(200, gin.H{
							"message": "12123",
						})
					},
				))

				w := httptest.NewRecorder()
				body := fmt.Sprintf(`{"hash": "%s"}`, item.Hash)
				req, _ := http.NewRequest(http.MethodPost, "/pings", bytes.NewBufferString(body))
				r.ServeHTTP(w, req)

				sprintf := fmt.Sprintf("anson:hash:%s", item.Hash)
				cacheValue := cache.loadCache(context.Background(), sprintf)

				equalJSON, _ := AreEqualJSON(`{"message": "12123"}`, cacheValue)
				assert.True(t, equalJSON)
			})
		}
	}
}

func Test_Post_Method_Should_Be_Evict(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for _, item := range []struct {
			Hash string
		}{
			{Hash: "hash111"},
			{Hash: "hash222"},
			{Hash: "hash333"},
		} {
			t.Run(fmt.Sprintf(`Not Error %s`, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.POST("/pings", cache.Handler(
					Caching{
						Cacheable: []Cacheable{
							{CacheName: "anson", Key: `hash:#hash#`, onCacheHit: CacheHitHook{func(c *gin.Context, cacheValue string) {
								// 这里会覆盖cache 实例的方法
								assert.True(t, len(cacheValue) > 0)
							}}},
						},
						Evict: []CacheEvict{
							{CacheName: []string{"anson"}, Key: `hash:#hash#`},
						},
					},
					func(c *gin.Context) {
						json := make(map[string]interface{})
						c.BindJSON(&json)
						c.JSON(200, gin.H{
							"message": "12123",
						})
					},
				))

				w := httptest.NewRecorder()
				body := fmt.Sprintf(`{"hash": "%s"}`, item.Hash)
				req, _ := http.NewRequest(http.MethodPost, "/pings", bytes.NewBufferString(body))
				r.ServeHTTP(w, req)

				sprintf := fmt.Sprintf("anson:hash:%s", item.Hash)
				cacheValue := cache.loadCache(context.Background(), sprintf)

				assert.Equal(t, cacheValue, "")

			})
		}
	}
}

func Test_Put_Method_Should_Be_Cache(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for _, item := range []struct {
			Hash    string
			doError bool
		}{
			{Hash: "hash111", doError: true},
			{Hash: "hash222", doError: false},
			{Hash: "hash333", doError: false},
		} {
			t.Run(fmt.Sprintf(`Not Error %s`, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.PUT("/pings", cache.Handler(
					Caching{
						Cacheable: []Cacheable{
							{CacheName: "anson", Key: `hash:#hash#`, onCacheHit: CacheHitHook{func(c *gin.Context, cacheValue string) {
								// 这里会覆盖cache 实例的方法
								assert.True(t, len(cacheValue) > 0)
							}}},
						},
					},
					func(c *gin.Context) {
						json := make(map[string]interface{})
						c.BindJSON(&json)
						c.JSON(200, gin.H{
							"message": "12123",
						})
					},
				))

				w := httptest.NewRecorder()
				var body string
				if item.doError {
					body = fmt.Sprintf(`{"hash1": "%s"`, item.Hash)
				} else {
					body = fmt.Sprintf(`{"hash": "%s"}`, item.Hash)
				}
				req, _ := http.NewRequest(http.MethodPut, "/pings", bytes.NewBufferString(body))
				r.ServeHTTP(w, req)

				sprintf := fmt.Sprintf("anson:hash:%s", item.Hash)
				cacheValue := cache.loadCache(context.Background(), sprintf)

				if item.doError {
					assert.Equal(t, cacheValue, "")
					equalJSON, _ := AreEqualJSON(`{"message": "12123"}`, cacheValue)
					assert.False(t, equalJSON)
				} else {
					equalJSON, _ := AreEqualJSON(`{"message": "12123"}`, cacheValue)
					assert.True(t, equalJSON)
				}
			})
		}
	}
}

func Test_Redis_Not_Option_Start_Up_Will_Fail(t *testing.T) {
	cache, err := NewRedisCache(time.Second*1, nil)
	assert.Error(t, err)
	assert.Nil(t, cache)
	cache, err = NewMemoryCache(time.Second*-1, nil)
	assert.Error(t, err)
	assert.Nil(t, cache)
}

func AreEqualJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}
