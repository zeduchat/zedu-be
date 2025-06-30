package postgresql

import "gorm.io/gorm"

func CountRecords(db *gorm.DB, model any) (int64, error) {
	var count int64
	result := db.Model(model).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func CountSpecificRecords(db *gorm.DB, model any, query string, args ...any) (int64, error) {
	var count int64
	result := db.Model(model).Where(query, args...).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}
