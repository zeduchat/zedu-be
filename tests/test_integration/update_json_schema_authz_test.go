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

type jsonSchemaUser struct {
	token string
	id    string
	org   string
}

func newJSONSchemaUser(t *testing.T, db *storage.Database, logger *utility.Logger, withOrg bool) jsonSchemaUser {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	token := initialise(currUUID, t, r, authCtrl, false)
	u := jsonSchemaUser{token: token, id: tst.GetUserIDFromToken(t, token, db)}
	if withOrg {
		u.org, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("Org %v", currUUID),
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			Description: "update json schema authz test",
			Type:        "type1",
			Location:    "wakanda",
			Country:     "wakanda",
		}, token)
	}
	return u
}

func TestUpdateJSONSchemaAuthz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	owner := newJSONSchemaUser(t, db, logger, true)
	member := newJSONSchemaUser(t, db, logger, false)
	outsider := newJSONSchemaUser(t, db, logger, true)

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
	if err := db.Postgresql.Create(&models.OrganisationIntegrations{
		ID: utility.GenerateUUID(), OrgID: owner.org, IntegrationID: agentID, OwnerID: owner.id,
		JSONSchema: models.JSONB{"version": "original"},
	}).Error; err != nil {
		t.Fatalf("insert org integration: %v", err)
	}

	ctrl := agents.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	g := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	g.PATCH("/:org_id/agents/change_status", middleware.RequireOrgMember(db.Postgresql), ctrl.ChangeAgentStatus)
	g.PATCH("/:org_id/agents/:agent_id/updatejson", ctrl.RequireAgentManager, ctrl.UpdateJSONSchema)

	do := func(method, p, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(body)
		req, _ := http.NewRequest(method, p, &buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	stored := func() any {
		var oi models.OrganisationIntegrations
		if err := db.Postgresql.Where("org_id = ? AND integration_id = ?", owner.org, agentID).First(&oi).Error; err != nil {
			t.Fatalf("load org integration: %v", err)
		}
		return oi.JSONSchema["version"]
	}

	path := fmt.Sprintf("/api/v1/organisations/%s/agents/%s/updatejson", owner.org, agentID)
	body := func(v string) any { return map[string]any{"json_schema": map[string]any{"version": v}} }

	if rr := do(http.MethodPatch, path, outsider.token, body("evil")); rr.Code != http.StatusForbidden {
		t.Errorf("outsider updatejson: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if rr := do(http.MethodPatch, path, member.token, body("evil")); rr.Code != http.StatusForbidden {
		t.Errorf("member without can_manage_agents updatejson: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if got := stored(); got != "original" {
		t.Fatalf("schema changed by a denied caller: %v", got)
	}

	// change_status also writes json_schema; outsiders must not reach it.
	statusPath := fmt.Sprintf("/api/v1/organisations/%s/agents/change_status", owner.org)
	if rr := do(http.MethodPatch, statusPath, outsider.token, map[string]any{"status": true, "integration_id": agentID, "json_schema": map[string]any{"version": "evil"}}); rr.Code != http.StatusForbidden {
		t.Errorf("outsider change_status: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if got := stored(); got != "original" {
		t.Fatalf("schema changed by outsider change_status: %v", got)
	}

	if rr := do(http.MethodPatch, path, owner.token, body("changed")); rr.Code != http.StatusOK {
		t.Fatalf("owner updatejson: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	if got := stored(); got != "changed" {
		t.Errorf("owner update not saved: %v", got)
	}
}
