package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/pygzfei/gin-cache/cmd/startup"
	"github.com/pygzfei/gin-cache/internal"
	"github.com/pygzfei/gin-cache/pkg/define"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
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

func givingCacheOfHttpServer(timeout time.Duration, runFor RunFor, onHit ...func(c *gin.Context, cacheValue *define.CacheItem)) (*gin.Engine, *internal.CacheHandler) {
	var cache *internal.CacheHandler

	if runFor == MemoryCache {
		cache, _ = startup.MemCache(timeout, onHit...)
	} else if runFor == RedisCache {
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost"
		}
		cache, _ = startup.RedisCache(
			timeout,
			&redis.Options{
				Addr:     fmt.Sprintf("%s:6379", redisHost),
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
		define.Caching{
			Cacheable: []define.Cacheable{
				{GenKey: func(params map[string]interface{}) string {
					return fmt.Sprintf("anson:userId:%v hash:%v", params["id"], params["hash"])
				}},
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

	r.GET("/ping/:id/:hash", cache.Handler(
		define.Caching{
			Cacheable: []define.Cacheable{
				{GenKey: func(params map[string]interface{}) string {
					return fmt.Sprintf("anson:userId:%v hash:%v", params["id"], params["hash"])
				}},
			},
		},
		func(c *gin.Context) {
			id := c.Param("id")
			hash := c.Param("hash")
			c.Header("X-ID", id)
			c.Header("X-Hash", hash)

			c.JSON(200, gin.H{
				"id":   id,
				"hash": hash,
			})
		},
	))

	r.POST("/ping", cache.Handler(
		define.Caching{
			Evict: []define.CacheEvict{
				func(params map[string]interface{}) string {
					return fmt.Sprintf("anson:userId:%s*", params["id"])
				},
			},
		},
		func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "delete startup",
			})
		},
	))

	return r, cache
}

func Test_Path_Variable_Not_Variable_Can_Cache_CanStore(t *testing.T) {

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
				r, cache := givingCacheOfHttpServer(time.Hour, runFor, func(c *gin.Context, cacheValue *define.CacheItem) {
					assert.True(t, cacheValue != nil)
				})

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
				r.ServeHTTP(w, req)

				cacheKey := "anson:userid:<nil> hash:<nil>"
				loadCache := cache.Load(context.Background(), cacheKey)
				assert.Equal(t, 200, w.Code)

				sprintf := `{"hash":"","id":""}`
				equalJSON, err := AreEqualJSON(sprintf, string(loadCache.Body))
				assert.Equal(t, equalJSON && err == nil, true)

				//test for startup hit hook
				w = httptest.NewRecorder()
				req, _ = http.NewRequest(http.MethodGet, "/ping", nil)
				r.ServeHTTP(w, req)
			})
		}
	}
}

