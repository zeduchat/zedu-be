package thread

import (
	"strings"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/utility"
)

func SaveOrMergeJoinSystemMessage(db *storage.Database, logger *utility.Logger, channelID, channelName, orgID string, newUsers []models.UserMention, adder *models.UserMention) error {
	if len(newUsers) == 0 {
		return nil
	}

	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id": channelID,
						},
					},
					{
						"term": map[string]any{
							"type": "system",
						},
					},
					{
						"range": map[string]any{
							"created_at": map[string]any{
								"gte": oneHourAgo,
							},
						},
					},
				},
			},
		},
		"sort": []map[string]any{
			{
				"created_at": map[string]any{
					"order": "desc",
				},
			},
		},
		"size": 1,
	}

	var threadData any
	err := elastic.SelectAll(db.Elastic, models.ThreadIndexName, query, &threadData)
	if err == nil {
		if threadDataMap, ok := threadData.(map[string]any); ok {
			if hits, ok := threadDataMap["hits"].(map[string]any); ok {
				if hitsArray, ok := hits["hits"].([]any); ok && len(hitsArray) > 0 {
					if hitMap, ok := hitsArray[0].(map[string]any); ok {
						if source, ok := hitMap["_source"].(map[string]any); ok {
							threadID := utility.GetString(source, "id")
							if threadID == "" {
								threadID = utility.GetString(source, "thread_id")
							}

							if threadID != "" {
								// Extract existing joined users & adder from stored metadata or payload
								existingUsers := extractUsersFromSource(db, source, orgID)
								combinedUsers := mergeUserMentions(existingUsers, newUsers)

								activeAdder := adder
								if activeAdder == nil {
									activeAdder = extractAdderFromSource(source)
								}

								newContent := models.FormatMergedChannelJoinMessage(combinedUsers, activeAdder, channelName)

								var mentions []models.Mention
								for _, u := range combinedUsers {
									mentions = append(mentions, models.Mention{ID: u.UserID, Type: "user"})
								}
								if activeAdder != nil && activeAdder.UserID != "" {
									mentions = append(mentions, models.Mention{ID: activeAdder.UserID, Type: "user"})
								}

								updateFields := map[string]any{
									"message":    newContent,
									"mentions":   mentions,
									"edited":     false, // Ensure (edited) indicator does not show
									"updated_at": time.Now().UTC(),
								}

								_ = elastic.UpdateDocument(db.Elastic, models.ThreadIndexName, threadID, updateFields)

								feed := models.FeedMessageRequest{
									ChannelID:   channelID,
									UserName:    combinedUsers[0].Username,
									CreatedAt:   time.Now().UTC().Format(time.RFC3339),
									Type:        "system",
									Content:     newContent,
									ThreadId:    threadID,
									UserId:      combinedUsers[0].UserID,
									OrgId:       orgID,
									ChannelName: channelName,
									Edited:      false,
								}
								_ = centrifuge.PublishChannel(logger, channelID, feed)
								return nil
							}
						}
					}
				}
			}
		}
	}

	// No recent system join message found within 1 hour: create a new one
	newContent := models.FormatMergedChannelJoinMessage(newUsers, adder, channelName)

	var mentions []models.Mention
	for _, u := range newUsers {
		mentions = append(mentions, models.Mention{ID: u.UserID, Type: "user"})
	}
	if adder != nil && adder.UserID != "" {
		mentions = append(mentions, models.Mention{ID: adder.UserID, Type: "user"})
	}

	systemMsg := models.CreateThreadMsgReq{
		Content:    newContent,
		Type:       "system",
		UserId:     newUsers[0].UserID,
		ChannelsID: channelID,
		OrgId:      orgID,
		ThreadId:   utility.GenerateUUID(),
		Mentions:   mentions,
	}

	_, err = SaveThreadMessage(systemMsg, db, logger)
	return err
}

func extractUsersFromSource(db *storage.Database, source map[string]any, orgID string) []models.UserMention {
	var users []models.UserMention
	seen := make(map[string]bool)

	if mentionsArray, ok := source["mentions"].([]any); ok {
		for _, m := range mentionsArray {
			if mMap, ok := m.(map[string]any); ok {
				uID := utility.GetString(mMap, "id")
				mType := utility.GetString(mMap, "type")
				if uID != "" && mType == "user" && !seen[uID] {
					seen[uID] = true
					var u models.User
					uDetails, err := u.GetUserByID(db.Postgresql, uID, orgID)
					uName := uID
					if err == nil {
						uName = uDetails.Profile.UserName
						if uName == "" {
							uName = utility.SplitEmailString(uDetails.Email)
						}
					}
					users = append(users, models.UserMention{UserID: uID, Username: uName})
				}
			}
		}
	}

	if len(users) == 0 {
		uID := utility.GetString(source, "user_id")
		uName := utility.GetString(source, "username")
		if uID != "" {
			users = append(users, models.UserMention{UserID: uID, Username: uName})
		}
	}

	return users
}

