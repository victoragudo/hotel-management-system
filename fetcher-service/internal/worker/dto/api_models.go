package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/victoragudo/hotel-management-system/pkg/entities"
)

type HotelAPIResponse struct {
	HotelID             int64       `json:"hotel_id"`
	CupidID             int         `json:"cupid_id"`
	MainImageTh         string      `json:"main_image_th"`
	HotelType           string      `json:"hotel_type"`
	HotelTypeID         int         `json:"hotel_type_id"`
	Chain               string      `json:"chain"`
	ChainID             int         `json:"chain_id"`
	Latitude            float64     `json:"latitude"`
	Longitude           float64     `json:"longitude"`
	HotelName           string      `json:"hotel_name"`
	Phone               string      `json:"phone"`
	Fax                 string      `json:"fax"`
	Email               string      `json:"email"`
	Address             Address     `json:"address"`
	Stars               int32       `json:"stars"`
	AirportCode         string      `json:"airport_code"`
	Rating              float64     `json:"rating"`
	ReviewCount         int         `json:"review_count"`
	Checkin             CheckinInfo `json:"checkin"`
	Parking             string      `json:"parking"`
	GroupRoomMin        any         `json:"group_room_min"`
	ChildAllowed        bool        `json:"child_allowed"`
	PetsAllowed         bool        `json:"pets_allowed"`
	Photos              []Photo     `json:"photos"`
	Description         string      `json:"description"`
	MarkdownDescription string      `json:"markdown_description"`
	ImportantInfo       string      `json:"important_info"`
	Facilities          []Facility  `json:"facilities"`
	Policies            []Policy    `json:"policies"`
	Rooms               []Room      `json:"rooms"`
	Reviews             any         `json:"reviews"`
}

type Address struct {
	Address    string `json:"address"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postal_code"`
}

type CheckinInfo struct {
	CheckinStart        string   `json:"checkin_start"`
	CheckinEnd          string   `json:"checkin_end"`
	Checkout            string   `json:"checkout"`
	Instructions        []string `json:"instructions"`
	SpecialInstructions string   `json:"special_instructions"`
}

type Photo struct {
	URL              string  `json:"url"`
	HDURL            string  `json:"hd_url"`
	ImageDescription string  `json:"image_description"`
	ImageClass1      string  `json:"image_class1"`
	ImageClass2      string  `json:"image_class2"`
	MainPhoto        bool    `json:"main_photo"`
	Score            float64 `json:"score"`
	ClassID          int     `json:"class_id"`
	ClassOrder       int     `json:"class_order"`
}

type Facility struct {
	FacilityID int    `json:"facility_id"`
	Name       string `json:"name"`
}

type Policy struct {
	PolicyType   string `json:"policy_type"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ChildAllowed string `json:"child_allowed"`
	PetsAllowed  string `json:"pets_allowed"`
	Parking      string `json:"parking"`
	ID           int    `json:"id"`
}

type Room struct {
	ID             int       `json:"id"`
	RoomName       string    `json:"room_name"`
	Description    string    `json:"description"`
	RoomSizeSquare float32   `json:"room_size_square"`
	RoomSizeUnit   string    `json:"room_size_unit"`
	HotelID        string    `json:"hotel_id"`
	MaxAdults      int       `json:"max_adults"`
	MaxChildren    int       `json:"max_children"`
	MaxOccupancy   int       `json:"max_occupancy"`
	BedRelation    string    `json:"bed_relation"`
	BedTypes       []BedType `json:"bed_types"`
	RoomAmenities  []Amenity `json:"room_amenities"`
	Photos         []Photo   `json:"photos"`
	Views          []any     `json:"views"`
}

type BedType struct {
	Quantity int    `json:"quantity"`
	BedType  string `json:"bed_type"`
	BedSize  string `json:"bed_size"`
	ID       int    `json:"id"`
}

type Amenity struct {
	AmenitiesID int    `json:"amenities_id"`
	Name        string `json:"name"`
	Sort        int    `json:"sort"`
}
type ReviewAPIResponse struct {
	ReviewID     int64  `json:"review_id"`
	AverageScore int32  `json:"average_score"`
	Country      string `json:"country"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Date         string `json:"date"`
	Headline     string `json:"headline"`
	Language     string `json:"language"`
	Pros         string `json:"pros"`
	Cons         string `json:"cons"`
	Source       string `json:"source"`
}

