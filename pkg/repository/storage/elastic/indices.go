package elastic

import "strings"

var (
	ThreadIndexName  = "threads"
	MessageIndexName = "messages"
)

func ResolveIndexPrefix(prefix string, appMode string) string {
	cleanPrefix := strings.ToLower(strings.TrimSpace(prefix))
	cleanMode := strings.ToLower(strings.TrimSpace(appMode))

	ignoredEnvs := map[string]bool{
		"prod":       true,
		"production": true,
		"staging":    true,
		"stage":      true,
	}

	if ignoredEnvs[cleanPrefix] || ignoredEnvs[cleanMode] {
		return ""
	}

	return strings.TrimSpace(prefix)
}

func InitIndexNames(prefix string, appMode string) {
	resolvedPrefix := ResolveIndexPrefix(prefix, appMode)
	if resolvedPrefix == "" {
		ThreadIndexName = "threads"
		MessageIndexName = "messages"
	} else {
		ThreadIndexName = resolvedPrefix + "threads"
		MessageIndexName = resolvedPrefix + "messages"
	}
}
