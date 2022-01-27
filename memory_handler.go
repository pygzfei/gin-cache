package gin_cache

import (
	"context"
	"strings"
	"sync"
	"time"
)

type Schedule struct {
	Key   string
	timer *time.Timer
}

type memoryHandler struct {
	cacheStore sync.Map
	cacheTime  time.Duration
	pubSub     chan Schedule
	schedules  map[string]*time.Timer
}

var mux sync.Mutex

func NewMemoryHandler(cacheTime time.Duration) *memoryHandler {
	return &memoryHandler{
		cacheStore: sync.Map{},
		cacheTime:  cacheTime,
		pubSub:     make(chan Schedule),
		schedules:  make(map[string]*time.Timer),
	}
}

func (this *memoryHandler) LoadCache(_ context.Context, key string) string {
	load, ok := this.cacheStore.Load(key)
	if ok {
		return load.(string)
	}
	return ""
}

func (this *memoryHandler) SetCache(ctx context.Context, key string, data string) {
	mux.Lock()
	this.cacheStore.Store(key, data)
	// timeout
	schedule := Schedule{Key: key, timer: time.NewTimer(this.cacheTime)}
	this.schedules[key] = schedule.timer
	defer mux.Unlock()

	go func(s Schedule) {
		select {
		case <-s.timer.C:
			this.DoCacheEvict(ctx, []string{s.Key})
		default:
			return
		}
	}(schedule)
}

func (this *memoryHandler) DoCacheEvict(_ context.Context, keys []string) {
	mux.Lock()
	deleteKeys := []string{}
	for _, key := range keys {
		isEndingStar := key[len(key)-1:]
		this.cacheStore.Range(func(keyInMap, value interface{}) bool {
			// match *
			if isEndingStar == "*" {
				if strings.Contains(keyInMap.(string), strings.ReplaceAll(key, "*", "")) {
					deleteKeys = append(deleteKeys, keyInMap.(string))
				}
			} else {
				if keyInMap == key {
					deleteKeys = append(deleteKeys, key)
				}
			}
			return true
		})
	}
	for _, key := range deleteKeys {
		this.cacheStore.Delete(key)
		timer := this.schedules[key]
		if timer != nil {
			timer.Stop()
		}
		delete(this.schedules, key)
	}
	defer mux.Unlock()
}
