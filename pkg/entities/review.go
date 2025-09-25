package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReviewData struct {
	ID           string         `gorm:"primaryKey;type:varchar(36)"`
	HotelID      int64          `gorm:"not null;index:idx_reviews_hotel_id"`
	ReviewID     int64          `gorm:"uniqueIndex"`
	AverageScore int32          `gorm:"not null"`
	Country      string         `gorm:"type:varchar(100)"`
	Type         string         `gorm:"type:varchar(50)"`
	Name         string         `gorm:"type:varchar(255)"`
	Date         time.Time      `gorm:"not null"`
	Headline     string         `gorm:"type:varchar(500)"`
	Language     string         `gorm:"type:varchar(10);default:en"`
	Pros         string         `gorm:"type:text"`
	Cons         string         `gorm:"type:text"`
	Source       string         `gorm:"type:varchar(50)"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	NextUpdateAt time.Time      `gorm:"not null"`

	Hotel HotelData `gorm:"foreignKey:HotelID;references:HotelID"`
}

func (r *ReviewData) BeforeCreate(_ *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	r.NextUpdateAt = time.Now()
	if r.Language == "" {
		r.Language = "en"
	}
	return
}

func (r *ReviewData) BeforeUpdate(_ *gorm.DB) (err error) {
	r.UpdatedAt = time.Now()
	return
}

func (r *ReviewData) TableName() string {
	return "reviews"
}
