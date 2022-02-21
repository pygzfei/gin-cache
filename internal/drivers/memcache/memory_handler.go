package memcache

import (
	"context"
	"github.com/pygzfei/gin-cache/pkg/define"
	"strings"
	"sync"
	"time"
)

// Schedule entity
type Schedule struct {
	Key   string
	Timer *time.Timer
}

// memoryHandler is private
type memoryHandler struct {
	cacheStore sync.Map
	cacheTime  time.Duration
	pubSub     chan Schedule
	schedules  map[string]*time.Timer
}

var mux sync.Mutex

// NewMemoryHandler do new memory startup object
func NewMemoryHandler(cacheTime time.Duration) *memoryHandler {
	return &memoryHandler{
		cacheStore: sync.Map{},
		cacheTime:  cacheTime,
		pubSub:     make(chan Schedule),
		schedules:  make(map[string]*time.Timer),
	}
}

func (m *memoryHandler) Load(_ context.Context, key string) *define.CacheItem {
	load, ok := m.cacheStore.Load(key)
	if ok {
		return load.(*define.CacheItem)
	}
	return nil
}

func (m *memoryHandler) Set(ctx context.Context, key string, data *define.CacheItem, timeout time.Duration) {
	mux.Lock()
	defer mux.Unlock()
	m.cacheStore.Store(key, data)
	// timeout
	var schedule Schedule
	if timeout > 0 {
		schedule = Schedule{Key: key, Timer: time.NewTimer(timeout)}
	} else {
		schedule = Schedule{Key: key, Timer: time.NewTimer(m.cacheTime)}
	}
	m.schedules[key] = schedule.Timer
	go func(s *Schedule) {
		select {
		case <-s.Timer.C:
			m.DoEvict(ctx, []string{s.Key})
		}
	}(&schedule)

}

func (m *memoryHandler) DoEvict(_ context.Context, keys []string) {
	mux.Lock()
	defer mux.Unlock()
	evictKeys := []string{}
	for _, key := range keys {
		isEndingStar := key[len(key)-1:]
		m.cacheStore.Range(func(keyInMap, value interface{}) bool {
			// match *
			if isEndingStar == "*" {
				if strings.Contains(keyInMap.(string), strings.ReplaceAll(key, "*", "")) {
					evictKeys = append(evictKeys, keyInMap.(string))
				}
			} else {
				if keyInMap == key {
					evictKeys = append(evictKeys, key)
				}
			}
			return true
		})
	}
	for _, key := range evictKeys {
		m.cacheStore.Delete(key)
		timer := m.schedules[key]
		if timer != nil {
			timer.Stop()
		}
		delete(m.schedules, key)
	}

}
