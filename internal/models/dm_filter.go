package models

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

func mapToStruct(raw map[string]interface{}) (*elastic.ESResponse, error) {
	m := &elastic.ESResponse{}

	hitsArray, ok := raw["hits"].(map[string]interface{})["hits"].([]interface{})
	if !ok {
		return nil, errors.New("invalid response format: missing 'hits' array")
	}
	for _, hit := range hitsArray {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}
		source, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		b, err := json.Marshal(source)
		if err != nil {
			return nil, err
		}
		var dm elastic.DmFilter
		if err := json.Unmarshal(b, &dm); err != nil {
			return nil, err
		}
		// m.Hits.Hits = append(m.Hits.Hits, dm)
	}
	return m, nil
}

func FilterDms(db *storage.Database, userId, orgId string, c *gin.Context) ([]elastic.DmFilter, *elastic.PaginationResponse, error) {
	chanIds, err := GetUserDmChannels(db.Postgresql, userId, orgId)
	if err != nil {
		return nil, nil, err
	}

	query := queryElasticForDms(chanIds, userId, orgId)
	dmFilter := make([]elastic.DmFilter, 0)
	rs := &elastic.ESResponse{}
	raw, err := elastic.PerformSearchWithMultipleIndicesPagination(db.Elastic, query, rs, c)

	if err != nil {
		return nil, nil, err
	}
	hits := rs.Hits.Hits
	for _, hit := range hits {
		dmFilter = append(dmFilter, hit.Source)
	}

	return dmFilter, raw, nil
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

	b, _ := json.MarshalIndent(query, "\n", " ")
	fmt.Println(string(b))
	return query
}

func GetUserDmChannels(db *gorm.DB, userId, orgId string) ([]string, error) {
	org := Organisation{}
	dm := DmChannels{}
	var channs []string
	if exists := postgresql.CheckExists(db, &org, "id = ?", orgId); exists == false {
		return nil, errors.New("Organisation does not exist")
	}
	if err := db.Model(&dm).
		Select("channel_id").
		Where("(user_id = ? AND org_id = ?) OR (participant_id = ? AND org_id = ?)", userId, orgId, userId, orgId).
		Scan(&channs).Error; err != nil {
		return nil, errors.New("user does not belong in this channel")
	}
	if channs == nil {
		return nil, errors.New("user does not belong in this channel")
	}
	return channs, nil
}
