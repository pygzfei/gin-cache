[![Build Status](https://github.com/pygzfei/gin-cache/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/pygzfei/gin-cache/actions?query=branch%3Amaster)
[![Go Report Card](https://goreportcard.com/badge/github.com/pygzfei/gin-cache?branch=main)](https://goreportcard.com/report/github.com/pygzfei/gin-cache)
[![codecov](https://codecov.io/gh/pygzfei/gin-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/pygzfei/gin-cache)

## Gin cache middleware
Easy use of caching with Gin Handler Func

## [中文](/README_CN.md)

## Driver
- [x] memory
- [x] redis
- [ ] more...

## Install
```
go get -u github.com/pygzfei/gin-cache
```
## Quick start
```
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pygzfei/gin-cache"
	"github.com/pygzfei/gin-cache/driver/memcache"
	"time"
)

func main() {

	cache, _ := memcache.NewCacheHandler(
		time.Minute * 30, // 每个条缓存的存活时间为30分钟, 不同的key值会有不同的失效时间, 互不影响
	)
	r := gin.Default()

	r.GET("/ping", cache.Handler(
		gincache.Caching{
			Cacheable: []gincache.Cacheable{
				// #id# 是请求数据, 来自于query 或者 post data, 例如: `/?id=1`, 缓存将会生成为: `anson:userid:1`
				{CacheName: "anson", Key: `id:#id#`},
			},
		},
		func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong", // 返回数据将会被缓存
			})
		},
	))

	r.Run()
}

```

## Trigger Cache evict
```
// Post Body Json: {"id": 1}
// The cache key value that will trigger invalidation is: `anson:userid:1`
r.POST("/ping", cache.Handler(
    Caching{
        Evict: []CacheEvict{
            // #id# Get `{"id": 1}` from Post Body Json
            {CacheName: []string{"anson"}, Key: "id:#id#"},
        },
    },
    func(c *gin.Context) {
        // ...
    },
))

// Wildcards '*' can also be used, e.g. 'anson:id:1*'
// If this data exists in the cache list: ["anson:id:1", "anson:id:12", "anson:id:3"]
// Then the cached data starting with `anson:id:1` will be deleted, and the cache list will remain: ["anson:id:3"]
r.POST("/ping", cache.Handler(
    Caching{
        Evict: []CacheEvict{
            // #id# 从Post Body Json获取 `{"id": 1}`
            {CacheName: []string{"anson"}, Key: "id:#id#*"},
        },
    },
    func(c *gin.Context) {
        // ...
    },
))
```

## Use Redis
```
cache, _ := NewRedisCache(time.Second*30, &redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
	
```

## Hooks
cache instance, returns "application/json; Charset=utf-8" by default
```
ctx.Writer.Header().Set("Content-Type", "application/json; Charset=utf-8")
ctx.String(http.StatusOK, cacheValue)
ctx.Abort()
````
also, can use the global Hook to intercept the return information
```
cache, _ := NewMemoryCache(timeout, func(c *gin.Context, cacheValue string) {
    // cached value, which can be intercepted globally
})

```
also, use a separate Hook to intercept a message return
```
cache, _ := NewMemoryCache(timeout, func(c *gin.Context, cacheValue string) {
    // will not be executed here
})

r.GET("/pings", cache.Handler(
    Caching{
        Cacheable: []Cacheable{
            {CacheName: "anson", Key: `userId:#id# hash:#hash#`,
             onCacheHit: CacheHitHook{func(c *gin.Context, cacheValue string) {
                // this will override the global interception of the cache
                assert.True(t, len(cacheValue) > 0)
            }}},
        },
    },
    func(c *gin.Context) {
       //...
    },
))
```

## Rules
    ...