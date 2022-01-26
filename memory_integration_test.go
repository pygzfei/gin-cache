package gin_cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func GivingHttpServer(timeout time.Duration) (*gin.Engine, *Cache) {
	cache := NewMemoryCache(
		timeout,
	)
	gin.ForceConsoleColor()
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	r.GET("/ping", cache.Handler(
		ApiCacheable{
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
		ApiCacheable{
			CacheEvict: []CacheEvict{
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

func Test_Memory_Cache_CanStore(t *testing.T) {

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
			r, cache := GivingHttpServer(time.Hour)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ping?id=%s&hash=%s", item.Id, item.Hash), nil)
			r.ServeHTTP(w, req)

			loadCache := cache.loadCache(context.Background(), fmt.Sprintf("anson:userid:%s hash:%s", item.Id, item.Hash))
			assert.Equal(t, 200, w.Code)

			assert.Equal(t, w.Body.String(), loadCache)

			equalJSON, err := AreEqualJSON(fmt.Sprintf(`{"id": "%s", "hash": "%s"}`, item.Id, item.Hash), w.Body.String())
			assert.Equal(t, equalJSON && err == nil, true)
		})
	}
}

func Test_Memory_Cache_Evict(t *testing.T) {

	for _, item := range []struct {
		Id   string
		Hash string
	}{
		{Id: "10", Hash: "anson"},
		{Id: "2", Hash: "anson"},
		{Id: "1", Hash: "anson"},
	} {
		t.Run(fmt.Sprintf(`can cache %s  %s`, item.Id, item.Hash), func(t *testing.T) {
			r, cache := GivingHttpServer(time.Hour)
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

func Test_MemoryCache_Fuzzy_Evict(t *testing.T) {

	for _, item := range []struct {
		Hash string
	}{
		{Hash: "hash111"},
		{Hash: "hash222"},
		{Hash: "hash333"},
	} {
		t.Run(fmt.Sprintf(`can like delete %s`, item.Hash), func(t *testing.T) {
			r, cache := GivingHttpServer(time.Hour)

			r.PUT("/ping", cache.Handler(
				ApiCacheable{
					CacheEvict: []CacheEvict{
						{CacheName: []string{"anson"}, Key: "hash*"},
					},
				},
				func(c *gin.Context) {
					json := make(map[string]interface{})
					c.ShouldBindBodyWith(&json, binding.JSON)
					c.JSON(200, gin.H{
						"message": "12123",
					})
				},
			))

			r.GET("/pings", cache.Handler(
				ApiCacheable{
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

			cacheValue := cache.loadCache(context.Background(), fmt.Sprintf("anson:hash:%s", item.Hash))

			w = httptest.NewRecorder()
			req, _ = http.NewRequest(http.MethodPut, "/ping", strings.NewReader(fmt.Sprintf(`{"hash": "%s"}`, item.Hash)))
			r.ServeHTTP(w, req)

			cacheValue = cache.loadCache(context.Background(), fmt.Sprintf("anson:hash:%s", item.Hash))
			assert.Equal(t, cacheValue, "")
		})
	}
}

func Test_Memory_Timeout_Event(t *testing.T) {

	for key, val := range map[string]string{
		"1": "anson",
		"2": "anson",
	} {
		t.Run("%s %s", func(t *testing.T) {
			r, cache := GivingHttpServer(time.Millisecond * 0)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ping?id=%s&hash=%s", key, val), nil)
			r.ServeHTTP(w, req)

			time.Sleep(time.Second * 1)
			loadCache := cache.loadCache(context.Background(), fmt.Sprintf("anson:userid:%s hash:%s", key, val))
			assert.Equal(t, loadCache, "")
		})
	}

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
