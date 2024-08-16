package typesense

import (
	"context"
	"time"

	"github.com/typesense/typesense-go/v2/typesense"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func ConnectToTypeSense(logger *utility.Logger, config config.TypeSense) *typesense.Client {
	utility.LogAndPrint(logger, "connecting to typesense server")

	client := typesense.NewClient(
		typesense.WithServer(config.TypeSense_API_URL),
		typesense.WithAPIKey(config.TypeSense_API_KEY),
	)

	healthy, err := client.Health(context.Background(), 10*time.Second)
	if err != nil {
		utility.LogAndPrint(logger, "failed to connect to typesense server: "+err.Error())
		return nil
	}
	if !healthy {
		utility.LogAndPrint(logger, "typesense server is not healthy")
		return nil
	}

	utility.LogAndPrint(logger, "connected to typesense server")

	storage.DB.TypeSense = client

	return client
}
