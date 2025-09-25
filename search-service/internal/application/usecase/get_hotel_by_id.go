package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/victoragudo/hotel-management-system/pkg/constants"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/hotel"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/search"
)

type GetHotelByIDUseCase struct {
	hotelRepo     hotel.Repository
	hotelProvider hotel.Provider
	searchEngine  search.Engine
	cache         hotel.CacheRepository
	logger        *slog.Logger
}

func NewGetHotelByIDUseCase(
	hotelRepo hotel.Repository,
	hotelProvider hotel.Provider,
	searchEngine search.Engine,
	cache hotel.CacheRepository,
	logger *slog.Logger,
) *GetHotelByIDUseCase {
	return &GetHotelByIDUseCase{
		hotelRepo:     hotelRepo,
		hotelProvider: hotelProvider,
		searchEngine:  searchEngine,
		cache:         cache,
		logger:        logger,
	}
}

func (getHotelByIdUseCase *GetHotelByIDUseCase) Execute(ctx context.Context, hotelID int64, reviewsCount int) (*hotel.Hotel, error) {
	startTime := time.Now()

	getHotelByIdUseCase.logger.Info("Getting hotel by ID", constants.HotelId, hotelID)

	cacheKey := fmt.Sprintf("hotel:%d", hotelID)
	if cachedData, err := getHotelByIdUseCase.cache.Get(ctx, cacheKey); err == nil {
		var cachedHotel hotel.Hotel
		if err := json.Unmarshal(cachedData, &cachedHotel); err == nil {
			return &cachedHotel, nil
		}
		getHotelByIdUseCase.logger.Warn("Failed to unmarshal cached hotel", constants.HotelId, hotelID, "error", err)
	}

	foundHotel, err := getHotelByIdUseCase.hotelRepo.FindByHotelID(ctx, hotelID)
	if err == nil && foundHotel != nil {
		if hotelData, err := json.Marshal(foundHotel); err == nil {
			_ = getHotelByIdUseCase.cache.Set(ctx, cacheKey, hotelData, 5*time.Minute)
		}
		go getHotelByIdUseCase.indexHotel(*foundHotel)
		return foundHotel, nil
	}
	if err != nil {
		getHotelByIdUseCase.logger.Warn("Error querying hotel from database", constants.HotelId, hotelID, "error", err)
	}

	getHotelByIdUseCase.logger.Info("Falling back to Cupid API", constants.HotelId, hotelID)

	externalHotel, err := getHotelByIdUseCase.hotelProvider.GetHotelByID(ctx, hotelID)
	if err != nil {
		getHotelByIdUseCase.logger.Error("Failed to fetch hotel from Cupid API", constants.HotelId, hotelID, "error", err)
		return nil, fmt.Errorf("hotel not found in database and failed to fetch from external API: %w", err)
	}

	if reviews, err := getHotelByIdUseCase.hotelProvider.GetHotelReviews(ctx, hotelID, reviewsCount); err == nil {
		reviewSlice := make([]hotel.Review, len(reviews))
		for i, review := range reviews {
			reviewSlice[i] = *review
		}
		externalHotel.Reviews = reviewSlice
	} else {
		getHotelByIdUseCase.logger.Warn("Failed to fetch hotel reviews", constants.HotelId, hotelID, "error", err)
	}

	if translations, err := getHotelByIdUseCase.hotelProvider.GetHotelTranslations(ctx, hotelID, constants.Languages); err == nil {
		translationSlice := make([]hotel.Translation, len(translations))
		for i, translation := range translations {
			translationSlice[i] = *translation
		}
		externalHotel.Translations = translationSlice
	} else {
		getHotelByIdUseCase.logger.Warn("Failed to fetch hotel translations", constants.HotelId, hotelID, "error", err)
	}

	if err := getHotelByIdUseCase.hotelRepo.Save(ctx, externalHotel); err != nil {
		getHotelByIdUseCase.logger.Error("Failed to save hotel from external API to database", constants.HotelId, hotelID, "error", err)
	} else {
		getHotelByIdUseCase.logger.Info("Hotel saved to database from external API", constants.HotelId, hotelID)
	}

	// Indexing in meilisearch is not relevant to the response API in a hotelById request, so we parallelize
	go getHotelByIdUseCase.indexHotel(*externalHotel)

	if hotelData, err := json.Marshal(externalHotel); err == nil {
		err = getHotelByIdUseCase.cache.Set(ctx, cacheKey, hotelData, 5*time.Minute)
		if err != nil {
			getHotelByIdUseCase.logger.Error("Failed to set hotel cache", "hotel_id", hotelID, "error", err)
		}
	}
	getHotelByIdUseCase.logger.Info("Hotel fetched from external API", "hotel_id", hotelID, "duration", time.Since(startTime))
	return externalHotel, nil
}

func (getHotelByIdUseCase *GetHotelByIDUseCase) indexHotel(h hotel.Hotel) {
	indexCtx, cancel := context.WithTimeout(context.Background(), time.Duration(3)*time.Minute)
	defer cancel()
	if err := getHotelByIdUseCase.searchEngine.UpdateHotel(indexCtx, &h); err != nil {
		getHotelByIdUseCase.logger.Error("Failed to index hotel in search engine", "hotel_id", h.HotelID, "error", err)
	} else {
		getHotelByIdUseCase.logger.Info("Hotel indexed in search engine", "hotel_id", h.HotelID)
	}
}
