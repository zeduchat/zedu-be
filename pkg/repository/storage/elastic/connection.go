package elastic

import (
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func ConnectToElastic(logger *utility.Logger, EConfig config.ElasticDb) *elasticsearch.Client {

	utility.LogAndPrint(logger, "connecting to Elastic DB...")

	cfg := elasticsearch.Config{
		APIKey: EConfig.ElasticApiKey,
		Addresses: []string{
			EConfig.ElasticEndpoint,
		},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Errorf("error creating Elasticsearch client: %w", err))
		return nil
	}

	utility.LogAndPrint(logger, "connected to Elastic DB  ✅ ")
	fmt.Println("connected to Elastic DB  ✅ ")

	info, err := client.Info()

	if err != nil {
		utility.LogAndPrint(logger, fmt.Errorf("error fetching Elasticsearch client info: %w", err))
		return nil
	}

	utility.LogAndPrint(logger, fmt.Sprintf("elastic client info: %v", info))

	storage.DB.Elastic = client

	return client
}
