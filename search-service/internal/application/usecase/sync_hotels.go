package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/hotel"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/search"
)

type SyncHotelsUseCase struct {
	hotelRepo    hotel.Repository
	searchEngine search.Engine
	cache        hotel.CacheRepository
	logger       *slog.Logger
}

func NewSyncHotelsUseCase(
	hotelRepo hotel.Repository,
	searchEngine search.Engine,
	cache hotel.CacheRepository,
	logger *slog.Logger,
) *SyncHotelsUseCase {
	return &SyncHotelsUseCase{
		hotelRepo:    hotelRepo,
		searchEngine: searchEngine,
		cache:        cache,
		logger:       logger,
	}
}

type SyncOptions struct {
	BatchSize        int
	FullSync         bool
	SinceTimestamp   time.Time
	ClearIndexFirst  bool
	UpdateCacheAfter bool
}

type SyncResult struct {
	TotalHotels       int
	IndexedHotels     int
	FailedHotels      int
	TotalTranslations int
	Duration          time.Duration
	StartTime         time.Time
	EndTime           time.Time
	LastSyncTime      time.Time
	Errors            []string
}

func (uc *SyncHotelsUseCase) Execute(ctx context.Context, options SyncOptions) (*SyncResult, error) {
	startTime := time.Now()

	uc.logger.Info("Starting hotel synchronization",
		"full_sync", options.FullSync,
		"batch_size", options.BatchSize,
		"clear_index_first", options.ClearIndexFirst)

	result := &SyncResult{
		StartTime: startTime,
		Errors:    make([]string, 0),
	}

	if options.BatchSize <= 0 {
		options.BatchSize = 100
	}

	if options.ClearIndexFirst {
		if err := uc.searchEngine.ClearIndex(ctx); err != nil {
			uc.logger.Error("Failed to clear search index", "error", err)
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to clear index: %v", err))
		} else {
			uc.logger.Info("Search index cleared")
		}
	}

	var hotels []*hotel.Hotel
	var err error

	if options.FullSync {
		hotels, err = uc.getAllHotels(ctx)
	} else if !options.SinceTimestamp.IsZero() {
		hotels, err = uc.hotelRepo.FindUpdatedAfter(ctx, options.SinceTimestamp)
	} else {
		since := time.Now().Add(-5 * time.Minute)
		hotels, err = uc.hotelRepo.FindUpdatedAfter(ctx, since)
	}

	if err != nil {
		uc.logger.Error("Failed to fetch hotels from database", "error", err)
		return result, fmt.Errorf("failed to fetch hotels: %w", err)
	}

	result.TotalHotels = len(hotels)
	uc.logger.Info("UpdateHotels fetched from database", "count", result.TotalHotels)

	if len(hotels) > 0 {
		result.IndexedHotels, result.FailedHotels, result.TotalTranslations = uc.indexHotelsInBatches(ctx, hotels, options.BatchSize)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.LastSyncTime = result.EndTime

	if options.UpdateCacheAfter {
		uc.updateLastSyncTime(ctx, result.LastSyncTime)
	}

	uc.logger.Info("Hotel synchronization completed",
		"total_hotels", result.TotalHotels,
		"indexed_hotels", result.IndexedHotels,
		"failed_hotels", result.FailedHotels,
		"total_translations", result.TotalTranslations,
		"duration", result.Duration,
		"errors", len(result.Errors))

	return result, nil
}

func (uc *SyncHotelsUseCase) getAllHotels(ctx context.Context) ([]*hotel.Hotel, error) {
	var allHotels []*hotel.Hotel
	limit := 1000
	offset := 0

	for {
		hotels, err := uc.hotelRepo.FindAll(ctx, limit, offset)
		if err != nil {
			return nil, err
		}

		if len(hotels) == 0 {
			break
		}

		allHotels = append(allHotels, hotels...)
		offset += len(hotels)

		uc.logger.Debug("Fetched hotels batch", "batch_size", len(hotels), "total_so_far", len(allHotels))

		if len(hotels) < limit {
			break
		}
	}

	return allHotels, nil
}

func (uc *SyncHotelsUseCase) indexHotelsInBatches(ctx context.Context, hotels []*hotel.Hotel, batchSize int) (indexed, failed, totalTranslations int) {
	for i := 0; i < len(hotels); i += batchSize {
		end := i + batchSize
		if end > len(hotels) {
			end = len(hotels)
		}

		batch := hotels[i:end]

		// Count translations in this batch
		batchTranslations := 0
		for _, h := range batch {
			batchTranslations += len(h.Translations)
		}

		uc.logger.Debug("Processing batch",
			"batch_start", i,
			"batch_size", len(batch),
			"batch_translations", batchTranslations)

		if err := uc.searchEngine.Index(ctx, batch); err != nil {
			uc.logger.Error("Failed to index batch", "batch_start", i, "batch_size", len(batch), "error", err)
			failed += len(batch)
		} else {
			uc.logger.Debug("Batch indexed successfully", "batch_start", i, "batch_size", len(batch))
			indexed += len(batch)
			totalTranslations += batchTranslations
		}

		time.Sleep(100 * time.Millisecond)
	}

	return indexed, failed, totalTranslations
}

func (uc *SyncHotelsUseCase) GetLastSyncTime(ctx context.Context) (*time.Time, error) {
	cacheKey := "last_sync_time"

	data, err := uc.cache.Get(ctx, cacheKey)
	if err != nil {
		return nil, nil // No previous sync time
	}

	var lastSyncTime time.Time
	if err := lastSyncTime.UnmarshalBinary(data); err != nil {
		uc.logger.Warn("Failed to unmarshal last sync time", "error", err)
		return nil, nil
	}

	return &lastSyncTime, nil
}

func (uc *SyncHotelsUseCase) updateLastSyncTime(ctx context.Context, syncTime time.Time) {
	cacheKey := "last_sync_time"

	data, err := syncTime.MarshalBinary()
	if err != nil {
		uc.logger.Warn("Failed to marshal sync time", "error", err)
		return
	}

	if err := uc.cache.Set(ctx, cacheKey, data, 24*time.Hour); err != nil {
		uc.logger.Warn("Failed to cache last sync time", "error", err)
	}
}

func (uc *SyncHotelsUseCase) GetSyncStats(ctx context.Context) (*search.IndexStats, error) {
	stats, err := uc.searchEngine.GetIndexStats(ctx)
	if err != nil {
		uc.logger.Error("Failed to get index stats", "error", err)
		return nil, fmt.Errorf("failed to get index stats: %w", err)
	}

	return stats, nil
}
