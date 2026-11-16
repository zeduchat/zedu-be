package test_links

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hngprojects/telex_be/services/links"
)

func TestExtractURLs(t *testing.T) {
	manyURLs := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		manyURLs = append(manyURLs, fmt.Sprintf("https://example.com/page/%d", i))
	}

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name:    "plain text url",
			content: "Check https://example.com/page for details",
			want:    []string{"https://example.com/page"},
		},
		{
			name:    "html anchor",
			content: `<p>See <a href="https://example.com/docs">docs</a></p>`,
			want:    []string{"https://example.com/docs"},
		},
		{
			name:    "duplicate urls",
			content: "https://a.com and https://a.com",
			want:    []string{"https://a.com"},
		},
		{
			name:    "trailing punctuation",
			content: "Visit https://a.com/foo, then https://b.com/bar.",
			want:    []string{"https://a.com/foo", "https://b.com/bar"},
		},
		{
			name:    "www link gets https scheme",
			content: "go to www.example.com/x now",
			want:    []string{"https://www.example.com/x"},
		},
		{
			name:    "mailto and javascript rejected",
			content: `<a href="mailto:x@y.com">mail</a> <a href="javascript:alert(1)">x</a>`,
			want:    nil,
		},
		{
			name:    "stuck together urls split",
			content: "https://a.comhttps://b.com",
			want:    []string{"https://a.com", "https://b.com"},
		},
		{
			name:    "markdown link",
			content: "[text](https://a.com)",
			want:    []string{"https://a.com"},
		},
		{
			name:    "markdown url runs return each once",
			content: "[https://a.com](https://a.com)[https://b.com](https://b.com)",
			want:    []string{"https://a.com", "https://b.com"},
		},
		{
			name:    "balanced parentheses kept",
			content: "https://en.wikipedia.org/wiki/Foo_(bar)",
			want:    []string{"https://en.wikipedia.org/wiki/Foo_(bar)"},
		},
		{
			name:    "scheme and host lowercased",
			content: "HTTPS://Example.COM/Path",
			want:    []string{"https://example.com/Path"},
		},
		{
			name:    "bare domain not matched",
			content: "example.com is not a link",
			want:    nil,
		},
		{
			name:    "first occurrence order kept",
			content: "https://b.com https://a.com https://b.com",
			want:    []string{"https://b.com", "https://a.com"},
		},
		{
			name:    "caps at fifty urls",
			content: strings.Join(manyURLs, " "),
			want:    manyURLs[:50],
		},
		{
			name:    "url over length limit rejected",
			content: "https://example.com/" + strings.Repeat("a", 2048),
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := links.ExtractURLs(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d urls, got %d: %v", len(tt.want), len(got), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("url %d: expected %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}
