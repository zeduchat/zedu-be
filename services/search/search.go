package search

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

var SortkeyWords = []string{"recency", "relevance"}

func ValidateSortKey(sortby string) bool {
	sortby = strings.ToLower(sortby)
	return slices.Contains(SortkeyWords, sortby)
}

type SearchRequest struct {
	DB     *storage.Database
	Ctx    *gin.Context
	Logger *utility.Logger
	UserID string
	OrgID  string
	Query  string
	SortBy string
}

func Search(req SearchRequest) ([]utility.SearchQueryResult, postgresql.PaginationResponse, int, error) {

	searchQuery := models.NewSearchQueryFilterKeywords()
	queryArr := utility.CheckQueryStringContainKeyword(req.Query)
	if len(queryArr) >= 1 {
		searchQuery.ProcessQueryString(queryArr)
		searchQuery.Message = utility.ExtractWordsBeforeKeywords(req.Query)
	} else if queryArr == nil && req.Query != "" {
		searchQuery.Message = req.Query
	} else {
		return nil, postgresql.PaginationResponse{}, http.StatusBadRequest, errors.New("invalid search query, empty query provided")
	}

	if req.SortBy != "" {
		if !ValidateSortKey(req.SortBy) {
			return nil, postgresql.PaginationResponse{}, http.StatusBadRequest, errors.New("invalid sort key provided")
		}
		searchQuery.SortBy = req.SortBy
	}

	modelReq := models.SearchQueryRequest{
		DB:     req.DB,
		Ctx:    req.Ctx,
		Logger: req.Logger,
		UserID: req.UserID,
		OrgID:  req.OrgID,
		Opts:   searchQuery,
	}
	searchResult, paginationResponse, err := models.SearchQuery(modelReq)
	if err != nil {
		if err.Error() == "no search results found" || strings.Contains(err.Error(), "Organisation does not exist") {
			return nil, paginationResponse, http.StatusNotFound, err
		} else if err.Error() == "error fetching channels" || strings.Contains(err.Error(), "User does not exist in the organisation") {
			return nil, paginationResponse, http.StatusBadRequest, err
		}
		return nil, paginationResponse, http.StatusInternalServerError, err
	}
	return searchResult, paginationResponse, http.StatusOK, nil
}

type SearchChannelRequest struct {
	DB        *storage.Database
	Ctx       *gin.Context
	Logger    *utility.Logger
	UserID    string
	ChannelID string
	Query     string
	SortBy    string
}

func SearchChannel(req SearchChannelRequest) ([]utility.SearchQueryResult, postgresql.PaginationResponse, int, error) {

	searchQuery := models.NewSearchQueryFilterKeywords()
	queryArr := utility.CheckQueryStringContainKeyword(req.Query)
	if len(queryArr) >= 1 {
		searchQuery.ProcessQueryString(queryArr)
		searchQuery.Message = utility.ExtractWordsBeforeKeywords(req.Query)
	} else if queryArr == nil && req.Query != "" {
		searchQuery.Message = req.Query
	} else {
		return nil, postgresql.PaginationResponse{}, http.StatusBadRequest, errors.New("invalid search query, empty query provided")
	}

	if req.SortBy != "" {
		if !ValidateSortKey(req.SortBy) {
			return nil, postgresql.PaginationResponse{}, http.StatusBadRequest, errors.New("invalid sort key provided")
		}
		searchQuery.SortBy = req.SortBy
	}

	modelReq := models.SearchChannelQueryRequest{
		DB:        req.DB,
		Ctx:       req.Ctx,
		Logger:    req.Logger,
		UserID:    req.UserID,
		ChannelID: req.ChannelID,
		Opts:      searchQuery,
	}
	searchResult, paginationResponse, err := models.SearchChannelQuery(modelReq)
	if err != nil {
		if err.Error() == "no search results found" {
			return nil, paginationResponse, http.StatusNotFound, err
		} else if strings.Contains(err.Error(), "unauthorized") {
			return nil, paginationResponse, http.StatusForbidden, err
		}
		return nil, paginationResponse, http.StatusInternalServerError, err
	}
	return searchResult, paginationResponse, http.StatusOK, nil
}
