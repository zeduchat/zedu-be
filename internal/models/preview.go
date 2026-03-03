package models

import (
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
