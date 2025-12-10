package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
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

type ReactionData struct {
	MessageID  string `gorm:"column:message_id"`
	ReactionID string `gorm:"column:reaction_id"`
	Reaction   string `gorm:"column:reaction"`
	UserID     string `gorm:"column:user_id"`
	UserName   string `gorm:"column:user_name"`
	AvatarURL  string `gorm:"column:avatar_url"`
}

type ThreadData struct {
	ThreadID     string  `gorm:"column:id"`
	MessageCount *int64  `gorm:"column:message_count"`
	LastReply    *string `gorm:"column:last_reply"`
}

type ReplyUserData struct {
	ThreadID  string `gorm:"column:thread_id"`
	UserID    string `gorm:"column:user_id"`
	UserName  string `gorm:"column:username"`
	AvatarURL string `gorm:"column:avatar_url"`
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

func SearchQuery(db *storage.Database, c *gin.Context, searchQuery *SearchQueryFiltersKeywords, userId string, orgId string) ([]utility.SearchQueryResult, error) {

	var qResults []utility.SearchQueryResult

	query, err := buildSearchQuery(db.Postgresql, searchQuery, userId, orgId)
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	res, err := elastic.PerformSearchWithMultipleIndices(db.Elastic, query)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	hitsData, ok := res["hits"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type for hits: %T", res["hits"])
	}

	hitsArray, ok := hitsData["hits"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type for hits.hits: %T", hitsData["hits"])
	}

	for _, hitItem := range hitsArray {
		hit, ok := hitItem.(map[string]any)
		if !ok {
			continue
		}

		index, ok := hit["_index"].(string)
		if !ok {
			continue
		}

		source, ok := hit["_source"].(map[string]any)
		if !ok {
			continue
		}

		// Process hit and append to results
		qResults = append(qResults, utility.ProcessMessageHit(index, source))
	}
	return qResults, nil
}

func buildSearchQuery(db *gorm.DB, opts *SearchQueryFiltersKeywords, userId string, orgId string) (map[string]any, error) {
	query := initializeQuery()
	boolQuery := query["query"].(map[string]any)["bool"].(map[string]any)
	channels, err := GetChannelsByOrgIDs(db, orgId, userId)
	if err != nil {
		return nil, err
	}

	addFullTextSearch(boolQuery, opts)

	addSenderFilter(boolQuery, opts)

	addChannelFilter(boolQuery, opts)

	addDateFilters(boolQuery, opts)

	addContentFilter(boolQuery, opts)
	addOrgOrChannelFilter(boolQuery, orgId, channels)

	addSorting(query, opts)

	return query, nil
}

func initializeQuery() map[string]any {
	return map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must":   []any{},
				"filter": []any{},
			},
		},
	}
}
func addOrgOrChannelFilter(boolQuery map[string]any, orgID string, channelIDs []string) {
	shouldClauses := []any{
		map[string]any{"term": map[string]any{"org_id.keyword": orgID}},
	}

	if len(channelIDs) > 0 {
		shouldClauses = append(shouldClauses, map[string]any{
			"terms": map[string]any{"channels_id.keyword": channelIDs},
		})
	}

	orgOrChannelFilter := map[string]any{
		"bool": map[string]any{
			"should":               shouldClauses,
			"minimum_should_match": 1,
		},
	}

	if existingFilters, ok := boolQuery["must"].([]any); ok {
		boolQuery["must"] = append(existingFilters, orgOrChannelFilter)
	} else {
		boolQuery["must"] = []any{orgOrChannelFilter}
	}
}

