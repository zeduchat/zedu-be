package migrations

import (
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"gorm.io/gorm"
)

func RunAllMigrations(db *storage.Database) {

	// verification migration
	MigrateModels(db.Postgresql, AuthMigrationModels(), AlterColumnModels())

}

func MigrateModels(db *gorm.DB, models []any, AlterColums []AlterColumn) {
	_ = db.AutoMigrate(models...)

	// for _, alter := range AlterColums {
	// 	if alter.Column != "" {
	// 		alter.AddColumn(db)
	// 	}
	// }
}