func extractAdderFromSource(source map[string]any) *models.UserMention {
	return nil
}

func mergeUserMentions(existing []models.UserMention, newUsers []models.UserMention) []models.UserMention {
	seen := make(map[string]bool)
	var merged []models.UserMention

	for _, u := range existing {
		if !seen[u.UserID] {
			seen[u.UserID] = true
			merged = append(merged, u)
		}
	}

	for _, u := range newUsers {
		if !seen[u.UserID] {
			seen[u.UserID] = true
			merged = append(merged, u)
		}
	}

	return merged
}

func SaveOrMergeLeaveSystemMessage(db *storage.Database, logger *utility.Logger, channelID, channelName, orgID string, leftUsers []models.UserMention, remover *models.UserMention) error {
	if len(leftUsers) == 0 {
		return nil
	}

	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id": channelID,
						},
					},
					{
						"term": map[string]any{
							"type": "system",
						},
					},
					{
						"range": map[string]any{
							"created_at": map[string]any{
								"gte": oneHourAgo,
							},
						},
					},
				},
			},
		},
		"sort": []map[string]any{
			{
				"created_at": map[string]any{
					"order": "desc",
				},
			},
		},
		"size": 1,
	}

	var threadData any
	err := elastic.SelectAll(db.Elastic, models.ThreadIndexName, query, &threadData)
	if err == nil {
		if threadDataMap, ok := threadData.(map[string]any); ok {
			if hits, ok := threadDataMap["hits"].(map[string]any); ok {
				if hitsArray, ok := hits["hits"].([]any); ok && len(hitsArray) > 0 {
					if hitMap, ok := hitsArray[0].(map[string]any); ok {
						if source, ok := hitMap["_source"].(map[string]any); ok {
							threadID := utility.GetString(source, "id")
							if threadID == "" {
								threadID = utility.GetString(source, "thread_id")
							}

							msgContent := utility.GetString(source, "message")
							if threadID != "" && (strings.Contains(strings.ToLower(msgContent), "left") || strings.Contains(strings.ToLower(msgContent), "removed")) {
								existingUsers := extractUsersFromSource(db, source, orgID)
								combinedUsers := mergeUserMentions(existingUsers, leftUsers)

								activeRemover := remover
								if activeRemover == nil {
									activeRemover = extractAdderFromSource(source)
								}

								newContent := models.FormatMergedChannelLeaveMessage(combinedUsers, activeRemover, channelName)

								var mentions []models.Mention
								for _, u := range combinedUsers {
									mentions = append(mentions, models.Mention{ID: u.UserID, Type: "user"})
								}
								if activeRemover != nil && activeRemover.UserID != "" {
									mentions = append(mentions, models.Mention{ID: activeRemover.UserID, Type: "user"})
								}

								updateFields := map[string]any{
									"message":    newContent,
									"mentions":   mentions,
									"edited":     false,
									"updated_at": time.Now().UTC(),
								}

								_ = elastic.UpdateDocument(db.Elastic, models.ThreadIndexName, threadID, updateFields)

								feed := models.FeedMessageRequest{
									ChannelID:   channelID,
									UserName:    combinedUsers[0].Username,
									CreatedAt:   time.Now().UTC().Format(time.RFC3339),
									Type:        "system",
									Content:     newContent,
									ThreadId:    threadID,
									UserId:      combinedUsers[0].UserID,
									OrgId:       orgID,
									ChannelName: channelName,
									Edited:      false,
								}
								_ = centrifuge.PublishChannel(logger, channelID, feed)
								return nil
							}
						}
					}
				}
			}
		}
	}

	newContent := models.FormatMergedChannelLeaveMessage(leftUsers, remover, channelName)

	var mentions []models.Mention
	for _, u := range leftUsers {
		mentions = append(mentions, models.Mention{ID: u.UserID, Type: "user"})
	}
	if remover != nil && remover.UserID != "" {
		mentions = append(mentions, models.Mention{ID: remover.UserID, Type: "user"})
	}

	systemMsg := models.CreateThreadMsgReq{
		Content:    newContent,
		Type:       "system",
		UserId:     leftUsers[0].UserID,
		ChannelsID: channelID,
		OrgId:      orgID,
		ThreadId:   utility.GenerateUUID(),
		Mentions:   mentions,
	}

	_, err = SaveThreadMessage(systemMsg, db, logger)
	return err
}
