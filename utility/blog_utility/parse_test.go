package blog_utility

import (
	"strings"
	"testing"
)

func TestConvertMarkdownToHTMLSkipsRawHTML(t *testing.T) {
	markdown := "# Safe heading\n\n<script>alert('xss')</script>\n\n<iframe src=\"https://evil.example\"></iframe>\n\n**Safe text**"

	html := ConvertMarkdownToHTML(markdown)

	for _, unsafe := range []string{"<script", "alert('xss')", "<iframe", "evil.example"} {
		if strings.Contains(html, unsafe) {
			t.Fatalf("generated HTML contains unsafe raw markup %q: %s", unsafe, html)
		}
	}

	for _, expected := range []string{"<h1>Safe heading</h1>", "<strong>Safe text</strong>"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("generated HTML is missing expected Markdown output %q: %s", expected, html)
		}
	}
}
