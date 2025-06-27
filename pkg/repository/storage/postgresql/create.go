package postgresql

import (
	"fmt"

	"gorm.io/gorm"
)

func CreateOneRecord(db *gorm.DB, model any) error {
	result := db.Create(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("record creation failed")
	}
	return nil
}

func CreateMultipleRecords(db *gorm.DB, model any, length int) error {
	result := db.Create(model)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != int64(length) {
		return fmt.Errorf("record creation failed")
	}
	return nil
}
