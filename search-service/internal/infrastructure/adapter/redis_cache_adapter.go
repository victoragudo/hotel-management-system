package adapter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCacheAdapter struct {
	client *redis.Client
	logger *slog.Logger
	prefix string
}

func NewRedisCacheAdapterWithClient(client *redis.Client, logger *slog.Logger) *RedisCacheAdapter {
	return &RedisCacheAdapter{
		client: client,
		logger: logger,
		prefix: "search-service:",
	}
}

func (r *RedisCacheAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := r.prefix + key

	result, err := r.client.Get(ctx, fullKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			r.logger.Debug("Cache miss", "key", key)
			return nil, fmt.Errorf("cache miss for key %s", key)
		}
		r.logger.Error("Failed to get from cache", "key", key, "error", err)
		return nil, fmt.Errorf("cache get error for key %s: %w", key, err)
	}

	r.logger.Debug("Cache hit", "key", key, "size", len(result))
	return []byte(result), nil
}

func (r *RedisCacheAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := r.prefix + key

	err := r.client.Set(ctx, fullKey, value, ttl).Err()
	if err != nil {
		r.logger.Error("Failed to set cache", "key", key, "ttl", ttl, "error", err)
		return fmt.Errorf("cache set error for key %s: %w", key, err)
	}

	r.logger.Debug("Cache set", "key", key, "ttl", ttl, "size", len(value))
	return nil
}

func (r *RedisCacheAdapter) Delete(ctx context.Context, key string) error {
	fullKey := r.prefix + key

	result, err := r.client.Del(ctx, fullKey).Result()
	if err != nil {
		r.logger.Error("Failed to delete from cache", "key", key, "error", err)
		return fmt.Errorf("cache delete error for key %s: %w", key, err)
	}

	r.logger.Debug("Cache delete", "key", key, "deleted_count", result)
	return nil
}

func (r *RedisCacheAdapter) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.prefix + key

	result, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		r.logger.Error("Failed to check cache existence", "key", key, "error", err)
		return false, fmt.Errorf("cache exists error for key %s: %w", key, err)
	}

	exists := result > 0
	r.logger.Debug("Cache exists check", "key", key, "exists", exists)
	return exists, nil
}

func (r *RedisCacheAdapter) SetWithExpiration(ctx context.Context, key string, value []byte, expiration time.Time) error {
	fullKey := r.prefix + key

	err := r.client.SetEx(ctx, fullKey, value, time.Until(expiration)).Err()
	if err != nil {
		r.logger.Error("Failed to set cache with expiration", "key", key, "expiration", expiration, "error", err)
		return fmt.Errorf("cache set with expiration error for key %s: %w", key, err)
	}

	r.logger.Debug("Cache set with expiration", "key", key, "expiration", expiration, "size", len(value))
	return nil
}

func (r *RedisCacheAdapter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := r.prefix + key

	ttl, err := r.client.TTL(ctx, fullKey).Result()
	if err != nil {
		r.logger.Error("Failed to get TTL", "key", key, "error", err)
		return 0, fmt.Errorf("cache TTL error for key %s: %w", key, err)
	}

	r.logger.Debug("Cache TTL", "key", key, "ttl", ttl)
	return ttl, nil
}

func (r *RedisCacheAdapter) Increment(ctx context.Context, key string) (int64, error) {
	fullKey := r.prefix + key

	result, err := r.client.Incr(ctx, fullKey).Result()
	if err != nil {
		r.logger.Error("Failed to increment counter", "key", key, "error", err)
		return 0, fmt.Errorf("cache increment error for key %s: %w", key, err)
	}

	r.logger.Debug("Cache increment", "key", key, "value", result)
	return result, nil
}

func (r *RedisCacheAdapter) IncrementWithExpiration(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	fullKey := r.prefix + key

	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Error("Failed to increment with expiration", "key", key, "error", err)
		return 0, fmt.Errorf("cache increment with expiration error for key %s: %w", key, err)
	}

	result := incrCmd.Val()
	r.logger.Debug("Cache increment with expiration", "key", key, "value", result, "ttl", ttl)
	return result, nil
}

func (r *RedisCacheAdapter) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.prefix + key
	}

	values, err := r.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		r.logger.Error("Failed to get multiple keys", "keys", keys, "error", err)
		return nil, fmt.Errorf("cache mget error: %w", err)
	}

	result := make(map[string][]byte)
	for i, value := range values {
		if value != nil {
			if strValue, ok := value.(string); ok {
				result[keys[i]] = []byte(strValue)
			}
		}
	}

	r.logger.Debug("Cache multiple get", "requested", len(keys), "found", len(result))
	return result, nil
}

func (r *RedisCacheAdapter) SetMultiple(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()

	for key, value := range items {
		fullKey := r.prefix + key
		pipe.Set(ctx, fullKey, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Error("Failed to set multiple keys", "count", len(items), "error", err)
		return fmt.Errorf("cache mset error: %w", err)
	}

	r.logger.Debug("Cache multiple set", "count", len(items), "ttl", ttl)
	return nil
}

func (r *RedisCacheAdapter) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := r.prefix + pattern

	keys, err := r.client.Keys(ctx, fullPattern).Result()
	if err != nil {
		r.logger.Error("Failed to get keys for pattern", "pattern", pattern, "error", err)
		return fmt.Errorf("cache keys error for pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		r.logger.Debug("No keys found for pattern", "pattern", pattern)
		return nil
	}

	result, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		r.logger.Error("Failed to delete pattern keys", "pattern", pattern, "keys_count", len(keys), "error", err)
		return fmt.Errorf("cache delete pattern error for %s: %w", pattern, err)
	}

	r.logger.Info("Cache pattern delete", "pattern", pattern, "deleted_count", result)
	return nil
}

func (r *RedisCacheAdapter) Ping(ctx context.Context) error {
	_, err := r.client.Ping(ctx).Result()
	if err != nil {
		r.logger.Error("Redis ping failed", "error", err)
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

func (r *RedisCacheAdapter) Close() error {
	return r.client.Close()
}

func (r *RedisCacheAdapter) GetStats(ctx context.Context) (map[string]string, error) {
	_, err := r.client.Info(ctx, "stats", "memory").Result()
	if err != nil {
		r.logger.Error("Failed to get Redis stats", "error", err)
		return nil, fmt.Errorf("redis info error: %w", err)
	}

	stats := map[string]string{
		"status": "connected",
	}

	return stats, nil
}
