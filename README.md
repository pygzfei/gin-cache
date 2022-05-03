[![Release](https://img.shields.io/github/v/release/pygzfei/gin-cache.svg?style=flat-square)](https://github.com/pygzfei/gin-cache/releases)
[![doc](https://img.shields.io/badge/go.dev-doc-007d9c?style=flat-square&logo=read-the-docs)](https://pkg.go.dev/github.com/pygzfei/gin-cache)
[![Build Status](https://github.com/pygzfei/gin-cache/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/pygzfei/gin-cache/actions?query=branch%3Amaster)
[![Go Report Card](https://goreportcard.com/badge/github.com/pygzfei/gin-cache?branch=main)](https://goreportcard.com/report/github.com/pygzfei/gin-cache)
[![codecov](https://codecov.io/gh/pygzfei/gin-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/pygzfei/gin-cache)
![](https://img.shields.io/badge/license-MIT-green)

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

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pygzfei/gin-cache/cmd/startup"
	"github.com/pygzfei/gin-cache/pkg/define"
	"time"
)

func main() {

	cache, _ := startup.MemCache()
	r := gin.Default()

	r.GET("/ping", cache.Handler(
		define.Caching{
		    Cacheable: []define.Cacheable{
                    // params["id"] is the request data from query or post data, for example: 
                    // http://domain/?id=1, the cache will be generated as: `anson:id:1`
                    {GenKey: func(params map[string]interface{}) string {
                        return fmt.Sprintf("anson:id:%s", params["id"])
                    }},
			},
		},
		func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong", // The returned data will be cached
			})
		},
	))

	r.Run()
}

```

## Overwrite global cache time

```go
cache, _ := startup.MemCache()

r := gin.Default()

r.GET("/ping_for_timeout", cache.Handler(
    define.Caching{
        Cacheable: []define.Cacheable{
            {GenKey: func(params map[string]interface{}) string {
                return fmt.Sprintf("anson:id:%s&name=%s", item.Id, item.Hash)
            }, 
            // The effective time of the cache will be based on this time value instead of the global value
            CacheTime: time.Second },
        },
    },
    func(c *gin.Context) {
       // ...
    },
))

```

## Trigger Cache evict

```go
// Post Body Json: {"id": 1}
// The cache key value that will trigger invalidation is: `anson:userid:1`
r.POST("/ping", cache.Handler(
    define.Caching{
        Evict: []define.CacheEvict{
            // params["id"]  from Post Body Json `{"id": 1}`
            func(params map[string]interface{}) string {
                return fmt.Sprintf("anson:id:%s", params["id"])
            },
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
    define.Caching{
        Evict: []define.CacheEvict{
            func(params map[string]interface{}) string {
                return fmt.Sprintf("anson:id:%s*", params["id"])
            },
        },
    },
    func(c *gin.Context) {
        // ...
    },
))
```

## Use Redis

```go
cache, _ := startup.RedisCache(time.Second*30, &redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
	
```

## Hooks

cache instance, returns "application/json; Charset=utf-8" by default

```go
ctx.Writer.Header().Set("Content-Type", "application/json; Charset=utf-8")
ctx.String(http.StatusOK, cacheValue)
ctx.Abort()
````

also, can use the global Hook to intercept the return information

```go
cache, _ := startup.MemCache(timeout, func(c *gin.Context, cacheValue string) {
    // cached value, which can be intercepted globally
})

```

also, use a separate Hook to intercept a message return

```go
cache, _ := startup.MemCache(timeout, func(c *gin.Context, cacheValue string) {
    // will not be executed here
})

r.GET("/pings", cache.Handler(
    define.Caching{
        Cacheable: []define.Cacheable{
            GenKey: func(params map[string]interface{}) string {
                return fmt.Sprintf("anson:userId:%s hash:%s", params["id"], params["hash"])
            },
             onCacheHit: define.CacheHitHook{func(c *gin.Context, cacheValue string) {
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