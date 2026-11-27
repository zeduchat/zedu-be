package test_integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	svcAgents "github.com/hngprojects/telex_be/services/agents"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// CreateSystemAgent must store the row under the URL-derived id, so a second
// create for the same card URL hits the duplicate guard.
func TestCreateSystemAgentRejectsSameCardURL(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	// The card name changes between fetches, so only the id guard can catch it.
	fetch := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetch++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":               "dup-agent-" + utility.GenerateUUID(),
			"description":        "duplicate guard test",
			"url":                "https://agent.example.com",
			"skills":             []any{},
			"defaultInputModes":  []any{"text"},
			"defaultOutputModes": []any{"text"},
			"provider":           map[string]any{"organization": "example", "url": "https://example.com"},
		})
	}))
	defer srv.Close()

	jsonURL := srv.URL + "/agent-" + utility.GenerateUUID() + ".json"
	wantID, err := utility.GenerateUUIDFromString(jsonURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Postgresql.Where("integration_id = ?", wantID).Delete(&models.IntegrationSettings{})
		db.Postgresql.Where("json_url = ?", jsonURL).Delete(&models.Integrations{})
	})

	extReq := request.ExternalRequest{Logger: logger, Test: false}
	req := models.CustomIntegrationRequest{JSONUrl: jsonURL}

	resp, err := svcAgents.CreateSystemAgent(req, db.Postgresql, extReq, utility.GenerateUUID())
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if resp.ID != wantID {
		t.Fatalf("system agent stored under %s, want the URL-derived id %s", resp.ID, wantID)
	}

	_, err = svcAgents.CreateSystemAgent(req, db.Postgresql, extReq, utility.GenerateUUID())
	if err == nil || err.Error() != "organisation already has that agent" {
		t.Fatalf("second create: got %v, want the duplicate error", err)
	}

	var n int64
	db.Postgresql.Model(&models.Integrations{}).Where("json_url = ?", jsonURL).Count(&n)
	if n != 1 {
		t.Errorf("got %d integrations rows for the card URL, want 1", n)
	}
	if fetch != 1 {
		t.Errorf("card fetched %d times, want 1 (the duplicate guard runs before the fetch)", fetch)
	}
}
