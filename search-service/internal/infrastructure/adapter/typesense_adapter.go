package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/hotel"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/search"
)

type TypesenseAdapter struct {
	client         *typesense.Client
	collectionName string
	logger         *slog.Logger
}

func NewTypesenseAdapter(hostURL, apiKey, collectionName string, logger *slog.Logger) (*TypesenseAdapter, error) {
	client := typesense.NewClient(
		typesense.WithServer(hostURL),
		typesense.WithAPIKey(apiKey),
	)

	adapter := &TypesenseAdapter{
		client:         client,
		collectionName: collectionName,
		logger:         logger,
	}

	if err := adapter.initializeCollection(); err != nil {
		return nil, fmt.Errorf("failed to initialize collection: %w", err)
	}

	return adapter, nil
}

type TypesenseDocument struct {
	HotelID      int64   `json:"hotel_id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Phone        string  `json:"phone"`
	Chain        string  `json:"chain"`
	Rating       float64 `json:"rating"`
	StarRating   int32   `json:"star_rating"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Fax          string  `json:"fax"`
	Email        string  `json:"email"`
	AirportCode  string  `json:"airport_code"`
	ReviewCount  int32   `json:"review_count"`
	ChildAllowed bool    `json:"child_allowed"`
	PetsAllowed  bool    `json:"pets_allowed"`
	CreatedAt    int64   `json:"created_at"`
	Parking      string  `json:"parking"`
	UpdatedAt    int64   `json:"updated_at"`
}

func (t *TypesenseAdapter) initializeCollection() error {
	collectionSchema := &api.CollectionSchema{
		Name: t.collectionName,
		Fields: []api.Field{
			{
				Name: "hotel_id",
				Type: "int64",
			},
			{
				Name: "name",
				Type: "string",
			},
			{
				Name: "description",
				Type: "string",
			},
			{
				Name:     "phone",
				Type:     "string",
				Optional: pointer.True(),
			},
			{
				Name:     "chain",
				Type:     "string",
				Facet:    pointer.True(),
				Optional: pointer.True(),
			},
			{
				Name:  "rating",
				Type:  "float",
				Facet: pointer.True(),
			},
			{
				Name:  "star_rating",
				Type:  "int32",
				Facet: pointer.True(),
			},
			{
				Name: "latitude",
				Type: "float",
			},
			{
				Name: "longitude",
				Type: "float",
			},
			{
				Name:     "fax",
				Type:     "string",
				Optional: pointer.True(),
			},
			{
				Name:     "email",
				Type:     "string",
				Optional: pointer.True(),
			},
			{
				Name:     "airport_code",
				Type:     "string",
				Facet:    pointer.True(),
				Optional: pointer.True(),
			},
			{
				Name:  "review_count",
				Type:  "int32",
				Facet: pointer.True(),
			},
			{
				Name:  "child_allowed",
				Type:  "bool",
				Facet: pointer.True(),
			},
			{
				Name:  "pets_allowed",
				Type:  "bool",
				Facet: pointer.True(),
			},
			{
				Name:  "created_at",
				Type:  "int64",
				Facet: pointer.True(),
			},
			{
				Name:     "parking",
				Type:     "string",
				Facet:    pointer.True(),
				Optional: pointer.True(),
			},
			{
				Name:  "updated_at",
				Type:  "int64",
				Facet: pointer.True(),
			},
		},
		DefaultSortingField: pointer.String("rating"),
	}

	_, err := t.client.Collections().Create(collectionSchema)
	if err != nil {
		t.logger.Warn("Collection creation result", "error", err)
	}

	t.logger.Info("Typesense collection initialized", "collection_name", t.collectionName)
	return nil
}

func (t *TypesenseAdapter) convertHotelToDocument(h *hotel.Hotel) *TypesenseDocument {
	document := &TypesenseDocument{
		HotelID:      h.HotelID,
		Name:         h.Name,
		Description:  h.Description,
		Phone:        h.Phone,
		Chain:        h.Chain,
		Rating:       h.Rating,
		StarRating:   h.StarRating,
		Latitude:     h.Latitude,
		Longitude:    h.Longitude,
		Fax:          h.Fax,
		Email:        h.Email,
		AirportCode:  h.AirportCode,
		ReviewCount:  h.ReviewCount,
		ChildAllowed: h.ChildAllowed,
		PetsAllowed:  h.PetsAllowed,
		UpdatedAt:    h.UpdatedAt.UTC().Unix(),
		Parking:      h.Parking,
		CreatedAt:    h.CreatedAt.UTC().Unix(),
	}

	return document
}

