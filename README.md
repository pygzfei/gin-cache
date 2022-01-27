[![Build Status](https://github.com/pygzfei/gin-cache/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/pygzfei/gin-cache/actions?query=branch%3Amaster)
[![Go Report Card](https://goreportcard.com/badge/github.com/pygzfei/gin-cache)](https://goreportcard.com/report/github.com/pygzfei/gin-cache)
[![codecov](https://codecov.io/gh/pygzfei/gin-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/pygzfei/gin-cache)

## Gin cache middleware
## Quick start 
Use memory cache 
```
package main

import (
	"github.com/gin-gonic/gin"
	"time"
)

func main() {
	cache := NewMemoryCache(
		time.Minute * 30, // 30 Minutes Cache will be invalid
	)
	r := gin.Default()

	r.GET("/ping", cache.Handler(
		Caching{
			Cacheable: []Cacheable{
				// #id# is your query or post data, if query `/?id=1`, kye in cache will be `anson:userid:1`
				{CacheName: "anson", Key: `id:#id#`},
			},
		},
		func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong", // response data will be cache
			})
		},
	))

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
```

## Trigger Cache evict
```
// post data: {"id": 1}
// accurately delete key: `anson:userid:1`
r.POST("/ping", cache.Handler(
    Caching{
        Evict: []CacheEvict{
            // #id# in your post data field is id, e.q `{"id": 1}`
            {CacheName: []string{"anson"}, Key: "id:#id#"},
        },
    },
    func(c *gin.Context) {
        // ...
    },
))

// And you can use wildcard '*'
// When this data in your cache ["anson:id:1", "anson:id:2", "anson:id:3"]
// The key start with `anson:id` will be delete in your cache 
r.POST("/ping", cache.Handler(
    Caching{
        Evict: []CacheEvict{
            // #id# in your post data field is id, e.q `{"id": 1}`
            {CacheName: []string{"anson"}, Key: "id:*"},
        },
    },
    func(c *gin.Context) {
        // ...
    },
))
```

## Use Redis
```
cache := NewRedisCache(time.Second*30, &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	
```