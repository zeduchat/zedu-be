package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type SearchQueryFiltersKeywords struct {
	Message string
	In      string
	From    string
	Has     string
	Before  time.Time
	On      time.Time
	After   time.Time
	Exact   string
	SortBy  string
}

func NewSearchQueryFilterKeywords() *SearchQueryFiltersKeywords {
	return &SearchQueryFiltersKeywords{}
}
func (s *SearchQueryFiltersKeywords) ProcessQueryString(queryArr [][]string) {

	for _, match := range queryArr {
		key, value := match[1], match[2]

		switch key {
		case "from":
			s.From = strings.TrimPrefix(value, "@")
		case "in":
			s.In = strings.TrimPrefix(value, "#")
		case "has":
			s.Has = strings.TrimPrefix(value, ":")
		case "before":
			t, _ := time.Parse(time.RFC3339, strings.TrimPrefix(value, ":"))
			s.Before = t
		case "on":
			t, _ := time.Parse(time.RFC3339, strings.TrimPrefix(value, ":"))
			s.On = t
		case "after":
			t, _ := time.Parse(time.RFC3339, strings.TrimPrefix(value, ":"))
			s.After = t
		case "exact":
			s.Exact = strings.TrimPrefix(value, ":")
		}
	}
}

func SearchQuery(db *storage.Database, c *gin.Context, searchQuery *SearchQueryFiltersKeywords, userId string) ([]utility.SearchQueryResult, error) {
	var qResults []utility.SearchQueryResult

	// Build the search query
	query, err := buildSearchQuery(db.Postgresql, searchQuery, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	// Perform search on ElasticSearch
	res, err := elastic.PerformSearchWithMultipleIndices(db.Elastic, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to perform search: %w", err)
	}

	// Extract hits data
	hitsData, ok := res["hits"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected type for hits: %T", res["hits"])
	}

	hitsArray, ok := hitsData["hits"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected type for hits.hits: %T", hitsData["hits"])
	}

	// Process each hit
	for _, hitItem := range hitsArray {
		hit, ok := hitItem.(map[string]interface{})
		if !ok {
			continue
		}

		index, ok := hit["_index"].(string)
		if !ok {
			continue
		}

		source, ok := hit["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		// Process hit and append to results
		qResults = append(qResults, utility.ProcessMessageHit(index, source))
	}
	return qResults, nil
}

func buildSearchQuery(db *gorm.DB, opts *SearchQueryFiltersKeywords, userId string) (map[string]interface{}, error) {
	query := initializeQuery()
	boolQuery := query["query"].(map[string]interface{})["bool"].(map[string]interface{})

	addFullTextSearch(boolQuery, opts)

	addSenderFilter(boolQuery, opts)

	addChannelFilter(boolQuery, opts)

	// // 4. Date filters: on, before, and after
	// addDateFilters(boolQuery, opts)

	// // 5. Additional content filter ("has")
	// addContentFilter(boolQuery, opts)

	// // 6. Sorting (recency or relevance)
	// addSorting(query, opts)

	return query, nil
}

func initializeQuery() map[string]interface{} {
	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   []interface{}{}, // for text matching
				"filter": []interface{}{}, // for term/range filters
			},
		},
	}
}
func addFullTextSearch(boolQuery map[string]interface{}, opts *SearchQueryFiltersKeywords) {
	mustClauses := boolQuery["must"].([]interface{})

	if opts.Exact != "" {
		// Ensure we only match exact messages from the logged-in user
		mustClauses = append(mustClauses, map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"match_phrase": map[string]interface{}{
							"message": opts.Exact,
						},
					},
				},
			},
		})
	} else if opts.Message != "" {
		// General search across multiple fields (does not filter by user_id)
		mustClauses = append(mustClauses, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": opts.Message,
				"type":  "best_fields",
				"fields": []string{
					"message",
					"messages.message",
					"messages.content",
					"full_name",
					"username",
					"user_name",
					"channel_name",
				},
			},
		})
	}

	boolQuery["must"] = mustClauses
}

func addSenderFilter(query map[string]interface{}, opts *SearchQueryFiltersKeywords) {
	var boolQuery map[string]interface{}

	if rawQuery, exists := query["query"]; exists {
		querySection, ok := rawQuery.(map[string]interface{})
		if !ok {
			querySection = make(map[string]interface{})
			query["query"] = querySection
		}
		if rawBool, exists := querySection["bool"]; exists {
			if b, ok := rawBool.(map[string]interface{}); ok {
				boolQuery = b
			} else {
				boolQuery = make(map[string]interface{})
				querySection["bool"] = boolQuery
			}
		} else {
			boolQuery = make(map[string]interface{})
			querySection["bool"] = boolQuery
		}
	} else {
		boolQuery = query
	}

	var shouldClauses []interface{}
	if rawShould, exists := boolQuery["should"]; exists {
		if arr, ok := rawShould.([]interface{}); ok {
			shouldClauses = arr
		} else {
			shouldClauses = []interface{}{}
		}
	} else {
		shouldClauses = []interface{}{}
	}

	if opts.From != "" {
		fromVal := strings.Trim(opts.From, "\"")
		senderFilters := []interface{}{
			map[string]interface{}{"match": map[string]interface{}{"user_name": fromVal}},
			map[string]interface{}{"match": map[string]interface{}{"username": fromVal}},
			map[string]interface{}{"match": map[string]interface{}{"full_name": fromVal}},
			map[string]interface{}{"match": map[string]interface{}{"messages.user_name": fromVal}},
			map[string]interface{}{"match": map[string]interface{}{"messages.username": fromVal}},
			map[string]interface{}{"match": map[string]interface{}{"messages.full_name": fromVal}},
		}

		shouldClauses = append(shouldClauses, senderFilters...)
		boolQuery["should"] = shouldClauses
		boolQuery["minimum_should_match"] = 1
	}

}

