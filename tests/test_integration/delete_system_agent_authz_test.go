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

type sysAgentUser struct {
	token string
	id    string
	org   string
}

func newSysAgentUser(t *testing.T, db *storage.Database, logger *utility.Logger, withOrg bool) sysAgentUser {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	token := initialise(currUUID, t, r, authCtrl, false)
	u := sysAgentUser{token: token, id: tst.GetUserIDFromToken(t, token, db)}
	if withOrg {
		u.org, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("Org %v", currUUID),
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			Description: "system agent delete authz test",
			Type:        "type1",
			Location:    "wakanda",
			Country:     "wakanda",
		}, token)
	}
	return u
}

func TestDeleteSystemAgentAuthz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	ownerA := newSysAgentUser(t, db, logger, true)
	ownerB := newSysAgentUser(t, db, logger, true)
	member := newSysAgentUser(t, db, logger, false)

	// Add the member to org A with the default "User" role (no can_manage_agents).
	var userRole models.OrgRole
	if err := db.Postgresql.Where("name = ?", models.OrgRoleNameUser).First(&userRole).Error; err != nil {
		t.Fatalf("load User role: %v", err)
	}
	orgCtrl := organisation.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	addR := gin.Default()
	addR.POST("/api/v1/organisations/:org_id/users", middleware.Authorize(db.Postgresql), orgCtrl.AddMemberToOrganisation)
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(models.OrgUserCreateRequest{UserID: member.id, RoleID: userRole.ID})
	addReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/users", ownerA.org), &b)
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+ownerA.token)
	addRR := httptest.NewRecorder()
	addR.ServeHTTP(addRR, addReq)
	if addRR.Code != http.StatusOK && addRR.Code != http.StatusCreated {
		t.Fatalf("add member: %d %s", addRR.Code, addRR.Body.String())
	}

	// One system agent, installed in org A and org B.
	agentID := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.Integrations{
		ID: agentID, Name: "sys-agent-" + agentID, IsSystem: true, PreSharedKey: utility.GenerateUUID(), OwnerID: utility.GenerateUUID(),
	}).Error; err != nil {
		t.Fatalf("insert integration: %v", err)
	}
	for _, org := range []string{ownerA.org, ownerB.org} {
		if err := db.Postgresql.Create(&models.OrganisationIntegrations{
			ID: utility.GenerateUUID(), OrgID: org, IntegrationID: agentID, OwnerID: ownerA.id, IsSystem: true,
		}).Error; err != nil {
			t.Fatalf("insert org integration: %v", err)
		}
	}
	t.Cleanup(func() {
		db.Postgresql.Where("integration_id = ?", agentID).Delete(&models.OrganisationIntegrations{})
		db.Postgresql.Where("id = ?", agentID).Delete(&models.Integrations{})
	})

	installed := func(org string) bool {
		var n int64
		db.Postgresql.Model(&models.OrganisationIntegrations{}).Where("org_id = ? AND integration_id = ?", org, agentID).Count(&n)
		return n > 0
	}
	agentRowExists := func() bool {
		var n int64
		db.Postgresql.Model(&models.Integrations{}).Where("id = ?", agentID).Count(&n)
		return n > 0
	}

	ctrl := agents.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	r.DELETE("/api/v1/organisations/:org_id/agents/system/:agent_id", middleware.Authorize(db.Postgresql), ctrl.DeleteAgentApp)
	del := func(org, agent, token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organisations/%s/agents/system/%s", org, agent), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	if rr := del(ownerA.org, agentID, ownerB.token); rr.Code != http.StatusForbidden {
		t.Fatalf("outsider: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if rr := del(ownerA.org, agentID, member.token); rr.Code != http.StatusForbidden {
		t.Fatalf("member without can_manage_agents: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if !installed(ownerA.org) || !installed(ownerB.org) || !agentRowExists() {
		t.Fatal("a denied request removed data")
	}

	if rr := del(ownerA.org, agentID, ownerA.token); rr.Code != http.StatusOK {
		t.Fatalf("owner of A: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	if installed(ownerA.org) {
		t.Error("org A's install was not removed")
	}
	if !installed(ownerB.org) {
		t.Error("org B's install was removed by org A's request")
	}
	if !agentRowExists() {
		t.Error("the global system agent row was deleted from an org route")
	}

	// Already uninstalled from A: 404, and B and the agent row stay.
	if rr := del(ownerA.org, agentID, ownerA.token); rr.Code != http.StatusNotFound {
		t.Fatalf("not installed: got %d want 404: %s", rr.Code, rr.Body.String())
	}
	if !installed(ownerB.org) || !agentRowExists() {
		t.Error("a not-installed delete removed data")
	}
}
