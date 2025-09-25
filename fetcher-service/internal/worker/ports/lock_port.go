package ports

import (
	"context"
	"time"
)

type LockPort interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key string) error
	Close() error
}