func (t *TypesenseAdapter) Index(_ context.Context, hotels []*hotel.Hotel) error {
	if len(hotels) == 0 {
		return nil
	}

	t.logger.Debug("Indexing hotels", "count", len(hotels))

	documents := make([]TypesenseDocument, len(hotels))
	for i, h := range hotels {
		documents[i] = *t.convertHotelToDocument(h)
	}

	documentsInterface := make([]interface{}, len(documents))
	for i, doc := range documents {
		documentsInterface[i] = doc
	}

	params := &api.ImportDocumentsParams{
		Action:    pointer.String("upsert"),
		BatchSize: pointer.Int(100),
	}

	_, err := t.client.Collection(t.collectionName).Documents().Import(documentsInterface, params)
	if err != nil {
		t.logger.Error("Failed to import documents", "error", err)
		return fmt.Errorf("failed to index hotels: %w", err)
	}

	t.logger.Info("Hotels indexed successfully", "count", len(hotels))
	return nil
}

func (t *TypesenseAdapter) Search(_ context.Context, params search.Params) (*search.Result, error) {
	queryBy := "name,description"
	query := "*"
	if params.Query != "" {
		query = params.Query
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	searchParams := &api.SearchCollectionParams{
		Q:       query,
		QueryBy: queryBy,
		Page:    &page,
		PerPage: &limit,
	}

	filters := t.buildFilters(params)
	if filters != "" {
		searchParams.FilterBy = &filters
	}

	sortBy := t.buildSort(params)
	if sortBy != "" {
		searchParams.SortBy = &sortBy
	}

	if params.HasLocationFilter() {
		geoFilter := fmt.Sprintf("location:(%f, %f, %f km)", params.Latitude, params.Longitude, params.Radius)
		if filters != "" {
			combinedFilter := filters + " && " + geoFilter
			searchParams.FilterBy = &combinedFilter
		} else {
			searchParams.FilterBy = &geoFilter
		}
	}

	t.logger.Debug("Executing Typesense search", "query", query, "filters", filters, "sort", sortBy)

	searchResponse, err := t.client.Collection(t.collectionName).Documents().Search(searchParams)
	if err != nil {
		t.logger.Error("Typesense search failed", "error", err)
		return nil, fmt.Errorf("typesense search error: %w", err)
	}

	hotels := make([]*hotel.Hotel, 0, len(*searchResponse.Hits))
	for _, hit := range *searchResponse.Hits {
		if h, err := t.convertDocumentToHotel(hit.Document); err == nil {
			hotels = append(hotels, h)
		} else {
			t.logger.Warn("Failed to convert document to hotel", "error", err)
		}
	}

	totalHits := int64(0)
	if searchResponse.Found != nil {
		totalHits = int64(*searchResponse.Found)
	}

	result := &search.Result{
		Hotels:    hotels,
		TotalHits: totalHits,
		Page:      page,
		Limit:     limit,
	}

	return result, nil
}

func (t *TypesenseAdapter) buildFilters(params search.Params) string {
	var filters []string

	if params.Name != "" {
		filters = append(filters, fmt.Sprintf("name:=%s", params.Name))
	}

	if params.Description != "" {
		filters = append(filters, fmt.Sprintf("description:=%s", params.Description))
	}

	if params.Phone != "" {
		filters = append(filters, fmt.Sprintf("phone:=%s", params.Phone))
	}

	if params.Chain != "" {
		filters = append(filters, fmt.Sprintf("chain:=%s", params.Chain))
	}

	if params.Email != "" {
		filters = append(filters, fmt.Sprintf("email:=%s", params.Email))
	}

	if params.Fax != "" {
		filters = append(filters, fmt.Sprintf("fax:=%s", params.Fax))
	}

	if params.AirportCode != "" {
		filters = append(filters, fmt.Sprintf("airport_code:=%s", params.AirportCode))
	}

	if params.Parking != "" {
		filters = append(filters, fmt.Sprintf("parking:=%s", params.Parking))
	}

	if params.City != "" {
		filters = append(filters, fmt.Sprintf("city:=%s", params.City))
	}

	if params.Country != "" {
		filters = append(filters, fmt.Sprintf("country:=%s", params.Country))
	}

	if params.RatingMin > 0 {
		filters = append(filters, fmt.Sprintf("rating:>=%f", params.RatingMin))
	}
	if params.RatingMax > 0 {
		filters = append(filters, fmt.Sprintf("rating:<=%f", params.RatingMax))
	}

	if params.StarRating > 0 {
		filters = append(filters, fmt.Sprintf("star_rating:>=%d", params.StarRating))
	}

	if params.ReviewCount > 0 {
		filters = append(filters, fmt.Sprintf("review_count:>=%d", params.ReviewCount))
	}

	if params.ChildAllowed != nil {
		filters = append(filters, fmt.Sprintf("child_allowed:=%t", *params.ChildAllowed))
	}

	if params.PetsAllowed != nil {
		filters = append(filters, fmt.Sprintf("pets_allowed:=%t", *params.PetsAllowed))
	}

	if len(params.Amenities) > 0 {
		amenityFilters := make([]string, len(params.Amenities))
		for i, amenity := range params.Amenities {
			amenityFilters[i] = fmt.Sprintf("amenities:=%s", amenity)
		}
		filters = append(filters, fmt.Sprintf("(%s)", strings.Join(amenityFilters, " || ")))
	}

	if len(params.Tags) > 0 {
		tagFilters := make([]string, len(params.Tags))
		for i, tag := range params.Tags {
			tagFilters[i] = fmt.Sprintf("tags:=%s", tag)
		}
		filters = append(filters, fmt.Sprintf("(%s)", strings.Join(tagFilters, " || ")))
	}

	if params.PriceMin > 0 && params.PriceMax > 0 {
		filters = append(filters,
			fmt.Sprintf("price_max:>=%f", params.PriceMin),
			fmt.Sprintf("price_min:<=%f", params.PriceMax),
		)
	} else if params.PriceMin > 0 {
		filters = append(filters, fmt.Sprintf("price_max:>=%f", params.PriceMin))
	} else if params.PriceMax > 0 {
		filters = append(filters, fmt.Sprintf("price_min:<=%f", params.PriceMax))
	}

	if params.Currency != "" {
		filters = append(filters, fmt.Sprintf("currency:=%s", params.Currency))
	}

	return strings.Join(filters, " && ")
}

func (t *TypesenseAdapter) buildSort(params search.Params) string {
	if params.SortBy == "" {
		return ""
	}

	sortOrder := "desc"
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}

	switch params.SortBy {
	case "price":
		return fmt.Sprintf("price_min:%s", sortOrder)
	case "distance":
		if params.HasLocationFilter() {
			return fmt.Sprintf("location(%f, %f):%s", params.Latitude, params.Longitude, sortOrder)
		}
		return ""
	default:
		return fmt.Sprintf("%s:%s", params.SortBy, sortOrder)
	}
}

