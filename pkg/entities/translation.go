package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type HotelTranslation struct {
	ID string `gorm:"primaryKey;type:varchar(36)"`

	HotelID int64 `gorm:"not null"`

	Name        string         `gorm:"not null;type:varchar(255)"`
	Description string         `gorm:"type:text"`
	Address     datatypes.JSON `gorm:"type:jsonb"`

	Policies            datatypes.JSON `gorm:"type:jsonb"`
	ContactInfo         datatypes.JSON `gorm:"type:jsonb"`
	Status              string         `gorm:"type:varchar(20);default:active;index:idx_hotels_status"`
	Source              string         `gorm:"type:varchar(50);default:cupid_api"`
	Chain               string         `gorm:"type:varchar(255)"`
	Checkin             datatypes.JSON `gorm:"type:jsonb"`
	Parking             string         `gorm:"type:varchar(50)"`
	GroupRoomMin        datatypes.JSON `gorm:"type:jsonb"`
	Photos              datatypes.JSON `gorm:"type:jsonb"`
	MarkdownDescription string         `gorm:"type:text"`
	ImportantInfo       string         `gorm:"type:text"`
	Facilities          datatypes.JSON `gorm:"type:jsonb"`
	Rooms               datatypes.JSON `gorm:"type:jsonb"`

	Lang string `gorm:"type:varchar(10)"`

	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	NextUpdateAt time.Time      `gorm:"not null"`

	HotelData []HotelData `gorm:"foreignKey:HotelID;references:HotelID"`
}

func (t *HotelTranslation) TableName() string {
	return "translations"
}

func (t *HotelTranslation) BeforeCreate(_ *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	t.NextUpdateAt = time.Now()

	if t.Status == "" {
		t.Status = "active"
	}
	if t.Source == "" {
		t.Source = "cupid_api"
	}
	return
}

func (t *HotelTranslation) BeforeUpdate(_ *gorm.DB) (err error) {
	t.UpdatedAt = time.Now()
	if !t.DeletedAt.Valid {
		t.Status = "active"
	}

	return
}

func (t *HotelTranslation) SetPolicies(policies map[string]any) error {
	if len(policies) == 0 {
		t.Policies = datatypes.JSON("")
		return nil
	}
	data, err := json.Marshal(policies)
	if err != nil {
		return err
	}
	t.Policies = data
	return nil
}

func (t *HotelTranslation) SetContactInfo(contact map[string]string) error {
	if len(contact) == 0 {
		t.ContactInfo = datatypes.JSON("")
		return nil
	}
	data, err := json.Marshal(contact)
	if err != nil {
		return err
	}
	t.ContactInfo = data
	return nil
}

func (t *HotelTranslation) SetAddress(address map[string]string) error {
	if len(address) == 0 {
		t.Address = datatypes.JSON("")
		return nil
	}
	data, err := json.Marshal(address)
	if err != nil {
		return err
	}
	t.Address = data
	return nil
}
