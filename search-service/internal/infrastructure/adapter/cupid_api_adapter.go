package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	apimodels "github.com/victoragudo/hotel-management-system/pkg/api-models"

	"github.com/google/uuid"
	"github.com/victoragudo/hotel-management-system/search-service/internal/domain/hotel"
)

type CupidAPIAdapter struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewCupidAPIAdapter(baseURL, apiKey string, timeout time.Duration, logger *slog.Logger) *CupidAPIAdapter {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &CupidAPIAdapter{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

func (cupidAPI *CupidAPIAdapter) makeAPIRequest(ctx context.Context, url string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("accept", "application/json")
	request.Header.Set("x-api-key", cupidAPI.apiKey)

	resp, err := cupidAPI.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	return resp, nil
}

func (cupidAPI *CupidAPIAdapter) GetHotelByID(ctx context.Context, hotelID int64) (*hotel.Hotel, error) {
	url := fmt.Sprintf("%s/property/%d", cupidAPI.baseURL, hotelID)

	cupidAPI.logger.Debug("Fetching hotel from Cupid API", "hotel_id", hotelID, "url", url)

	startTime := time.Now()
	resp, err := cupidAPI.makeAPIRequest(ctx, url)
	if err != nil {
		cupidAPI.logger.Error("Failed to call Cupid API", "hotel_id", hotelID, "error", err, "duration", time.Since(startTime))
		return nil, fmt.Errorf("cupid API request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	cupidAPI.logger.Debug("Cupid API response received", "hotel_id", hotelID, "status_code", resp.StatusCode, "duration", time.Since(startTime))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		cupidAPI.logger.Warn("Hotel not found in Cupid API", "hotel_id", hotelID)
		return nil, fmt.Errorf("hotel %d not found in Cupid API", hotelID)
	}

	if resp.StatusCode != http.StatusOK {
		cupidAPI.logger.Error("Cupid API returned error", "hotel_id", hotelID, "status_code", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("cupid API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResponse apimodels.HotelAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		cupidAPI.logger.Error("Failed to unmarshal Cupid API response", "hotel_id", hotelID, "error", err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	hotelEntity, err := cupidAPI.convertCupidToHotel(apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Cupid data to hotel: %w", err)
	}

	cupidAPI.logger.Info("Successfully fetched hotel from Cupid API", "hotel_id", hotelID, "hotel_name", hotelEntity.Name)
	return hotelEntity, nil
}

func (cupidAPI *CupidAPIAdapter) GetHotelReviews(ctx context.Context, hotelID int64, reviewsCount int) ([]*hotel.Review, error) {
	url := fmt.Sprintf("%s/property/reviews/%d/%d", cupidAPI.baseURL, hotelID, reviewsCount)

	cupidAPI.logger.Debug("Fetching hotel reviews from Cupid API", "hotel_id", hotelID)

	resp, err := cupidAPI.makeAPIRequest(ctx, url)
	if err != nil {
		cupidAPI.logger.Error("Failed to call Cupid API for reviews", "hotel_id", hotelID, "error", err)
		return nil, fmt.Errorf("cupid API reviews request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		cupidAPI.logger.Warn("Cupid API returned non-OK status for reviews", "hotel_id", hotelID, "status_code", resp.StatusCode)
		return nil, fmt.Errorf("cupid API returned status %d for reviews", resp.StatusCode)
	}

	var reviewsResponse []apimodels.ReviewAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&reviewsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode reviews response: %w", err)
	}

	reviews := make([]*hotel.Review, 0, len(reviewsResponse))
	for _, cupidReview := range reviewsResponse {
		review, err := cupidAPI.convertCupidToReview(hotelID, cupidReview)
		if err != nil {
			cupidAPI.logger.Warn("Failed to convert review", "review_id", cupidReview.ReviewID, "error", err)
			continue
		}
		reviews = append(reviews, review)
	}

	cupidAPI.logger.Info("Successfully fetched reviews from Cupid API", "hotel_id", hotelID, "count", len(reviews))
	return reviews, nil
}

func (cupidAPI *CupidAPIAdapter) GetHotelTranslations(ctx context.Context, hotelID int64, languages []string) ([]*hotel.Translation, error) {
	baseURL := fmt.Sprintf("%s/property/%d/lang/", cupidAPI.baseURL, hotelID)

	var translations []*hotel.Translation

	for _, language := range languages {
		url := baseURL + language + "/"

		cupidAPI.logger.Debug("Fetching hotel translations from Cupid API", "hotel_id", hotelID, "language", language)

		request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		request.Header.Set("accept", "application/json")
		request.Header.Set("x-api-key", cupidAPI.apiKey)

		resp, err := cupidAPI.httpClient.Do(request)
		if err != nil {
			cupidAPI.logger.Error("Failed to call Cupid API for translations", "hotel_id", hotelID, "language", language, "error", err)
			return nil, fmt.Errorf("cupid API translations request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			cupidAPI.logger.Warn("Cupid API returned non-OK status for translations", "hotel_id", hotelID, "language", language, "status_code", resp.StatusCode)
			continue
		}

		var translationsResponse apimodels.TranslationAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&translationsResponse); err != nil {
			cupidAPI.logger.Warn("Failed to decode translations response", "hotel_id", hotelID, "language", language, "error", err)
			continue
		}

		translation, err := cupidAPI.convertCupidToTranslation(translationsResponse, language)
		if err != nil {
			cupidAPI.logger.Warn("Failed to convert translation", "hotel_id", hotelID, "language", language, "error", err)
			continue
		}
		translations = append(translations, translation)
	}

	cupidAPI.logger.Info("Successfully fetched translations from Cupid API", "hotel_id", hotelID, "count", len(translations))
	return translations, nil
}

func (cupidAPI *CupidAPIAdapter) convertFacilities(apiFacilities []apimodels.Facility) []hotel.Facility {
	facilities := make([]hotel.Facility, 0, len(apiFacilities))
	for _, facility := range apiFacilities {
		facilities = append(facilities, hotel.Facility{
			Name: facility.Name,
			ID:   facility.ID,
		})
	}
	return facilities
}

func (cupidAPI *CupidAPIAdapter) convertPolicies(apiPolicies []apimodels.Policy) []hotel.Policy {
	policies := make([]hotel.Policy, 0, len(apiPolicies))
	for _, policy := range apiPolicies {
		policies = append(policies, hotel.Policy{
			PolicyType:   policy.PolicyType,
			Name:         policy.Name,
			Description:  policy.Description,
			ChildAllowed: policy.ChildAllowed,
			PetsAllowed:  policy.PetsAllowed,
			Parking:      policy.Parking,
		})
	}
	return policies
}

func (cupidAPI *CupidAPIAdapter) convertImages(apiPhotos []apimodels.Photo) []string {
	images := make([]string, 0, len(apiPhotos))
	for _, photo := range apiPhotos {
		images = append(images, photo.URL)
	}
	return images
}

func (cupidAPI *CupidAPIAdapter) convertBedTypes(apiBedTypes []apimodels.BedType) []hotel.BedType {
	bedTypes := make([]hotel.BedType, 0, len(apiBedTypes))
	for _, bedType := range apiBedTypes {
		bedTypes = append(bedTypes, hotel.BedType{
			Quantity: bedType.Quantity,
			BedType:  bedType.BedType,
			BedSize:  bedType.BedSize,
			ID:       bedType.ID,
		})
	}
	return bedTypes
}

func (cupidAPI *CupidAPIAdapter) convertRoomAmenities(apiAmenities []apimodels.Amenity) []hotel.Amenity {
	amenities := make([]hotel.Amenity, 0, len(apiAmenities))
	for _, amenity := range apiAmenities {
		amenities = append(amenities, hotel.Amenity{
			AmenitiesID: amenity.AmenityID,
			Name:        amenity.Name,
			Sort:        amenity.Sort,
		})
	}
	return amenities
}

func (cupidAPI *CupidAPIAdapter) convertRoomPhotos(apiPhotos []apimodels.Photo) []hotel.RoomPhoto {
	photos := make([]hotel.RoomPhoto, 0, len(apiPhotos))
	for _, photo := range apiPhotos {
		photos = append(photos, hotel.RoomPhoto{
			URL:              photo.URL,
			HDURL:            photo.HDURL,
			ImageDescription: photo.ImageDescription,
			ImageClass1:      photo.ImageClass1,
			ImageClass2:      photo.ImageClass2,
			MainPhoto:        photo.MainPhoto,
			Score:            photo.Score,
			ClassID:          photo.ClassID,
			ClassOrder:       photo.ClassOrder,
		})
	}
	return photos
}

func (cupidAPI *CupidAPIAdapter) convertPhotos(apiPhotos []apimodels.Photo) []hotel.Photo {
	photos := make([]hotel.Photo, 0, len(apiPhotos))
	for _, photo := range apiPhotos {
		photos = append(photos, hotel.Photo{
			URL:              photo.URL,
			HDURL:            photo.HDURL,
			ImageDescription: photo.ImageDescription,
			ImageClass1:      photo.ImageClass1,
			ImageClass2:      photo.ImageClass2,
			MainPhoto:        photo.MainPhoto,
			Score:            photo.Score,
			ClassID:          photo.ClassID,
			ClassOrder:       photo.ClassOrder,
		})
	}
	return photos
}

func (cupidAPI *CupidAPIAdapter) convertRooms(apiRooms []apimodels.Room) []hotel.Room {
	rooms := make([]hotel.Room, 0, len(apiRooms))
	for _, room := range apiRooms {
		rooms = append(rooms, hotel.Room{
			ID:             room.ID,
			RoomName:       room.RoomName,
			Description:    room.Description,
			RoomSizeSquare: room.RoomSizeSquare,
			RoomSizeUnit:   room.RoomSizeUnit,
			HotelID:        room.HotelID,
			MaxAdults:      room.MaxAdults,
			MaxChildren:    room.MaxChildren,
			MaxOccupancy:   room.MaxOccupancy,
			BedRelation:    room.BedRelation,
			BedTypes:       cupidAPI.convertBedTypes(room.BedTypes),
			RoomAmenities:  cupidAPI.convertRoomAmenities(room.RoomAmenities),
			Photos:         cupidAPI.convertRoomPhotos(room.Photos),
			Views:          room.Views,
		})
	}
	return rooms
}

func (cupidAPI *CupidAPIAdapter) convertCupidToHotel(hotelAPIResponse apimodels.HotelAPIResponse) (*hotel.Hotel, error) {
	h := &hotel.Hotel{
		HotelID:             hotelAPIResponse.HotelID,
		CupidID:             int64(hotelAPIResponse.CupidID),
		HotelTypeID:         int64(hotelAPIResponse.HotelTypeID),
		Name:                hotelAPIResponse.HotelName,
		Description:         hotelAPIResponse.Description,
		Rating:              hotelAPIResponse.Rating,
		StarRating:          hotelAPIResponse.Stars,
		Latitude:            hotelAPIResponse.Latitude,
		Longitude:           hotelAPIResponse.Longitude,
		MainImageTh:         hotelAPIResponse.MainImageTh,
		HotelType:           hotelAPIResponse.HotelType,
		Chain:               hotelAPIResponse.Chain,
		ChainID:             int32(hotelAPIResponse.ChainID),
		Phone:               hotelAPIResponse.Phone,
		Fax:                 hotelAPIResponse.Fax,
		Email:               hotelAPIResponse.Email,
		AirportCode:         hotelAPIResponse.AirportCode,
		ReviewCount:         int32(hotelAPIResponse.ReviewCount),
		Parking:             hotelAPIResponse.Parking,
		ChildAllowed:        hotelAPIResponse.ChildAllowed,
		PetsAllowed:         hotelAPIResponse.PetsAllowed,
		MarkdownDescription: hotelAPIResponse.MarkdownDescription,
		ImportantInfo:       hotelAPIResponse.ImportantInfo,
	}

	h.Address = hotel.Address{
		Street:     hotelAPIResponse.Address.Address,
		City:       hotelAPIResponse.Address.City,
		State:      hotelAPIResponse.Address.State,
		Country:    hotelAPIResponse.Address.Country,
		PostalCode: hotelAPIResponse.Address.PostalCode,
	}

	h.ContactInfo = hotel.ContactInfo{
		Phone: hotelAPIResponse.Phone,
		Fax:   hotelAPIResponse.Fax,
		Email: hotelAPIResponse.Email,
	}

	h.CheckinInfo = hotel.CheckinInfo{
		CheckinStart:        cupidAPI.parseTimeString(hotelAPIResponse.Checkin.CheckinStart),
		CheckinEnd:          cupidAPI.parseTimeString(hotelAPIResponse.Checkin.CheckinEnd),
		Checkout:            cupidAPI.parseTimeString(hotelAPIResponse.Checkin.Checkout),
		Instructions:        hotelAPIResponse.Checkin.Instructions,
		SpecialInstructions: hotelAPIResponse.Checkin.SpecialInstructions,
	}

	h.Facilities = cupidAPI.convertFacilities(hotelAPIResponse.Facilities)
	h.Policies = cupidAPI.convertPolicies(hotelAPIResponse.Policies)
	h.Images = cupidAPI.convertImages(hotelAPIResponse.Photos)
	h.Rooms = cupidAPI.convertRooms(hotelAPIResponse.Rooms)
	h.Photos = cupidAPI.convertPhotos(hotelAPIResponse.Photos)

	return h, nil
}

func (cupidAPI *CupidAPIAdapter) convertCupidToTranslation(translationAPIResponse apimodels.TranslationAPIResponse, lang string) (*hotel.Translation, error) {
	translation := &hotel.Translation{
		HotelID:             translationAPIResponse.HotelID,
		HotelTypeID:         int64(translationAPIResponse.HotelTypeID),
		Name:                translationAPIResponse.HotelName,
		Description:         translationAPIResponse.Description,
		Latitude:            translationAPIResponse.Latitude,
		Longitude:           translationAPIResponse.Longitude,
		Chain:               translationAPIResponse.Chain,
		Parking:             translationAPIResponse.Parking,
		MarkdownDescription: translationAPIResponse.MarkdownDescription,
		ImportantInfo:       translationAPIResponse.ImportantInfo,
		Lang:                lang,
	}

	translation.Address = hotel.Address{
		Street:     translationAPIResponse.Address.Address,
		City:       translationAPIResponse.Address.City,
		State:      translationAPIResponse.Address.State,
		Country:    translationAPIResponse.Address.Country,
		PostalCode: translationAPIResponse.Address.PostalCode,
	}

	translation.ContactInfo = hotel.ContactInfo{
		Phone: translationAPIResponse.Phone,
		Fax:   translationAPIResponse.Fax,
		Email: translationAPIResponse.Email,
	}

	translation.CheckinInfo = hotel.CheckinInfo{
		CheckinStart:        cupidAPI.parseTimeString(translationAPIResponse.Checkin.CheckinStart),
		CheckinEnd:          cupidAPI.parseTimeString(translationAPIResponse.Checkin.CheckinEnd),
		Checkout:            cupidAPI.parseTimeString(translationAPIResponse.Checkin.Checkout),
		Instructions:        translationAPIResponse.Checkin.Instructions,
		SpecialInstructions: translationAPIResponse.Checkin.SpecialInstructions,
	}

	translation.Facilities = cupidAPI.convertFacilities(translationAPIResponse.Facilities)
	translation.Policies = cupidAPI.convertPolicies(translationAPIResponse.Policies)
	translation.Images = cupidAPI.convertImages(translationAPIResponse.Photos)
	translation.Rooms = cupidAPI.convertRooms(translationAPIResponse.Rooms)
	translation.Photos = cupidAPI.convertPhotos(translationAPIResponse.Photos)

	return translation, nil
}

func (cupidAPI *CupidAPIAdapter) convertCupidToReview(hotelId int64, cupidReview apimodels.ReviewAPIResponse) (*hotel.Review, error) {
	var reviewDate time.Time
	if cupidReview.Date != "" {
		if t, err := time.Parse(time.RFC3339, cupidReview.Date); err == nil {
			reviewDate = t
		}
	}

	return &hotel.Review{
		ID:           uuid.NewString(),
		HotelID:      hotelId,
		ReviewID:     cupidReview.ReviewID,
		AverageScore: cupidReview.AverageScore,
		Country:      cupidReview.Country,
		Type:         cupidReview.Type,
		Name:         cupidReview.Name,
		Date:         reviewDate,
		Headline:     cupidReview.Headline,
		Language:     cupidReview.Language,
		Pros:         cupidReview.Pros,
		Cons:         cupidReview.Cons,
		Source:       cupidReview.Source,
	}, nil
}

func (cupidAPI *CupidAPIAdapter) parseTimeString(timeStr string) time.Time {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}
	}
	now := time.Now().UTC()
	result := time.Date(now.Year(), now.Month(), now.Day(),
		t.Hour(), t.Minute(), 0, 0, time.UTC)

	return result
}