func (t *TypesenseAdapter) convertDocumentToHotel(hit any) (*hotel.Hotel, error) {
	data, err := json.Marshal(hit)
	if err != nil {
		return nil, err
	}

	var typesenseDocument TypesenseDocument
	if err := json.Unmarshal(data, &typesenseDocument); err != nil {
		return nil, err
	}

	h := &hotel.Hotel{
		HotelID:      typesenseDocument.HotelID,
		Name:         typesenseDocument.Name,
		Description:  typesenseDocument.Description,
		Phone:        typesenseDocument.Phone,
		Chain:        typesenseDocument.Chain,
		Rating:       typesenseDocument.Rating,
		StarRating:   typesenseDocument.StarRating,
		Latitude:     typesenseDocument.Latitude,
		Longitude:    typesenseDocument.Longitude,
		Fax:          typesenseDocument.Fax,
		Email:        typesenseDocument.Email,
		AirportCode:  typesenseDocument.AirportCode,
		ReviewCount:  typesenseDocument.ReviewCount,
		ChildAllowed: typesenseDocument.ChildAllowed,
		PetsAllowed:  typesenseDocument.PetsAllowed,
		CreatedAt:    time.Unix(typesenseDocument.CreatedAt, 0),
		Parking:      typesenseDocument.Parking,
		UpdatedAt:    time.Unix(typesenseDocument.UpdatedAt, 0),
	}

	return h, nil
}

func (t *TypesenseAdapter) UpdateHotel(ctx context.Context, h *hotel.Hotel) error {
	return t.Index(ctx, []*hotel.Hotel{h})
}

func (t *TypesenseAdapter) DeleteHotel(ctx context.Context, hotelID string) error {
	_, err := t.client.Collection(t.collectionName).Document(hotelID).Delete()
	if err != nil {
		return fmt.Errorf("failed to delete hotel %s: %w", hotelID, err)
	}

	t.logger.Debug("Hotel deleted from collection", "hotel_id", hotelID)
	return nil
}

