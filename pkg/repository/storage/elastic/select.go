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

const (
	ThreadIndex  = "threads"
	MessageIndex = "messages"
	PersonIndex  = "persons"
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

func CheckExists(Client *elasticsearch.Client, indexName string, query map[string]interface{}) (bool, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return false, fmt.Errorf("failed to marshal query: %v", err)
	}

	ctx := context.Background()
	res, err := Client.Search(
		Client.Search.WithContext(ctx),
		Client.Search.WithIndex(indexName),
		Client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return false, fmt.Errorf("failed to execute search query: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return false, fmt.Errorf("error in search response: %s", res.String())
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return false, fmt.Errorf("failed to decode raw response: %v", err)
	}

	// Check if any hits exist.
	hitsObj, ok := raw["hits"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("unexpected response format: missing hits")
	}

	// Elasticsearch 7.x returns total as a map.
	if totalObj, ok := hitsObj["total"].(map[string]interface{}); ok {
		if value, ok := totalObj["value"].(float64); ok {
			return value > 0, nil
		}
		return false, fmt.Errorf("unexpected response format: total.value is missing")
	}

	// In case total is returned as a number (older versions)
	if total, ok := hitsObj["total"].(float64); ok {
		return total > 0, nil
	}

	return false, fmt.Errorf("unexpected response format for total")
}

func PerformSearchWithMultipleIndices(client *elasticsearch.Client, query map[string]interface{}, reciever interface{}) (map[string]interface{}, error) {

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("error encoding query: %w", err)
	}

	// Perform the search request
	ctx := context.Background()
	res, err := client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex(ThreadIndex, MessageIndex),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
		// client.Search.WithFrom((opts.PageNumber-1)*opts.PageSize),
		// client.Search.WithSize(opts.PageSize),
	)

	if err != nil {
		return nil, fmt.Errorf("error performing search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, fmt.Errorf("error parsing error response: %w", err)
		}
		return nil, fmt.Errorf("search error: %v", e)
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	return result, nil
}
