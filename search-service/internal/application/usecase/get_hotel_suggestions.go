package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/hotel"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/search"
)

type GetHotelSuggestionsUseCase struct {
	searchEngine search.Engine
	cache        hotel.CacheRepository
	logger       *slog.Logger
}

func NewGetHotelSuggestionsUseCase(
	searchEngine search.Engine,
	cache hotel.CacheRepository,
	logger *slog.Logger,
) *GetHotelSuggestionsUseCase {
	return &GetHotelSuggestionsUseCase{
		searchEngine: searchEngine,
		cache:        cache,
		logger:       logger,
	}
}

func (uc *GetHotelSuggestionsUseCase) Execute(ctx context.Context, query string, limit int) ([]*search.Suggestion, error) {
	startTime := time.Now()

	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	uc.logger.Debug("Getting hotel suggestions", "query", query, "limit", limit)

	cacheKey := fmt.Sprintf("suggestions:%s:%d", query, limit)

	if cachedData, err := uc.cache.Get(ctx, cacheKey); err == nil {
		var cachedSuggestions []*search.Suggestion
		if err := json.Unmarshal(cachedData, &cachedSuggestions); err == nil {
			uc.logger.Debug("Suggestions found in cache",
				"query", query,
				"count", len(cachedSuggestions),
				"duration", time.Since(startTime))
			return cachedSuggestions, nil
		}
		uc.logger.Warn("Failed to unmarshal cached suggestions", "error", err)
	}

	suggestions, err := uc.searchEngine.GetSuggestions(ctx, query, limit)
	if err != nil {
		uc.logger.Error("Failed to get suggestions from search engine", "error", err)
		return nil, fmt.Errorf("failed to get suggestions: %w", err)
	}

	if suggestionsData, err := json.Marshal(suggestions); err == nil {
		if err := uc.cache.Set(ctx, cacheKey, suggestionsData, 30*time.Minute); err != nil {
			uc.logger.Warn("Failed to cache suggestions", "error", err)
		}
	}

	uc.logger.Info("Hotel suggestions retrieved",
		"query", query,
		"count", len(suggestions),
		"duration", time.Since(startTime))

	return suggestions, nil
}

func (uc *GetHotelSuggestionsUseCase) GetTrendingSuggestions(ctx context.Context, limit int) ([]*search.Suggestion, error) {
	if limit <= 0 {
		limit = 10
	}

	cacheKey := fmt.Sprintf("trending_suggestions:%d", limit)

	if cachedData, err := uc.cache.Get(ctx, cacheKey); err == nil {
		var cachedSuggestions []*search.Suggestion
		if err := json.Unmarshal(cachedData, &cachedSuggestions); err == nil {
			return cachedSuggestions, nil
		}
	}

	trendingSuggestions := []*search.Suggestion{
		{Text: "luxury hotels", Type: "category", Score: 0.95},
		{Text: "beach resorts", Type: "category", Score: 0.90},
		{Text: "city center hotels", Type: "location", Score: 0.85},
		{Text: "spa hotels", Type: "amenity", Score: 0.80},
		{Text: "business hotels", Type: "category", Score: 0.75},
		{Text: "family hotels", Type: "category", Score: 0.70},
		{Text: "boutique hotels", Type: "category", Score: 0.65},
		{Text: "airport hotels", Type: "location", Score: 0.60},
		{Text: "mountain resorts", Type: "location", Score: 0.55},
		{Text: "pet-friendly hotels", Type: "amenity", Score: 0.50},
	}

	if limit < len(trendingSuggestions) {
		trendingSuggestions = trendingSuggestions[:limit]
	}

	if data, err := json.Marshal(trendingSuggestions); err == nil {
		_ = uc.cache.Set(ctx, cacheKey, data, 2*time.Hour)
	}

	return trendingSuggestions, nil
}

func (uc *GetHotelSuggestionsUseCase) GetLocationSuggestions(ctx context.Context, query string, limit int) ([]*search.Suggestion, error) {
	if limit <= 0 {
		limit = 10
	}

	locationSuggestions := []*search.Suggestion{
		{Text: "New York", Type: "city", Score: 0.95, Metadata: map[string]interface{}{"country": "USA"}},
		{Text: "London", Type: "city", Score: 0.90, Metadata: map[string]interface{}{"country": "UK"}},
		{Text: "Paris", Type: "city", Score: 0.85, Metadata: map[string]interface{}{"country": "France"}},
		{Text: "Tokyo", Type: "city", Score: 0.80, Metadata: map[string]interface{}{"country": "Japan"}},
		{Text: "Dubai", Type: "city", Score: 0.75, Metadata: map[string]interface{}{"country": "UAE"}},
	}

	if query != "" {
		var filtered []*search.Suggestion
		for _, suggestion := range locationSuggestions {
			if len(suggestion.Text) >= len(query) &&
				suggestion.Text[:len(query)] == query {
				filtered = append(filtered, suggestion)
			}
		}
		locationSuggestions = filtered
	}

	if limit < len(locationSuggestions) {
		locationSuggestions = locationSuggestions[:limit]
	}

	return locationSuggestions, nil
}
