package models

import (
	"fmt"
	"regexp"
	"strings"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func stripHTMLTags(content string) string {
	stripped := htmlTagRegex.ReplaceAllString(content, "")
	return strings.TrimSpace(stripped)
}

func mimeTypeToPreviewLabel(mimeType string) string {
	mime := strings.ToLower(mimeType)

	switch {
	case strings.HasPrefix(mime, "image/"):
		return "📷 Photo"
	case strings.HasPrefix(mime, "video/"):
		return "🎥 Video"
	case strings.HasPrefix(mime, "audio/"):
		return "🎵 Audio"
	case mime == "application/pdf":
		return "📄 PDF"
	case strings.Contains(mime, "word") || strings.Contains(mime, "msword") || strings.Contains(mime, "wordprocessingml"):
		return "📝 Document"
	case strings.Contains(mime, "excel") || strings.Contains(mime, "spreadsheetml") || strings.Contains(mime, "ms-excel"):
		return "📊 Spreadsheet"
	case strings.Contains(mime, "powerpoint") || strings.Contains(mime, "presentationml"):
		return "📊 Presentation"
	case mime == "application/zip" || strings.Contains(mime, "rar") || strings.Contains(mime, "7z") || strings.Contains(mime, "tar"):
		return "🗜️ Archive"
	default:
		return "📎 File"
	}
}

func BuildPreviewMessage(content string, media []File) string {
	plainText := stripHTMLTags(content)
	if plainText != "" {
		return content
	}
	if len(media) == 0 {
		return ""
	}
	return mimeTypeToPreviewLabel(media[0].MimeType)
}

type UserMention struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

func FormatMergedChannelJoinMessage(users []UserMention, adder *UserMention, channelName string) string {
	if len(users) == 0 {
		return ""
	}

	firstUser := fmt.Sprintf(`<span class="mention" data-type="mention" data-id="%s" data-label="%s" data-mention-suggestion-char="@">@%s</span>`, users[0].UserID, users[0].Username, users[0].Username)

	var base string
	if adder != nil && adder.UserID != "" {
		adderMention := fmt.Sprintf(`<span class="mention" data-type="mention" data-id="%s" data-label="%s" data-mention-suggestion-char="@">@%s</span>`, adder.UserID, adder.Username, adder.Username)
		base = fmt.Sprintf("%s has been added to this channel by %s", firstUser, adderMention)
	} else {
		base = fmt.Sprintf("%s joined this channel", firstUser)
	}

	if len(users) == 1 {
		return fmt.Sprintf("<p>%s</p><p></p>", base)
	}

	secondUser := fmt.Sprintf(`<span class="mention" data-type="mention" data-id="%s" data-label="%s" data-mention-suggestion-char="@">@%s</span>`, users[1].UserID, users[1].Username, users[1].Username)

	if len(users) == 2 {
		return fmt.Sprintf("<p>%s. %s also joined.</p><p></p>", base, secondUser)
	}

	othersCount := len(users) - 2
	if othersCount == 1 {
		return fmt.Sprintf("<p>%s. %s and 1 other also joined.</p><p></p>", base, secondUser)
	}
	return fmt.Sprintf("<p>%s. %s and %d others also joined.</p><p></p>", base, secondUser, othersCount)
}

func FormatMergedChannelLeaveMessage(users []UserMention, remover *UserMention, channelName string) string {
	if len(users) == 0 {
		return ""
	}

	firstUser := fmt.Sprintf(`<span class="mention" data-type="mention" data-id="%s" data-label="%s" data-mention-suggestion-char="@">@%s</span>`, users[0].UserID, users[0].Username, users[0].Username)

	var base string
	if remover != nil && remover.UserID != "" {
		removerMention := fmt.Sprintf(`<span class="mention" data-type="mention" data-id="%s" data-label="%s" data-mention-suggestion-char="@">@%s</span>`, remover.UserID, remover.Username, remover.Username)
		base = fmt.Sprintf("%s was removed from this channel by %s", firstUser, removerMention)
	} else {
		base = fmt.Sprintf("%s left this channel", firstUser)
	}

	if len(users) == 1 {
		return fmt.Sprintf("<p>%s</p><p></p>", base)
	}

	secondUser := fmt.Sprintf(`<span class="mention" data-type="mention" data-id="%s" data-label="%s" data-mention-suggestion-char="@">@%s</span>`, users[1].UserID, users[1].Username, users[1].Username)

	if len(users) == 2 {
		return fmt.Sprintf("<p>%s. %s also left.</p><p></p>", base, secondUser)
	}

	othersCount := len(users) - 2
	if othersCount == 1 {
		return fmt.Sprintf("<p>%s. %s and 1 other also left.</p><p></p>", base, secondUser)
	}
	return fmt.Sprintf("<p>%s. %s and %d others also left.</p><p></p>", base, secondUser, othersCount)
}
