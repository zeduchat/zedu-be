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

type sendBackUser struct {
	token string
	id    string
	org   string
}

func newSendBackUser(t *testing.T, db *storage.Database, logger *utility.Logger, withOrg bool) sendBackUser {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	token := initialise(currUUID, t, r, authCtrl, false)
	u := sendBackUser{token: token, id: tst.GetUserIDFromToken(t, token, db)}
	if withOrg {
		u.org, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("Org %v", currUUID),
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			Description: "change send-back status authz test",
			Type:        "type1",
			Location:    "wakanda",
			Country:     "wakanda",
		}, token)
	}
	return u
}

func TestChangeSendBackStatusAuthz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	owner := newSendBackUser(t, db, logger, true)
	member := newSendBackUser(t, db, logger, false)
	outsider := newSendBackUser(t, db, logger, true)

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
	if err := db.Postgresql.Create(&models.Integrations{ID: agentID, Name: "sendback authz " + agentID, OwnerID: owner.id, PreSharedKey: utility.GenerateUUID()}).Error; err != nil {
		t.Fatalf("insert integration: %v", err)
	}
	if err := db.Postgresql.Create(&models.OrganisationIntegrations{ID: utility.GenerateUUID(), OrgID: owner.org, IntegrationID: agentID, OwnerID: owner.id, IsActive: true}).Error; err != nil {
		t.Fatalf("insert org integration: %v", err)
	}
	channelA := utility.GenerateUUID()
	channelB := utility.GenerateUUID()
	for _, ch := range []models.Channels{
		{ID: channelA, Name: "sendback-a-" + channelA[:8], Description: "a", OrganisationID: owner.org, OwnerId: owner.id},
		{ID: channelB, Name: "sendback-b-" + channelB[:8], Description: "b", OrganisationID: outsider.org, OwnerId: outsider.id},
	} {
		if err := db.Postgresql.Create(&ch).Error; err != nil {
			t.Fatalf("insert channel: %v", err)
		}
	}
	for _, ch := range []string{channelA, channelB} {
		if err := db.Postgresql.Create(&models.OrganisationChannelsIntegrations{ID: utility.GenerateUUID(), OrgID: owner.org, IntegrationID: agentID, ChannelID: ch, IsActive: true, SendBack: true}).Error; err != nil {
			t.Fatalf("insert channel integration: %v", err)
		}
	}

	ctrl := agents.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	g := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	g.PATCH("/:org_id/channels/:channel_id/agents/change-sendback-status", ctrl.RequireAgentManager, ctrl.ChangeOrgChannelIntSendBackStatus)

	do := func(channelID, token string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(map[string]any{"status": false, "integration_id": agentID})
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/organisations/%s/channels/%s/agents/change-sendback-status", owner.org, channelID), &buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	sendBack := func(channelID string) bool {
		var oci models.OrganisationChannelsIntegrations
		if err := db.Postgresql.Where("org_id = ? AND integration_id = ? AND channel_id = ?", owner.org, agentID, channelID).First(&oci).Error; err != nil {
			t.Fatalf("load channel integration: %v", err)
		}
		return oci.SendBack
	}

	if rr := do(channelA, outsider.token); rr.Code != http.StatusForbidden {
		t.Errorf("outsider: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if rr := do(channelA, member.token); rr.Code != http.StatusForbidden {
		t.Errorf("member without can_manage_agents: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if !sendBack(channelA) {
		t.Fatalf("send_back changed by a denied caller")
	}
	if rr := do(channelB, owner.token); rr.Code != http.StatusBadRequest {
		t.Errorf("owner with another org's channel: got %d want 400: %s", rr.Code, rr.Body.String())
	}
	if !sendBack(channelB) {
		t.Fatalf("send_back changed for a channel outside the path org")
	}
	if rr := do(channelA, owner.token); rr.Code != http.StatusOK {
		t.Fatalf("owner: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	if sendBack(channelA) {
		t.Errorf("owner's send_back change was not saved")
	}
}
