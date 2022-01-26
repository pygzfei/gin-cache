package main

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"time"
)

func main() {
	//r := gin.Default()
	//cache := NewRedisCache(
	//	time.Minute*30,
	//	&redis.Options{
	//		Addr:     "localhost:6379",
	//		Password: "",
	//		DB:       0,
	//	})
	//
	////cache := middleware.NewMemoryCache(
	////	time.Hour,
	////)
	//
	//r.GET("/ping", cache.Cacheable(
	//	ApiCacheable{
	//		Cacheable: []Cacheable{
	//			{CacheName: "anson", Key: `userId:#id# hash:#hash#`},
	//		},
	//	},
	//	func(c *gin.Context) {
	//		query, _ := c.GetQuery("id")
	//		c.JSON(200, gin.H{
	//			"message": query,
	//		})
	//	},
	//))
	//
	//r.POST("/ping", cache.Cacheable(
	//	ApiCacheable{
	//		CacheEvict: []CacheEvict{
	//			{CacheName: []string{"anson"}, Key: "userId:#id#*"},
	//		},
	//	},
	//	func(c *gin.Context) {
	//		c.JSON(200, gin.H{
	//			"message": "delete cache",
	//		})
	//	},
	//))
	//
	//r.Run(":9988")
}
