package elastic

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/hngprojects/telex_be/utility"
)

func CreateIndex(client *elasticsearch.Client, indexName string, mapping interface{}, logger *utility.Logger) error {

	// Check if index exists

	res, err := client.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {

		// Create index
		body, _ := json.Marshal(mapping)
		_, err = client.Indices.Create(indexName, client.Indices.Create.WithBody(bytes.NewReader(body)))
		if err != nil {
			logger.Error("an error occurred while creating index: %v", err)
			return err
		}
	}

	logger.Info(fmt.Sprintf("Index %s exist, resulted in %v", indexName, res.StatusCode))

	return nil
}

func AddDocument(client *elasticsearch.Client, indexName, docID string, doc interface{}, logger *utility.Logger) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("error marshalling document: %w", err)
	}

	res, err := client.Index(
		indexName,
		bytes.NewReader(data),
		client.Index.WithDocumentID(docID),
		client.Index.WithRefresh("true"),
	)

	logger.Info(res.String())

	if err != nil {
		return fmt.Errorf("error indexing document: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		logger.Error(fmt.Errorf("error response from Elasticsearch: %s", res.String()))
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	logger.Info(fmt.Sprintf("Document %s added successfully\n", docID))

	return nil
}
