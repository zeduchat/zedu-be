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

func mapToStruct(raw map[string]interface{}) (*elastic.RawValues, error) {
	m := &elastic.RawValues{}

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
		m.Hits.Hits = append(m.Hits.Hits, dm)
	}
	return m, nil
}

func FilterDms(db *storage.Database, userId, orgId string, c *gin.Context) ([]elastic.DmFilter, *elastic.PaginationResponse, error) {
	chanIds, err := GetUserDmChannels(db.Postgresql, userId, orgId)
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	query := queryElasticForDms(chanIds, userId, orgId)

	rs := &elastic.RawValues{}
	raw, err := elastic.PerformSearchWithMultipleIndicesPagination(db.Elastic, query, rs, c)

	if err != nil {
		return nil, nil, err
	}

	hits := rs.Hits.Hits

	fmt.Printf("%+v", raw)
	fmt.Printf("%+v", hits)
	return hits, raw, nil
}

func queryElasticForDms(chanIds []string, userId, orgId string) map[string]interface{} {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"terms": map[string]interface{}{
							"channels_id.keyword": chanIds,
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"org_id.keyword": orgId,
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
		"aggs": map[string]interface{}{
			"unique_senders": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "user_id.keyword",
					"size":  100,
					"order": map[string]interface{}{
						"max_created_at": "desc",
					},
				},
				"aggs": map[string]interface{}{
					"max_created_at": map[string]interface{}{
						"max": map[string]interface{}{
							"field": "created_at",
						},
					},
					"latest_message": map[string]interface{}{
						"top_hits": map[string]interface{}{
							"sort": []interface{}{
								map[string]interface{}{
									"created_at": map[string]interface{}{
										"order": "desc",
									},
								},
							},
							"size": 1,
						},
					},
				},
			},
		},
		"size": 0,
	}
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
		Where("user_id = ? AND org_id = ?", userId, orgId).
		Scan(&channs).Error; err != nil {
		return nil, fmt.Errorf("%s", "User does not exist in this organization")
	}
	fmt.Println(channs)
	return channs, nil
}
