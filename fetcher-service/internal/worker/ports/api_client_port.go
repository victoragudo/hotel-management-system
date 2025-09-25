package ports

import (
	"context"

	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/worker/dto"
)

type APIClientPort interface {
	FetchHotelData(ctx context.Context, hotelId int64) (*dto.HotelAPIResponse, error)
	FetchHotelReviews(ctx context.Context, hotelID int64, options *dto.ReviewFetchOptions) (*dto.ReviewDataList, error)
	FetchTranslations(ctx context.Context, hotelID string, options *dto.TranslationFetchOptions) (*dto.TranslationAPIResponse, error)
}
