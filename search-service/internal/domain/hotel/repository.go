package hotel

import (
	"context"
	"time"
)

type Repository interface {
	FindByHotelID(ctx context.Context, hotelID int64) (*Hotel, error)
	Save(ctx context.Context, hotel *Hotel) error
	Update(ctx context.Context, hotel *Hotel) error
	FindAll(ctx context.Context, limit, offset int) ([]*Hotel, error)
	FindUpdatedAfter(ctx context.Context, timestamp time.Time) ([]*Hotel, error)
	Delete(ctx context.Context, id string) error
}

type Provider interface {
	GetHotelByID(ctx context.Context, hotelID int64) (*Hotel, error)
	GetHotelReviews(ctx context.Context, hotelID int64, reviewsCount int) ([]*Review, error)
	GetHotelTranslations(ctx context.Context, hotelID int64, languages []string) ([]*Translation, error)
}

type CacheRepository interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
