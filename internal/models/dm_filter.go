package models

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
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
func queryElasticForDms(chanIds []string, userId, orgId string) map[string]interface{} {
	query := map[string]interface{}{
		"size": 100,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"terms": map[string]interface{}{
							"channels_id.keyword": chanIds,
						},
					},
				},
				"must_not": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"user_id.keyword": userId,
						},
					},
				},
			},
		},
		"sort": []interface{}{
			map[string]interface{}{
				"created_at": map[string]interface{}{
					"order": "desc",
				},
			},
		},
		"collapse": map[string]interface{}{
			"field": "user_id.keyword",
		},
		"aggs": map[string]interface{}{
			"unique_users": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "user_id.keyword",
					"size":  100,
				},
				"aggs": map[string]interface{}{
					"latest_message": map[string]interface{}{
						"top_hits": map[string]interface{}{
							"size": 1,
							"sort": []interface{}{
								map[string]interface{}{
									"created_at": map[string]interface{}{
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
