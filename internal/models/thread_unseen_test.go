package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestParseUniqueThreadIDs(t *testing.T) {
	var data any
	raw := `{"aggregations":{"unique_thread_ids":{"buckets":[{"key":"t1","doc_count":3},{"key":"","doc_count":1},{"key":"t2","doc_count":1}]}}}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}

	ids, err := parseUniqueThreadIDs(data)
	if err != nil {
		t.Fatalf("parseUniqueThreadIDs returned error: %v", err)
	}
	if want := []string{"t1", "t2"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("got %v, want %v", ids, want)
	}
}

func TestParseUniqueThreadIDsWithoutAggregations(t *testing.T) {
	ids, err := parseUniqueThreadIDs(map[string]any{"hits": map[string]any{}})
	if err != nil {
		t.Fatalf("parseUniqueThreadIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected no ids, got %v", ids)
	}
}

// The badge count and the Threads list must use the same visibility rules,
// scoped to the user's current channels.
func TestUserThreadsVisibilityQueryScopesToUserAndChannels(t *testing.T) {
	query := userThreadsVisibilityQuery("user-1", []string{"chan-a", "dm-b"})

	raw, err := json.Marshal(query)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)

	for _, want := range []string{
		`"channels_id":["chan-a","dm-b"]`,
		`"user_id":"user-1"`,
		`"mentions.id":"user-1"`,
		`"messages.user_id":"user-1"`,
		`"minimum_should_match":1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("visibility query is missing %s\nquery: %s", want, body)
		}
	}
}