func addFullTextSearch(boolQuery map[string]any, opts *SearchQueryFiltersKeywords) {
	mustClauses := boolQuery["must"].([]any)

	if opts.Exact != "" {
		mustClauses = append(mustClauses, map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{
						"match_phrase": map[string]any{
							"message": opts.Exact,
						},
					},
				},
			},
		})
	} else if opts.Message != "" {
		mustClauses = append(mustClauses, map[string]any{
			"multi_match": map[string]any{
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

func addSenderFilter(query map[string]any, opts *SearchQueryFiltersKeywords) {
	var boolQuery map[string]any

	if rawQuery, exists := query["query"]; exists {
		querySection, ok := rawQuery.(map[string]any)
		if !ok {
			querySection = make(map[string]any)
			query["query"] = querySection
		}
		if rawBool, exists := querySection["bool"]; exists {
			if b, ok := rawBool.(map[string]any); ok {
				boolQuery = b
			} else {
				boolQuery = make(map[string]any)
				querySection["bool"] = boolQuery
			}
		} else {
			boolQuery = make(map[string]any)
			querySection["bool"] = boolQuery
		}
	} else {
		boolQuery = query
	}

	var shouldClauses []any
	if rawShould, exists := boolQuery["should"]; exists {
		if arr, ok := rawShould.([]any); ok {
			shouldClauses = arr
		} else {
			shouldClauses = []any{}
		}
	} else {
		shouldClauses = []any{}
	}

	if opts.From != "" {
		fromVal := strings.Trim(opts.From, "\"")
		senderFilters := []any{
			map[string]any{"match": map[string]any{"user_name": fromVal}},
			map[string]any{"match": map[string]any{"username": fromVal}},
			map[string]any{"match": map[string]any{"full_name": fromVal}},
			map[string]any{"match": map[string]any{"messages.user_name": fromVal}},
			map[string]any{"match": map[string]any{"messages.username": fromVal}},
			map[string]any{"match": map[string]any{"messages.full_name": fromVal}},
		}

		shouldClauses = append(shouldClauses, senderFilters...)
		boolQuery["should"] = shouldClauses
		boolQuery["minimum_should_match"] = 1
	}
}

func addChannelFilter(boolQuery map[string]any, opts *SearchQueryFiltersKeywords) {
	if opts.In != "" {
		channelName := strings.Trim(opts.In, "\"")

		if _, exists := boolQuery["must"]; !exists {
			boolQuery["must"] = []any{}
		}

		mustClauses := boolQuery["must"].([]any)

		mustClauses = append(mustClauses, map[string]any{
			"match_phrase": map[string]any{
				"channel_name": channelName,
			},
		})

		boolQuery["must"] = mustClauses
	}
}

func addDateFilters(boolQuery map[string]any, opts *SearchQueryFiltersKeywords) {
	filterClauses := boolQuery["filter"].([]any)

	if !opts.On.IsZero() {
		startOfDay := opts.On.Truncate(24 * time.Hour)
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)
		onFilter := map[string]any{
			"range": map[string]any{
				"created_at": map[string]any{
					"gte": startOfDay.Format(time.RFC3339),
					"lte": endOfDay.Format(time.RFC3339),
				},
			},
		}
		onFilter2 := map[string]any{
			"range": map[string]any{
				"messages.created_at": map[string]any{
					"gte": startOfDay.Format(time.RFC3339),
					"lte": endOfDay.Format(time.RFC3339),
				},
			},
		}
		filterClauses = append(filterClauses, onFilter, onFilter2)
	}

	if !opts.Before.IsZero() {
		beforeFilter := map[string]any{
			"range": map[string]any{
				"created_at": map[string]any{
					"lt": opts.Before.Format(time.RFC3339),
				},
			},
		}
		beforeFilter2 := map[string]any{
			"range": map[string]any{
				"messages.created_at": map[string]any{
					"lt": opts.Before.Format(time.RFC3339),
				},
			},
		}
		filterClauses = append(filterClauses, beforeFilter, beforeFilter2)
	}

	if !opts.After.IsZero() {
		afterFilter := map[string]any{
			"range": map[string]any{
				"created_at": map[string]any{
					"gt": opts.After.Format(time.RFC3339),
				},
			},
		}
		afterFilter2 := map[string]any{
			"range": map[string]any{
				"messages.created_at": map[string]any{
					"gt": opts.After.Format(time.RFC3339),
				},
			},
		}
		filterClauses = append(filterClauses, afterFilter, afterFilter2)
	}

	boolQuery["filter"] = filterClauses
}

func addContentFilter(boolQuery map[string]any, opts *SearchQueryFiltersKeywords) {
	mustClauses := boolQuery["must"].([]any)
	if opts.Has != "" {
		mustClauses = append(mustClauses, map[string]any{
			"multi_match": map[string]any{
				"query":  opts.Has,
				"type":   "best_fields",
				"fields": []string{"message", "messages.message"},
			},
		})
	}
	boolQuery["must"] = mustClauses
}

func addSorting(query map[string]any, opts *SearchQueryFiltersKeywords) (map[string]any, error) {
	var sorting []any
	if opts.SortBy == "recency" {
		sorting = []any{
			map[string]any{
				"created_at": map[string]any{
					"order": "desc",
				},
			},
		}
	} else {
		// Default to relevance (_score descending)
		sorting = []any{
			map[string]any{
				"_score": map[string]any{
					"order": "desc",
				},
			},
		}
	}
	query["sort"] = sorting
	return query, nil
}

func GetChannelsByOrgIDs(db *gorm.DB, orgId string, userId string) ([]string, error) {
	var channels Channels
	var channs []string

	org := Organisation{}

	if exists := postgresql.CheckExists(db, &org, "id = ?", orgId); exists == false {
		return nil, errors.New("Organisation does not exist")
	}

	orgs, err := org.GetUserOrganisations(db, userId)
	if err != nil && orgs == nil {
		return nil, err
	} else if orgs == nil {
		return nil, errors.New("User does not exist in this organisation")
	}

	if err := db.Model(&channels).
		Select("channels.id").
		Where("channels.organisation_id = ?", orgId).
		Scan(&channs).Error; err != nil {
		return nil, errors.New("error fetching channels")
	}
	return channs, nil
}

// FetchReactionsForMessages fetches all reactions for given message IDs
func FetchReactionsForMessages(db *gorm.DB, messageIDs []string) (map[string][]ReactionData, error) {
	if len(messageIDs) == 0 {
		return make(map[string][]ReactionData), nil
	}

	var reactions []ReactionData
	err := db.Table("reactions").
		Select(`
			reactions.message_id,
			reactions.reaction_id,
			reactions.reaction,
			reactions.user_id,
			users.user_name,
			users.avatar_url
		`).
		Joins("LEFT JOIN users ON reactions.user_id = users.id").
		Where("reactions.message_id IN ? AND reactions.message_id IS NOT NULL", messageIDs).
		Scan(&reactions).Error

	if err != nil {
		return nil, err
	}

	// Group reactions by message_id
	reactionMap := make(map[string][]ReactionData)
	for _, reaction := range reactions {
		reactionMap[reaction.MessageID] = append(reactionMap[reaction.MessageID], reaction)
	}

	return reactionMap, nil
}

// FetchThreadDataForMessages fetches thread metadata for messages
func FetchThreadDataForMessages(db *gorm.DB, messageIDs []string) (map[string]ThreadData, error) {
	if len(messageIDs) == 0 {
		return make(map[string]ThreadData), nil
	}

	var threadIDs []string
	err := db.Table("messages").
		Select("DISTINCT thread_id").
		Where("id IN ? AND thread_id IS NOT NULL", messageIDs).
		Pluck("thread_id", &threadIDs).Error

	if err != nil || len(threadIDs) == 0 {
		return make(map[string]ThreadData), nil
	}

	var threads []ThreadData
	err = db.Table("threads").
		Select("id, message_count, last_reply").
		Where("id IN ?", threadIDs).
		Scan(&threads).Error

	if err != nil {
		return nil, err
	}

	threadMap := make(map[string]ThreadData)
	for _, thread := range threads {
		threadMap[thread.ThreadID] = thread
	}

	return threadMap, nil
}

// FetchReplyUsersForThreads fetches users who replied in threads
func FetchReplyUsersForThreads(db *gorm.DB, threadIDs []string) (map[string][]ReplyUserData, error) {
	if len(threadIDs) == 0 {
		return make(map[string][]ReplyUserData), nil
	}

	var replyUsers []ReplyUserData
	err := db.Table("messages").
		Select(`
			DISTINCT ON (messages.thread_id, messages.user_id)
			messages.thread_id,
			messages.user_id,
			messages.username,
			messages.avatar_url
		`).
		Where("messages.thread_id IN ? AND messages.user_id IS NOT NULL", threadIDs).
		Order("messages.thread_id, messages.user_id, messages.created_at DESC").
		Scan(&replyUsers).Error

	if err != nil {
		return nil, err
	}

	replyUserMap := make(map[string][]ReplyUserData)
	for _, user := range replyUsers {
		replyUserMap[user.ThreadID] = append(replyUserMap[user.ThreadID], user)
	}

	return replyUserMap, nil
}

// AggregateReactions groups reactions by emoji and counts them
func AggregateReactions(reactions []ReactionData) []utility.ReactionInfo {
	if len(reactions) == 0 {
		return nil
	}

	reactionGroups := make(map[string]*utility.ReactionInfo)

	for _, reaction := range reactions {
		if _, exists := reactionGroups[reaction.Reaction]; !exists {
			reactionGroups[reaction.Reaction] = &utility.ReactionInfo{
				ReactionID: reaction.ReactionID,
				Emoji:      reaction.Reaction,
				Count:      0,
				Users:      []utility.ReactionUser{},
			}
		}

		reactionGroups[reaction.Reaction].Count++
		reactionGroups[reaction.Reaction].Users = append(
			reactionGroups[reaction.Reaction].Users,
			utility.ReactionUser{
				UserID:    reaction.UserID,
				UserName:  reaction.UserName,
				AvatarURL: reaction.AvatarURL,
			},
		)
	}

	result := make([]utility.ReactionInfo, 0, len(reactionGroups))
	for _, reactionInfo := range reactionGroups {
		result = append(result, *reactionInfo)
	}

	return result
}

// ConvertReplyUsersToUtilityFormat converts database reply users to utility format
func ConvertReplyUsersToUtilityFormat(replyUsers []ReplyUserData) []utility.ReplyUser {
	if len(replyUsers) == 0 {
		return nil
	}

	result := make([]utility.ReplyUser, 0, len(replyUsers))
	for _, user := range replyUsers {
		result = append(result, utility.ReplyUser{
			UserID:    user.UserID,
			UserName:  user.UserName,
			AvatarURL: user.AvatarURL,
		})
	}

	return result
}
