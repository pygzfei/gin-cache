[![Release](https://img.shields.io/github/v/release/pygzfei/gin-cache.svg?style=flat-square)](https://github.com/pygzfei/gin-cache/releases)
[![doc](https://img.shields.io/badge/go.dev-doc-007d9c?style=flat-square&logo=read-the-docs)](https://pkg.go.dev/github.com/pygzfei/gin-cache)
[![Build Status](https://github.com/pygzfei/gin-cache/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/pygzfei/gin-cache/actions?query=branch%3Amaster)
[![Go Report Card](https://goreportcard.com/badge/github.com/pygzfei/gin-cache?branch=main)](https://goreportcard.com/report/github.com/pygzfei/gin-cache)
[![codecov](https://codecov.io/gh/pygzfei/gin-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/pygzfei/gin-cache)
![](https://img.shields.io/badge/license-MIT-green)

## Gin cache middleware
以 Gin Handler Func 方式轻松使用缓存

## 驱动
- [x] memory
- [x] redis
- [ ] more...
## 安装
```
go get -u github.com/pygzfei/gin-cache
```
## 快速开始
```
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pygzfei/gin-cache/cmd/startup"
	"github.com/pygzfei/gin-cache/pkg/define"
	"time"
)

func main() {

	cache, _ := startup.MemCache(
		time.Minute * 30, // 每个条缓存的存活时间为30分钟, 不同的key值会有不同的失效时间, 互不影响
	)
	r := gin.Default()

	r.GET("/ping", cache.Handler(
		define.Caching{
			Cacheable: []define.Cacheable{
				// params["id"] 是请求数据, 来自于query 或者 post data, 例如: `/?id=1`, 缓存将会生成为: `anson:userid:1`
				{GenKey: func(params map[string]interface{}) string {
					return fmt.Sprintf("anson:id:%s", params["id"])
				}},
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

## 触发缓存驱逐
```
// Post Body Json: {"id": 1}
// 将会触发失效的缓存Key值为: `anson:userid:1`
r.POST("/ping", cache.Handler(
    define.Caching{
        Evict: []define.CacheEvict{
            // params["id"] 从 Post Body Json获取 `{"id": 1}`
            func(params map[string]interface{}) string {
				return fmt.Sprintf("anson:id:%s", params["id"])
			},
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

## 使用Redis
```
cache, _ := startup.RedisCache(time.Second*30, &redis.Options{
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
cache, _ := startup.MemCache(timeout, func(c *gin.Context, cacheValue string) {
    // 被缓存的值, 可以在全局拦截
})

```
也可以使用独立的Hook去拦截某个消息返回
```
cache, _ := startup.MemCache(timeout, func(c *gin.Context, cacheValue string) {
    // 这里不会被执行
})

r.GET("/pings", cache.Handler(
    define.Caching{
        Cacheable: []define.Cacheable{
            GenKey: func(params map[string]interface{}) string {
				return fmt.Sprintf("anson:userId:%s hash:%s", params["id"], params["hash"])
			},
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