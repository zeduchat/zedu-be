package elastic

import (
	"bytes"
	"encoding/json"
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

func DeleteDocument(client *elasticsearch.Client, indexName, docID string) error {
	res, err := client.Delete(indexName, docID, client.Delete.WithRefresh("true"))
	if err != nil {
		return fmt.Errorf("error deleting document: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}
	return nil
}

func DeleteByQuery(client *elasticsearch.Client, indexName string, query map[string]any) error {
	// Perform the delete by query request

	// Convert the query to JSON
	data, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("error marshalling delete script: %w", err)
	}

	res, err := client.DeleteByQuery(
		[]string{indexName},                    // Index name(s)
		bytes.NewReader(data),                  // Query to match documents
		client.DeleteByQuery.WithRefresh(true), // Refresh the index after deletion
	)
	if err != nil {
		return fmt.Errorf("error performing delete by query: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	return nil
}
