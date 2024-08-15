package typesense

import (
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

	utility.LogAndPrint(logger, "connected to typesense server")

	storage.DB.TypeSense = client

	return client
}
