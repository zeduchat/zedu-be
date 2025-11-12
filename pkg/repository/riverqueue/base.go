package riverqueueBg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

// SetupRiver initializes River using pgx driver
func SetupRiver(ctx context.Context, configDatabase config.Database, logger *utility.Logger, db *storage.Database) (*river.Client[pgx.Tx], error) {

	dbsCV := configDatabase
	utility.LogAndPrint(logger, "connecting to database for riverqueue")
	dsnString := dsn(dbsCV.DB_HOST, dbsCV.USERNAME, dbsCV.PASSWORD, dbsCV.DB_NAME, dbsCV.DB_PORT, dbsCV.SSLMODE, dbsCV.TIMEZONE)
	dbPool, err := pgxpool.New(context.Background(), dsnString)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	// Create river driver for pgx
	driver := riverpgxv5.New(dbPool)

	// Run migrations
	riverMigration, err := rivermigrate.New(driver, &rivermigrate.Config{})
	if err != nil {
		return nil, fmt.Errorf("river migration failed to connect: %w", err)
	}

	res, err := riverMigration.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return nil, fmt.Errorf("river migration failed: %w", err)
	}

	logger.Info("<<<<<<<<<<<<<<<<<< River migration done, res: %v >>>>>>>>>>>>>>>>>>>>", res)

	// Create River client

	slogger := utility.NewRotatingLogger()

	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: registerWorkers(logger, db),
		Logger:  slogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize river client: %w", err)
	}

	storage.DB.River = client

	return client, nil
}

func dsn(host, user, password, dbname, port, sslmode, timezone string) string {
	return fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=%v TimeZone=%v", host, user, password, dbname, port, sslmode, timezone)
}
