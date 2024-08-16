package typesense

import (
	"context"

	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
)

func CreateCollection(client *typesense.Client, collectionName string, fields []api.Field) error {

	collectionSchema := api.CollectionSchema{
		Name:   collectionName,
		Fields: fields,
	}

	_, err := client.Collections().Create(context.Background(), &collectionSchema)
	if err != nil {
		return err
	}

	return nil
}

func InsertDocument(client *typesense.Client, collectionName string, document interface{}) error {
	_, err := client.Collection(collectionName).Documents().Create(context.Background(), document)
	if err != nil {
		return err
	}
	return nil
}
