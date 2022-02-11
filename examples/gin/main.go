package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pygzfei/gin-cache/drivers/memcache"
	"github.com/pygzfei/gin-cache/internal"
	"time"
)

func main() {

	cache, _ := memcache.NewCacheHandler(
		time.Minute * 30, // 每个条缓存的存活时间为30分钟, 不同的key值会有不同的失效时间, 互不影响
	)
	r := gin.Default()

	r.GET("/ping", cache.Handler(
		internal.Caching{
			Cacheable: []internal.Cacheable{
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
