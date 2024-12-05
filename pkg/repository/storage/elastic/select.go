package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
)

var (
	defaultPage  = 1
	defaultLimit = 20
)

type Pagination struct {
	Page  int
	Limit int
}
type PaginationResponse struct {
	CurrentPage     int `json:"current_page"`
	PageCount       int `json:"page_count"`
	TotalPagesCount int `json:"total_pages_count"`
	SummaryCount    int `json:"summary_count,omitempty"`
}

func GetPagination(c *gin.Context) Pagination {
	var (
		page  *int
		limit *int
	)
	if c.Query("page") != "" {
		pageInt, err := strconv.Atoi(c.Query("page"))
		if err == nil {
			page = &pageInt
		}
	}
	if c.Query("limit") != "" {
		limitInt, err := strconv.Atoi(c.Query("limit"))
		if err == nil {
			limit = &limitInt
		}
	}

	if page != nil && limit != nil {
		return Pagination{Page: *page, Limit: *limit}
	} else if page == nil && limit != nil {
		return Pagination{Page: defaultPage, Limit: *limit}
	} else if page != nil && limit == nil {
		return Pagination{Page: *page, Limit: defaultLimit}
	} else {
		return Pagination{Page: defaultPage, Limit: defaultLimit}
	}
}

func SelectWithPagination(Client *elasticsearch.Client, indexName string, query map[string]interface{}, reciever *interface{}, c *gin.Context) (*PaginationResponse, error) {

	pag := GetPagination(c)

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %v", err)
	}

	ctx := context.Background()

	res, err := Client.Search(
		Client.Search.WithContext(ctx),
		Client.Search.WithIndex(indexName),
		Client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error in search response: %s", res.String())
	}

	var raw map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw response: %v", err)
	}

	*reciever = raw

	var rawResult struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []json.RawMessage `json:"hits"`
		} `json:"hits"`
	}

	rawJSON, _ := json.MarshalIndent(raw, "", "  ")

	if err := json.Unmarshal(rawJSON, &rawResult); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %v", err)

	}

	totalPages := int(math.Ceil(float64(rawResult.Hits.Total.Value) / float64(pag.Limit)))

	return &PaginationResponse{
		PageCount:       int(rawResult.Hits.Total.Value),
		CurrentPage:     pag.Page,
		TotalPagesCount: totalPages,
	}, nil

}

func SelectAll(Client *elasticsearch.Client, indexName string, query map[string]interface{}, reciever *interface{}) error {

	body, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("failed to marshal query: %v", err)
	}

	ctx := context.Background()

	res, err := Client.Search(
		Client.Search.WithContext(ctx),
		Client.Search.WithIndex(indexName),
		Client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return fmt.Errorf("failed to execute search query: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error in search response: %s", res.String())
	}

	var raw map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return fmt.Errorf("failed to decode raw response: %v", err)
	}

	*reciever = raw

	return nil
}

func SelectByID(Client *elasticsearch.Client, indexName string, docID string, reciever *interface{}) error {

	res, err := Client.Get(indexName, docID)

	if err != nil {
		return fmt.Errorf("failed to execute search query: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error in search response: %s", res.String())
	}

	var raw map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return fmt.Errorf("failed to decode raw response: %v", err)
	}

	*reciever = raw["_source"]

	return nil
}
