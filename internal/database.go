package internal

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

var ctx = context.Background()

type DatabaseHandler interface {
	Put(key string, value string)
	Get(key string) string
}

type RedisDatabase struct {
	client *redis.Client
}

func NewRedisDatabase(address string, password string, db int) *RedisDatabase {
	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	}).WithTimeout(time.Second * 5)

	return &RedisDatabase{rdb}
}

func (d *RedisDatabase) Put(key string, value string) {
	_, err := d.client.Set(ctx, key, value, 0).Result()

	// TODO: return the error
	if err != nil {
		fmt.Println(err)
	}
}

func (d *RedisDatabase) Get(key string) string {
	// TODO: return the error if there is one
	return d.client.Get(ctx, key).Val()
}

type MemoryDatabase struct {
	dataStore map[string]string
	mux       sync.Mutex
}

func NewMemoryDatabase() *MemoryDatabase {
	m := make(map[string]string)
	return &MemoryDatabase{dataStore: m}
}

func (d *MemoryDatabase) Put(key string, value string) {
	d.mux.Lock()
	d.dataStore[key] = value
	d.mux.Unlock()
}

func (d *MemoryDatabase) Get(key string) string {
	d.mux.Lock()
	defer d.mux.Unlock()
	return d.dataStore[key]
}