func Test_Path_Variable_Cache_CanStore(t *testing.T) {

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
				r, cache := givingCacheOfHttpServer(time.Hour, runFor, func(c *gin.Context, cacheValue *define.CacheItem) {
					assert.True(t, cacheValue != nil)
				})

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ping/%s/%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)

				cacheKey := fmt.Sprintf("anson:userid:%s hash:%s", item.Id, item.Hash)
				loadCache := cache.Load(context.Background(), cacheKey)
				assert.Equal(t, 200, w.Code)

				sprintf := fmt.Sprintf(`{"id": "%s", "hash": "%s"}`, item.Id, item.Hash)
				equalJSON, err := AreEqualJSON(sprintf, string(loadCache.Body))
				assert.Equal(t, equalJSON && err == nil, true)

				// test for header cache
				assert.Equal(t, loadCache.Header.Get("X-ID"), item.Id)
				assert.Equal(t, loadCache.Header.Get("X-Hash"), item.Hash)

				//test for startup hit hook
				w = httptest.NewRecorder()
				req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/ping/%s/%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)
			})
		}
	}
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
				r, cache := givingCacheOfHttpServer(time.Hour, runFor, func(c *gin.Context, cacheValue *define.CacheItem) {
					assert.True(t, cacheValue != nil)
				})

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ping?id=%s&hash=%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)

				cacheKey := fmt.Sprintf("anson:userid:%s hash:%s", item.Id, item.Hash)
				loadCache := cache.Load(context.Background(), cacheKey)
				assert.Equal(t, 200, w.Code)

				sprintf := fmt.Sprintf(`{"id": "%s", "hash": "%s"}`, item.Id, item.Hash)
				equalJSON, err := AreEqualJSON(sprintf, string(loadCache.Body))
				assert.Equal(t, equalJSON && err == nil, true)

				//test for startup hit hook
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
				r, cache := givingCacheOfHttpServer(time.Hour, runFor, func(c *gin.Context, cacheValue *define.CacheItem) {
					// 这里不会被触发
					err := errors.New("should not trigger this func")
					assert.NoError(t, err)
				})

				r.GET("/pings", cache.Handler(
					define.Caching{
						Cacheable: []define.Cacheable{
							{GenKey: func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:userId:%s hash:%s", params["id"], params["hash"])
							}, OnCacheHit: define.CacheHitHook{func(c *gin.Context, cacheValue *define.CacheItem) {
								// 这里会覆盖cache 实例的方法
								assert.True(t, cacheValue != nil)
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
					define.Caching{
						Evict: []define.CacheEvict{
							func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:userId:%s*", params["id"])
							},
						},
					},
					func(c *gin.Context) {
						c.JSON(200, gin.H{
							"message": "delete startup",
						})
					},
				))

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/pings?id=%s&hash=%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)

				loadCache := cache.Load(context.Background(), fmt.Sprintf("anson:userid:%s hash:%s", item.Id, item.Hash))
				assert.Equal(t, 200, w.Code)

				equalJSON, err := AreEqualJSON(fmt.Sprintf(`{"id": "%s", "hash": "%s"}`, item.Id, item.Hash), string(loadCache.Body))
				assert.Equal(t, equalJSON && err == nil, true)

				// test for startup hit hook
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
			t.Run(fmt.Sprintf(`can startup %s  %s`, item.Id, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ping?id=%s&hash=%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)

				w = httptest.NewRecorder()
				req, _ = http.NewRequest(http.MethodPost, "/ping", strings.NewReader(fmt.Sprintf(`{"id": "%s", "hash": "%s"}`, item.Id, item.Hash)))
				r.ServeHTTP(w, req)

				loadCache := cache.Load(context.Background(), fmt.Sprintf("anson:userid:%s hash:%s", item.Id, item.Hash))
				assert.Equal(t, loadCache == nil, true)
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
					define.Caching{
						Evict: []define.CacheEvict{
							func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:hash*")
							},
						},
					},
					func(c *gin.Context) {
						body := make(map[string]interface{})
						_ = c.BindJSON(&body)
						c.JSON(200, gin.H{
							"message": "12123",
						})
					},
				))

				r.DELETE("/pings", cache.Handler(
					define.Caching{
						Evict: []define.CacheEvict{
							func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:name:%s", params["name"])
							},
						},
					},
					func(c *gin.Context) {
						body := make(map[string]interface{})
						_ = c.BindJSON(&body)
						c.JSON(200, gin.H{
							"message": "12123",
						})
					},
				))

				r.GET("/pings", cache.Handler(
					define.Caching{
						Cacheable: []define.Cacheable{
							{GenKey: func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:hash:%s", params["hash"])
							}},
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

				cacheValue := cache.Load(context.Background(), fmt.Sprintf("anson:hash:%s", item.Hash))
				assert.Equal(t, cacheValue == nil, true)
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
				loadCache := cache.Load(context.Background(), cacheKey)
				assert.Equal(t, loadCache == nil, true)
			})
		}
	}
}

