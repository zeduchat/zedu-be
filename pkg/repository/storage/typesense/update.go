package typesense

import (
	"context"

	"github.com/typesense/typesense-go/v2/typesense"
)

func UpdateDocument(client *typesense.Client, collectionName string, document interface{}) error {
	_, err := client.Collection(collectionName).Documents().Upsert(context.Background(), document)
	if err != nil {
		return err
	}
	return nil
}
