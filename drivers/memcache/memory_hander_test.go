package memcache

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_Not_Option_Start_Up_Will_Fail(t *testing.T) {
	cache, err := NewCacheHandler(time.Second * -1)
	assert.Error(t, err)
	assert.Nil(t, cache)
}