func (t *TypesenseAdapter) GetSuggestions(ctx context.Context, query string, limit int) ([]*search.Suggestion, error) {
	searchParams := &api.SearchCollectionParams{
		Q:       query,
		QueryBy: "name,city,country",
		PerPage: pointer.Int(limit),
		Page:    pointer.Int(1),
	}

	searchResponse, err := t.client.Collection(t.collectionName).Documents().Search(searchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get suggestions: %w", err)
	}

	suggestions := make([]*search.Suggestion, 0, len(*searchResponse.Hits))
	for _, hit := range *searchResponse.Hits {
		if suggestion := t.convertHitToSuggestion(hit.Document); suggestion != nil {
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

func (t *TypesenseAdapter) convertHitToSuggestion(hit any) *search.Suggestion {
	data, err := json.Marshal(hit)
	if err != nil {
		return nil
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}

	name, _ := doc["name"].(string)
	city, _ := doc["city"].(string)
	country, _ := doc["country"].(string)

	suggestion := &search.Suggestion{
		Text:  name,
		Type:  "hotel",
		Score: 1.0,
	}

	if hotelIDFloat, ok := doc["hotel_id"].(float64); ok {
		hotelID := int64(hotelIDFloat)
		suggestion.HotelID = &hotelID
	}

	if city != "" || country != "" {
		suggestion.Metadata = map[string]any{
			"city":    city,
			"country": country,
		}
	}

	return suggestion
}

func (t *TypesenseAdapter) GetFacets(ctx context.Context) (*search.Facets, error) {
	searchParams := &api.SearchCollectionParams{
		Q:       "*",
		QueryBy: "name",
		PerPage: pointer.Int(0),
		FacetBy: pointer.String("city,country,star_rating,amenities,price_range,chain"),
	}

	searchResponse, err := t.client.Collection(t.collectionName).Documents().Search(searchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get facets: %w", err)
	}

	facets := &search.Facets{
		Cities:       make([]search.FacetItem, 0),
		Countries:    make([]search.FacetItem, 0),
		StarRatings:  make([]search.FacetItem, 0),
		Amenities:    make([]search.FacetItem, 0),
		PriceRanges:  make([]search.FacetItem, 0),
		HotelChains:  make([]search.FacetItem, 0),
		RatingRanges: make([]search.FacetItem, 0),
	}

	if searchResponse.FacetCounts != nil {
		for _, facetCount := range *searchResponse.FacetCounts {
			switch *facetCount.FieldName {
			case "city":
				for _, count := range *facetCount.Counts {
					facets.Cities = append(facets.Cities, search.FacetItem{
						Value: *count.Value,
						Count: int64(*count.Count),
					})
				}
			case "country":
				for _, count := range *facetCount.Counts {
					facets.Countries = append(facets.Countries, search.FacetItem{
						Value: *count.Value,
						Count: int64(*count.Count),
					})
				}
			case "star_rating":
				for _, count := range *facetCount.Counts {
					facets.StarRatings = append(facets.StarRatings, search.FacetItem{
						Value: *count.Value,
						Count: int64(*count.Count),
					})
				}
			case "amenities":
				for _, count := range *facetCount.Counts {
					facets.Amenities = append(facets.Amenities, search.FacetItem{
						Value: *count.Value,
						Count: int64(*count.Count),
					})
				}
			case "price_range":
				for _, count := range *facetCount.Counts {
					facets.PriceRanges = append(facets.PriceRanges, search.FacetItem{
						Value: *count.Value,
						Count: int64(*count.Count),
					})
				}
			case "chain":
				for _, count := range *facetCount.Counts {
					facets.HotelChains = append(facets.HotelChains, search.FacetItem{
						Value: *count.Value,
						Count: int64(*count.Count),
					})
				}
			}
		}
	}

	return facets, nil
}

func (t *TypesenseAdapter) ClearIndex(ctx context.Context) error {
	_, err := t.client.Collection(t.collectionName).Retrieve()
	if err == nil {
		_, err := t.client.Collection(t.collectionName).Delete()
		if err != nil {
			return fmt.Errorf("failed to clear collection: %w", err)
		}
	}

	if err := t.initializeCollection(); err != nil {
		return fmt.Errorf("failed to reinitialize collection: %w", err)
	}

	t.logger.Info("Collection cleared and reinitialized")
	return nil
}

func (t *TypesenseAdapter) GetIndexStats(ctx context.Context) (*search.IndexStats, error) {
	collection, err := t.client.Collection(t.collectionName).Retrieve()
	if err != nil {
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	collectionResponses, _ := t.client.Collections().Retrieve()

	_ = collectionResponses

	for _, document := range collectionResponses {
		println(document.NumDocuments)
	}

	return &search.IndexStats{
		TotalDocuments: int64(*collection.NumDocuments),
		IndexSize:      0,
		LastUpdated:    time.Now(),
		Version:        "typesense",
	}, nil
}

func (t *TypesenseAdapter) HealthCheck(ctx context.Context) error {
	_, err := t.client.Health(5 * time.Second)
	if err != nil {
		return fmt.Errorf("typesense health check failed: %w", err)
	}
	return nil
}
