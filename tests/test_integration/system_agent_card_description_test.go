package test_integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	svcAgents "github.com/hngprojects/telex_be/services/agents"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func cardWithoutDescription(name string) map[string]any {
	return map[string]any{
		"name":               name,
		"url":                "https://agent.example.com",
		"skills":             []any{},
		"defaultInputModes":  []any{"text"},
		"defaultOutputModes": []any{"text"},
		"provider":           map[string]any{"organization": "example", "url": "https://example.com"},
	}
}

func TestValidateAgentDataRequiresStringDescription(t *testing.T) {
	cases := map[string]any{
		"missing": nil,
		"number":  42,
		"object":  map[string]any{"x": 1},
		"array":   []any{"x"},
	}
	for name, desc := range cases {
		t.Run(name, func(t *testing.T) {
			card := cardWithoutDescription("card")
			if desc != nil {
				card["description"] = desc
			}
			err := models.ValidateAgentData(card)
			if err == nil || !strings.Contains(err.Error(), "description") {
				t.Fatalf("want a description validation error, got %v", err)
			}
		})
	}

	card := cardWithoutDescription("card")
	card["description"] = "an agent"
	if err := models.ValidateAgentData(card); err != nil {
		t.Fatalf("valid card rejected: %v", err)
	}
}

// CreateSystemAgent used to panic on data_r["description"].(string) for a card
// without a description. It must now return a validation error and store nothing.
func TestCreateSystemAgentRejectsCardWithoutDescription(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	name := "no-desc-agent-" + utility.GenerateUUID()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cardWithoutDescription(name))
	}))
	defer srv.Close()
	t.Cleanup(func() { db.Postgresql.Where("name = ?", name).Delete(&models.Integrations{}) })

	extReq := request.ExternalRequest{Logger: logger, Test: false}
	req := models.CustomIntegrationRequest{JSONUrl: srv.URL + "/agent-" + utility.GenerateUUID() + ".json"}

	var err error
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("CreateSystemAgent panicked: %v", p)
			}
		}()
		_, err = svcAgents.CreateSystemAgent(req, db.Postgresql, extReq, utility.GenerateUUID())
	}()
	if err == nil || !strings.Contains(err.Error(), "description") {
		t.Fatalf("want a description validation error, got %v", err)
	}

	var n int64
	db.Postgresql.Model(&models.Integrations{}).Where("name = ?", name).Count(&n)
	if n != 0 {
		t.Errorf("an invalid card created %d integrations rows", n)
	}
}