func Test_Post_Method_Should_Be_Cache(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {
		rand.Seed(time.Now().Unix() + 1)
		for _, item := range []struct {
			Hash string
		}{
			{Hash: fmt.Sprintf("hash%v", rand.Int())},
			{Hash: fmt.Sprintf("hash%v", rand.Int())},
			{Hash: fmt.Sprintf("hash%v", rand.Int())},
		} {
			t.Run(fmt.Sprintf(`Not Error %s`, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.POST("/pings", cache.Handler(
					define.Caching{
						Cacheable: []define.Cacheable{
							{
								GenKey: func(params map[string]interface{}) string {
									return fmt.Sprintf("anson:hash:%s", params["hash"])
								}, OnCacheHit: define.CacheHitHook{func(c *gin.Context, cacheValue *define.CacheItem) {
									// 这里会覆盖cache 实例的方法
									assert.True(t, cacheValue != nil)
								}}},
						},
					},
					func(c *gin.Context) {
						body := make(map[string]interface{})
						_ = c.BindJSON(&body)
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
				cacheValue := cache.Load(context.Background(), sprintf)

				equalJSON, _ := AreEqualJSON(`{"message": "12123"}`, string(cacheValue.Body))
				assert.True(t, equalJSON)
			})
		}
	}
}

func Test_Post_Method_Should_Be_Evict_Old_Data_And_Cache_New_Data(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {
		rand.Seed(time.Now().Unix() + 2)

		for _, item := range []struct {
			Hash string
		}{
			{Hash: fmt.Sprintf("%v", rand.Int())},
			{Hash: fmt.Sprintf("%v", rand.Int())},
			{Hash: fmt.Sprintf("%v", rand.Int())},
		} {
			t.Run(fmt.Sprintf(`Not Error %s`, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.POST("/pings", cache.Handler(
					define.Caching{
						Cacheable: []define.Cacheable{
							//{CacheName: "anson", Key: `hash:#hash#`},
							{GenKey: func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:hash:%s", params["hash"])
							}},
						},
						Evict: []define.CacheEvict{
							//{CacheName: []string{"anson"}, Key: `hash:#hash#`},
							func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:hash:%s", params["hash"])
							},
						},
					},
					func(c *gin.Context) {
						body := make(map[string]interface{})
						err := c.BindJSON(&body)
						fmt.Println(err)
						c.JSON(200, body)
					},
				))

				w := httptest.NewRecorder()
				body := fmt.Sprintf(`{"hash":"%s"}`, item.Hash)
				req, _ := http.NewRequest(http.MethodPost, "/pings", bytes.NewBufferString(body))
				r.ServeHTTP(w, req)

				sprintf := fmt.Sprintf("anson:hash:%s", item.Hash)
				cacheValue := cache.Load(context.Background(), sprintf)
				equalJSON, _ := AreEqualJSON(string(cacheValue.Body), body)
				assert.True(t, equalJSON)

			})
		}
	}
}

func Test_Post_Method_Should_Be_Evict(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {
		rand.Seed(time.Now().Unix() + 2)

		for _, item := range []struct {
			Hash string
		}{
			{Hash: fmt.Sprintf("hash%v", rand.Int())},
			{Hash: fmt.Sprintf("hash%v", rand.Int())},
			{Hash: fmt.Sprintf("hash%v", rand.Int())},
		} {
			t.Run(fmt.Sprintf(`Not Error %s`, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.GET("/ping_for_get", cache.Handler(
					define.Caching{
						Cacheable: []define.Cacheable{
							{GenKey: func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:hash:%s", params["hash"])
							}, OnCacheHit: define.CacheHitHook{func(c *gin.Context, cacheValue *define.CacheItem) {
								// 这里会覆盖cache 实例的方法
								assert.True(t, cacheValue != nil)
							}}},
						},
					},
					func(c *gin.Context) {
						query, _ := c.GetQuery("hash")
						c.JSON(200, query)
					},
				))

				r.POST("/ping_for_post", cache.Handler(
					define.Caching{
						Evict: []define.CacheEvict{
							func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:hash:%s", params["hash"])
							},
						},
					},
					func(c *gin.Context) {
						body := make(map[string]interface{})
						_ = c.BindJSON(&body)
						c.JSON(200, body)
					},
				))

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, "/ping_for_get?hash="+item.Hash, nil)
				r.ServeHTTP(w, req)

				sprintf := fmt.Sprintf("anson:hash:%s", item.Hash)
				cacheValue := cache.Load(context.Background(), sprintf)

				w = httptest.NewRecorder()
				body := fmt.Sprintf(`{"hash":"%s"}`, item.Hash)
				req, _ = http.NewRequest(http.MethodPost, "/ping_for_post", bytes.NewBufferString(body))
				r.ServeHTTP(w, req)

				cacheValue = cache.Load(context.Background(), sprintf)

				assert.Equal(t, cacheValue == nil, true)

			})
		}
	}
}

