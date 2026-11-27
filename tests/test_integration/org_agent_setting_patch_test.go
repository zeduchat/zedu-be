package test_integration

import (
	"bytes"
	"encoding/json"
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

type orgSettingUser struct {
	token string
	id    string
	org   string
}

func newOrgSettingUser(t *testing.T, db *storage.Database, logger *utility.Logger, withOrg bool) orgSettingUser {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	token := initialise(currUUID, t, r, authCtrl, false)
	u := orgSettingUser{token: token, id: tst.GetUserIDFromToken(t, token, db)}
	if withOrg {
		u.org, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("Org %v", currUUID),
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			Description: "org agent setting patch test",
			Type:        "type1",
			Location:    "wakanda",
			Country:     "wakanda",
		}, token)
	}
	return u
}

func TestUpdateOrgAgentSetting(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	owner := newOrgSettingUser(t, db, logger, true)
	member := newOrgSettingUser(t, db, logger, false)
	outsider := newOrgSettingUser(t, db, logger, true)

	var userRole models.OrgRole
	if err := db.Postgresql.Where("name = ?", models.OrgRoleNameUser).First(&userRole).Error; err != nil {
		t.Fatalf("load User role: %v", err)
	}
	orgCtrl := organisation.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	addR := gin.Default()
	addR.POST("/api/v1/organisations/:org_id/users", middleware.Authorize(db.Postgresql), orgCtrl.AddMemberToOrganisation)
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(models.OrgUserCreateRequest{UserID: member.id, RoleID: userRole.ID})
	addReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/users", owner.org), &b)
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+owner.token)
	addRR := httptest.NewRecorder()
	addR.ServeHTTP(addRR, addReq)
	if addRR.Code != http.StatusOK && addRR.Code != http.StatusCreated {
		t.Fatalf("add member: %d %s", addRR.Code, addRR.Body.String())
	}

	agentID := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.Integrations{ID: agentID, Name: "org setting patch " + agentID, OwnerID: owner.id, PreSharedKey: utility.GenerateUUID()}).Error; err != nil {
		t.Fatalf("insert integration: %v", err)
	}
	if err := db.Postgresql.Create(&models.OrganisationIntegrations{ID: utility.GenerateUUID(), OrgID: owner.org, IntegrationID: agentID, OwnerID: owner.id, IsActive: true}).Error; err != nil {
		t.Fatalf("insert org integration: %v", err)
	}
	orgID := owner.org
	first := utility.GenerateUUID()
	second := utility.GenerateUUID()
	for _, s := range []models.IntegrationSettings{
		{ID: first, OrgID: &orgID, IntegrationID: agentID, FormFieldLabel: "first", FormFieldValue: "1"},
		{ID: second, OrgID: &orgID, IntegrationID: agentID, FormFieldLabel: "second", FormFieldValue: "2"},
	} {
		if err := db.Postgresql.Create(&s).Error; err != nil {
			t.Fatalf("insert setting: %v", err)
		}
	}

	ctrl := agents.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	g := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	g.PATCH("/:org_id/agents/:agent_id/settings/:setting_id", ctrl.RequireAgentManager, ctrl.UpdateAgentSetting)

	do := func(settingID, token string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(map[string]any{"form_field_value": "changed"})
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/organisations/%s/agents/%s/settings/%s", owner.org, agentID, settingID), &buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	value := func(id string) string {
		var s models.IntegrationSettings
		if err := db.Postgresql.Where("id = ?", id).First(&s).Error; err != nil {
			t.Fatalf("load setting: %v", err)
		}
		return s.FormFieldValue
	}

	if rr := do(second, outsider.token); rr.Code != http.StatusForbidden {
		t.Errorf("outsider: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if rr := do(second, member.token); rr.Code != http.StatusForbidden {
		t.Errorf("member without can_manage_agents: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if rr := do(utility.GenerateUUID(), owner.token); rr.Code != http.StatusBadRequest {
		t.Errorf("owner with unknown setting id: got %d want 400: %s", rr.Code, rr.Body.String())
	}
	if value(first) != "1" || value(second) != "2" {
		t.Fatalf("settings changed by a denied or invalid request")
	}

	// Before the fix this route always answered 400 "invalid channel agent id format".
	if rr := do(second, owner.token); rr.Code != http.StatusOK {
		t.Fatalf("owner: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	if value(second) != "changed" {
		t.Errorf("targeted setting was not updated: %q", value(second))
	}
	if value(first) != "1" {
		t.Errorf("a different setting was updated: %q", value(first))
	}
}
