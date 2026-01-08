package main

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/stripe/stripe-go/v72"

	"github.com/hngprojects/telex_be/cronjobs"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/internal/models/migrations"
	"github.com/hngprojects/telex_be/internal/models/seed"
	"github.com/hngprojects/telex_be/pkg/repository/agora"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/firebase"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/onesignal"
	"github.com/hngprojects/telex_be/pkg/repository/rabbitmq"
	riverqueueBg "github.com/hngprojects/telex_be/pkg/repository/riverqueue"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/pkg/repository/storage/mongodb"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
	"github.com/hngprojects/telex_be/pkg/repository/webpush"
	"github.com/hngprojects/telex_be/pkg/router"
	np "github.com/hngprojects/telex_be/services/notification_processor"
	"github.com/hngprojects/telex_be/utility"
)

func main() {

	logger := utility.NewLogger() //Warning !!!!! Do not recreate this action anywhere on the apps

	configuration := config.Setup(logger, "./app")
	stripe.Key = configuration.Stripe.STRIPE_KEY
	postgresql.ConnectToDatabase(logger, configuration.Database)
	redis.ConnectToRedis(logger, configuration.Redis)
	minio.ConnectToMinio(logger, configuration.Minio)
	centrifuge.NewCentrifugoService(logger, configuration.Centrifuge)
	agora.NewAgoraService(logger, configuration.Agora)
	typesense.ConnectToTypeSense(logger, configuration.TypeSense)
	models.SetStripeMap(configuration.Stripe)
	models.SetMapPackagePriceID(configuration.Stripe)
	rabbitmq.QueueClient.QM = rabbitmq.NewQueueManager(configuration.RabbitMQ)
	rabbitmq.QueueClient.QM.Start(logger)
	elastic.ConnectToElastic(logger, configuration.Elastic)
	firebase.ConnectFirebase(logger, configuration.Firebase)
	onesignal.ConnectOneSignal(logger, configuration.OneSignal)
	mongodb.StartMongoDBConnection(logger, config.Config.MongoDB)
	webpush.NewPushClient(logger, configuration.WebPush)

	db := storage.Connection()

	// start river
	ctx := context.Background()
	riverClient, err := riverqueueBg.SetupRiver(ctx, configuration.Database, logger, db)

	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		riverClient.Stop(ctx)
	}()

	err = riverClient.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}

	validatorRef := validator.New()
	validatorRef.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("json")
		if name == "-" {
			return ""
		}
		return name
	})
	utility.RegisterCustomValidations(validatorRef)

	cronjobs.StartCronJob(request.ExternalRequest{Logger: logger}, *storage.DB, "send-notifications")
	dispatcher := np.NewDispatcher(0, 15, db, logger)
	dispatcher.Run()
	go np.FeedDispatcher(dispatcher)

	if configuration.Database.Migrate {
		migrations.RunAllMigrations(db)
		seed.SeedRolesAndPermissions(logger, db.Postgresql)
		seed.SeedPlans(logger, db.Postgresql)
		seed.SeedAssistants(logger, db.Postgresql)
		seed.SeedIndex(logger, db.Elastic)
		seed.SeedCreditPackages(logger, db.Postgresql)
		seed.SeedTranslatorPrompt(logger, db.Postgresql)
	}

	r := router.Setup(logger, validatorRef, db, &configuration.App)

	utility.LogAndPrint(logger, fmt.Sprintf("Server is starting at 127.0.0.1:%s", configuration.Server.Port))
	log.Fatal(r.Run(":" + configuration.Server.Port))
}
