package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Channel struct {
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
}

type MessageQuery struct {
	MessageID string `json:"message_id"`
	Message   string `json:"message"`
}

type UserQuery struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	AvatarURL string `json:"avatar_url"`
}

type SearchQueryResult struct {
	User     UserQuery      `json:"user"`
	Channel  []Channel      `json:"channel"`
	Messages []MessageQuery `json:"messages"`
}

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

func SearchQuery(db *storage.Database, c *gin.Context, searchQuery *SearchQueryFiltersKeywords, userId string) (*SearchQueryResult, error) {
	var qResult SearchQueryResult
	query, err := buildSearchQuery(db.Postgresql, searchQuery, userId)
	// pagination := elastic.GetPagination(c)
	if err != nil {
		return nil, err
	}
	res, err := elastic.PerformSearchWithMultipleIndices(db.Elastic, query, &qResult)
	if err != nil {
		return nil, err
	}

	fmt.Println(res, "AAAA")
	return nil, nil
}
func buildSearchQuery(db *gorm.DB, opts *SearchQueryFiltersKeywords, userId string) (map[string]interface{}, error) {
	query := initializeQuery()
	boolQuery := query["query"].(map[string]interface{})["bool"].(map[string]interface{})

	addFullTextSearch(boolQuery, opts)
	addSenderFilter(boolQuery, opts)
	if err := addChannelFilter(db, boolQuery, opts, userId); err != nil {
		return nil, err
	}
	addDateFilters(boolQuery, opts)
	addContentFilter(boolQuery, opts)
	addSorting(query, opts)

	return query, nil
}

func initializeQuery() map[string]interface{} {
	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   []interface{}{},
				"filter": []interface{}{},
			},
		},
	}
}

func addFullTextSearch(boolQuery map[string]interface{}, opts *SearchQueryFiltersKeywords) {
	if opts.Message != "" {
		mustClauses := boolQuery["must"].([]interface{})
		searchQuery := opts.Message
		searchType := "best_fields"
		if opts.Exact != "" {
			searchQuery = opts.Exact
			searchType = "phrase"
		}
		mustClauses = append(mustClauses, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  searchQuery,
				"type":   searchType,
				"fields": []string{"message", "messages.message", "messages.content", "threads.full_name", "threads.username", "messages.full_name", "messages.username"},
			},
		})
	}
}

func addSenderFilter(boolQuery map[string]interface{}, opts *SearchQueryFiltersKeywords) {
	if opts.From != "" {
		filterClauses := boolQuery["filter"].([]interface{})
		filterClauses = append(filterClauses, map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{"match": map[string]interface{}{"username": opts.From}},
					map[string]interface{}{"match": map[string]interface{}{"full_name": opts.From}},
					map[string]interface{}{"match": map[string]interface{}{"messages.username": opts.From}},
					map[string]interface{}{"match": map[string]interface{}{"messages.full_name": opts.From}},
				},
				"minimum_should_match": 1,
			},
		})
		boolQuery["filter"] = filterClauses
	}
}

func addChannelFilter(db *gorm.DB, boolQuery map[string]interface{}, opts *SearchQueryFiltersKeywords, userId string) error {
	if opts.In != "" {
		channelID, err := GetChannelByName(db, opts.In, userId)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New("channel not found")
			}
			return err
		}
		filterClauses := boolQuery["filter"].([]interface{})
		filterClauses = append(filterClauses, map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{"term": map[string]interface{}{"channels_id.keyword": channelID}},
					map[string]interface{}{"term": map[string]interface{}{"messages.channels_id.keyword": channelID}},
					map[string]interface{}{"term": map[string]interface{}{"channel_name.keyword": channelID}},
				},
				"minimum_should_match": 1,
			},
		})
		boolQuery["filter"] = filterClauses
	}
	return nil
}

func addDateFilters(boolQuery map[string]interface{}, opts *SearchQueryFiltersKeywords) {
	var dateFilters []interface{}
	if !opts.On.IsZero() {
		startOfDay := opts.On.Truncate(24 * time.Hour)
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)
		dateFilters = append(dateFilters, dateRangeFilter("created_at", startOfDay, endOfDay))
		dateFilters = append(dateFilters, dateRangeFilter("messages.created_at", startOfDay, endOfDay))
	}
	if len(dateFilters) > 0 {
		boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{"bool": map[string]interface{}{"should": dateFilters, "minimum_should_match": 1}})
	}
}

func dateRangeFilter(field string, start, end time.Time) map[string]interface{} {
	return map[string]interface{}{"range": map[string]interface{}{field: map[string]interface{}{"gte": start.Format(time.RFC3339), "lte": end.Format(time.RFC3339)}}}
}

func addContentFilter(boolQuery map[string]interface{}, opts *SearchQueryFiltersKeywords) {
	if opts.Has != "" {
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{"match": map[string]interface{}{"message": opts.Has}},
					map[string]interface{}{"match": map[string]interface{}{"messages.message": opts.Has}},
				},
				"minimum_should_match": 1,
			},
		})
	}
}

func addSorting(query map[string]interface{}, opts *SearchQueryFiltersKeywords) (map[string]interface{}, error) {
	if opts.SortBy == "recency" {
		query["sort"] = []interface{}{map[string]interface{}{"created_at": map[string]interface{}{"order": "desc"}}}
	}

	// For debugging: print the final query
	jsonQuery, _ := json.MarshalIndent(query, "", "  ")
	fmt.Println(string(jsonQuery))

	return query, nil
}

func GetChannelByName(db *gorm.DB, channel string, userId string) (*Channel, error) {
	var ch Channel
	err := db.First(&ch, "channel_name = ? AND user_id = ?", channel, userId)
	postgresql.SelectOneFromDb(db, &ch, "channel_name = ? AND user_id = ?", channel, userId)
	if err.Error != nil {
		return nil, err.Error
	}
	return &ch, nil
}
