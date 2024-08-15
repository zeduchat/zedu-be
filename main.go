package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/cronjobs"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/internal/models/migrations"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/pkg/router"
	"github.com/hngprojects/telex_be/utility"
)

func main() {
	logger := utility.NewLogger() //Warning !!!!! Do not recreate this action anywhere on the app

	configuration := config.Setup(logger, "./app")

	postgresql.ConnectToDatabase(logger, configuration.Database)
	redis.ConnectToRedis(logger, configuration.Redis)
	minio.ConnectToMinio(logger, configuration.Minio)
	centrifuge.NewCentrifugoService(logger, configuration.Centrifuge)
	centrifuge.NewCentrifugod(logger, configuration.Centrifuge)

	d := models.FeedWebHookRequest{
		EventName: "testtt",
		UserName:  "cyberguru",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		ActionType: "action_test",
		ChannelID: "0191524f-6adc-7c38-a9fb-9c6d98859fe6",
	}

	err := centrifuge.BroadcastChannel(logger, "0191524f-6adc-7c38-a9fb-9c6d98859fe6", d)

	if err != nil {
		fmt.Println(err)
	}

	validatorRef := validator.New()

	db := storage.Connection()

	cronjobs.StartCronJob(request.ExternalRequest{Logger: logger}, *storage.DB, "send-notifications")

	if configuration.Database.Migrate {
		migrations.RunAllMigrations(db)
	}

	r := router.Setup(logger, validatorRef, db, &configuration.App)

	utility.LogAndPrint(logger, fmt.Sprintf("Server is starting at 127.0.0.1:%s", configuration.Server.Port))
	log.Fatal(r.Run(":" + configuration.Server.Port))
}
