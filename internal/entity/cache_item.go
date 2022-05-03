package entity

import "time"

type CacheItem struct {
	Value    string
	CreateAt time.Time
	ExpireAt time.Time
	Hits     uint64
}