func Test_Put_Method_Should_Be_Cache(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {
		rand.Seed(time.Now().Unix() + 3)
		for _, item := range []struct {
			Hash    string
			doError bool
		}{
			{Hash: fmt.Sprintf("hash%v", rand.Int()), doError: true},
			{Hash: fmt.Sprintf("hash%v", rand.Int()), doError: false},
			{Hash: fmt.Sprintf("hash%v", rand.Int()), doError: false},
		} {
			t.Run(fmt.Sprintf(`Not Error %s`, item.Hash), func(t *testing.T) {
				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.PUT("/pings", cache.Handler(
					define.Caching{
						Cacheable: []define.Cacheable{
							{GenKey: func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:hash:%s", params["hash"])
							}, OnCacheHit: define.CacheHitHook{func(c *gin.Context, cacheValue *define.CacheItem) {
								// 这里会覆盖cache 实例的方法
								assert.True(t, cacheValue != nil)
							}}},
						},
					},
					func(c *gin.Context) {
						body := make(map[string]interface{})
						_ = c.BindJSON(&body)
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
				cacheValue := cache.Load(context.Background(), sprintf)

				if item.doError {
					assert.Equal(t, cacheValue == nil, true)
				} else {
					equalJSON, _ := AreEqualJSON(`{"message": "12123"}`, string(cacheValue.Body))
					assert.True(t, equalJSON)
				}
			})
		}
	}
}

func Test_Diff_Timeout_Cache_Evict(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for _, item := range []struct {
			Id   string
			Hash string
		}{
			{Id: "1", Hash: "anson1"},
			{Id: "2", Hash: "anson2"},
		} {
			t.Run(fmt.Sprintf(`can startup %s  %s`, item.Id, item.Hash), func(t *testing.T) {

				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.GET("/ping_for_timeout", cache.Handler(
					define.Caching{
						Cacheable: []define.Cacheable{
							{GenKey: func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:id:%s&name=%s", item.Id, item.Hash)
							}, CacheTime: time.Second},
						},
					},
					func(c *gin.Context) {
						c.JSON(200, gin.H{
							"id":   item.Id,
							"hash": item.Hash,
						})
					},
				))

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ping_for_timeout?id=%s&name=%s", item.Id, item.Hash), nil)
				r.ServeHTTP(w, req)

				time.Sleep(time.Second * 2)

				loadCache := cache.Load(context.Background(), fmt.Sprintf("anson:id:%s&name=%s", item.Id, item.Hash))

				assert.Equal(t, loadCache == nil, true)
			})

		}
	}
}

func Test_All_Cache_Evict(t *testing.T) {

	for _, runFor := range []RunFor{MemoryCache, RedisCache} {

		for _, item := range []struct {
			Id   string
			Hash string
		}{
			{Id: "10", Hash: "anson"},
		} {
			t.Run(fmt.Sprintf(`can startup %s  %s`, item.Id, item.Hash), func(t *testing.T) {

				r, cache := givingCacheOfHttpServer(time.Hour, runFor)

				r.POST("/ping_for_post", cache.Handler(
					define.Caching{
						Evict: []define.CacheEvict{
							func(params map[string]interface{}) string {
								return fmt.Sprintf("anson:*")
							},
						},
					},
					func(c *gin.Context) {
						body := make(map[string]interface{})
						_ = c.BindJSON(&body)
						c.JSON(200, body)
					},
				))

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodPost, "/ping_for_post", strings.NewReader(fmt.Sprintf(`{"id": "%s", "hash": "%s"}`, item.Id, item.Hash)))
				r.ServeHTTP(w, req)

				loadCache := cache.Load(context.Background(), fmt.Sprintf("anson:userid:%s hash:%s", item.Id, item.Hash))

				assert.Equal(t, loadCache == nil, true)
			})

		}
	}
}

func AreEqualJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}
