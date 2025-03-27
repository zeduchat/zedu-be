package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type DmFilter struct {
	UserId    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	OrgId     string    `json:"org_id,omitempty"`
	AvatarUrl string    `json:"avatar_url,omitempty"`
	ChannelId string    `json:"channel_id"`
	Message   string    `json:"message"`
	TimeStamp time.Time `json:"timestamp,omitempty"`
}

type RawValues struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []DmFilter `json:"hits"`
	} `json:"hits"`
}

func mapToStruct(raw map[string]interface{}) (*RawValues, error) {
	m := &RawValues{}

	b, err := json.Marshal(raw)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if err := json.Unmarshal(b, m); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return m, nil
}

func FilterDms(db *storage.Database, userId, orgId string) (*RawValues, error) {
	chanIds, err := GetUserDmChannels(db.Postgresql, userId, orgId)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	query := queryElasticForDms(chanIds, userId, orgId)

	raw, err := elastic.PerformSearchWithMultipleIndices(db.Elastic, query)

	if err != nil {
		return nil, err
	}

	toStruct, err := mapToStruct(raw)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%+v", toStruct)
	return toStruct, nil
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
