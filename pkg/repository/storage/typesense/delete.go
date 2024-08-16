package typesense

import (
	"context"

	"github.com/typesense/typesense-go/v2/typesense"
)

func DeleteCollection(client *typesense.Client, collectionName string) error {

	_, err := client.Collection(collectionName).Delete(context.Background())
	if err != nil {
		return err
	}

	return nil
}
