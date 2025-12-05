package postgresql

import (
	"gorm.io/gorm"
)

func SaveAllFields(db *gorm.DB, model any) (*gorm.DB, error) {
	result := db.Save(model)
	if result.Error != nil {
		return result, result.Error
	}
	return result, nil
}

func SaveAllModelsFields(db *gorm.DB, models []any) (*gorm.DB, error) {
	// Use a transaction to ensure atomicity of updates
	tx := db.Begin()
	if tx.Error != nil {
		return tx, tx.Error
	}

	// Loop through each model and update it
	for _, model := range models {
		result := tx.Save(model)
		if result.Error != nil {
			// If any update fails, rollback the transaction and return the error
			tx.Rollback()
			return result, result.Error
		}
	}

	// Commit the transaction if all updates are successful
	if err := tx.Commit().Error; err != nil {
		return tx, err
	}

	return tx, nil
}

func UpdateFields(db *gorm.DB, model any, updates any, query any, args ...any) (*gorm.DB, error) {
	result := db.Model(model).Where(query, args...).Updates(updates)
	if result.Error != nil {
		return result, result.Error
	}
	return result, nil
}

func UpdateFieldsAndReturn(db *gorm.DB, model any, updates any, query any, args ...any) (*gorm.DB, error) {
	result := db.Model(model).
		Where(query, args...).
		Updates(updates)

	if result.Error != nil {
		return result, result.Error
	}

	// Fetch the updated record
	if err := db.Where(query, args...).First(model).Error; err != nil {
		return result, err
	}

	return result, nil
}

func UpdateFieldsInTransaction(db *gorm.DB, updates []ModelUpdate) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			result := tx.Model(update.Model).Where(update.Where, update.Args...).Updates(update.Updates)
			if result.Error != nil {
				tx.Rollback()
				return result.Error
			}
		}
		return nil
	})
}

type ModelUpdate struct {
	Model   any
	Updates any
	Where   string
	Args    []any
}
