package activity

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	activity "github.com/hngprojects/telex_be/internal/models/activity"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func GetUserMentionsActivity(userID, orgID string, db *storage.Database, c *gin.Context, logger *utility.Logger) ([]activity.MentionActivityItem, *elastic.PaginationResponse, int, error) {
	var org models.Organisation

	exists := postgresql.CheckExists(db.Postgresql, &org, "id = ?", orgID)
	if !exists {
		return nil, nil, http.StatusNotFound, errors.New("organisation not found")
	}

	query := buildMentionsQuery(userID, orgID)

	var mentionsData []activity.MentionActivityItem

	paginationResp, err := elastic.PerformSearchWithMultipleIndicesPagination(
		db.Elastic,
		query,
		activity.MentionActivityItem{},
		&mentionsData,
		c,
	)

	if err != nil {
		logger.Error("failed to fetch mentions from elasticsearch: %v", err)
		return []activity.MentionActivityItem{}, &elastic.PaginationResponse{}, http.StatusOK, nil
	}

	return mentionsData, paginationResp, http.StatusOK, nil
}

func buildMentionsQuery(userID, orgID string) map[string]any {
	return map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{
						"term": map[string]any{
							"org_id.keyword": orgID,
						},
					},
					map[string]any{
						"term": map[string]any{
							"mentions.id.keyword": userID,
						},
					},
				},
			},
		},
		"sort": []any{
			map[string]any{
				"created_at": map[string]any{
					"order": "desc",
				},
			},
		},
	}
}
