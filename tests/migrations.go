package tests

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

func buildDatabaseURL(db config.Database) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		db.USERNAME,
		db.PASSWORD,
		db.DB_HOST,
		db.DB_PORT,
		db.DB_NAME,
		db.SSLMODE,
	)
}

func getMigrationsPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		path := filepath.Join(dir, "db", "migrations")
		if _, err := os.Stat(path); err == nil {
			return filepath.Abs(path)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	migrationsPath := filepath.Join(wd, "..", "db", "migrations")
	return filepath.Abs(migrationsPath)
}

func fixDirtyMigrationState(m *migrate.Migrate, logger *utility.Logger) error {
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if dirty {
		logger.Warning("Dirty migration state detected at version %d. Attempting to fix...", version)
		prevVersion := int(version) - 1
		if prevVersion < 0 {
			prevVersion = 0
		}
		if err := m.Force(prevVersion); err != nil {
			return fmt.Errorf("failed to force migration to version %d: %w", prevVersion, err)
		}
		logger.Info("Successfully cleared dirty migration state, forced to version %d", prevVersion)
	}
	return nil
}

func RunSQLMigrations(db config.Database, logger *utility.Logger) (bool, error) {
	dbURL := buildDatabaseURL(db)
	migrationsPath, err := getMigrationsPath()
	if err != nil {
		return false, fmt.Errorf("failed to get migrations path: %w", err)
	}

	logger.Info("Running SQL migrations from: %s", migrationsPath)
	logger.Info("Database: %s", db.DB_NAME)

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		dbURL,
	)
	if err != nil {
		return false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := fixDirtyMigrationState(m, logger); err != nil {
		return false, fmt.Errorf("failed to fix dirty migration state: %w", err)
	}

	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return false, fmt.Errorf("failed to check current version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		currentVersion = 0
	}

	if dirty {
		return false, fmt.Errorf("migration state is dirty at version %d", currentVersion)
	}

	logger.Info("Current migration version: %d", currentVersion)

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Info("No new migrations to apply")
			return true, nil
		}
		return false, fmt.Errorf("failed to run migrations: %w", err)
	}

	newVersion, _, _ := m.Version()
	logger.Info("Successfully applied migrations. Current version: %d", newVersion)

	return true, nil
}
