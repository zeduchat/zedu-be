package test_integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/agents"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// newSlashCmdOwner signs a user up and creates an org they own.
func newSlashCmdOwner(t *testing.T, db *storage.Database, logger *utility.Logger) (token, orgID string) {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	token = initialise(currUUID, t, r, authCtrl, false)
	orgID, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("Org %v", currUUID),
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		Description: "slash command delete test",
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}, token)
	return token, orgID
}

func TestDeleteAgentSlashCommandDeletesTheNamedCommand(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	ownerToken, orgID := newSlashCmdOwner(t, db, logger)
	outsiderToken, _ := newSlashCmdOwner(t, db, logger)

	agentID := utility.GenerateUUID()
	oneID, twoID := utility.GenerateUUID(), utility.GenerateUUID()
	for _, sc := range []models.SlashCommand{
		{ID: oneID, OrgID: &orgID, IntegrationID: &agentID, Command: "/one", ProcessingURL: "https://example.com/one"},
		{ID: twoID, OrgID: &orgID, IntegrationID: &agentID, Command: "/two", ProcessingURL: "https://example.com/two"},
	} {
		if err := db.Postgresql.Create(&sc).Error; err != nil {
			t.Fatalf("insert slash command: %v", err)
		}
	}
	t.Cleanup(func() {
		db.Postgresql.Where("integration_id = ?", agentID).Delete(&models.SlashCommand{})
	})
	exists := func(id string) bool {
		var n int64
		db.Postgresql.Model(&models.SlashCommand{}).Where("id = ?", id).Count(&n)
		return n > 0
	}

	ctrl := agents.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	r.DELETE("/api/v1/organisations/:org_id/agents/:agent_id/slash-commands/:command_id", middleware.Authorize(db.Postgresql), ctrl.DeleteAgentSlashCommand)
	del := func(cmdID, token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organisations/%s/agents/%s/slash-commands/%s", orgID, agentID, cmdID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	if rr := del(twoID, outsiderToken); rr.Code != http.StatusForbidden {
		t.Fatalf("outsider: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if !exists(oneID) || !exists(twoID) {
		t.Fatal("outsider's request deleted a command")
	}

	if rr := del(twoID, ownerToken); rr.Code != http.StatusOK {
		t.Fatalf("owner delete /two: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	if exists(twoID) {
		t.Error("/two was not deleted")
	}
	if !exists(oneID) {
		t.Error("/one was deleted instead of /two")
	}

	// A command id that is not this agent's command deletes nothing.
	if rr := del(utility.GenerateUUID(), ownerToken); rr.Code == http.StatusOK {
		t.Fatalf("unknown command id: got 200: %s", rr.Body.String())
	}
	if !exists(oneID) {
		t.Error("an unknown command id deleted /one")
	}
}
