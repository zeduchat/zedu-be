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
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/router"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

type groupAuthzUser struct {
	token string
	id    string
	org   string
}

func newGroupAuthzUser(t *testing.T, db *storage.Database, logger *utility.Logger, withOrg bool) groupAuthzUser {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	token := initialise(currUUID, t, r, authCtrl, false)
	u := groupAuthzUser{token: token, id: tst.GetUserIDFromToken(t, token, db)}
	if withOrg {
		u.org, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("Org %v", currUUID),
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			Description: "group authz test",
			Type:        "type1",
			Location:    "wakanda",
			Country:     "wakanda",
		}, token)
	}
	return u
}

func addGroupAuthzMember(t *testing.T, db *storage.Database, logger *utility.Logger, owner groupAuthzUser, memberID, roleName string) {
	var role models.OrgRole
	if err := db.Postgresql.Where("name = ?", roleName).First(&role).Error; err != nil {
		t.Fatalf("load %s role: %v", roleName, err)
	}
	orgCtrl := organisation.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	r.POST("/api/v1/organisations/:org_id/users", middleware.Authorize(db.Postgresql), orgCtrl.AddMemberToOrganisation)
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(models.OrgUserCreateRequest{UserID: memberID, RoleID: role.ID})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/users", owner.org), &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+owner.token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Fatalf("add member: %d %s", rr.Code, rr.Body.String())
	}
}

// TestGroupRoutesAuthz covers Ziki #2074, #1629 and #1504: every group route
// needs membership of :org_id, and group writes need can_manage_channels.
func TestGroupRoutesAuthz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	owner := newGroupAuthzUser(t, db, logger, true)
	member := newGroupAuthzUser(t, db, logger, false)
	manager := newGroupAuthzUser(t, db, logger, false)
	outsider := newGroupAuthzUser(t, db, logger, true)
	addGroupAuthzMember(t, db, logger, owner, member.id, models.OrgRoleNameUser)
	addGroupAuthzMember(t, db, logger, owner, manager.id, models.OrgRoleNameManager)

	r := gin.Default()
	router.Group(r, "/api/v1", validator.New(), db, logger)

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
	groupExists := func(id string) bool {
		var n int64
		db.Postgresql.Model(&models.Group{}).Where("id = ?", id).Count(&n)
		return n == 1
	}

	base := fmt.Sprintf("/api/v1/organisations/%s/groups", owner.org)
	newGroup := func(token string) *httptest.ResponseRecorder {
		return do(http.MethodPost, base, token, map[string]string{"name": "group " + utility.GenerateUUID()})
	}

	// Writes by an outsider or a plain member are refused before anything is created.
	if rr := newGroup(outsider.token); rr.Code != http.StatusForbidden {
		t.Fatalf("outsider create: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	if rr := newGroup(member.token); rr.Code != http.StatusForbidden {
		t.Fatalf("member create: got %d want 403: %s", rr.Code, rr.Body.String())
	}
	rr := newGroup(owner.token)
	if rr.Code != http.StatusOK {
		t.Fatalf("owner create: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	data, _ := tst.ParseResponse(rr)["data"].(map[string]any)
	groupID, _ := data["id"].(string)
	if groupID == "" {
		t.Fatalf("owner create returned no group id: %s", rr.Body.String())
	}

	channelID := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.Channels{ID: channelID, Name: "ch " + channelID, Description: "group authz", OrganisationID: owner.org, OwnerId: owner.id}).Error; err != nil {
		t.Fatalf("insert channel: %v", err)
	}
	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM channels WHERE id = ?", channelID)
		db.Postgresql.Exec("DELETE FROM groups WHERE id = ?", groupID)
	})

	groupPath := fmt.Sprintf("%s/%s", base, groupID)
	channelPath := fmt.Sprintf("%s/channels/%s", groupPath, channelID)
	rename := map[string]string{"name": "renamed " + utility.GenerateUUID()}

	cases := []struct {
		name   string
		method string
		path   string
		token  string
		body   any
		want   int
	}{
		{"outsider GET group", http.MethodGet, groupPath, outsider.token, nil, http.StatusForbidden},
		{"outsider GET groups", http.MethodGet, base, outsider.token, nil, http.StatusForbidden},
		{"outsider GET group channels", http.MethodGet, groupPath + "/channels", outsider.token, nil, http.StatusForbidden},
		{"outsider GET no-group channels", http.MethodGet, base + "/channels/no-group", outsider.token, nil, http.StatusForbidden},
		{"outsider GET group via own org", http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/groups/%s", outsider.org, groupID), outsider.token, nil, http.StatusBadRequest},
		{"outsider PATCH group", http.MethodPatch, groupPath, outsider.token, rename, http.StatusForbidden},
		{"outsider DELETE group", http.MethodDelete, groupPath, outsider.token, nil, http.StatusForbidden},
		{"outsider assign channel", http.MethodPost, channelPath, outsider.token, nil, http.StatusForbidden},
		{"outsider move channel", http.MethodPatch, channelPath, outsider.token, nil, http.StatusForbidden},
		{"outsider unassign channel", http.MethodDelete, channelPath, outsider.token, nil, http.StatusForbidden},
		{"member GET group", http.MethodGet, groupPath, member.token, nil, http.StatusOK},
		{"member PATCH group", http.MethodPatch, groupPath, member.token, rename, http.StatusForbidden},
		{"member DELETE group", http.MethodDelete, groupPath, member.token, nil, http.StatusForbidden},
		{"member assign channel", http.MethodPost, channelPath, member.token, nil, http.StatusForbidden},
		{"manager PATCH group", http.MethodPatch, groupPath, manager.token, rename, http.StatusOK},
		{"owner assign channel", http.MethodPost, channelPath, owner.token, nil, http.StatusOK},
		{"owner DELETE invalid group id", http.MethodDelete, base + "/not-a-uuid", owner.token, nil, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := do(tc.method, tc.path, tc.token, tc.body)
			if rr.Code != tc.want {
				t.Fatalf("got %d want %d: %s", rr.Code, tc.want, rr.Body.String())
			}
			if tc.want != http.StatusOK {
				if d, _ := tst.ParseResponse(rr)["data"].(map[string]any); len(d) > 0 {
					t.Errorf("refused response carried data: %v", d)
				}
			}
		})
	}

	if !groupExists(groupID) {
		t.Fatalf("group was deleted by a refused request")
	}
	var ch models.Channels
	db.Postgresql.Where("id = ?", channelID).First(&ch)
	if ch.GroupID == nil || *ch.GroupID != groupID {
		t.Errorf("channel assignment was changed by a refused request: %v", ch.GroupID)
	}

	if rr := do(http.MethodDelete, groupPath, owner.token, nil); rr.Code != http.StatusOK {
		t.Fatalf("owner DELETE group: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	if groupExists(groupID) {
		t.Errorf("owner DELETE did not remove the group")
	}
}
