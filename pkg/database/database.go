package database

import (
	"context"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GormOpen(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
}

func RunMigrations(db *gorm.DB, entities ...interface{}) error {
	if err := db.AutoMigrate(entities...); err != nil {
		return err
	}
	return nil
}

type IDWithHotelID struct {
	ID      string `json:"id"`
	HotelID int64  `json:"hotel_id"`
}

type HotelMissingLang struct {
	HotelID     int64  `json:"hotel_id"`
	MissingLang string `json:"missing_lang"`
}

func QueryHotelIDsByID(ctx context.Context, db *gorm.DB, lastHotelID int64, limit int) ([]IDWithHotelID, error) {
	var results []IDWithHotelID
	query := db.WithContext(ctx).
		Table("hotels").
		Select("id, hotel_id").
		Where("next_update_at < NOW() AND hotel_id > 0").
		Order("hotel_id ASC").
		Limit(limit)

	if lastHotelID > 0 {
		query = query.Where("hotel_id > ?", lastHotelID)
	}

	err := query.Find(&results).Error
	return results, err
}

func QueryReviewIDsByID(ctx context.Context, db *gorm.DB, lastHotelID int64, limit int) ([]IDWithHotelID, error) {
	var results []IDWithHotelID
	query := db.WithContext(ctx).
		Table("reviews").
		Select("id, hotel_id").
		Where("next_update_at < NOW() AND hotel_id > 0").
		Order("hotel_id ASC").
		Limit(limit)

	if lastHotelID > 0 {
		query = query.Where("hotel_id > ?", lastHotelID)
	}

	err := query.Find(&results).Error
	return results, err
}

func QueryTranslationIDsByID(ctx context.Context, db *gorm.DB, lastHotelID int64, limit int) ([]IDWithHotelID, error) {
	var results []IDWithHotelID
	query := db.WithContext(ctx).
		Table("translations").
		Select("id, hotel_id").
		Where("next_update_at < NOW() AND hotel_id > 0").
		Order("hotel_id ASC, lang ASC").
		Limit(limit)

	if lastHotelID > 0 {
		query = query.Where("hotel_id > ?", lastHotelID)
	}

	err := query.Find(&results).Error
	return results, err
}

func GetHotelsWithMissingTranslationsRaw(ctx context.Context, db *gorm.DB, lastHotelID int64, limit int) ([]HotelMissingLang, error) {
	var results []HotelMissingLang

	baseQuery := `SELECT h.hotel_id as hotel_id, 'es' as missing_lang, h.hotel_id as sort_key
FROM hotels h
WHERE NOT EXISTS (
    SELECT 1 
    FROM translations t 
    WHERE t.hotel_id = h.hotel_id AND t.lang = 'es'
) AND h.hotel_id > 0`

	if lastHotelID > 0 {
		baseQuery += ` AND h.hotel_id > ?`
	}

	baseQuery += `
UNION ALL
SELECT h.hotel_id as hotel_id, 'fr' as missing_lang, h.hotel_id as sort_key
FROM hotels h
WHERE NOT EXISTS (
    SELECT 1 
    FROM translations t 
    WHERE t.hotel_id = h.hotel_id AND t.lang = 'fr'
) AND h.hotel_id > 0`

	if lastHotelID > 0 {
		baseQuery += ` AND h.hotel_id > ?`
	}

	query := `SELECT hotel_id, missing_lang FROM (` + baseQuery + `) AS combined
ORDER BY sort_key ASC, missing_lang ASC
LIMIT ?`

	var err *gorm.DB
	if lastHotelID > 0 {
		err = db.WithContext(ctx).Raw(query, lastHotelID, lastHotelID, limit).Scan(&results)
	} else {
		err = db.WithContext(ctx).Raw(query, limit).Scan(&results)
	}

	if err.RowsAffected > 0 {
		return results, nil
	}
	return results, err.Error
}

func GetMissingReviewsFromHotelID(ctx context.Context, db *gorm.DB, lastHotelID int64, limit int) ([]IDWithHotelID, error) {
	var results []IDWithHotelID

	baseQuery := `SELECT h.id as id, h.hotel_id as hotel_id
FROM hotels h
WHERE NOT EXISTS (
    SELECT 1 
    FROM reviews r 
    WHERE r.hotel_id = h.hotel_id
) AND h.hotel_id > 0`

	if lastHotelID > 0 {
		baseQuery += ` AND h.hotel_id > ?`
	}

	query := baseQuery + `
ORDER BY h.hotel_id ASC
LIMIT ?`

	var err error
	if lastHotelID > 0 {
		err = db.WithContext(ctx).Raw(query, lastHotelID, limit).Scan(&results).Error
	} else {
		err = db.WithContext(ctx).Raw(query, limit).Scan(&results).Error
	}

	return results, err
}
