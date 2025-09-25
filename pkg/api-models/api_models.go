package apimodels

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
	ID   int    `json:"id"`
	Name string `json:"name"`
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
	AmenityID int    `json:"amenities_id"`
	Name      string `json:"name"`
	Sort      int    `json:"sort"`
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

type ReviewFetchOptions struct {
	Language string `json:"language"`
}

type TranslationFetchOptions struct {
	Lang string `json:"lang"`
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

	addressData := map[string]string{
		"address":     hotelAPIResponse.Address.Address,
		"city":        hotelAPIResponse.Address.City,
		"state":       hotelAPIResponse.Address.State,
		"country":     hotelAPIResponse.Address.Country,
		"postal_code": hotelAPIResponse.Address.PostalCode,
	}
	if err := hotelData.SetAddress(addressData); err != nil {
		return nil, fmt.Errorf("error setting address: %w", err)
	}

	facilities := make([]string, len(hotelAPIResponse.Facilities))
	for i, facility := range hotelAPIResponse.Facilities {
		facilities[i] = facility.Name
	}
	if err := hotelData.SetFacilities(facilities); err != nil {
		return nil, fmt.Errorf("error setting facilities: %w", err)
	}

	policies := make(map[string]any)
	for _, policy := range hotelAPIResponse.Policies {
		policies[policy.PolicyType] = map[string]interface{}{
			"name":          policy.Name,
			"description":   policy.Description,
			"child_allowed": policy.ChildAllowed,
			"pets_allowed":  policy.PetsAllowed,
			"parking":       policy.Parking,
		}
	}
	if err := hotelData.SetPolicies(policies); err != nil {
		return nil, fmt.Errorf("error setting policies: %w", err)
	}

	contact := map[string]string{
		"phone": hotelAPIResponse.Phone,
		"fax":   hotelAPIResponse.Fax,
		"email": hotelAPIResponse.Email,
	}
	if err := hotelData.SetContactInfo(contact); err != nil {
		return nil, fmt.Errorf("error setting contact info: %w", err)
	}

	checkinData := map[string]interface{}{
		"checkin_start":        hotelAPIResponse.Checkin.CheckinStart,
		"checkin_end":          hotelAPIResponse.Checkin.CheckinEnd,
		"checkout":             hotelAPIResponse.Checkin.Checkout,
		"instructions":         hotelAPIResponse.Checkin.Instructions,
		"special_instructions": hotelAPIResponse.Checkin.SpecialInstructions,
	}
	checkinBytes, err := json.Marshal(checkinData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling checkin data: %w", err)
	}
	hotelData.Checkin = checkinBytes

	groupRoomMinBytes, err := json.Marshal(hotelAPIResponse.GroupRoomMin)
	if err != nil {
		return nil, fmt.Errorf("error marshaling group room min: %w", err)
	}
	hotelData.GroupRoomMin = groupRoomMinBytes

	photosBytes, err := json.Marshal(hotelAPIResponse.Photos)
	if err != nil {
		return nil, fmt.Errorf("error marshaling photos: %w", err)
	}
	hotelData.Photos = photosBytes

	roomsBytes, err := json.Marshal(hotelAPIResponse.Rooms)
	if err != nil {
		return nil, fmt.Errorf("error marshaling rooms: %w", err)
	}
	hotelData.Rooms = roomsBytes

	return hotelData, nil
}

