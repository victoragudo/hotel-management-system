package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/victoragudo/hotel-management-system/search-service/internal/application/usecase"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/search"
)

type HotelHandler struct {
	getHotelByIDUseCase        *usecase.GetHotelByIDUseCase
	searchHotelsUseCase        *usecase.SearchHotelsUseCase
	getHotelSuggestionsUseCase *usecase.GetHotelSuggestionsUseCase
	syncHotelsUseCase          *usecase.SyncHotelsUseCase
	logger                     *slog.Logger
}

func NewHotelHandler(
	getHotelByIDUseCase *usecase.GetHotelByIDUseCase,
	searchHotelsUseCase *usecase.SearchHotelsUseCase,
	getHotelSuggestionsUseCase *usecase.GetHotelSuggestionsUseCase,
	syncHotelsUseCase *usecase.SyncHotelsUseCase,
	logger *slog.Logger,
) *HotelHandler {
	return &HotelHandler{
		getHotelByIDUseCase:        getHotelByIDUseCase,
		searchHotelsUseCase:        searchHotelsUseCase,
		getHotelSuggestionsUseCase: getHotelSuggestionsUseCase,
		syncHotelsUseCase:          syncHotelsUseCase,
		logger:                     logger,
	}
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// GetHotelByID retrieves a hotel by its ID
// @Summary Get hotel by ID
// @Description Get detailed information about a specific hotel by its ID with optional reviews limit
// @Tags hotels
// @Accept json
// @Produce json
// @Param id path integer true "Hotel ID"
// @Param reviewsLimit query integer false "Limit the number of reviews to return" minimum(1)
// @Success 200 {object} APIResponse "Hotel details"
// @Failure 400 {object} APIResponse "Bad Request - Invalid parameters"
// @Failure 404 {object} APIResponse "Not Found - Hotel not found"
// @Failure 500 {object} APIResponse "Internal Server Error"
// @Router /api/v1/hotels/{id} [get]
func (h *HotelHandler) GetHotelByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	hotelID := vars["id"]
	if hotelID == "" {
		h.writeErrorResponse(w, "hotel ID is required", http.StatusBadRequest)
		return
	}

	hotelIDInt, err := strconv.ParseInt(hotelID, 10, 64)
	if err != nil {
		h.logger.Error("Failed to convert hotel_id to int", "hotel_id", hotelID)
		h.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	queryParams := r.URL.Query()
	reviewsLimitStr := queryParams.Get("reviewsLimit")
	var reviewsCountInt int
	if reviewsLimitStr != "" {
		reviewsCountInt, err = strconv.Atoi(reviewsLimitStr)
		if err != nil {
			h.logger.Error("Failed to convert hotel_id to int", "hotel_id", hotelID)
			h.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	hotel, err := h.getHotelByIDUseCase.Execute(r.Context(), hotelIDInt, reviewsCountInt)
	if err != nil {
		h.logger.Error("Failed to get hotel by ID", "hotel_id", hotelID, "error", err)
		h.writeErrorResponse(w, err.Error(), http.StatusNotFound)
		return
	}

	h.writeSuccessResponse(w, hotel, nil)
}

// SearchHotels searches for hotels based on various criteria
// @Summary Search hotels
// @Description Search for hotels using various filters based on TypesenseDocument fields
// @Tags search
// @Accept json
// @Produce json
// @Param q query string false "Search query for hotel name, description, or location"
// @Param name query string false "Filter by hotel name"
// @Param description query string false "Filter by hotel description"
// @Param phone query string false "Filter by hotel phone number"
// @Param chain query string false "Filter by hotel chain"
// @Param email query string false "Filter by hotel email"
// @Param fax query string false "Filter by hotel fax number"
// @Param airport_code query string false "Filter by airport code"
// @Param parking query string false "Filter by parking information"
// @Param city query string false "Filter by city"
// @Param country query string false "Filter by country"
// @Param rating_min query number false "Minimum rating (0-5)"
// @Param rating_max query number false "Maximum rating (0-5)"
// @Param star_rating query integer false "Minimum star rating (1-5)"
// @Param review_count query integer false "Filter by review count"
// @Param child_allowed query boolean false "Filter by child allowed status"
// @Param pets_allowed query boolean false "Filter by pets allowed status"
// @Param amenities query array false "Filter by amenities" collectionFormat(multi)
// @Param tags query array false "Filter by tags" collectionFormat(multi)
// @Param price_min query number false "Minimum price"
// @Param price_max query number false "Maximum price"
// @Param currency query string false "Price currency (e.g., USD, EUR)"
// @Param sort_by query string false "Sort by field (rating, price, distance, relevance, name, created_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Param page query integer false "Page number (default: 1)"
// @Param limit query integer false "Results per page (max: 100, default: 20)"
// @Param latitude query number false "Latitude for location-based search"
// @Param longitude query number false "Longitude for location-based search"
// @Param radius query number false "Search radius in kilometers"
// @Success 200 {object} APIResponse{data=[]hotel.Hotel,meta=object} "Search results with hotels and pagination"
// @Failure 400 {object} APIResponse "Bad Request - Invalid search parameters"
// @Failure 500 {object} APIResponse "Internal Server Error"
// @Router /api/v1/search/hotels [get]
func (h *HotelHandler) SearchHotels(w http.ResponseWriter, r *http.Request) {
	params := h.parseSearchParams(r)

	result, err := h.searchHotelsUseCase.Execute(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to search hotels", "error", err)
		h.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	meta := map[string]interface{}{
		"total_hits":      result.TotalHits,
		"page":            result.Page,
		"limit":           result.Limit,
		"total_pages":     result.TotalPages,
		"processing_time": result.ProcessingTime.String(),
		"query":           result.Query,
	}

	if result.Facets != nil {
		meta["facets"] = result.Facets
	}

	h.writeSuccessResponse(w, result.Hotels, meta)
}

// GetHotelSuggestions provides search suggestions based on query input
// @Summary Get hotel search suggestions
// @Description Get autocomplete suggestions for hotel search based on partial query input
// @Tags search
// @Accept json
// @Produce json
// @Param q query string true "Search query for suggestions"
// @Param limit query integer false "Maximum number of suggestions to return (default: 10)"
// @Success 200 {object} APIResponse "List of search suggestions"
// @Failure 400 {object} APIResponse "Bad Request - Query parameter is required"
// @Failure 500 {object} APIResponse "Internal Server Error"
// @Router /api/v1/search/suggestions [get]
func (h *HotelHandler) GetHotelSuggestions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.writeErrorResponse(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	h.logger.Debug("Getting suggestions", "query", query, "limit", limit)

	suggestions, err := h.getHotelSuggestionsUseCase.Execute(r.Context(), query, limit)
	if err != nil {
		h.logger.Error("Failed to get suggestions", "query", query, "error", err)
		h.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.writeSuccessResponse(w, suggestions, nil)
}

// GetFacets returns available search facets for filtering
// @Summary Get search facets
// @Description Get available facets for filtering hotel search results (cities, countries, star ratings, amenities, etc.)
// @Tags search
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse "Available search facets with counts"
// @Failure 500 {object} APIResponse "Internal Server Error"
// @Router /api/v1/search/facets [get]
func (h *HotelHandler) GetFacets(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Getting search facets")

	facets := &search.Facets{
		Cities: []search.FacetItem{
			{Value: "New York", Count: 150},
			{Value: "London", Count: 120},
			{Value: "Paris", Count: 98},
			{Value: "Tokyo", Count: 87},
			{Value: "Dubai", Count: 76},
		},
		Countries: []search.FacetItem{
			{Value: "United States", Count: 300},
			{Value: "United Kingdom", Count: 250},
			{Value: "France", Count: 200},
			{Value: "Japan", Count: 150},
			{Value: "UAE", Count: 100},
		},
		StarRatings: []search.FacetItem{
			{Value: "5", Count: 45},
			{Value: "4", Count: 123},
			{Value: "3", Count: 167},
			{Value: "2", Count: 89},
			{Value: "1", Count: 34},
		},
		Amenities: []search.FacetItem{
			{Value: "wifi", Count: 890},
			{Value: "pool", Count: 456},
			{Value: "gym", Count: 334},
			{Value: "spa", Count: 223},
			{Value: "restaurant", Count: 567},
			{Value: "parking", Count: 445},
			{Value: "pets allowed", Count: 156},
		},
	}

	h.writeSuccessResponse(w, facets, nil)
}

type CustomSyncOptions struct {
	usecase.SyncOptions
}

func (c *CustomSyncOptions) UnmarshalJSON(data []byte) error {
	type Alias struct {
		FullSync         bool            `json:"fullSync"`
		BatchSize        int             `json:"batchSize"`
		UpdateCacheAfter bool            `json:"updateCacheAfter"`
		SinceTimestamp   json.RawMessage `json:"sinceTimestamp"`
	}

	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.FullSync = aux.FullSync
	c.BatchSize = aux.BatchSize
	c.UpdateCacheAfter = aux.UpdateCacheAfter

	defaultTime := time.Now().AddDate(0, -1, 0)

	if len(aux.SinceTimestamp) == 0 || string(aux.SinceTimestamp) == "null" {
		c.SinceTimestamp = defaultTime
		return nil
	}

	var timeStr string
	if err := json.Unmarshal(aux.SinceTimestamp, &timeStr); err == nil {
		parsedTime, err := parseTimestamp(timeStr)
		if err != nil {
			c.SinceTimestamp = defaultTime
		} else {
			c.SinceTimestamp = parsedTime
		}
		return nil
	}

	var timestamp int64
	if err := json.Unmarshal(aux.SinceTimestamp, &timestamp); err == nil {
		if timestamp > 0 {
			c.SinceTimestamp = time.Unix(timestamp, 0)
		} else {
			c.SinceTimestamp = defaultTime
		}
		return nil
	}

	c.SinceTimestamp = defaultTime
	return nil
}

// TriggerSync manually triggers hotel data synchronization
// @Summary Trigger manual sync
// @Description Manually trigger synchronization of hotel data from external sources
// @Tags admin
// @Accept json
// @Produce json
// @Param options body usecase.SyncOptions false "Synchronization options"
// @Success 200 {object} APIResponse "Synchronization result with statistics"
// @Failure 500 {object} APIResponse "Internal Server Error"
// @Router /api/v1/admin/sync [post]
// CustomSyncOptions wraps SyncOptions to handle unmarshalling
func (h *HotelHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	var customOptions CustomSyncOptions
	customOptions.SinceTimestamp = time.Now().AddDate(0, -1, 0)

	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&customOptions); err != nil {
			h.logger.Warn("Failed to decode sync options, using defaults", "error", err)
		}
	}

	options := customOptions.SyncOptions

	if options.BatchSize == 0 {
		options.BatchSize = 100
	}
	options.UpdateCacheAfter = true

	h.logger.Info("Triggering manual sync",
		"full_sync", options.FullSync,
		"batch_size", options.BatchSize,
		"since_timestamp", options.SinceTimestamp.Format(time.RFC3339),
		"remote_addr", r.RemoteAddr)

	result, err := h.syncHotelsUseCase.Execute(r.Context(), options)
	if err != nil {
		h.logger.Error("Sync failed", "error", err)
		h.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.writeSuccessResponse(w, result, nil)
}

func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
		"2006-01-02T15:04:05-07:00",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	if timestamp, err := strconv.ParseInt(s, 10, 64); err == nil {
		if timestamp > 0 {
			return time.Unix(timestamp, 0), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", s)
}

// GetSyncStats returns current synchronization statistics
// @Summary Get sync statistics
// @Description Get current statistics about hotel data synchronization including index size, document count, and last update time
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse "Synchronization statistics"
// @Failure 500 {object} APIResponse "Internal Server Error"
// @Router /api/v1/admin/sync/stats [get]
func (h *HotelHandler) GetSyncStats(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Getting sync stats")

	stats, err := h.syncHotelsUseCase.GetSyncStats(r.Context())
	if err != nil {
		h.logger.Error("Failed to get sync stats", "error", err)
		h.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lastSyncTime, err := h.syncHotelsUseCase.GetLastSyncTime(r.Context())
	if err == nil && lastSyncTime != nil {
		stats.LastUpdated = *lastSyncTime
	}

	h.writeSuccessResponse(w, stats, nil)
}

// GetTrendingSuggestions returns trending hotel search suggestions
// @Summary Get trending search suggestions
// @Description Get currently trending hotel search suggestions based on popular searches
// @Tags search
// @Accept json
// @Produce json
// @Param limit query integer false "Maximum number of trending suggestions to return (default: 10)"
// @Success 200 {object} APIResponse{data=[]search.Suggestion} "List of trending search suggestions"
// @Failure 500 {object} APIResponse "Internal Server Error"
// @Router /api/v1/search/trending [get]
func (h *HotelHandler) GetTrendingSuggestions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	h.logger.Debug("Getting trending suggestions", "limit", limit)

	suggestions, err := h.getHotelSuggestionsUseCase.GetTrendingSuggestions(r.Context(), limit)
	if err != nil {
		h.logger.Error("Failed to get trending suggestions", "error", err)
		h.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.writeSuccessResponse(w, suggestions, nil)
}

func (h *HotelHandler) parseSearchParams(r *http.Request) search.Params {
	query := r.URL.Query()

	params := search.Params{
		Query:       query.Get("q"),
		Name:        query.Get("name"),
		Description: query.Get("description"),
		Phone:       query.Get("phone"),
		Chain:       query.Get("chain"),
		Email:       query.Get("email"),
		Fax:         query.Get("fax"),
		AirportCode: query.Get("airport_code"),
		Parking:     query.Get("parking"),
		City:        query.Get("city"),
		Country:     query.Get("country"),
		Currency:    query.Get("currency"),
		SortBy:      query.Get("sort_by"),
		SortOrder:   query.Get("sort_order"),
		Amenities:   query["amenities"],
		Tags:        query["tags"],
	}

	if ratingMin := query.Get("rating_min"); ratingMin != "" {
		if val, err := strconv.ParseFloat(ratingMin, 64); err == nil {
			params.RatingMin = val
		}
	}

	if ratingMax := query.Get("rating_max"); ratingMax != "" {
		if val, err := strconv.ParseFloat(ratingMax, 64); err == nil {
			params.RatingMax = val
		}
	}

	if starRating := query.Get("star_rating"); starRating != "" {
		if val, err := strconv.ParseInt(starRating, 10, 8); err == nil {
			params.StarRating = int8(val)
		}
	}

	if priceMin := query.Get("price_min"); priceMin != "" {
		if val, err := strconv.ParseFloat(priceMin, 64); err == nil {
			params.PriceMin = val
		}
	}

	if priceMax := query.Get("price_max"); priceMax != "" {
		if val, err := strconv.ParseFloat(priceMax, 64); err == nil {
			params.PriceMax = val
		}
	}

	if page := query.Get("page"); page != "" {
		if val, err := strconv.Atoi(page); err == nil {
			params.Page = val
		}
	}

	if limit := query.Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			params.Limit = val
		}
	}

	if latitude := query.Get("latitude"); latitude != "" {
		if val, err := strconv.ParseFloat(latitude, 64); err == nil {
			params.Latitude = val
		}
	}

	if longitude := query.Get("longitude"); longitude != "" {
		if val, err := strconv.ParseFloat(longitude, 64); err == nil {
			params.Longitude = val
		}
	}

	if radius := query.Get("radius"); radius != "" {
		if val, err := strconv.ParseFloat(radius, 64); err == nil {
			params.Radius = val
		}
	}

	if reviewCount := query.Get("review_count"); reviewCount != "" {
		if val, err := strconv.ParseInt(reviewCount, 10, 32); err == nil {
			params.ReviewCount = int32(val)
		}
	}

	if childAllowed := query.Get("child_allowed"); childAllowed != "" {
		if val, err := strconv.ParseBool(childAllowed); err == nil {
			params.ChildAllowed = &val
		}
	}

	if petsAllowed := query.Get("pets_allowed"); petsAllowed != "" {
		if val, err := strconv.ParseBool(petsAllowed); err == nil {
			params.PetsAllowed = &val
		}
	}

	return params
}

func (h *HotelHandler) writeSuccessResponse(w http.ResponseWriter, data interface{}, meta interface{}) {
	response := APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

func (h *HotelHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := APIResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
	}
}

// HealthCheck returns the health status of the search service
// @Summary Health check
// @Description Get the current health status of the search service
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=object} "Service health status with timestamp and version"
// @Router /health [get]
// @BasePath /
func (h *HotelHandler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "search-service",
		"version":   "1.0.0",
	}

	h.writeSuccessResponse(w, health, nil)
}
