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

type ReactionUser struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	AvatarURL string `json:"avatar_url"`
}

type ReactionInfo struct {
	ReactionID string         `json:"reaction_id"`
	Emoji      string         `json:"emoji"`
	Count      int            `json:"count"`
	Users      []ReactionUser `json:"users"`
}

type ReplyUser struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	AvatarURL string `json:"avatar_url"`
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
	return MessageQuery{
		MessageID: getString(source, "id"),
		Message:   getString(source, "message"),
		TimeStamp: getTime(source, "created_at"),
	}
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