type TranslationAPIResponse struct {
	HotelID             int64       `json:"hotel_id"`
	CupidID             int         `json:"cupid_id"`
	MainImageTh         string      `json:"main_image_th"`
	HotelType           string      `json:"hotel_type"`
	HotelTypeID         int         `json:"hotel_type_id"`
	Chain               string      `json:"chain"`
	ChainID             int         `json:"chain_id"`
	Latitude            float64     `json:"latitude"`
	Longitude           float64     `json:"longitude"`
	HotelName           string      `json:"hotel_name"`
	Phone               string      `json:"phone"`
	Fax                 string      `json:"fax"`
	Email               string      `json:"email"`
	Address             Address     `json:"address"`
	Stars               int8        `json:"stars"`
	AirportCode         string      `json:"airport_code"`
	Rating              float64     `json:"rating"`
	ReviewCount         int         `json:"review_count"`
	Checkin             CheckinInfo `json:"checkin"`
	Parking             string      `json:"parking"`
	GroupRoomMin        any         `json:"group_room_min"`
	ChildAllowed        bool        `json:"child_allowed"`
	PetsAllowed         bool        `json:"pets_allowed"`
	Photos              []Photo     `json:"photos"`
	Description         string      `json:"description"`
	MarkdownDescription string      `json:"markdown_description"`
	ImportantInfo       string      `json:"important_info"`
	Facilities          []Facility  `json:"facilities"`
	Policies            []Policy    `json:"policies"`
	Rooms               []Room      `json:"rooms"`
	Reviews             any         `json:"reviews"`
}

