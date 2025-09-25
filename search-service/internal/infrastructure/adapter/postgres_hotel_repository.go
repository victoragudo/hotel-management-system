package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/victoragudo/hotel-management-system/pkg/entities"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/hotel"
	"gorm.io/gorm"
)

const HOTEL_ID = "hotel_id"

type PostgresHotelRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewPostgresHotelRepository(db *gorm.DB, logger *slog.Logger) *PostgresHotelRepository {
	return &PostgresHotelRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PostgresHotelRepository) FindByHotelID(ctx context.Context, hotelID int64) (*hotel.Hotel, error) {
	var hotelModel entities.HotelData

	err := r.db.WithContext(ctx).
		Preload("ReviewsData").
		Preload("TranslationsData").
		Where(HOTEL_ID+" = ?", hotelID).First(&hotelModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to find hotel by hotel ID", "hotel_id", hotelID, "error", err)
		return nil, fmt.Errorf("failed to find hotel by hotel ID %d: %w", hotelID, err)
	}

	return r.convertModelToDomain(&hotelModel)
}

func (r *PostgresHotelRepository) Save(ctx context.Context, h *hotel.Hotel) error {
	hotelModel, err := r.convertDomainToModel(h)
	if err != nil {
		return fmt.Errorf("failed to convert domain to model: %w", err)
	}

	now := time.Now()
	hotelModel.CreatedAt = now
	hotelModel.UpdatedAt = now

	if err := r.db.WithContext(ctx).Create(hotelModel).Error; err != nil {
		r.logger.Error("Failed to save hotel", "hotel_id", h.HotelID, "error", err)
		return fmt.Errorf("failed to save hotel %d: %w", h.HotelID, err)
	}
	h.ID = hotelModel.ID
	r.logger.Debug("Hotel saved successfully", "hotel_id", h.HotelID)
	return nil
}

func (r *PostgresHotelRepository) Update(ctx context.Context, h *hotel.Hotel) error {
	hotelModel, err := r.convertDomainToModel(h)
	if err != nil {
		return fmt.Errorf("failed to convert domain to model: %w", err)
	}

	now := time.Now()
	hotelModel.UpdatedAt = now

	if err := r.db.WithContext(ctx).Save(hotelModel).Error; err != nil {
		r.logger.Error("Failed to update hotel", "hotel_id", h.HotelID, "error", err)
		return fmt.Errorf("failed to update hotel %d: %w", h.HotelID, err)
	}

	r.logger.Debug("Hotel updated successfully", "hotel_id", h.HotelID)
	return nil
}

func (r *PostgresHotelRepository) FindAll(ctx context.Context, limit, offset int) ([]*hotel.Hotel, error) {
	var hotelModels []entities.HotelData

	query := r.db.WithContext(ctx).
		Preload("ReviewsData").
		Preload("TranslationsData").
		Where("status = ?", "active")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&hotelModels).Error
	if err != nil {
		r.logger.Error("Failed to find hotels", "error", err)
		return nil, fmt.Errorf("failed to find hotels: %w", err)
	}

	hotels := make([]*hotel.Hotel, len(hotelModels))
	for i, model := range hotelModels {
		if h, err := r.convertModelToDomain(&model); err == nil {
			hotels[i] = h
		} else {
			r.logger.Warn("Failed to convert hotel model to domain", "hotel_id", model.HotelID, "error", err)
		}
	}

	return hotels, nil
}

func (r *PostgresHotelRepository) FindUpdatedAfter(ctx context.Context, timestamp time.Time) ([]*hotel.Hotel, error) {
	var hotelModels []entities.HotelData

	err := r.db.WithContext(ctx).
		Preload("TranslationsData").
		Where("updated_at > ? AND status = ?", timestamp, "active").
		Order("updated_at ASC").
		Find(&hotelModels).Error
	if err != nil {
		r.logger.Error("Failed to find updated hotels", "timestamp", timestamp, "error", err)
		return nil, fmt.Errorf("failed to find hotels updated after %v: %w", timestamp, err)
	}

	hotels := make([]*hotel.Hotel, len(hotelModels))
	for i, model := range hotelModels {
		if h, err := r.convertModelToDomain(&model); err == nil {
			hotels[i] = h
		} else {
			r.logger.Warn("Failed to convert hotel model to domain", "hotel_id", model.HotelID, "error", err)
		}
	}

	return hotels, nil
}

