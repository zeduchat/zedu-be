package test_integration

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/router"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// The org-scoped PATCH edited the global integrations row; it is removed and
// system agents are only changed through the back office.
func TestOrgSystemAgentPatchRouteRemoved(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	r := gin.New()
	router.Agents(r, "/api/v1", validator.New(), db, logger)

	registered := map[string]bool{}
	for _, rt := range r.Routes() {
		registered[rt.Method+" "+rt.Path] = true
	}
	if registered["PATCH /api/v1/organisations/:org_id/agents/system/:agent_id"] {
		t.Fatal("PATCH /organisations/:org_id/agents/system/:agent_id is still registered")
	}
	if !registered["PUT /api/v1/backoffice/agents/:agent_id"] {
		t.Fatal("back-office PUT /backoffice/agents/:agent_id is missing")
	}

	agentID := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.Integrations{
		ID: agentID, Name: "sys-agent-" + agentID, IsSystem: true, PreSharedKey: utility.GenerateUUID(), OwnerID: utility.GenerateUUID(),
	}).Error; err != nil {
		t.Fatalf("insert integration: %v", err)
	}
	t.Cleanup(func() { db.Postgresql.Where("id = ?", agentID).Delete(&models.Integrations{}) })

	authCtrl := auth.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	token := initialise(utility.GenerateUUID(), t, gin.Default(), authCtrl, false)

	req, _ := http.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/v1/organisations/%s/agents/system/%s", utility.GenerateUUID(), agentID),
		bytes.NewBufferString(`{"name":"Hacked"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("PATCH by a normal user: got %d want 404: %s", rr.Code, rr.Body.String())
	}

	var agent models.Integrations
	if err := db.Postgresql.Where("id = ?", agentID).First(&agent).Error; err != nil {
		t.Fatalf("load integration: %v", err)
	}
	if agent.Name != "sys-agent-"+agentID {
		t.Errorf("system agent was renamed to %q", agent.Name)
	}
}
