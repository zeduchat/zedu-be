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

	var messageIDs []string
	messageIDToResultIndex := make(map[string]int)

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

		result := utility.ProcessMessageHit(index, source)

		for _, msg := range result.Messages {
			if msg.MessageID != "" {
				messageIDs = append(messageIDs, msg.MessageID)
				messageIDToResultIndex[msg.MessageID] = len(qResults)
			}
		}

		qResults = append(qResults, result)
	}

	if len(messageIDs) == 0 {
		return qResults, nil
	}

	// Build reaction map from ES hits when available, otherwise fetch from ES reactions index
	reactionMap := make(map[string][]ReactionData)
	var missingMessageIDs []string

	// build thread map from ES hits
	threadIDMap := make(map[string]string)
	threadIDsSet := make(map[string]struct{})

	for i := range qResults {
		for j := range qResults[i].Messages {
			mid := qResults[i].Messages[j].MessageID
			// reactions present in hit source are already populated via utility.ProcessMessageHit
			if len(qResults[i].Messages[j].Reactions) > 0 {
				// convert utility.ReactionInfo to local ReactionData slice
				for _, ri := range qResults[i].Messages[j].Reactions {
					for _, u := range ri.Users {
						reactionMap[mid] = append(reactionMap[mid], ReactionData{
							MessageID:  mid,
							ReactionID: ri.ReactionID,
							Reaction:   ri.Emoji,
							UserID:     u.UserID,
							UserName:   u.UserName,
							AvatarURL:  u.AvatarURL,
						})
					}
				}
			} else {
				missingMessageIDs = append(missingMessageIDs, mid)
			}

			// thread id extracted from ES hit (if present)
			tid := qResults[i].Thread.ID
			if tid != "" {
				threadIDMap[mid] = tid
				threadIDsSet[tid] = struct{}{}
			}
		}
	}

	// Fetch reactions from ES for messages missing reactions
	if len(missingMessageIDs) > 0 {
		esReactions, err := FetchReactionsForMessages(db, missingMessageIDs)
		if err != nil {
			fmt.Printf("Warning: failed to fetch reactions from ES: %v\n", err)
		} else {
			for k, v := range esReactions {
				reactionMap[k] = append(reactionMap[k], v...)
			}
		}
	}

	// Build list of thread IDs
	var threadIDs []string
	for tid := range threadIDsSet {
		threadIDs = append(threadIDs, tid)
	}

	// Fetch thread metadata from ES (threads index)
	threadDataMap := make(map[string]ThreadData)
	if len(threadIDs) > 0 {
		td, err := FetchThreadDataForThreads(db, threadIDs)
		if err != nil {
			fmt.Printf("Warning: failed to fetch thread data from ES: %v\n", err)
		} else {
			threadDataMap = td
		}
	}

	// Fetch reply users from ES (messages index) for threads
	replyUsersMap := make(map[string][]ReplyUserData)
	if len(threadIDs) > 0 {
		rum, err := FetchReplyUsersForThreads(db, threadIDs)
		if err != nil {
			fmt.Printf("Warning: failed to fetch reply users from ES: %v\n", err)
		} else {
			replyUsersMap = rum
		}
	}

	for i := range qResults {
		for j := range qResults[i].Messages {
			messageID := qResults[i].Messages[j].MessageID

			if reactions, exists := reactionMap[messageID]; exists {
				qResults[i].Messages[j].Reactions = AggregateReactions(reactions)
			}

			if threadID, hasThread := threadIDMap[messageID]; hasThread {
				if threadData, exists := threadDataMap[threadID]; exists {
					if threadData.MessageCount != nil {
						count := int(*threadData.MessageCount)
						qResults[i].Messages[j].ReplyCount = &count
					}

					qResults[i].Messages[j].LastReplyTimestamp = threadData.LastReply
				}

				if replyUsers, exists := replyUsersMap[threadID]; exists {
					qResults[i].Messages[j].ReplyUsers = ConvertReplyUsersToUtilityFormat(replyUsers)
				}
			}
		}
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

// FetchReactionsForMessages fetches reactions from Elasticsearch 'reactions' index
func FetchReactionsForMessages(db *storage.Database, messageIDs []string) (map[string][]ReactionData, error) {
	result := make(map[string][]ReactionData)
	if len(messageIDs) == 0 || db == nil || db.Elastic == nil {
		return result, nil
	}

	query := map[string]any{
		"query": map[string]any{
			"terms": map[string]any{"message_id.keyword": messageIDs},
		},
		"size": 10000,
	}

	var raw any
	if err := elastic.SelectAll(db.Elastic, "reactions", query, &raw); err != nil {
		return nil, err
	}

	rawMap, ok := raw.(map[string]any)
	if !ok {
		return result, nil
	}

	hits, ok := rawMap["hits"].(map[string]any)
	if !ok {
		return result, nil
	}
	hitsArr, _ := hits["hits"].([]any)
	for _, h := range hitsArr {
		if hm, ok := h.(map[string]any); ok {
			if src, ok := hm["_source"].(map[string]any); ok {
				var mid, rid, react, uid, uname, aurl string
				if v, ok := src["message_id"].(string); ok {
					mid = v
				}
				if v, ok := src["reaction_id"].(string); ok {
					rid = v
				}
				if v, ok := src["reaction"].(string); ok {
					react = v
				}
				if v, ok := src["user_id"].(string); ok {
					uid = v
				}
				if v, ok := src["user_name"].(string); ok {
					uname = v
				}
				if v, ok := src["avatar_url"].(string); ok {
					aurl = v
				}
				rd := ReactionData{
					MessageID:  mid,
					ReactionID: rid,
					Reaction:   react,
					UserID:     uid,
					UserName:   uname,
					AvatarURL:  aurl,
				}
				result[mid] = append(result[mid], rd)
			}
		}
	}

	return result, nil
}

// FetchThreadDataForThreads fetches thread metadata from ES 'threads' index
func FetchThreadDataForThreads(db *storage.Database, threadIDs []string) (map[string]ThreadData, error) {
	result := make(map[string]ThreadData)
	if len(threadIDs) == 0 || db == nil || db.Elastic == nil {
		return result, nil
	}

	query := map[string]any{
		"query": map[string]any{
			"terms": map[string]any{"id.keyword": threadIDs},
		},
		"size": 10000,
	}

	var raw any
	if err := elastic.SelectAll(db.Elastic, "threads", query, &raw); err != nil {
		return nil, err
	}

	rawMap, ok := raw.(map[string]any)
	if !ok {
		return result, nil
	}
	hits, ok := rawMap["hits"].(map[string]any)
	if !ok {
		return result, nil
	}
	hitsArr, _ := hits["hits"].([]any)
	for _, h := range hitsArr {
		if hm, ok := h.(map[string]any); ok {
			if src, ok := hm["_source"].(map[string]any); ok {
				var id string
				if v, ok := src["id"].(string); ok {
					id = v
				}
				td := ThreadData{
					ThreadID:     id,
					MessageCount: nil,
					LastReply:    nil,
				}
				if mc, ok := src["message_count"]; ok {
					switch v := mc.(type) {
					case float64:
						c := int64(v)
						td.MessageCount = &c
					case int64:
						td.MessageCount = &v
					}
				}
				if lr, ok := src["last_reply"]; ok {
					if s, ok := lr.(string); ok {
						td.LastReply = &s
					}
				}
				result[id] = td
			}
		}
	}

	return result, nil
}

// FetchReplyUsersForThreads aggregates distinct users who replied in threads by scanning messages index
func FetchReplyUsersForThreads(db *storage.Database, threadIDs []string) (map[string][]ReplyUserData, error) {
	result := make(map[string][]ReplyUserData)
	if len(threadIDs) == 0 || db == nil || db.Elastic == nil {
		return result, nil
	}

	query := map[string]any{
		"query": map[string]any{
			"terms": map[string]any{"thread_id.keyword": threadIDs},
		},
		"size": 10000,
	}

	var raw any
	if err := elastic.SelectAll(db.Elastic, "messages", query, &raw); err != nil {
		return nil, err
	}

	rawMap, ok := raw.(map[string]any)
	if !ok {
		return result, nil
	}
	hits, ok := rawMap["hits"].(map[string]any)
	if !ok {
		return result, nil
	}
	hitsArr, _ := hits["hits"].([]any)

	seen := make(map[string]map[string]struct{}) // threadID -> set of userID
	for _, h := range hitsArr {
		if hm, ok := h.(map[string]any); ok {
			if src, ok := hm["_source"].(map[string]any); ok {
				var tid, uid string
				if v, ok := src["thread_id"].(string); ok {
					tid = v
				}
				if v, ok := src["user_id"].(string); ok {
					uid = v
				}
				if tid == "" || uid == "" {
					continue
				}
				if _, ok := seen[tid]; !ok {
					seen[tid] = make(map[string]struct{})
				}
				if _, exists := seen[tid][uid]; exists {
					continue
				}
				seen[tid][uid] = struct{}{}
				var uname, aurl string
				if v, ok := src["username"].(string); ok {
					uname = v
				}
				if v, ok := src["avatar_url"].(string); ok {
					aurl = v
				}
				rud := ReplyUserData{
					ThreadID:  tid,
					UserID:    uid,
					UserName:  uname,
					AvatarURL: aurl,
				}
				result[tid] = append(result[tid], rud)
			}
		}
	}

	return result, nil
}