func (translationAPIResponse *TranslationAPIResponse) ToHotelTranslations(lang string) (*entities.HotelTranslation, error) {
	translation := &entities.HotelTranslation{
		Lang:                lang,
		HotelID:             translationAPIResponse.HotelID,
		Name:                translationAPIResponse.HotelName,
		Description:         translationAPIResponse.Description,
		Chain:               translationAPIResponse.Chain,
		Parking:             translationAPIResponse.Parking,
		MarkdownDescription: translationAPIResponse.MarkdownDescription,
		ImportantInfo:       translationAPIResponse.ImportantInfo,
	}

	addressData := map[string]string{
		"address":     translationAPIResponse.Address.Address,
		"city":        translationAPIResponse.Address.City,
		"state":       translationAPIResponse.Address.State,
		"country":     translationAPIResponse.Address.Country,
		"postal_code": translationAPIResponse.Address.PostalCode,
	}
	if err := translation.SetAddress(addressData); err != nil {
		return nil, fmt.Errorf("error setting address: %w", err)
	}

	facilities := make([]string, len(translationAPIResponse.Facilities))
	for i, facility := range translationAPIResponse.Facilities {
		facilities[i] = facility.Name
	}
	facilitiesBytes, err := json.Marshal(facilities)
	if err != nil {
		return nil, fmt.Errorf("error marshaling facilities: %w", err)
	}
	translation.Facilities = facilitiesBytes

	policies := make(map[string]any)
	for _, policy := range translationAPIResponse.Policies {
		policies[policy.PolicyType] = map[string]interface{}{
			"name":          policy.Name,
			"description":   policy.Description,
			"child_allowed": policy.ChildAllowed,
			"pets_allowed":  policy.PetsAllowed,
			"parking":       policy.Parking,
		}
	}
	if err := translation.SetPolicies(policies); err != nil {
		return nil, fmt.Errorf("error setting policies: %w", err)
	}

	contact := map[string]string{
		"phone": translationAPIResponse.Phone,
		"fax":   translationAPIResponse.Fax,
		"email": translationAPIResponse.Email,
	}
	if err := translation.SetContactInfo(contact); err != nil {
		return nil, fmt.Errorf("error setting contact info: %w", err)
	}

	checkinData := map[string]interface{}{
		"checkin_start":        translationAPIResponse.Checkin.CheckinStart,
		"checkin_end":          translationAPIResponse.Checkin.CheckinEnd,
		"checkout":             translationAPIResponse.Checkin.Checkout,
		"instructions":         translationAPIResponse.Checkin.Instructions,
		"special_instructions": translationAPIResponse.Checkin.SpecialInstructions,
	}
	checkinBytes, err := json.Marshal(checkinData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling checkin data: %w", err)
	}
	translation.Checkin = checkinBytes

	groupRoomMinBytes, err := json.Marshal(translationAPIResponse.GroupRoomMin)
	if err != nil {
		return nil, fmt.Errorf("error marshaling group room min: %w", err)
	}
	translation.GroupRoomMin = groupRoomMinBytes

	photosBytes, err := json.Marshal(translationAPIResponse.Photos)
	if err != nil {
		return nil, fmt.Errorf("error marshaling photos: %w", err)
	}
	translation.Photos = photosBytes

	roomsBytes, err := json.Marshal(translationAPIResponse.Rooms)
	if err != nil {
		return nil, fmt.Errorf("error marshaling rooms: %w", err)
	}
	translation.Rooms = roomsBytes

	return translation, nil
}

func (reviewApiResponse *ReviewAPIResponse) ToReviewData(hotelID int64) (*entities.ReviewData, error) {
	reviewData := &entities.ReviewData{
		HotelID:      hotelID,
		ReviewID:     reviewApiResponse.ReviewID,
		AverageScore: int32(reviewApiResponse.AverageScore),
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
		if parsedDate, err := time.Parse(time.RFC3339, reviewApiResponse.Date); err == nil {
			reviewData.Date = parsedDate
		}
	}

	return reviewData, nil
}

type ReviewDataList []*ReviewAPIResponse

func (reviewDataList ReviewDataList) ToReviewDataList(hotelID int64) ([]*entities.ReviewData, error) {
	reviews := make([]*entities.ReviewData, 0, len(reviewDataList))
	for _, reviewAPIResponse := range reviewDataList {
		if reviewData, err := reviewAPIResponse.ToReviewData(hotelID); err == nil {
			reviews = append(reviews, reviewData)
		}
	}
	return reviews, nil
}
