package migrations

import (
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"gorm.io/gorm"
)

func RunAllMigrations(db *storage.Database) {

	// verification migration
	MigrateModels(db.Postgresql, AuthMigrationModels(), AlterColumnModels(db.Postgresql))

}

func MigrateModels(db *gorm.DB, models []interface{}, AlterColums []AlterColumn) {
	_ = db.AutoMigrate(models...)

}
