package utility

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

type Thread struct {
	ID string `json:"id"`
}

type Channel struct {
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name,omitempty"`
}

type MessageQuery struct {
	MessageID          string         `json:"message_id"`
	Message            string         `json:"message"`
	TimeStamp          time.Time      `json:"timestamp,omitempty"`
	Reactions          []ReactionInfo `json:"reactions,omitempty"`
	ReplyCount         *int           `json:"reply_count,omitempty"`
	LastReplyTimestamp *string        `json:"last_reply_timestamp,omitempty"`
	ReplyUsers         []ReplyUser    `json:"reply_users,omitempty"`
}

type ReactionUser struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	AvatarURL string `json:"avatar_url"`
}

type ReactionInfo struct {
	ReactionID string         `json:"reaction_id"`
	Emoji      string         `json:"emoji"`
	Count      int            `json:"count"`
	Users      []ReactionUser `json:"users,omitempty"`
}

type ReplyUser struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	AvatarURL string `json:"avatar_url"`
}

type UserQuery struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	AvatarURL string `json:"avatar_url"`
}

type SearchQueryResult struct {
	User     UserQuery      `json:"user,omitempty"`
	Messages []MessageQuery `json:"messages,omitempty"`
	Channel  Channel        `json:"channel,omitempty"`
	Thread   Thread         `json:"thread,omitempty"`
}

func CheckQueryStringContainKeyword(query string) [][]string {
	re := regexp.MustCompile(`(\w+):([^\s]+(?:\s[^\s]+)*)`)
	matches := re.FindAllStringSubmatch(query, -1)
	return matches
}

func ExtractWordsBeforeKeywords(input string) string {
	// Use regex to extract text before " in: ", " from: ", etc.
	re := regexp.MustCompile(`^(.*?)\s*\b(in|from|has|before|on|after|exact):`)

	// Match the regex in the input string
	match := re.FindStringSubmatch(input)

	if len(match) > 1 {
		result := strings.TrimSpace(match[1]) // Remove extra spaces
		if result == `"` {
			return "" // Ensure no empty spaces are returned
		}
		return result
	}
	return strings.TrimSpace(input) // If no keyword is found, return the input
}

// processMessageHit processes a message or thread and returns a SearchQueryResult
func ProcessMessageHit(index string, source map[string]any) SearchQueryResult {
	result := SearchQueryResult{
		User: extractUserInfo(source),
		Messages: []MessageQuery{
			extractMessageInfo(source),
		},
		Channel: extractChannelInfo(source, index),
		Thread:  extractThreadInfo(source),
	}
	return result
}

// extractUserInfo extracts user-related data
func extractUserInfo(source map[string]any) UserQuery {
	return UserQuery{
		UserID:    getString(source, "user_id"),
		UserName:  getString(source, "username"),
		AvatarURL: getString(source, "avatar_url"),
	}
}

// extractMessageInfo extracts message-related data
func extractMessageInfo(source map[string]any) MessageQuery {
	mq := MessageQuery{
		MessageID: getString(source, "id"),
		Message:   getString(source, "message"),
		TimeStamp: getTime(source, "created_at"),
	}

	// Extract and aggregate reactions from nested structure
	if reactionsAny, ok := source["reactions"]; ok {
		if arr, ok := reactionsAny.([]any); ok && len(arr) > 0 {
			reactionGroups := make(map[string]*ReactionInfo)

			for _, r := range arr {
				if rm, ok := r.(map[string]any); ok {
					emoji := getString(rm, "emoji")
					if emoji == "" {
						continue
					}

					if _, exists := reactionGroups[emoji]; !exists {
						reactionGroups[emoji] = &ReactionInfo{
							ReactionID: getString(rm, "id"),
							Emoji:      emoji,
							Count:      0,
							Users:      []ReactionUser{},
						}
					}

					reactionGroups[emoji].Count++
					reactionGroups[emoji].Users = append(
						reactionGroups[emoji].Users,
						ReactionUser{
							UserID:    getString(rm, "user_id"),
							UserName:  getString(rm, "user_name"),
							AvatarURL: getString(rm, "avatar_url"),
						},
					)
				}
			}

			for _, reactionInfo := range reactionGroups {
				mq.Reactions = append(mq.Reactions, *reactionInfo)
			}
		}
	}

	// Extract reply metadata if present (only set when > 0)
	if rc, ok := source["reply_count"]; ok {
		if v := toInt(rc); v > 0 {
			mq.ReplyCount = ptrInt(v)
		}
	} else if rc2, ok := source["message_count"]; ok {
		if v := toInt(rc2); v > 0 {
			mq.ReplyCount = ptrInt(v)
		}
	}

	// Only set last reply timestamp when it's a valid, non-zero time
	if lrt, ok := source["last_reply"]; ok {
		if s, ok := lrt.(string); ok && s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil && !t.IsZero() {
				mq.LastReplyTimestamp = &s
			}
		}
	} else if lrt2, ok := source["last_reply_timestamp"]; ok {
		if s, ok := lrt2.(string); ok && s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil && !t.IsZero() {
				mq.LastReplyTimestamp = &s
			}
		}
	}

	if ruAny, ok := source["reply_users"]; ok {
		if arr, ok := ruAny.([]any); ok {
			for _, u := range arr {
				if um, ok := u.(map[string]any); ok {
					r := ReplyUser{
						UserID:    getString(um, "user_id"),
						UserName:  getString(um, "user_name"),
						AvatarURL: getString(um, "avatar_url"),
					}
					mq.ReplyUsers = append(mq.ReplyUsers, r)
				}
			}
		}
	}

	return mq
}

// extractChannelInfo extracts channel-related data and sets channel_name for threads
func extractChannelInfo(source map[string]any, index string) Channel {
	channel := Channel{
		ChannelID: getString(source, "channels_id"),
	}
	if index == "threads" { // Only threads have channel_name
		channel.ChannelName = getString(source, "username")
	}
	return channel
}

// extractThreadInfo extracts thread-related data
func extractThreadInfo(source map[string]any) Thread {
	return Thread{
		ID: getString(source, "thread_id"),
	}
}

// getString safely retrieves a string from a map
func getString(source map[string]any, key string) string {
	if val, ok := source[key].(string); ok {
		return val
	}
	return ""
}

// getTime safely retrieves a time.Time value from a map
func getTime(source map[string]any, key string) time.Time {
	if val, ok := source[key].(string); ok {
		parsedTime, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return parsedTime
		}
	}
	return time.Time{}
}

// toInt converts various numeric types to int (safe for float64 from json)
func toInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case float32:
		return int(t)
	case float64:
		return int(t)
	case string:
		// attempt parse
		return 0
	default:
		return 0
	}
}

func ptrInt(i int) *int { return &i }

func ExtractCollectionName(fullName string) string {
	if len(fullName) > 84 {
		return fullName[84:]
	}
	return fullName
}

func ExtractBaseURL(raw string) (string, error) {
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "http://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	host := u.Hostname()
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host, nil
	}
	return domain, nil
}

func ReadSystemPromptFromFile(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path is empty")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}