func addChannelFilter(boolQuery map[string]interface{}, opts *SearchQueryFiltersKeywords) {
	if opts.In != "" {
		channelName := strings.Trim(opts.In, "\"")

		// Ensure "should" exists
		if _, exists := boolQuery["should"]; !exists {
			boolQuery["should"] = []interface{}{}
		}

		shouldClauses := boolQuery["should"].([]interface{})

		shouldClauses = append(shouldClauses, map[string]interface{}{
			"match": map[string]interface{}{
				"channel_name": map[string]interface{}{
					"query":     channelName,
					"fuzziness": "AUTO", // Allows minor spelling variations
				},
			},
		})

		// Assign back to boolQuery
		boolQuery["should"] = shouldClauses
		boolQuery["minimum_should_match"] = 2
	}

	// Print the generated Elasticsearch query
	b, _ := json.MarshalIndent(map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
	}, "", "  ")
	fmt.Println(string(b))
}

func addDateFilters(boolQuery map[string]interface{}, opts *SearchQueryFiltersKeywords) {
	filterClauses := boolQuery["filter"].([]interface{})

	// "On" filter: match documents on a specific day.
	if !opts.On.IsZero() {
		startOfDay := opts.On.Truncate(24 * time.Hour)
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)
		onFilter := map[string]interface{}{
			"range": map[string]interface{}{
				"created_at": map[string]interface{}{
					"gte": startOfDay.Format(time.RFC3339),
					"lte": endOfDay.Format(time.RFC3339),
				},
			},
		}
		onFilter2 := map[string]interface{}{
			"range": map[string]interface{}{
				"messages.created_at": map[string]interface{}{
					"gte": startOfDay.Format(time.RFC3339),
					"lte": endOfDay.Format(time.RFC3339),
				},
			},
		}
		filterClauses = append(filterClauses, onFilter, onFilter2)
	}

	// "Before" filter: match documents before the given time.
	if !opts.Before.IsZero() {
		beforeFilter := map[string]interface{}{
			"range": map[string]interface{}{
				"created_at": map[string]interface{}{
					"lt": opts.Before.Format(time.RFC3339),
				},
			},
		}
		beforeFilter2 := map[string]interface{}{
			"range": map[string]interface{}{
				"messages.created_at": map[string]interface{}{
					"lt": opts.Before.Format(time.RFC3339),
				},
			},
		}
		filterClauses = append(filterClauses, beforeFilter, beforeFilter2)
	}

	// "After" filter: match documents after the given time.
	if !opts.After.IsZero() {
		afterFilter := map[string]interface{}{
			"range": map[string]interface{}{
				"created_at": map[string]interface{}{
					"gt": opts.After.Format(time.RFC3339),
				},
			},
		}
		afterFilter2 := map[string]interface{}{
			"range": map[string]interface{}{
				"messages.created_at": map[string]interface{}{
					"gt": opts.After.Format(time.RFC3339),
				},
			},
		}
		filterClauses = append(filterClauses, afterFilter, afterFilter2)
	}

	boolQuery["filter"] = filterClauses
}

func addContentFilter(boolQuery map[string]interface{}, opts *SearchQueryFiltersKeywords) {
	mustClauses := boolQuery["must"].([]interface{})
	if opts.Has != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  opts.Has,
				"type":   "best_fields",
				"fields": []string{"message", "messages.message"},
			},
		})
	}
	boolQuery["must"] = mustClauses
}

func addSorting(query map[string]interface{}, opts *SearchQueryFiltersKeywords) (map[string]interface{}, error) {
	var sorting []interface{}
	if opts.SortBy == "recency" {
		sorting = []interface{}{
			map[string]interface{}{
				"created_at": map[string]interface{}{
					"order": "desc",
				},
			},
		}
	} else {
		// Default to relevance (_score descending)
		sorting = []interface{}{
			map[string]interface{}{
				"_score": map[string]interface{}{
					"order": "desc",
				},
			},
		}
	}
	query["sort"] = sorting
	return query, nil
}

// func GetUserChannelsByName(db *gorm.DB, userId string, channelName string) (*Channels, error) {
// 	var channel Channels
// 	err := db.Joins("JOIN user_channels ON user_channels.channels_id = channels.id").
// 		Where("user_channels.user_id = ? AND channels.name ILIKE ?", userId, "%"+channelName+"%").
// 		First(&channel).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	fmt.Printf("%+v", channel)
// 	return &channel, nil
// }

// // using ts_vector
// func SearchUserChannelByFullText(db *gorm.DB, userId string, searchTerm string) (*Channels, error) {
// 	var channel Channels
// 	err := db.
// 		Joins("JOIN user_channels ON user_channels.channels_id = channels.id").
// 		Where("user_channels.user_id = ? AND to_tsvector('english', channels.name) @@ plainto_tsquery(?)", userId, searchTerm).
// 		First(&channel).Error
// 	if err != nil {
// 		return nil, err
// 	}

// 	fmt.Printf("%+v", channel)
// 	return &channel, nil
// }

// // implement get channel by ID
// func GetChannelById(db *gorm.DB, channel_id string) (*Channels, error) {
// 	var channel Channels
// 	err := db.Where("id = ?", channel_id).First(&channel).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &channel, nil
// }