func (r *PostgresHotelRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&entities.HotelData{}).Error
	if err != nil {
		r.logger.Error("Failed to delete hotel", "id", id, "error", err)
		return fmt.Errorf("failed to delete hotel %s: %w", id, err)
	}

	r.logger.Debug("Hotel deleted successfully", "id", id)
	return nil
}

func (r *PostgresHotelRepository) convertModelToDomain(model *entities.HotelData) (*hotel.Hotel, error) {
	h := &hotel.Hotel{
		ID:                  model.ID,
		HotelID:             model.HotelID,
		CupidID:             model.CupidID,
		Name:                model.Name,
		Description:         model.Description,
		Rating:              model.Rating,
		StarRating:          model.StarRating,
		Location:            hotel.Location{Latitude: model.Latitude, Longitude: model.Longitude},
		Status:              model.Status,
		Source:              model.Source,
		MainImageTh:         model.MainImageTh,
		HotelType:           model.HotelType,
		Chain:               model.Chain,
		ChainID:             model.ChainID,
		Phone:               model.Phone,
		Fax:                 model.Fax,
		Email:               model.Email,
		AirportCode:         model.AirportCode,
		ReviewCount:         model.ReviewCount,
		Parking:             model.Parking,
		ChildAllowed:        model.ChildAllowed,
		PetsAllowed:         model.PetsAllowed,
		MarkdownDescription: model.MarkdownDescription,
		ImportantInfo:       model.ImportantInfo,
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
		NextUpdateAt:        model.NextUpdateAt,
	}

	if len(model.Address) > 0 {
		var address hotel.Address
		if err := json.Unmarshal(model.Address, &address); err == nil {
			h.Address = address
		}
	}

	if len(model.Amenities) > 0 {
		var amenities []string
		if err := json.Unmarshal(model.Amenities, &amenities); err == nil {
			h.Amenities = amenities
		}
	}

	if len(model.Policies) > 0 {
		var policies []hotel.Policy
		if err := json.Unmarshal(model.Policies, &policies); err == nil {
			h.Policies = policies
		}
	}

	if len(model.ContactInfo) > 0 {
		var contactInfo hotel.ContactInfo
		if err := json.Unmarshal(model.ContactInfo, &contactInfo); err == nil {
			h.ContactInfo = contactInfo
		}
	}

	if len(model.Checkin) > 0 {
		var checkinInfo hotel.CheckinInfo
		if err := json.Unmarshal(model.Checkin, &checkinInfo); err == nil {
			h.CheckinInfo = checkinInfo
		}
	}

	if len(model.Photos) > 0 {
		var photos []hotel.Photo
		if err := json.Unmarshal(model.Photos, &photos); err == nil {
			h.Photos = photos
		}
	}

	if len(model.Facilities) > 0 {
		var facilities []hotel.Facility
		if err := json.Unmarshal(model.Facilities, &facilities); err == nil {
			h.Facilities = facilities
		}
	}

	if len(model.Rooms) > 0 {
		var rooms []hotel.Room
		if err := json.Unmarshal(model.Rooms, &rooms); err == nil {
			h.Rooms = rooms
		}
	}

	if len(model.ReviewsData) > 0 {
		var reviews []hotel.Review

		for _, reviewData := range model.ReviewsData {
			reviews = append(reviews, hotel.Review{
				ID:           reviewData.ID,
				HotelID:      reviewData.HotelID,
				ReviewID:     reviewData.ReviewID,
				AverageScore: reviewData.AverageScore,
				Country:      reviewData.Country,
				Type:         reviewData.Type,
				Name:         reviewData.Name,
				Date:         reviewData.Date,
				Headline:     reviewData.Headline,
				Language:     reviewData.Language,
				Pros:         reviewData.Pros,
				Cons:         reviewData.Cons,
				Source:       reviewData.Source,
			})
		}
		h.Reviews = reviews
	}

	if len(model.TranslationsData) > 0 {
		var translations []hotel.Translation

		for _, translationData := range model.TranslationsData {
			translation := hotel.Translation{
				ID:                  translationData.ID,
				HotelID:             translationData.HotelID,
				Name:                translationData.Name,
				Description:         translationData.Description,
				Status:              translationData.Status,
				Source:              translationData.Source,
				Chain:               translationData.Chain,
				Parking:             translationData.Parking,
				MarkdownDescription: translationData.MarkdownDescription,
				ImportantInfo:       translationData.ImportantInfo,
				CreatedAt:           translationData.CreatedAt,
				UpdatedAt:           translationData.UpdatedAt,
				NextUpdateAt:        translationData.NextUpdateAt,
				Lang:                translationData.Lang,
			}

			if len(translationData.Address) > 0 {
				var address hotel.Address
				if err := json.Unmarshal(translationData.Address, &address); err == nil {
					translation.Address = address
				}
			}

			if len(translationData.Policies) > 0 {
				var policies []hotel.Policy
				if err := json.Unmarshal(translationData.Policies, &policies); err == nil {
					translation.Policies = policies
				}
			}

			if len(translationData.ContactInfo) > 0 {
				var contactInfo hotel.ContactInfo
				if err := json.Unmarshal(translationData.ContactInfo, &contactInfo); err == nil {
					translation.ContactInfo = contactInfo
				}
			}

			if len(translationData.Checkin) > 0 {
				var checkinInfo hotel.CheckinInfo
				if err := json.Unmarshal(translationData.Checkin, &checkinInfo); err == nil {
					translation.CheckinInfo = checkinInfo
				}
			}

			if len(translationData.Photos) > 0 {
				var photos []hotel.Photo
				if err := json.Unmarshal(translationData.Photos, &photos); err == nil {
					translation.Photos = photos
				}
			}

			if len(translationData.Facilities) > 0 {
				var facilities []hotel.Facility
				if err := json.Unmarshal(translationData.Facilities, &facilities); err == nil {
					translation.Facilities = facilities
				}
			}

			if len(translationData.Rooms) > 0 {
				var rooms []hotel.Room
				if err := json.Unmarshal(translationData.Rooms, &rooms); err == nil {
					translation.Rooms = rooms
				}
			}

			translations = append(translations, translation)
		}
		h.Translations = translations
	}

	return h, nil
}

