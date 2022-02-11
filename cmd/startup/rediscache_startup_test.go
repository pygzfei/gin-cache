package startup

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRedisCache(t *testing.T) {
	type args struct {
		cacheTime  time.Duration
		options    *redis.Options
		onCacheHit []func(c *gin.Context, cacheValue string)
	}
	tests := []struct {
		name    string
		args    args
		success bool
	}{
		{name: "init success", args: args{cacheTime: time.Second}, success: true},
		{name: "init error", args: args{cacheTime: time.Second * -1}, success: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RedisCache(tt.args.cacheTime, &redis.Options{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			}, tt.args.onCacheHit...)
			if err != nil {
				assert.Error(t, err)
				return
			}
			assert.True(t, got != nil, tt.success)
		})
	}
}
