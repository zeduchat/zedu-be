package seed

import (
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/utility"
)

var Indexs = map[string]any{
	"threads":  models.Thread_mapping,
	"messages": models.Message_mapping,
}

func SeedIndex(logger *utility.Logger, es *elasticsearch.Client) {

	for index, mapping := range Indexs {

		logger.Info("Creating Index for: " + index)

		err := elastic.CreateIndex(es, index, mapping, logger)
		if err != nil {
			logger.Error(fmt.Sprintf("An error occurred while indexing: %s, error: %v", index, err))
		} else {
			logger.Info("Done indexing: " + index)
		}
	}
}
