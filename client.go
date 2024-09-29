package redis_lock

import (
	"github.com/redis/go-redis/v9"
)

type MyRedis struct {
	*redis.Client
}

func NewClient(addr, password string, dbNumber int) *MyRedis {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       dbNumber,
	})
	return &MyRedis{rdb}
}
