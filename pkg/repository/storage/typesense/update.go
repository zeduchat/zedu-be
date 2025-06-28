package typesense

import (
	"context"

	"github.com/typesense/typesense-go/v2/typesense"
)

func UpdateDocument(client *typesense.Client, collectionName string, document any) error {

	if client == nil {
		return nil
	}
	_, err := client.Collection(collectionName).Documents().Upsert(context.Background(), document)
	if err != nil {
		return err
	}
	return nil
}
