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

type chanSettingUser struct {
	token string
	id    string
	org   string
}

func newChanSettingUser(t *testing.T, db *storage.Database, logger *utility.Logger, withOrg bool) chanSettingUser {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	token := initialise(currUUID, t, r, authCtrl, false)
	u := chanSettingUser{token: token, id: tst.GetUserIDFromToken(t, token, db)}
	if withOrg {
		u.org, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("Org %v", currUUID),
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			Description: "channel agent settings authz test",
			Type:        "type1",
			Location:    "wakanda",
			Country:     "wakanda",
		}, token)
	}
	return u
}

func TestChannelAgentSettingsAuthz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	owner := newChanSettingUser(t, db, logger, true)
	member := newChanSettingUser(t, db, logger, false)
	outsider := newChanSettingUser(t, db, logger, true)

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
	archivedID := utility.GenerateUUID()
	for _, id := range []string{agentID, archivedID} {
		if err := db.Postgresql.Create(&models.Integrations{ID: id, Name: "channel settings authz " + id, OwnerID: owner.id, PreSharedKey: utility.GenerateUUID()}).Error; err != nil {
			t.Fatalf("insert integration: %v", err)
		}
	}
	if err := db.Postgresql.Create(&models.OrganisationIntegrations{ID: utility.GenerateUUID(), OrgID: owner.org, IntegrationID: agentID, OwnerID: owner.id, IsActive: true}).Error; err != nil {
		t.Fatalf("insert org integration: %v", err)
	}
	if err := db.Postgresql.Create(&models.OrganisationIntegrations{ID: utility.GenerateUUID(), OrgID: owner.org, IntegrationID: archivedID, OwnerID: owner.id, IsActive: true, IsArchived: true}).Error; err != nil {
		t.Fatalf("insert archived org integration: %v", err)
	}
	channelA := utility.GenerateUUID()
	channelB := utility.GenerateUUID()
	for _, ch := range []models.Channels{
		{ID: channelA, Name: "chset-a-" + channelA[:8], Description: "a", OrganisationID: owner.org, OwnerId: owner.id},
		{ID: channelB, Name: "chset-b-" + channelB[:8], Description: "b", OrganisationID: outsider.org, OwnerId: outsider.id},
	} {
		if err := db.Postgresql.Create(&ch).Error; err != nil {
			t.Fatalf("insert channel: %v", err)
		}
	}
	// A row pairing the owner's org with the outsider's channel, as a bad write could leave.
	if err := db.Postgresql.Create(&models.ChannelIntegrationSettings{ID: utility.GenerateUUID(), OrgID: owner.org, IntegrationID: agentID, ChannelID: channelB, FormFieldLabel: "stray", FormFieldValue: "x"}).Error; err != nil {
		t.Fatalf("insert stray setting: %v", err)
	}

	ctrl := agents.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	g := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	g.POST("/:org_id/agents/:agent_id/channels/:channel_id/settings", ctrl.RequireAgentManager, ctrl.AddChannelAgentSetting)
	g.GET("/:org_id/agents/:agent_id/channels/:channel_id/settings", middleware.RequireOrgMember(db.Postgresql), ctrl.GetChannelAgentSetting)
	g.PATCH("/:org_id/agents/:agent_id/channels/:channel_id/settings/:setting_id", ctrl.RequireAgentManager, ctrl.UpdateChannelAgentSetting)
	g.DELETE("/:org_id/agents/:agent_id/channels/:channel_id/settings/:setting_id", ctrl.RequireAgentManager, ctrl.DeleteChannelAgentSetting)

	do := func(method, p, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			json.NewEncoder(&buf).Encode(body)
		}
		req, _ := http.NewRequest(method, p, &buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	base := func(agent, channel string) string {
		return fmt.Sprintf("/api/v1/organisations/%s/agents/%s/channels/%s/settings", owner.org, agent, channel)
	}
	add := func(label string) map[string]any {
		return map[string]any{"form_field_label": label, "form_field_value": "v"}
	}
	settingIDs := func() map[string]string {
		var rows []models.ChannelIntegrationSettings
		db.Postgresql.Where("org_id = ? AND integration_id = ? AND channel_id = ?", owner.org, agentID, channelA).Find(&rows)
		out := map[string]string{}
		for _, row := range rows {
			out[row.FormFieldLabel] = row.ID
		}
		return out
	}

	cases := []struct {
		name   string
		method string
		path   string
		token  string
		body   any
		want   int
	}{
		{"outsider get", http.MethodGet, base(agentID, channelA), outsider.token, nil, http.StatusForbidden},
		{"outsider add", http.MethodPost, base(agentID, channelA), outsider.token, add("evil"), http.StatusForbidden},
		{"member add", http.MethodPost, base(agentID, channelA), member.token, add("evil"), http.StatusForbidden},
		{"owner add first", http.MethodPost, base(agentID, channelA), owner.token, add("first"), http.StatusOK},
		{"owner add second", http.MethodPost, base(agentID, channelA), owner.token, add("second"), http.StatusOK},
		{"owner add for archived install", http.MethodPost, base(archivedID, channelA), owner.token, add("archived"), http.StatusBadRequest},
		{"member get", http.MethodGet, base(agentID, channelA), member.token, nil, http.StatusOK},
		{"owner get with another org's channel", http.MethodGet, base(agentID, channelB), owner.token, nil, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rr := do(tc.method, tc.path, tc.token, tc.body); rr.Code != tc.want {
				t.Fatalf("got %d want %d: %s", rr.Code, tc.want, rr.Body.String())
			}
		})
	}

	ids := settingIDs()
	if len(ids) != 2 || ids["first"] == "" || ids["second"] == "" {
		t.Fatalf("expected the owner's two settings only, got %v", ids)
	}

	if rr := do(http.MethodDelete, base(agentID, channelA)+"/"+ids["second"], outsider.token, nil); rr.Code != http.StatusForbidden {
		t.Errorf("outsider delete: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if rr := do(http.MethodPatch, base(agentID, channelA)+"/"+ids["second"], member.token, map[string]any{"form_field_value": "evil"}); rr.Code != http.StatusForbidden {
		t.Errorf("member patch: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if rr := do(http.MethodDelete, base(agentID, channelA)+"/"+ids["second"], owner.token, nil); rr.Code != http.StatusOK {
		t.Fatalf("owner delete: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	after := settingIDs()
	if after["first"] != ids["first"] {
		t.Errorf("deleting the second setting removed the first one: %v", after)
	}
	if after["second"] != "" {
		t.Errorf("the requested setting was not deleted: %v", after)
	}
}
