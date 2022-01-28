[![Build Status](https://github.com/pygzfei/gin-cache/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/pygzfei/gin-cache/actions?query=branch%3Amaster)
[![Go Report Card](https://goreportcard.com/badge/github.com/pygzfei/gin-cache?branch=main)](https://goreportcard.com/report/github.com/pygzfei/gin-cache)
[![codecov](https://codecov.io/gh/pygzfei/gin-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/pygzfei/gin-cache)

## Gin cache middleware
实现了内存缓存 以及 Redis缓存的方式
## Install
```
go get -u github.com/pygzfei/gin-cache
```
## Quick start 
内存缓存: 内部维护着一个map
```
package main

import (
	"github.com/gin-gonic/gin"
	"time"
)

func main() {
	cache := NewMemoryCache(
		time.Minute * 30, // 每个条缓存的存活时间为30分钟, 不同的key值会有不同的失效时间, 互不影响
	)
	r := gin.Default()

	r.GET("/ping", cache.Handler(
		Caching{
			Cacheable: []Cacheable{
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
// 将会触发失效的缓存Key值为: `anson:userid:1`
r.POST("/ping", cache.Handler(
    Caching{
        Evict: []CacheEvict{
            // #id# 从Post Body Json获取 `{"id": 1}`
            {CacheName: []string{"anson"}, Key: "id:#id#"},
        },
    },
    func(c *gin.Context) {
        // ...
    },
))

// 也可以使用通配符 '*', 例如 'anson:id:1*'
// 如果缓存列表里面存在这些数据: ["anson:id:1", "anson:id:12", "anson:id:3"]
// 那么 `anson:id:1` 开头的缓存数据, 将会被删除, 缓存列表将剩余: ["anson:id:3"]
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
cache := NewRedisCache(time.Second*30, &redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
	
```

## Hooks
缓存实例, 默认返回"application/json; Charset=utf-8", 类似的代码如下:
```
ctx.Writer.Header().Set("Content-Type", "application/json; Charset=utf-8")
ctx.String(http.StatusOK, cacheValue)
ctx.Abort()
````
可以使用全局的Hook拦截返回信息
```
cache := NewMemoryCache(timeout, func(c *gin.Context, cacheValue string) {
    // 被缓存的值, 可以在全局拦截
})

```
也可以使用独立的Hook去拦截某个消息返回
```
cache := NewMemoryCache(timeout, func(c *gin.Context, cacheValue string) {
    // 这里不会被执行
})

r.GET("/pings", cache.Handler(
    Caching{
        Cacheable: []Cacheable{
            {CacheName: "anson", Key: `userId:#id# hash:#hash#`,
             onCacheHit: CacheHitHook{func(c *gin.Context, cacheValue string) {
                // 这里会覆盖cache的全局拦截
                assert.True(t, len(cacheValue) > 0)
            }}},
        },
    },
    func(c *gin.Context) {
       //...
    },
))
```