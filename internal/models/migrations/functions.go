package migrations

type AlterColumn struct {
	Model     interface{}
	TableName string
	Column    string
	Type      string
}

// func (a *AlterColumn) UpdateColumnType(db *gorm.DB) error {
// 	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s::%s", a.TableName, a.Column, a.Type, a.Column, a.Type)).Error; err != nil {
// 		return err
// 	}

// 	// Update the GORM model to reflect the changes
// 	if err := db.Migrator().AlterColumn(a.Model, a.Column); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (a *AlterColumn) UpdateNull(db *gorm.DB) error {
// 	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", a.TableName, a.Column)).Error; err != nil {
// 		return err
// 	}

// 	if err := db.Migrator().AlterColumn(a.Model, a.Column); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (a *AlterColumn) AddColumn(db *gorm.DB) error {
// 	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s NULL", a.TableName, a.Column, a.Type)).Error; err != nil {
// 		return err
// 	}

// 	if err := db.Migrator().AlterColumn(a.Model, a.Column); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (a *AlterColumn) DropColumn(db *gorm.DB) error {
// 	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s %s", a.TableName, a.Column, a.Type)).Error; err != nil {
// 		return err
// 	}

// 	if err := db.Migrator().AlterColumn(a.Model, a.Column); err != nil {
// 		return err
// 	}

// 	return nil
// }
