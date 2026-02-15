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

func SelectWithPagination(Client *elasticsearch.Client, indexName []string, query map[string]any, reciever *any, c *gin.Context) (*PaginationResponse, error) {

	pag := GetPagination(c)

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %v", err)
	}

	ctx := context.Background()

	res, err := Client.Search(
		Client.Search.WithContext(ctx),
		Client.Search.WithIndex(indexName...),
		Client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error in search response: %s", res.String())
	}

	var raw map[string]any

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

func SelectAll(Client *elasticsearch.Client, indexName string, query map[string]any, reciever *any) error {

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

	var raw map[string]any

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return fmt.Errorf("failed to decode raw response: %v", err)
	}

	*reciever = raw

	return nil
}

func SelectByID(Client *elasticsearch.Client, indexName string, docID string, reciever *any) error {

	res, err := Client.Get(indexName, docID)

	if err != nil {
		return fmt.Errorf("failed to execute search query: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error in search response: %s", res.String())
	}

	var raw map[string]any

	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return fmt.Errorf("failed to decode raw response: %v", err)
	}

	*reciever = raw["_source"]

	return nil
}

func CheckExists(Client *elasticsearch.Client, indexName string, query map[string]any) (bool, error) {
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

	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return false, fmt.Errorf("failed to decode raw response: %v", err)
	}

	// Check if any hits exist.
	hitsObj, ok := raw["hits"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("unexpected response format: missing hits")
	}

	// Elasticsearch 7.x returns total as a map.
	if totalObj, ok := hitsObj["total"].(map[string]any); ok {
		if value, ok := totalObj["value"].(float64); ok {
			return value > 0, nil
		}
		return false, fmt.Errorf("unexpected response format: total.value is missing")
	}

	if total, ok := hitsObj["total"].(float64); ok {
		return total > 0, nil
	}

	return false, fmt.Errorf("unexpected response format for total")
}

func PerformSearchWithMultipleIndices(client *elasticsearch.Client, query map[string]any) (map[string]any, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("error encoding query: %w", err)
	}

	ctx := context.Background()
	res, err := client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex(ThreadIndex, MessageIndex),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
	)

	if err != nil {
		return nil, fmt.Errorf("error performing search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]any
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, fmt.Errorf("error parsing error response: %w", err)
		}
		return nil, fmt.Errorf("Elasticsearch error: %v", e)
	}

	var result map[string]any
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	hits, ok := result["hits"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response structure, 'hits' field missing")
	}

	hitsArray, ok := hits["hits"].([]any)
	if !ok || len(hitsArray) == 0 {
		return nil, fmt.Errorf("no search results found")
	}

	return result, nil
}

func PerformSearchWithMultipleIndicesPagination[T any](client *elasticsearch.Client, query map[string]any, inter T, result *[]T, c *gin.Context) (*PaginationResponse, error) {
	var ESResponse struct {
		Took     int  `json:"took"`
		TimedOut bool `json:"timed_out"`
		Shards   any  `json:"_shards"`
		Hits     struct {
			Total struct {
				Value    int    `json:"value"`
				Relation string `json:"relation"`
			} `json:"total"`
			MaxScore *float64 `json:"max_score,omitempty"`
			Hits     []struct {
				Index  string          `json:"_index,omitempty"`
				Id     string          `json:"_id,omitempty"`
				Score  float64         `json:"_score,omitempty"`
				Source json.RawMessage `json:"_source"`
				Sort   []any           `json:"sort,omitempty"`
			} `json:"hits"`
		} `json:"hits"`
	}
	pag := GetPagination(c)

	query["from"] = (pag.Page - 1) * pag.Limit
	query["size"] = pag.Limit

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("error encoding query: %w", err)
	}

	ctx := context.Background()
	res, err := client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex(ThreadIndex, MessageIndex),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
	)

	if err != nil {
		return nil, fmt.Errorf("error performing search: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("error in search response: %s", res.String())
	}

	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&ESResponse); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	rawJSON, _ := json.Marshal(raw)
	if err := json.Unmarshal(rawJSON, &ESResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %v", err)
	}

	hits := ESResponse.Hits.Hits
	for _, r := range hits {
		b, err := r.Source.MarshalJSON()
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, &inter)
		if err != nil {
			return nil, err
		}
		*result = append(*result, inter)
	}
	totalPages := int(math.Ceil(float64(ESResponse.Hits.Total.Value) / float64(pag.Limit)))

	return &PaginationResponse{
		PageCount:       int(ESResponse.Hits.Total.Value),
		CurrentPage:     pag.Page,
		TotalPagesCount: totalPages,
	}, nil
}
