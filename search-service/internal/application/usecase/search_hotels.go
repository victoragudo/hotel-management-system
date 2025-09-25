package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/hotel"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/search"
)

type SearchHotelsUseCase struct {
	searchEngine search.Engine
	cache        hotel.CacheRepository
	logger       *slog.Logger
}

func NewSearchHotelsUseCase(
	searchEngine search.Engine,
	cache hotel.CacheRepository,
	logger *slog.Logger,
) *SearchHotelsUseCase {
	return &SearchHotelsUseCase{
		searchEngine: searchEngine,
		cache:        cache,
		logger:       logger,
	}
}

func (uc *SearchHotelsUseCase) Execute(ctx context.Context, params search.Params) (*search.Result, error) {
	startTime := time.Now()

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid search parameters: %w", err)
	}

	cacheKey := uc.generateCacheKey(params)
	if cachedResult, err := uc.cache.Get(ctx, cacheKey); err == nil {
		uc.logger.Debug("Cache hit for search", "cache_key", cacheKey)
		var result search.Result
		if err := json.Unmarshal(cachedResult, &result); err == nil {
			result.ProcessingTime = time.Since(startTime)
			return &result, nil
		}
	}

	result, err := uc.searchEngine.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search engine error: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	result.Query = params.Query
	result.Page = params.Page
	result.Limit = params.Limit
	result.CalculateTotalPages()

	if resultData, err := json.Marshal(result); err == nil {
		cacheTTL := time.Minute * 5
		if err := uc.cache.Set(ctx, cacheKey, resultData, cacheTTL); err != nil {
			uc.logger.Warn("Failed to cache search result", "error", err)
		}
	}

	return result, nil
}

func (uc *SearchHotelsUseCase) generateCacheKey(params search.Params) string {
	data, _ := json.Marshal(params)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("search:%s", hex.EncodeToString(hash[:])[:16])
}

func (uc *SearchHotelsUseCase) ExecuteWithFacets(ctx context.Context, params search.Params) (*search.Result, error) {
	result, err := uc.Execute(ctx, params)
	if err != nil {
		return nil, err
	}

	if result.Facets == nil && result.TotalHits > 0 {
		facets, err := uc.searchEngine.GetFacets(ctx)
		if err != nil {
			uc.logger.Warn("Failed to get facets", "error", err)
		} else {
			result.Facets = facets
		}
	}

	return result, nil
}

func (uc *SearchHotelsUseCase) GetPopularSearches(ctx context.Context, limit int) ([]string, error) {
	cacheKey := fmt.Sprintf("popular_searches:%d", limit)

	if cachedData, err := uc.cache.Get(ctx, cacheKey); err == nil {
		var searches []string
		if err := json.Unmarshal(cachedData, &searches); err == nil {
			return searches, nil
		}
	}

	popularSearches := []string{
		"luxury hotels",
		"beach resort",
		"city center",
		"business hotel",
		"spa hotel",
		"family hotel",
		"boutique hotel",
		"airport hotel",
	}

	if limit > 0 && limit < len(popularSearches) {
		popularSearches = popularSearches[:limit]
	}

	if data, err := json.Marshal(popularSearches); err == nil {
		_ = uc.cache.Set(ctx, cacheKey, data, time.Hour)
	}

	return popularSearches, nil
}
