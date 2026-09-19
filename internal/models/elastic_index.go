package models

import "github.com/hngprojects/telex_be/pkg/repository/storage/elastic"

func ResolveIndexPrefix(prefix string, appMode string) string {
	return elastic.ResolveIndexPrefix(prefix, appMode)
}

func InitIndexNames(prefix string, appMode string) {
	elastic.InitIndexNames(prefix, appMode)
	ThreadIndexName = elastic.ThreadIndexName
	MessageIndexName = elastic.MessageIndexName
}