type TranslationInfo struct {
	SourceLanguage string         `json:"source_language"`
	TargetLanguage string         `json:"target_language"`
	FieldName      string         `json:"field_name"`
	OriginalText   string         `json:"original_text"`
	TranslatedText string         `json:"translated_text"`
	Quality        float32        `json:"quality"`
	Confidence     float32        `json:"confidence"`
	Provider       string         `json:"provider"`
	Method         string         `json:"method"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type ReviewFetchOptions struct {
	ReviewCount int64
}

type TranslationFetchOptions struct {
	Lang string
}

func (hotelAPIResponse *HotelAPIResponse) ToHotelData() (*entities.HotelData, error) {
	hotelData := &entities.HotelData{
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

	addressMap := map[string]string{
		"address":     hotelAPIResponse.Address.Address,
		"city":        hotelAPIResponse.Address.City,
		"state":       hotelAPIResponse.Address.State,
		"country":     hotelAPIResponse.Address.Country,
		"postal_code": hotelAPIResponse.Address.PostalCode,
	}
	if err := hotelData.SetAddress(addressMap); err != nil {
		return nil, fmt.Errorf("failed to set address: %w", err)
	}

	contactMap := map[string]string{
		"phone": hotelAPIResponse.Phone,
		"fax":   hotelAPIResponse.Fax,
		"email": hotelAPIResponse.Email,
	}
	if err := hotelData.SetContactInfo(contactMap); err != nil {
		return nil, fmt.Errorf("failed to set contact info: %w", err)
	}

	policiesMap := make(map[string]any)
	for i, policy := range hotelAPIResponse.Policies {
		policyKey := fmt.Sprintf("policy_%d", i)
		description := policy.Description
		name := policy.Name

		policiesMap[policyKey] = map[string]any{
			"type":          policy.PolicyType,
			"name":          name,
			"description":   description,
			"child_allowed": policy.ChildAllowed,
			"pets_allowed":  policy.PetsAllowed,
			"parking":       policy.Parking,
			"id":            policy.ID,
		}
	}
	if err := hotelData.SetPolicies(policiesMap); err != nil {
		return nil, fmt.Errorf("failed to set policies: %w", err)
	}

	if len(hotelAPIResponse.Photos) > 0 {
		photosData, err := json.Marshal(hotelAPIResponse.Photos)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal photos: %w", err)
		}
		hotelData.Photos = photosData
	}

	if len(hotelAPIResponse.Facilities) > 0 {
		facilitiesData, err := json.Marshal(hotelAPIResponse.Facilities)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal facilities: %w", err)
		}
		hotelData.Facilities = facilitiesData
	}

	if hotelAPIResponse.Checkin.CheckinStart != "" || hotelAPIResponse.Checkin.CheckinEnd != "" || hotelAPIResponse.Checkin.Checkout != "" {
		checkinData, err := json.Marshal(hotelAPIResponse.Checkin)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal checkin: %w", err)
		}
		hotelData.Checkin = checkinData
	}

	if len(hotelAPIResponse.Rooms) > 0 {
		roomsData, err := json.Marshal(hotelAPIResponse.Rooms)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal rooms: %w", err)
		}
		hotelData.Rooms = roomsData
	}

	if hotelAPIResponse.GroupRoomMin != nil {
		groupRoomMinData, err := json.Marshal(hotelAPIResponse.GroupRoomMin)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal group_room_min: %w", err)
		}
		hotelData.GroupRoomMin = groupRoomMinData
	}

	return hotelData, nil
}

func (translationAPIResponse *TranslationAPIResponse) ToHotelTranslations(lang string) (*entities.HotelTranslation, error) {
	hotelData := &entities.HotelTranslation{
		HotelID:             translationAPIResponse.HotelID,
		Name:                translationAPIResponse.HotelName,
		Description:         translationAPIResponse.Description,
		Chain:               translationAPIResponse.Chain,
		Parking:             translationAPIResponse.Parking,
		MarkdownDescription: translationAPIResponse.MarkdownDescription,
		ImportantInfo:       translationAPIResponse.ImportantInfo,
		Lang:                lang,
	}

	addressMap := map[string]string{
		"address":     translationAPIResponse.Address.Address,
		"city":        translationAPIResponse.Address.City,
		"state":       translationAPIResponse.Address.State,
		"country":     translationAPIResponse.Address.Country,
		"postal_code": translationAPIResponse.Address.PostalCode,
	}
	if err := hotelData.SetAddress(addressMap); err != nil {
		return nil, fmt.Errorf("failed to set address: %w", err)
	}

	contactMap := map[string]string{
		"phone": translationAPIResponse.Phone,
		"fax":   translationAPIResponse.Fax,
		"email": translationAPIResponse.Email,
	}
	if err := hotelData.SetContactInfo(contactMap); err != nil {
		return nil, fmt.Errorf("failed to set contact info: %w", err)
	}

	policiesMap := make(map[string]any)
	for i, policy := range translationAPIResponse.Policies {
		policyKey := fmt.Sprintf("policy_%d", i)
		description := policy.Description
		name := policy.Name

		policiesMap[policyKey] = map[string]any{
			"type":          policy.PolicyType,
			"name":          name,
			"description":   description,
			"child_allowed": policy.ChildAllowed,
			"pets_allowed":  policy.PetsAllowed,
			"parking":       policy.Parking,
			"id":            policy.ID,
		}
	}
	if err := hotelData.SetPolicies(policiesMap); err != nil {
		return nil, fmt.Errorf("failed to set policies: %w", err)
	}

	if len(translationAPIResponse.Photos) > 0 {
		photosData, err := json.Marshal(translationAPIResponse.Photos)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal photos: %w", err)
		}
		hotelData.Photos = photosData
	}

	if len(translationAPIResponse.Facilities) > 0 {
		facilitiesData, err := json.Marshal(translationAPIResponse.Facilities)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal facilities: %w", err)
		}
		hotelData.Facilities = facilitiesData
	}

	if translationAPIResponse.Checkin.CheckinStart != "" || translationAPIResponse.Checkin.CheckinEnd != "" || translationAPIResponse.Checkin.Checkout != "" {
		checkinData, err := json.Marshal(translationAPIResponse.Checkin)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal checkin: %w", err)
		}
		hotelData.Checkin = checkinData
	}

	if len(translationAPIResponse.Rooms) > 0 {
		roomsData, err := json.Marshal(translationAPIResponse.Rooms)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal rooms: %w", err)
		}
		hotelData.Rooms = roomsData
	}

	if translationAPIResponse.GroupRoomMin != nil {
		groupRoomMinData, err := json.Marshal(translationAPIResponse.GroupRoomMin)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal group_room_min: %w", err)
		}
		hotelData.GroupRoomMin = groupRoomMinData
	}

	return hotelData, nil
}

func (reviewApiResponse *ReviewAPIResponse) ToReviewData(hotelID int64) (*entities.ReviewData, error) {
	review := &entities.ReviewData{
		HotelID:      hotelID,
		ReviewID:     reviewApiResponse.ReviewID,
		AverageScore: reviewApiResponse.AverageScore,
		Country:      reviewApiResponse.Country,
		Type:         reviewApiResponse.Type,
		Name:         reviewApiResponse.Name,
		Headline:     reviewApiResponse.Headline,
		Language:     reviewApiResponse.Language,
		Pros:         reviewApiResponse.Pros,
		Cons:         reviewApiResponse.Cons,
		Source:       reviewApiResponse.Source,
	}

	if reviewApiResponse.Date != "" {
		if parsedTime, err := time.Parse("2006-01-02 15:04:05", reviewApiResponse.Date); err == nil {
			review.Date = parsedTime
		}
	}

	return review, nil
}

type ReviewDataList []*ReviewAPIResponse

func (reviewDataList ReviewDataList) ToReviewDataList(hotelID int64) ([]*entities.ReviewData, error) {
	list := make([]*entities.ReviewData, 0, len(reviewDataList))
	for _, ri := range reviewDataList {
		mapped, err := ri.ToReviewData(hotelID)
		if err != nil {
			return nil, err
		}
		list = append(list, mapped)
	}
	return list, nil
}

type TranslationsAPIResponse struct {
	HotelAPIResponse
}
