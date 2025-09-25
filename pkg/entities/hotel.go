package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type HotelData struct {
	ID string `gorm:"primaryKey;type:varchar(36)"`

	HotelID     int64 `gorm:"not null"`
	CupidID     int64 `gorm:"not null"`
	HotelTypeID int64 `gorm:"type:integer"`

	Name                string         `gorm:"not null;type:varchar(255)"`
	Description         string         `gorm:"type:text"`
	Address             datatypes.JSON `gorm:"type:jsonb"`
	Rating              float64        `gorm:"type:decimal(3,2)"`
	StarRating          int32          `gorm:"type:smallint"`
	Latitude            float64        `gorm:"type:decimal(10,8)"`
	Longitude           float64        `gorm:"type:decimal(11,8)"`
	Amenities           datatypes.JSON `gorm:"type:jsonb"`
	Policies            datatypes.JSON `gorm:"type:jsonb"`
	ContactInfo         datatypes.JSON `gorm:"type:jsonb"`
	Status              string         `gorm:"type:varchar(20);default:active;index:idx_hotels_status"`
	Source              string         `gorm:"type:varchar(50);default:cupid_api"`
	MainImageTh         string         `gorm:"type:varchar(500)"`
	HotelType           string         `gorm:"type:varchar(100)"`
	Chain               string         `gorm:"type:varchar(255)"`
	ChainID             int32          `gorm:"type:integer"`
	Phone               string         `gorm:"type:varchar(50)"`
	Fax                 string         `gorm:"type:varchar(50)"`
	Email               string         `gorm:"type:varchar(255)"`
	AirportCode         string         `gorm:"type:varchar(10)"`
	ReviewCount         int32          `gorm:"type:integer"`
	Checkin             datatypes.JSON `gorm:"type:jsonb"`
	Parking             string         `gorm:"type:varchar(50)"`
	GroupRoomMin        datatypes.JSON `gorm:"type:jsonb"`
	ChildAllowed        bool           `gorm:"type:boolean"`
	PetsAllowed         bool           `gorm:"type:boolean"`
	Photos              datatypes.JSON `gorm:"type:jsonb"`
	MarkdownDescription string         `gorm:"type:text"`
	ImportantInfo       string         `gorm:"type:text"`
	Facilities          datatypes.JSON `gorm:"type:jsonb"`
	Rooms               datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	NextUpdateAt time.Time      `gorm:"not null"`

	ReviewsData      []ReviewData       `gorm:"foreignKey:HotelID;references:HotelID"`
	TranslationsData []HotelTranslation `gorm:"foreignKey:HotelID;references:HotelID"`
}

func (h *HotelData) BeforeCreate(_ *gorm.DB) (err error) {
	if h.ID == "" {
		h.ID = uuid.New().String()
	}
	h.CreatedAt = time.Now()
	h.UpdatedAt = time.Now()
	h.NextUpdateAt = time.Now()

	if h.Status == "" {
		h.Status = "active"
	}
	if h.Source == "" {
		h.Source = "cupid_api"
	}
	return
}

func (h *HotelData) BeforeUpdate(_ *gorm.DB) (err error) {
	h.UpdatedAt = time.Now()
	if !h.DeletedAt.Valid {
		h.Status = "active"
	}

	return
}

func (h *HotelData) TableName() string {
	return "hotels"
}

func (h *HotelData) SetAmenities(amenities []string) error {
	if len(amenities) == 0 {
		h.Amenities = datatypes.JSON("")
		return nil
	}
	data, err := json.Marshal(amenities)
	if err != nil {
		return err
	}
	h.Amenities = data
	return nil
}

func (h *HotelData) SetPolicies(policies map[string]any) error {
	if len(policies) == 0 {
		h.Policies = datatypes.JSON("")
		return nil
	}
	data, err := json.Marshal(policies)
	if err != nil {
		return err
	}
	h.Policies = data
	return nil
}

func (h *HotelData) SetContactInfo(contact map[string]string) error {
	if len(contact) == 0 {
		h.ContactInfo = datatypes.JSON("")
		return nil
	}
	data, err := json.Marshal(contact)
	if err != nil {
		return err
	}
	h.ContactInfo = data
	return nil
}

func (h *HotelData) SetAddress(address map[string]string) error {
	if len(address) == 0 {
		h.Address = datatypes.JSON("")
		return nil
	}
	data, err := json.Marshal(address)
	if err != nil {
		return err
	}
	h.Address = data
	return nil
}

func (h *HotelData) SetFacilities(facilities []string) error {
	if len(facilities) == 0 {
		h.Facilities = datatypes.JSON("")
		return nil
	}
	data, err := json.Marshal(facilities)
	if err != nil {
		return err
	}
	h.Facilities = data
	return nil
}
