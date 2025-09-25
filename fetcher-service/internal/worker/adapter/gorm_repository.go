package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/worker/ports"
	"github.com/victoragudo/hotel-management-system/pkg/constants"
	"github.com/victoragudo/hotel-management-system/pkg/entities"
	"gorm.io/gorm"
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(database *gorm.DB) (ports.RepositoryPort, error) {
	return &GormRepository{db: database}, nil
}

func (r *GormRepository) UpsertHotel(ctx context.Context, hotel *entities.HotelData) error {
	var existingHotel entities.HotelData
	err := r.db.WithContext(ctx).Where(constants.HotelId+" = ?", hotel.HotelID).First(&existingHotel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(hotel).Error
		}
		return err
	}

	hotel.ID = existingHotel.ID
	hotel.CreatedAt = existingHotel.CreatedAt
	return r.db.WithContext(ctx).Save(hotel).Error
}

func (r *GormRepository) UpsertHotelTranslations(ctx context.Context, translations *entities.HotelTranslation) error {
	var existingTranslations entities.HotelTranslation
	err := r.db.WithContext(ctx).Where(fmt.Sprintf("%s = ? AND %s = ?", constants.HotelId, constants.Lang), translations.HotelID, translations.Lang).First(&existingTranslations).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(translations).Error
		}
		return err
	}

	translations.ID = existingTranslations.ID
	translations.CreatedAt = existingTranslations.CreatedAt
	return r.db.WithContext(ctx).Save(translations).Error
}

func (r *GormRepository) CreateReview(ctx context.Context, review *entities.ReviewData) error {
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *GormRepository) UpdateReview(ctx context.Context, review *entities.ReviewData) error {
	return r.db.WithContext(ctx).Save(review).Error
}

func (r *GormRepository) GetReviewByReviewID(ctx context.Context, reviewID int64) (*entities.ReviewData, error) {
	var e entities.ReviewData
	err := r.db.WithContext(ctx).First(&e, constants.ReviewId+" = ?", reviewID).Error
	return &e, err
}

func (r *GormRepository) GetHotelIdByPk(ctx context.Context, id string) int64 {
	var hotelId int64
	err := r.db.WithContext(ctx).Model(&entities.HotelData{}).
		Where(constants.Id+" = ?", id).
		Select(constants.HotelId).
		First(&hotelId).Error
	if err != nil {
		return 0
	}
	return hotelId
}

func (r *GormRepository) ReviewCountByHotelId(ctx context.Context, hotelId int64) int64 {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.ReviewData{}).Where("hotel_id = ?", hotelId).Count(&count)
	if err.RowsAffected == 0 {
		return 0
	}
	return count
}

func (r *GormRepository) GetHotelIdByTranslationId(ctx context.Context, id string) int64 {
	var hotelId int64
	err := r.db.WithContext(ctx).Model(&entities.HotelTranslation{}).
		Where("id = ?", id).
		Select(constants.HotelId).
		First(&hotelId).Error
	if err != nil {
		return 0
	}
	return hotelId
}

func (r *GormRepository) GetHotelIdFromReviewByPk(ctx context.Context, id string) int64 {
	var hotelId int64
	err := r.db.WithContext(ctx).Model(&entities.ReviewData{}).
		Where("id = ?", id).
		Select(constants.HotelId).
		First(&hotelId).Error
	if err != nil {
		return 0
	}
	return hotelId
}

func (r *GormRepository) GetLangById(ctx context.Context, id string) string {
	var lang string
	err := r.db.WithContext(ctx).Model(&entities.HotelTranslation{}).
		Where("id = ?", id).
		Select(constants.Lang).
		First(&lang).Error
	if err != nil {
		return ""
	}
	return lang
}
