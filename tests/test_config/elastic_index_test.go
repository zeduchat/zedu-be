package test_config

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
)

func TestResolveIndexPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		appMode  string
		expected string
	}{
		{
			name:     "Empty prefix and mode",
			prefix:   "",
			appMode:  "",
			expected: "",
		},
		{
			name:     "Dev prefix with local mode",
			prefix:   "dev_",
			appMode:  "local",
			expected: "dev_",
		},
		{
			name:     "Production mode overrides prefix",
			prefix:   "dev_",
			appMode:  "production",
			expected: "",
		},
		{
			name:     "Prod mode short form overrides prefix",
			prefix:   "myprefix_",
			appMode:  "prod",
			expected: "",
		},
		{
			name:     "Staging mode overrides prefix",
			prefix:   "myprefix_",
			appMode:  "staging",
			expected: "",
		},
		{
			name:     "Stage mode overrides prefix",
			prefix:   "myprefix_",
			appMode:  "stage",
			expected: "",
		},
		{
			name:     "Prefix set to production overrides itself",
			prefix:   "production",
			appMode:  "development",
			expected: "",
		},
		{
			name:     "Case insensitive matching",
			prefix:   "  DEV_  ",
			appMode:  "LOCAL",
			expected: "DEV_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := elastic.ResolveIndexPrefix(tt.prefix, tt.appMode)
			if got != tt.expected {
				t.Errorf("ResolveIndexPrefix(%q, %q) = %q; want %q", tt.prefix, tt.appMode, got, tt.expected)
			}
		})
	}
}

func TestInitIndexNames(t *testing.T) {
	models.InitIndexNames("test_", "development")
	if models.ThreadIndexName != "test_threads" {
		t.Errorf("expected models.ThreadIndexName to be 'test_threads', got %q", models.ThreadIndexName)
	}
	if models.MessageIndexName != "test_messages" {
		t.Errorf("expected models.MessageIndexName to be 'test_messages', got %q", models.MessageIndexName)
	}

	// Reset for production
	models.InitIndexNames("test_", "production")
	if models.ThreadIndexName != "threads" {
		t.Errorf("expected models.ThreadIndexName to be 'threads', got %q", models.ThreadIndexName)
	}
	if models.MessageIndexName != "messages" {
		t.Errorf("expected models.MessageIndexName to be 'messages', got %q", models.MessageIndexName)
	}
}
