package hotel

import (
	"time"
)

type Facility struct {
	ID   int
	Name string
}

type Hotel struct {
	ID                  string
	HotelID             int64
	CupidID             int64
	Name                string
	Description         string
	Address             Address
	Rating              float64
	StarRating          int32
	Location            Location
	Images              []string
	Amenities           []string
	Policies            []Policy
	ContactInfo         ContactInfo
	Status              string
	Source              string
	MainImageTh         string
	HotelType           string
	Chain               string
	ChainID             int32
	Phone               string
	Fax                 string
	Email               string
	AirportCode         string
	ReviewCount         int32
	CheckinInfo         CheckinInfo
	Parking             string
	ChildAllowed        bool
	PetsAllowed         bool
	Photos              []Photo
	MarkdownDescription string
	ImportantInfo       string
	Facilities          []Facility
	Rooms               []Room
	Reviews             []Review
	Translations        []Translation
	CreatedAt           time.Time
	UpdatedAt           time.Time
	NextUpdateAt        time.Time
	HotelTypeID         int64
	Latitude            float64
	Longitude           float64
}

type Address struct {
	Street     string
	City       string
	State      string
	Country    string
	PostalCode string
}

type Location struct {
	Latitude  float64
	Longitude float64
}

type ContactInfo struct {
	Phone string
	Fax   string
	Email string
}

type CheckinInfo struct {
	CheckinStart        time.Time
	CheckinEnd          time.Time
	Checkout            time.Time
	Instructions        []string
	SpecialInstructions string
}

type Photo struct {
	URL              string
	HDURL            string
	ImageDescription string
	ImageClass1      string
	ImageClass2      string
	MainPhoto        bool
	Score            float64
	ClassID          int
	ClassOrder       int
}

type Room struct {
	ID             int
	RoomName       string
	Description    string
	RoomSizeSquare float32
	RoomSizeUnit   string
	HotelID        string
	MaxAdults      int
	MaxChildren    int
	MaxOccupancy   int
	BedRelation    string
	BedTypes       []BedType
	RoomAmenities  []Amenity
	Photos         []RoomPhoto
	Views          []interface{}
}

type BedType struct {
	Quantity int
	BedType  string
	BedSize  string
	ID       int
}

type Amenity struct {
	AmenitiesID int
	Name        string
	Sort        int
}

type RoomPhoto struct {
	URL              string
	HDURL            string
	ImageDescription string
	ImageClass1      string
	ImageClass2      string
	MainPhoto        bool
	Score            float64
	ClassID          int
	ClassOrder       int
}

type Review struct {
	ID           string
	HotelID      int64
	ReviewID     int64
	AverageScore int32
	Country      string
	Type         string
	Name         string
	Date         time.Time
	Headline     string
	Language     string
	Pros         string
	Cons         string
	Source       string
}

type Policy struct {
	PolicyType   string
	Name         string
	Description  string
	ChildAllowed string
	PetsAllowed  string
	Parking      string
	ID           int
}

type Translation struct {
	ID                  string
	HotelID             int64
	Name                string
	Description         string
	Address             Address
	Images              []string
	Amenities           []string
	Policies            []Policy
	ContactInfo         ContactInfo
	Status              string
	Source              string
	Chain               string
	CheckinInfo         CheckinInfo
	Parking             string
	Photos              []Photo
	MarkdownDescription string
	ImportantInfo       string
	Facilities          []Facility
	Rooms               []Room
	Reviews             []Review
	Translations        []Translation
	CreatedAt           time.Time
	UpdatedAt           time.Time
	NextUpdateAt        time.Time
	Lang                string
	HotelTypeID         int64
	Latitude            float64
	Longitude           float64
}
