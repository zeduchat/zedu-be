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
	svcAgents "github.com/hngprojects/telex_be/services/agents"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

type settingsUser struct {
	token string
	id    string
	org   string
}

// newSettingsUser signs a user up on a fresh engine (the tst helpers register
// routes, so each user needs its own) and, when withOrg is set, creates an org.
func newSettingsUser(t *testing.T, db *storage.Database, logger *utility.Logger, withOrg bool) settingsUser {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	token := initialise(currUUID, t, r, authCtrl, false)
	u := settingsUser{token: token, id: tst.GetUserIDFromToken(t, token, db)}
	if withOrg {
		u.org, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("Org %v", currUUID),
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			Description: "agent settings authz test",
			Type:        "type1",
			Location:    "wakanda",
			Country:     "wakanda",
		}, token)
	}
	return u
}

func storedAgentAPIKey(t *testing.T, db *storage.Database, orgID, agentID string) any {
	var ucis models.CustomIntegrationsSetting
	if err := db.Postgresql.Where("org_id = ? AND integration_id = ?", orgID, agentID).First(&ucis).Error; err != nil {
		t.Fatalf("load settings: %v", err)
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(ucis.SettingEntry), &entry); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	creds, _ := entry["auth_credentials"].(map[string]any)
	return creds["agent_api_key"]
}

func TestCustomAgentSettingsAuthz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	owner := newSettingsUser(t, db, logger, true)
	member := newSettingsUser(t, db, logger, false)
	outsider := newSettingsUser(t, db, logger, true)

	// Add the member to the owner's org with the default "User" role.
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

	// The owner's custom agent and its stored settings.
	agentID := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.OrganisationIntegrations{ID: utility.GenerateUUID(), OrgID: owner.org, IntegrationID: agentID, OwnerID: owner.id}).Error; err != nil {
		t.Fatalf("insert org integration: %v", err)
	}
	if err := db.Postgresql.Create(&models.CustomIntegrationsSetting{
		ID: utility.GenerateUUID(), OrgID: owner.org, IntegrationID: agentID,
		SettingEntry: `{"settings":{"mode":"original"},"auth_credentials":{"agent_api_key":"k1","integration_auth_credentials":"b3JpZ2luYWw="}}`,
	}).Error; err != nil {
		t.Fatalf("insert settings: %v", err)
	}

	ctrl := agents.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	g := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	g.GET("/:org_id/agents/:agent_id/settings", middleware.RequireOrgMember(db.Postgresql), ctrl.GetCustomAgentSettings)
	g.GET("/:org_id/agents/:agent_id/status", middleware.RequireOrgMember(db.Postgresql), ctrl.GetCustomAgentStatus)
	g.PUT("/:org_id/agents/:agent_id/settings", middleware.RequireOrgMember(db.Postgresql), ctrl.UpdateCustomAgentSettings)

	path := fmt.Sprintf("/api/v1/organisations/%s/agents/%s/settings", owner.org, agentID)
	statusPath := fmt.Sprintf("/api/v1/organisations/%s/agents/%s/status", owner.org, agentID)
	put := func(api string) map[string]any {
		return map[string]any{"setting_entry": map[string]any{
			"settings":         map[string]any{"mode": "changed"},
			"auth_credentials": map[string]any{"agent_api_key": api, "integration_auth_credentials": "Y2hhbmdlZA=="},
		}}
	}
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

	cases := []struct {
		name   string
		method string
		path   string
		token  string
		body   any
		want   int
	}{
		{"outsider GET settings", http.MethodGet, path, outsider.token, nil, http.StatusForbidden},
		{"outsider GET status", http.MethodGet, statusPath, outsider.token, nil, http.StatusForbidden},
		{"outsider PUT settings", http.MethodPut, path, outsider.token, put("evil"), http.StatusForbidden},
		{"member GET settings", http.MethodGet, path, member.token, nil, http.StatusOK},
		{"member PUT settings", http.MethodPut, path, member.token, put("evil"), http.StatusForbidden},
		{"owner GET settings", http.MethodGet, path, owner.token, nil, http.StatusOK},
		{"owner PUT non-object auth_credentials", http.MethodPut, path, owner.token,
			map[string]any{"setting_entry": map[string]any{"settings": map[string]any{}, "auth_credentials": "x"}}, http.StatusBadRequest},
		{"owner PUT settings", http.MethodPut, path, owner.token, put("evil"), http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := do(tc.method, tc.path, tc.token, tc.body)
			if rr.Code != tc.want {
				t.Fatalf("got %d want %d: %s", rr.Code, tc.want, rr.Body.String())
			}
			if tc.want == http.StatusForbidden {
				if data, _ := tst.ParseResponse(rr)["data"].(map[string]any); len(data) > 0 {
					t.Errorf("forbidden response carried data: %v", data)
				}
			}
		})
	}

	// The service checks membership too, for callers wired without the middleware.
	ids := map[string]string{"org_id": owner.org, "agent_id": agentID, "user_id": outsider.id}
	if _, code, err := svcAgents.GetCustomAgentSettings(ids, db.Postgresql, ctrl.ExtReq); err == nil || code != http.StatusForbidden {
		t.Errorf("service GET for outsider: got %d, %v; want 403", code, err)
	}
	if code, err := svcAgents.UpdateCustomAgentSettings(ids, models.CustomIntegrationSettingRequest{SettingEntry: map[string]any{"settings": map[string]any{}}}, db.Postgresql, ctrl.ExtReq); err == nil || code != http.StatusForbidden {
		t.Errorf("service PUT for outsider: got %d, %v; want 403", code, err)
	}

	// The owner's PUT changed the settings but kept the server-issued key.
	if got := storedAgentAPIKey(t, db, owner.org, agentID); got != "k1" {
		t.Errorf("agent_api_key was replaced through PUT settings: got %v want k1", got)
	}
	rr := do(http.MethodGet, path, owner.token, nil)
	data, _ := tst.ParseResponse(rr)["data"].(map[string]any)
	if s, _ := data["settings"].(map[string]any); s["mode"] != "changed" {
		t.Errorf("owner's settings update was not saved: %v", data)
	}
}
