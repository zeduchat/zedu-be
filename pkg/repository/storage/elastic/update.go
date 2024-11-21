package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

func UpdateDocument(client *elasticsearch.Client, indexName, docID string, updates map[string]interface{}) error {
	data, err := json.Marshal(map[string]interface{}{"doc": updates})
	if err != nil {
		return fmt.Errorf("error marshalling update: %w", err)
	}

	res, err := client.Update(indexName, docID, bytes.NewReader(data), client.Update.WithRefresh("true"))
	if err != nil {
		return fmt.Errorf("error updating document: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}
	return nil
}

func UpdateDocWithScript(client *elasticsearch.Client, indexName, docID string, req map[string]interface{}) error {

	// Convert payload req to JSON
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return fmt.Errorf("Error encoding update payload: %s", err)
	}

	res, err := client.Update(
		indexName, docID,
		&buf, client.Update.WithRefresh("true"),
		client.Update.WithContext(context.Background()))

	if err != nil {
		return fmt.Errorf("error updating document: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	return nil
}
