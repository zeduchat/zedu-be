package postgresql

import "gorm.io/gorm"

func DeleteRecordFromDb(db *gorm.DB, record interface{}) error {
	tx := db.Delete(record)
	return tx.Error
}

func DeleteSpecificRecord(db *gorm.DB, model interface{}, query string, args ...interface{}) error {
	if err := db.Where(query, args...).Delete(model).Error; err != nil {
		return err
	}
	return nil
}

func DeleteRecordWithNoModel(db *gorm.DB, query string, args ...interface{}) error {
	if err := db.Exec(query, args...).Error; err != nil {
		return err
	}
	return nil
}

func HardDeleteRecordFromDb(db *gorm.DB, record interface{}) error {
	tx := db.Unscoped().Delete(record)
	return tx.Error
}
