package typesense

import (
	"context"
	"fmt"

	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
)

func SearchDocuments(client *typesense.Client, collectionName, query, searchField string) ([]map[string]interface{}, error) {
	searchParams := &api.SearchCollectionParams{
		Q:       pointer.String(query),
		QueryBy: pointer.String(searchField),
	}

	searchResult, err := client.Collection(collectionName).Documents().Search(context.Background(), searchParams)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, hit := range *searchResult.Hits {
		if hit.Document != nil {
			results = append(results, *hit.Document)
		}
	}

	return results, nil
}

// Testing returns
func ListCollections(client *typesense.Client) error {
	collections, err := client.Collections().Retrieve(context.Background())
	if err != nil {
		return err
	}

	for _, collection := range collections {
		fmt.Printf("Collection Name: %s\n", collection.Name)
	}

	return nil
}
