package search

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Search(db *storage.Database, c *gin.Context, userId string, query string) ([]utility.SearchQueryResult, int, error) {
	searchQuery := models.NewSearchQueryFilterKeywords()
	queryArr := utility.CheckQueryStringContainKeyword(query)
	if queryArr != nil && len(queryArr) >= 1 {
		searchQuery.ProcessQueryString(queryArr)
		searchQuery.Message = utility.ExtractWordsBeforeKeywords(query)
	} else if queryArr == nil && query != "" {
		searchQuery.Message = query
	} else {
		fmt.Println(query, queryArr)
		return nil, http.StatusBadRequest, errors.New("invalid search query, empty query provided")
	}

	searchResult, err := models.SearchQuery(db, c, searchQuery, userId)

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return searchResult, http.StatusOK, nil
}
