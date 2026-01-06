package tests

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

func TestBuildDatabaseURL(t *testing.T) {
	db := config.Database{
		USERNAME: "test_user",
		PASSWORD: "test_pass",
		DB_HOST:  "localhost",
		DB_PORT:  "5432",
		DB_NAME:  "test_db",
		SSLMODE:  "disable",
	}

	url := buildDatabaseURL(db)
	expected := "postgres://test_user:test_pass@localhost:5432/test_db?sslmode=disable"

	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestBuildDatabaseURLWithSSL(t *testing.T) {
	db := config.Database{
		USERNAME: "test_user",
		PASSWORD: "test_pass",
		DB_HOST:  "localhost",
		DB_PORT:  "5432",
		DB_NAME:  "test_db",
		SSLMODE:  "require",
	}

	url := buildDatabaseURL(db)
	expected := "postgres://test_user:test_pass@localhost:5432/test_db?sslmode=require"

	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestGetMigrationsPath(t *testing.T) {
	logger := utility.NewLogger()

	path, err := getMigrationsPath()
	if err != nil {
		t.Fatalf("failed to get migrations path: %v", err)
	}

	logger.Info("Migrations path: %s", path)

	expectedSuffix := "db/migrations"
	actual := path[len(path)-len(expectedSuffix):]
	if actual != expectedSuffix {
		t.Errorf("expected path to end with %s, got %s", expectedSuffix, path)
	}
}
