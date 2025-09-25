package ports

import (
	"context"
	"time"
)

type CachePort interface {
	Get(ctx context.Context, key string, dest any) (bool, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Close() error
}
