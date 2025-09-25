package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/worker/ports"
)

type RedisCacheAdapter struct {
	client *redis.Client
}

func NewRedisCacheAdapter(addr, password string, db int) ports.CachePort {
	c := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db, PoolSize: 50})
	return &RedisCacheAdapter{client: c}
}

func (r *RedisCacheAdapter) Get(ctx context.Context, key string, dest any) (bool, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(val, dest)
}

func (r *RedisCacheAdapter) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, b, ttl).Err()
}

func (r *RedisCacheAdapter) Close() error {
	return r.client.Close()
}
