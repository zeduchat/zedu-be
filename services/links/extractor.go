package links

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

const (
	maxURLsPerMessage = 50
	maxURLLength      = 2048
)

var (
	hrefPattern = regexp.MustCompile(`(?i)<a\s[^>]*href\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	urlMarkers  = []string{"http://", "https://", "www."}
)

// ExtractURLs returns the normalized http(s) URLs found in a message body, in order of first appearance.
func ExtractURLs(content string) []string {
	if content == "" {
		return nil
	}

	candidates := make([]string, 0, 4)
	for _, match := range hrefPattern.FindAllStringSubmatch(content, -1) {
		for _, href := range match[1:] {
			if href != "" {
				candidates = append(candidates, href)
				break
			}
		}
	}
	candidates = append(candidates, scanPlainText(content)...)

	seen := make(map[string]bool, len(candidates))
	urls := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		normalized, ok := normalizeURL(candidate)
		if !ok || seen[normalized] {
			continue
		}
		seen[normalized] = true
		urls = append(urls, normalized)
		if len(urls) == maxURLsPerMessage {
			break
		}
	}
	return urls
}

func scanPlainText(content string) []string {
	lower := strings.ToLower(content)
	starts := markerPositions(lower)
	urls := make([]string, 0, len(starts))
	for i, start := range starts {
		end := len(content)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		urls = append(urls, trimURL(content[start:end]))
	}
	return urls
}

// shortcut: every www. counts as a URL start, so a www. inside another URL's query yields a second link; refine if users report it.
func markerPositions(lower string) []int {
	starts := make([]int, 0, 4)
	for _, marker := range urlMarkers {
		for offset := 0; ; {
			idx := strings.Index(lower[offset:], marker)
			if idx < 0 {
				break
			}
			starts = append(starts, offset+idx)
			offset += idx + len(marker)
		}
	}
	sort.Ints(starts)

	unique := starts[:0]
	for i, start := range starts {
		if i == 0 || start != starts[i-1] {
			unique = append(unique, start)
		}
	}
	return unique
}

func trimURL(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if i := strings.IndexAny(candidate, " \t\r\n<>\"'"); i >= 0 {
		candidate = candidate[:i]
	}
	candidate = strings.TrimRight(candidate, ".,;:!?*_`[]}>(")
	for strings.HasSuffix(candidate, ")") && strings.Count(candidate, ")") > strings.Count(candidate, "(") {
		candidate = strings.TrimSuffix(candidate, ")")
	}
	return candidate
}

func normalizeURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxURLLength {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(raw), "www.") {
		raw = "https://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Host == "" {
		return "", false
	}

	parsed.Scheme = scheme
	parsed.Host = strings.ToLower(parsed.Host)
	if parsed.Path == "/" && parsed.RawQuery == "" && parsed.Fragment == "" {
		parsed.Path = ""
	}

	normalized := parsed.String()
	if len(normalized) > maxURLLength {
		return "", false
	}
	return normalized, true
}
