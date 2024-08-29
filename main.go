package main

import (
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/stripe/stripe-go/v72"

	"github.com/hngprojects/telex_be/cronjobs"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/internal/models/migrations"
	"github.com/hngprojects/telex_be/internal/models/seed"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
	"github.com/hngprojects/telex_be/pkg/router"
	"github.com/hngprojects/telex_be/utility"
)

func main() {
	logger := utility.NewLogger() //Warning !!!!! Do not recreate this action anywhere on the app

	configuration := config.Setup(logger, "./app")
	//connect to stripe
	stripe.Key = configuration.Stripe.STRIPE_KEY
	postgresql.ConnectToDatabase(logger, configuration.Database)
	redis.ConnectToRedis(logger, configuration.Redis)
	minio.ConnectToMinio(logger, configuration.Minio)
	centrifuge.NewCentrifugoService(logger, configuration.Centrifuge)
	typesense.ConnectToTypeSense(logger, configuration.TypeSense)
	models.SetStripeMap(configuration.Stripe)

	validatorRef := validator.New()
	db := storage.Connection()

	cronjobs.StartCronJob(request.ExternalRequest{Logger: logger}, *storage.DB, "send-notifications")

	if configuration.Database.Migrate {
		migrations.RunAllMigrations(db)
		seed.SeedRolesAndPermissions(logger, db.Postgresql)
	}

	r := router.Setup(logger, validatorRef, db, &configuration.App)

	utility.LogAndPrint(logger, fmt.Sprintf("Server is starting at 127.0.0.1:%s", configuration.Server.Port))
	log.Fatal(r.Run(":" + configuration.Server.Port))
}
