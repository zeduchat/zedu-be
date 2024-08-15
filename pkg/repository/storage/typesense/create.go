package typesense

import (
	"context"
	"fmt"
	"log"

	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
)

func CreateDocument(client *typesense.Client) {

	newDocument1 := struct {
		ID           string `json:"id"`
		CompanyName  string `json:"company_name"`
		NumEmployees int    `json:"num_employees"`
		Country      string `json:"country"`
	}{
		ID:           "482",
		CompanyName:  "MF Legion",
		NumEmployees: 232,
		Country:      "Barbados",
	}

	_, err := client.Collection("companies").Documents().Upsert(context.Background(), newDocument1)
	if err != nil {
		log.Printf("Document upsert error: %v", err)
	}

	searchParams := &api.SearchCollectionParams{
		Q:       pointer.String("*"),
		QueryBy: pointer.String("company_name"),
	}

	searchResult, err := client.Collection("companies").Documents().Search(context.Background(), searchParams)
	if err != nil {
		log.Fatalf("Error searching documents: %v", err)
	}

	for _, hit := range *searchResult.Hits {
		fmt.Printf("Document found: %+v\n", hit.Document)
	}

}
