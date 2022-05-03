package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pygzfei/gin-cache/cmd/startup"
	"github.com/pygzfei/gin-cache/pkg/define"
)

func main() {
	cache, _ := startup.MemCache()
	r := gin.Default()

	r.GET("/ping", cache.Handler(
		define.Caching{
			Cacheable: []define.Cacheable{
				// params["id"] 是请求数据, 来自于query 或者 post data, 例如: `/?id=1`, 缓存将会生成为: `anson:id:1`
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
