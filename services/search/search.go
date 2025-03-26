package search

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

var SortkeyWords = []string{"recency", "relevance"}

func ValidateSortKey(sortby string) bool {
	sortby = strings.ToLower(sortby)
	return slices.Contains(SortkeyWords, sortby)
}

func Search(db *storage.Database, c *gin.Context, userId string,
	orgId string, query string, sortby string) ([]utility.SearchQueryResult, int, error) {

	searchQuery := models.NewSearchQueryFilterKeywords()
	queryArr := utility.CheckQueryStringContainKeyword(query)
	if queryArr != nil && len(queryArr) >= 1 {
		searchQuery.ProcessQueryString(queryArr)
		searchQuery.Message = utility.ExtractWordsBeforeKeywords(query)
	} else if queryArr == nil && query != "" {
		searchQuery.Message = query
	} else {
		return nil, http.StatusBadRequest, errors.New("invalid search query, empty query provided")
	}

	if sortby != "" {
		if !ValidateSortKey(sortby) {
			return nil, http.StatusBadRequest, errors.New("invalid sort key provided")
		}
		searchQuery.SortBy = sortby
	}

	searchResult, err := models.SearchQuery(db, c, searchQuery, userId, orgId)

	if err != nil {
		if err.Error() == "no search results found" || err.Error() == "Organisation does not exist" {
			return nil, http.StatusNotFound, err
		} else if err.Error() == "error fetching channels" || err.Error() == "User does not exist in this organisation" {
			return nil, http.StatusBadRequest, err
		}
		return nil, http.StatusInternalServerError, err
	}

	return searchResult, http.StatusOK, nil
}
