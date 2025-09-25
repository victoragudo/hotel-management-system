package search

import (
	"context"
	"time"

	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/hotel"
)

type Params struct {
	Query        string   `json:"q,omitempty"`
	Name         string   `json:"name,omitempty"`
	Description  string   `json:"description,omitempty"`
	Phone        string   `json:"phone,omitempty"`
	Chain        string   `json:"chain,omitempty"`
	Email        string   `json:"email,omitempty"`
	Fax          string   `json:"fax,omitempty"`
	AirportCode  string   `json:"airport_code,omitempty"`
	Parking      string   `json:"parking,omitempty"`
	City         string   `json:"city,omitempty"`
	Country      string   `json:"country,omitempty"`
	RatingMin    float64  `json:"rating_min,omitempty"`
	RatingMax    float64  `json:"rating_max,omitempty"`
	StarRating   int8     `json:"star_rating,omitempty"`
	ReviewCount  int32    `json:"review_count,omitempty"`
	ChildAllowed *bool    `json:"child_allowed,omitempty"`
	PetsAllowed  *bool    `json:"pets_allowed,omitempty"`
	Amenities    []string `json:"amenities,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	PriceMin     float64  `json:"price_min,omitempty"`
	PriceMax     float64  `json:"price_max,omitempty"`
	Currency     string   `json:"currency,omitempty"`
	SortBy       string   `json:"sort_by,omitempty"`
	SortOrder    string   `json:"sort_order,omitempty"`
	Page         int      `json:"page,omitempty"`
	Limit        int      `json:"limit,omitempty"`
	Latitude     float64  `json:"latitude,omitempty"`
	Longitude    float64  `json:"longitude,omitempty"`
	Radius       float64  `json:"radius,omitempty"`
}

type Result struct {
	Hotels         []*hotel.Hotel `json:"hotels"`
	TotalHits      int64          `json:"total_hits"`
	Page           int            `json:"page"`
	Limit          int            `json:"limit"`
	TotalPages     int            `json:"total_pages"`
	ProcessingTime time.Duration  `json:"processing_time"`
	Facets         *Facets        `json:"facets,omitempty"`
	Query          string         `json:"query,omitempty"`
}

type Facets struct {
	Cities       []FacetItem `json:"cities,omitempty"`
	Countries    []FacetItem `json:"countries,omitempty"`
	StarRatings  []FacetItem `json:"star_ratings,omitempty"`
	Amenities    []FacetItem `json:"amenities,omitempty"`
	PriceRanges  []FacetItem `json:"price_ranges,omitempty"`
	HotelChains  []FacetItem `json:"hotel_chains,omitempty"`
	RatingRanges []FacetItem `json:"rating_ranges,omitempty"`
}

type FacetItem struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type Suggestion struct {
	Text     string                 `json:"text"`
	Type     string                 `json:"type"`
	Score    float64                `json:"score"`
	HotelID  *int64                 `json:"hotel_id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Engine interface {
	Index(ctx context.Context, hotels []*hotel.Hotel) error
	Search(ctx context.Context, params Params) (*Result, error)
	GetSuggestions(ctx context.Context, query string, limit int) ([]*Suggestion, error)
	GetFacets(ctx context.Context) (*Facets, error)
	UpdateHotel(ctx context.Context, hotel *hotel.Hotel) error
	DeleteHotel(ctx context.Context, hotelID string) error
	ClearIndex(ctx context.Context) error
	GetIndexStats(ctx context.Context) (*IndexStats, error)
	HealthCheck(ctx context.Context) error
}

type IndexStats struct {
	TotalDocuments int64     `json:"total_documents"`
	IndexSize      int64     `json:"index_size_bytes"`
	LastUpdated    time.Time `json:"last_updated"`
	Version        string    `json:"version"`
}

func (p *Params) Validate() error {
	if p.Page < 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.RatingMin < 0 {
		p.RatingMin = 0
	}
	if p.RatingMax > 5 {
		p.RatingMax = 5
	}
	if p.StarRating < 0 {
		p.StarRating = 0
	}
	if p.StarRating > 5 {
		p.StarRating = 0
	}

	validSortFields := map[string]bool{
		"rating":     true,
		"price":      true,
		"distance":   true,
		"relevance":  true,
		"name":       true,
		"created_at": true,
	}
	if p.SortBy != "" && !validSortFields[p.SortBy] {
		p.SortBy = "relevance"
	}

	if p.SortOrder != "asc" && p.SortOrder != "desc" {
		p.SortOrder = "desc"
	}

	return nil
}

func (p *Params) HasLocationFilter() bool {
	return p.Latitude != 0 && p.Longitude != 0 && p.Radius > 0
}

func (p *Params) HasPriceFilter() bool {
	return p.PriceMin > 0 || p.PriceMax > 0
}

func (p *Params) HasRatingFilter() bool {
	return p.RatingMin > 0 || p.RatingMax > 0 || p.StarRating > 0
}

func (r *Result) CalculateTotalPages() {
	if r.Limit > 0 {
		r.TotalPages = int((r.TotalHits + int64(r.Limit) - 1) / int64(r.Limit))
	}
}

func (r *Result) HasNextPage() bool {
	return r.Page < r.TotalPages
}

func (r *Result) HasPreviousPage() bool {
	return r.Page > 1
}
