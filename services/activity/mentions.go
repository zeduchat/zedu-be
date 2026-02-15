package activity

import (
	"errors"
	"net/http"
	"time"

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

	// get channels with > 0 mentions and their LastReadAt
	var userChannels models.UserChannels
	channelMap, err := userChannels.GetChannelsWithMentions(db.Postgresql, userID)
	if err != nil {
		logger.Error("failed to fetch user channels with mentions: %v", err)
		return nil, nil, http.StatusInternalServerError, err
	}

	// get dms with > 0 unread threads and their LastReadAt
	var dmChannels models.DmChannels
	dmMap, err := dmChannels.GetDMsWithUnread(db.Postgresql, userID)
	if err != nil {
		logger.Error("failed to fetch dm channels with unread: %v", err)
		return nil, nil, http.StatusInternalServerError, err
	}

	// combine values
	activeChannels := make(map[string]time.Time)
	for id, t := range channelMap {
		activeChannels[id] = t
	}
	for id, t := range dmMap {
		activeChannels[id] = t
	}

	// if no channels have new mentions, return empty
	if len(activeChannels) == 0 {
		return []activity.MentionActivityItem{}, &elastic.PaginationResponse{}, http.StatusOK, nil
	}

	// build query to match unseen mentions
	query := buildMentionsQuery(userID, orgID, activeChannels)

	var mentionsData []activity.MentionActivityItem

	/*
		perform search across both "threads" and "messages"
		indices to capture mentions in parent threads (channels, dms, group dms)
		and reply messages.
		this query is filtered by channel_id and created_at > last_read_at
	*/
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

func buildMentionsQuery(userID, orgID string, activeChannels map[string]time.Time) map[string]any {

	shouldClauses := []any{}

	for channelID, lastReadAt := range activeChannels {
		clause := map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{
						"term": map[string]any{
							"channels_id": channelID,
						},
					},
					map[string]any{
						"range": map[string]any{
							"created_at": map[string]any{
								"gt": lastReadAt,
							},
						},
					},
				},
			},
		}

		shouldClauses = append(shouldClauses, clause)
	}

	return map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{
						"term": map[string]any{
							"org_id": orgID,
						},
					},
					map[string]any{
						"term": map[string]any{
							"mentions.id.keyword": userID,
						},
					},
				},
				"must": []any{
					map[string]any{
						"bool": map[string]any{
							"should":               shouldClauses,
							"minimum_should_match": 1,
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
