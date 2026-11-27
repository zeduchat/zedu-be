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

type deleteAgentUser struct {
	token string
	id    string
	org   string
}

func newDeleteAgentUser(t *testing.T, db *storage.Database, logger *utility.Logger, withOrg bool) deleteAgentUser {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	token := initialise(currUUID, t, r, authCtrl, false)
	u := deleteAgentUser{token: token, id: tst.GetUserIDFromToken(t, token, db)}
	if withOrg {
		u.org, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("Org %v", currUUID),
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			Description: "delete custom agent authz test",
			Type:        "type1",
			Location:    "wakanda",
			Country:     "wakanda",
		}, token)
	}
	return u
}

func TestDeleteCustomAgentAuthz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	owner := newDeleteAgentUser(t, db, logger, true)
	member := newDeleteAgentUser(t, db, logger, false)
	outsider := newDeleteAgentUser(t, db, logger, true)

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

	// The same custom agent is installed in both orgs, each with settings.
	agentID := utility.GenerateUUID()
	for _, u := range []deleteAgentUser{owner, outsider} {
		if err := db.Postgresql.Create(&models.OrganisationIntegrations{ID: utility.GenerateUUID(), OrgID: u.org, IntegrationID: agentID, OwnerID: u.id, IsActive: true}).Error; err != nil {
			t.Fatalf("insert org integration: %v", err)
		}
		if err := db.Postgresql.Create(&models.CustomIntegrationsSetting{ID: utility.GenerateUUID(), OrgID: u.org, IntegrationID: agentID, SettingEntry: "{\"settings\":{}}"}).Error; err != nil {
			t.Fatalf("insert custom settings: %v", err)
		}
	}
	onlyInOutsider := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.OrganisationIntegrations{ID: utility.GenerateUUID(), OrgID: outsider.org, IntegrationID: onlyInOutsider, OwnerID: outsider.id, IsActive: true}).Error; err != nil {
		t.Fatalf("insert org integration: %v", err)
	}

	ctrl := agents.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	g := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	g.DELETE("/:org_id/agents/:agent_id", ctrl.DeleteCustomAgentApp)

	do := func(orgID, agent, token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organisations/%s/agents/%s", orgID, agent), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	count := func(model any, orgID, agent string) int64 {
		var n int64
		db.Postgresql.Model(model).Where("org_id = ? AND integration_id = ?", orgID, agent).Count(&n)
		return n
	}

	if rr := do(owner.org, agentID, outsider.token); rr.Code != http.StatusForbidden {
		t.Errorf("outsider: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if rr := do(owner.org, agentID, member.token); rr.Code != http.StatusForbidden {
		t.Errorf("member who neither installed it nor can manage agents: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if count(&models.OrganisationIntegrations{}, owner.org, agentID) != 1 || count(&models.CustomIntegrationsSetting{}, owner.org, agentID) != 1 {
		t.Fatalf("denied caller deleted the owner's agent")
	}
	if rr := do(owner.org, onlyInOutsider, owner.token); rr.Code != http.StatusNotFound {
		t.Errorf("owner deleting an agent their org lacks: got %d want 404: %s", rr.Code, rr.Body.String())
	}
	if count(&models.OrganisationIntegrations{}, outsider.org, onlyInOutsider) != 1 {
		t.Fatalf("another org's install was deleted")
	}

	if rr := do(owner.org, agentID, owner.token); rr.Code != http.StatusOK {
		t.Fatalf("owner: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	if count(&models.OrganisationIntegrations{}, owner.org, agentID) != 0 || count(&models.CustomIntegrationsSetting{}, owner.org, agentID) != 0 {
		t.Errorf("owner's install or settings were not deleted")
	}
	if count(&models.OrganisationIntegrations{}, outsider.org, agentID) != 1 || count(&models.CustomIntegrationsSetting{}, outsider.org, agentID) != 1 {
		t.Errorf("deleting in one org removed the other org's install or settings")
	}
}
