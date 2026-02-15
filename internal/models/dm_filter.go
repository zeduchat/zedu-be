package models

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type DmFilter struct {
	UserId     string    `json:"user_id"`
	UserName   string    `json:"user_name,omitempty"`
	UsersName  string    `json:"username,omitempty"`
	OrgId      string    `json:"org_id,omitempty"`
	AvatarUrl  string    `json:"avatar_url"`
	ChannelsId string    `json:"channels_id,omitempty"`
	ChannelId  string    `json:"channel_id,omitempty"`
	Message    string    `json:"message"`
	Content    string    `json:"content,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

func FilterDms(db *storage.Database, userId, orgId string, c *gin.Context) ([]DmFilter, *elastic.PaginationResponse, int, error) {
	dmFilter := make([]DmFilter, 0)
	var infer DmFilter
	chanIds, err := GetUserDmChannels(db.Postgresql, userId, orgId)
	if err != nil {
		if err.Error() == "Organisation does not exist" {
			return nil, nil, http.StatusNotFound, err
		} else if strings.Contains(err.Error(), "user does not belong in this channel") {
			return nil, nil, http.StatusBadRequest, err
		}
		return nil, nil, http.StatusInternalServerError, err
	} else if len(chanIds) == 0 {
		return dmFilter, nil, http.StatusOK, nil // user haven't chatted yet...
	}

	query := queryElasticForDms(chanIds, userId, orgId)
	raw, err := elastic.PerformSearchWithMultipleIndicesPagination(db.Elastic, query, infer, &dmFilter, c)

	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}

	return dmFilter, raw, http.StatusOK, nil
}

func queryElasticForDms(chanIds []string, userId, orgId string) map[string]any {
	query := map[string]any{
		"size": 100,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{
						"terms": map[string]any{
							"channels_id": chanIds,
						},
					},
				},
				"must_not": []any{
					map[string]any{
						"term": map[string]any{
							"user_id": userId,
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
		"collapse": map[string]any{
			"field": "user_id",
		},
		"aggs": map[string]any{
			"unique_users": map[string]any{
				"terms": map[string]any{
					"field": "user_id",
					"size":  100,
				},
				"aggs": map[string]any{
					"latest_message": map[string]any{
						"top_hits": map[string]any{
							"size": 1,
							"sort": []any{
								map[string]any{
									"created_at": map[string]any{
										"order": "desc",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return query
}

func GetUserDmChannels(db *gorm.DB, userId, orgId string) ([]string, error) {
	org := Organisation{}
	dm := DmChannels{}
	var channs []string
	if exists := postgresql.CheckExists(db, &org, "id = ?", orgId); !exists {
		return nil, errors.New("Organisation does not exist")
	}
	if err := db.Model(&dm).
		Select("channel_id").
		Where("(user_id = ? AND org_id = ?) OR (participant_id = ? AND org_id = ?)", userId, orgId, userId, orgId).
		Scan(&channs).Error; err != nil {
		return nil, errors.New("user does not belong in this channel")
	}
	return channs, nil
}