func (r *PostgresHotelRepository) convertDomainToModel(h *hotel.Hotel) (*entities.HotelData, error) {
	model := &entities.HotelData{
		ID:                  h.ID,
		HotelID:             h.HotelID,
		CupidID:             h.CupidID,
		Name:                h.Name,
		Description:         h.Description,
		Rating:              h.Rating,
		StarRating:          h.StarRating,
		Latitude:            h.Location.Latitude,
		Longitude:           h.Location.Longitude,
		Status:              h.Status,
		Source:              h.Source,
		MainImageTh:         h.MainImageTh,
		HotelType:           h.HotelType,
		Chain:               h.Chain,
		ChainID:             h.ChainID,
		Phone:               h.Phone,
		Fax:                 h.Fax,
		Email:               h.Email,
		AirportCode:         h.AirportCode,
		ReviewCount:         h.ReviewCount,
		Parking:             h.Parking,
		ChildAllowed:        h.ChildAllowed,
		PetsAllowed:         h.PetsAllowed,
		MarkdownDescription: h.MarkdownDescription,
		ImportantInfo:       h.ImportantInfo,
		CreatedAt:           h.CreatedAt,
		UpdatedAt:           h.UpdatedAt,
		NextUpdateAt:        h.NextUpdateAt,
	}

	if addressJSON, err := json.Marshal(h.Address); err == nil {
		model.Address = addressJSON
	}

	if amenitiesJSON, err := json.Marshal(h.Amenities); err == nil {
		model.Amenities = amenitiesJSON
	}

	if policiesJSON, err := json.Marshal(h.Policies); err == nil {
		model.Policies = policiesJSON
	}

	if contactInfoJSON, err := json.Marshal(h.ContactInfo); err == nil {
		model.ContactInfo = contactInfoJSON
	}

	if checkinInfoJSON, err := json.Marshal(h.CheckinInfo); err == nil {
		model.Checkin = checkinInfoJSON
	}

	if photosJSON, err := json.Marshal(h.Photos); err == nil {
		model.Photos = photosJSON
	}

	if facilitiesJSON, err := json.Marshal(h.Facilities); err == nil {
		model.Facilities = facilitiesJSON
	}

	if roomsJSON, err := json.Marshal(h.Rooms); err == nil {
		model.Rooms = roomsJSON
	}

	return model, nil
}
