package utility

import (
	"strings"
)

func ExtractHashedFileName(generatedUrl string) string {
	urlParts := strings.Split(generatedUrl, "/")
	hashedFileName := urlParts[len(urlParts)-1]
	return hashedFileName
}
