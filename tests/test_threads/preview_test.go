package test_threads

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
)

func TestBuildPreviewMessage(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		media    []models.File
		expected string
	}{
		{
			name:     "plain text returned as-is",
			content:  "Hello world",
			media:    nil,
			expected: "Hello world",
		},
		{
			name:     "HTML-only content strips to empty then falls through to media",
			content:  "<p></p><p></p>",
			media:    []models.File{{MimeType: "image/png"}},
			expected: "📷 Photo",
		},
		{
			name:     "mixed HTML and text extracts text",
			content:  "<p>Hello</p>",
			media:    nil,
			expected: "<p>Hello</p>",
		},
		{
			name:     "nested HTML tags stripped",
			content:  "<div><strong>Bold text</strong></div>",
			media:    nil,
			expected: "<div><strong>Bold text</strong></div>",
		},
		{
			name:     "empty content and no media returns empty string",
			content:  "",
			media:    nil,
			expected: "",
		},
		{
			name:     "empty content with image/png media",
			content:  "",
			media:    []models.File{{MimeType: "image/png"}},
			expected: "📷 Photo",
		},
		{
			name:     "empty content with image/jpeg media",
			content:  "",
			media:    []models.File{{MimeType: "image/jpeg"}},
			expected: "📷 Photo",
		},
		{
			name:     "empty content with video/mp4 media",
			content:  "",
			media:    []models.File{{MimeType: "video/mp4"}},
			expected: "🎥 Video",
		},
		{
			name:     "empty content with audio/mpeg media",
			content:  "",
			media:    []models.File{{MimeType: "audio/mpeg"}},
			expected: "🎵 Audio",
		},
		{
			name:     "empty content with application/pdf media",
			content:  "",
			media:    []models.File{{MimeType: "application/pdf"}},
			expected: "📄 PDF",
		},
		{
			name:     "empty content with word document media",
			content:  "",
			media:    []models.File{{MimeType: "application/msword"}},
			expected: "📝 Document",
		},
		{
			name:     "empty content with docx media",
			content:  "",
			media:    []models.File{{MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}},
			expected: "📝 Document",
		},
		{
			name:     "empty content with excel media",
			content:  "",
			media:    []models.File{{MimeType: "application/vnd.ms-excel"}},
			expected: "📊 Spreadsheet",
		},
		{
			name:     "empty content with xlsx media",
			content:  "",
			media:    []models.File{{MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}},
			expected: "📊 Spreadsheet",
		},
		{
			name:     "empty content with powerpoint media",
			content:  "",
			media:    []models.File{{MimeType: "application/vnd.ms-powerpoint"}},
			expected: "📊 Presentation",
		},
		{
			name:     "empty content with zip archive media",
			content:  "",
			media:    []models.File{{MimeType: "application/zip"}},
			expected: "🗜️ Archive",
		},
		{
			name:     "empty content with unknown mime type",
			content:  "",
			media:    []models.File{{MimeType: "application/octet-stream"}},
			expected: "📎 File",
		},
		{
			name:     "HTML-only with no media returns empty string",
			content:  "<p></p>",
			media:    nil,
			expected: "",
		},
		{
			name:     "whitespace-only HTML returns empty string",
			content:  "<p>   </p>",
			media:    nil,
			expected: "",
		},
		{
			name:     "plain text takes priority over media",
			content:  "Some message",
			media:    []models.File{{MimeType: "image/png"}},
			expected: "Some message",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := models.BuildPreviewMessage(tc.content, tc.media)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
