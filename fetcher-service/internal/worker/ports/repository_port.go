package ports

import (
	"context"

	"github.com/victoragudo/hotel-management-system/pkg/entities"
)

type RepositoryPort interface {
	UpsertHotel(ctx context.Context, hotel *entities.HotelData) error
	UpsertHotelTranslations(ctx context.Context, translations *entities.HotelTranslation) error
	CreateReview(ctx context.Context, review *entities.ReviewData) error
	UpdateReview(ctx context.Context, review *entities.ReviewData) error
	GetReviewByReviewID(ctx context.Context, reviewID int64) (*entities.ReviewData, error)
	GetHotelIdByPk(ctx context.Context, id string) int64
	ReviewCountByHotelId(ctx context.Context, hotelId int64) int64
	GetHotelIdByTranslationId(ctx context.Context, id string) int64
	GetHotelIdFromReviewByPk(ctx context.Context, id string) int64
	GetLangById(ctx context.Context, id string) string
}
