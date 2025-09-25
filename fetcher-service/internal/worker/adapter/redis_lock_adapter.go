package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/worker/ports"
)

type RedisLockAdapter struct {
	client *redis.Client
}

func NewRedisLockAdapter(addr, password string, db int) ports.LockPort {
	c := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db, PoolSize: 50})
	return &RedisLockAdapter{client: c}
}

func (r *RedisLockAdapter) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	ok, err := r.client.SetNX(ctx, key, fmt.Sprintf("%d", time.Now().UnixNano()), ttl).Result()
	return ok, err
}

func (r *RedisLockAdapter) Release(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisLockAdapter) Close() error {
	return r.client.Close()
}
