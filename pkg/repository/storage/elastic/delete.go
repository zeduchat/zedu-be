package elastic

import (
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/hngprojects/telex_be/utility"
)

func DeleteIndex(client *elasticsearch.Client, indexName string, logger *utility.Logger) error {
	res, err := client.Indices.Delete([]string{indexName})

	if err != nil {
		logger.Error(fmt.Errorf("error response from Elasticsearch: %s", res.String()))
		return fmt.Errorf("error deleting index: %w", err)
	}

	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	logger.Info("Index %s deleted successfully\n", indexName)
	
	return nil
}
