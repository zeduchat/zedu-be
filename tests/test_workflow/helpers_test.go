package test_workflow

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
	"github.com/hngprojects/telex_be/pkg/controller/workflow"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

type wfUser struct {
	token string
	id    string
	org   string
}

// newWfUser signs a user up on a fresh engine (the tst helpers register
// routes, so each user needs its own) and, when withOrg is set, creates an org.
func newWfUser(t *testing.T, db *storage.Database, logger *utility.Logger, withOrg bool) wfUser {
	t.Helper()
	currUUID := utility.GenerateUUID()
	v := validator.New()
	extReq := request.ExternalRequest{Logger: logger, Test: true}
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: extReq}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: extReq}

	r := gin.Default()
	signup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username%v", currUUID),
	}
	tst.SignupUser(t, r, authCtrl, signup, false)
	token := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: signup.Email, Password: signup.Password})

	u := wfUser{token: token, id: tst.GetUserIDFromToken(t, token, db)}
	if withOrg {
		u.org, _, _ = tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("Org %v", currUUID),
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			Description: "workflow authz test",
			Type:        "type1",
			Location:    "wakanda",
			Country:     "wakanda",
		}, token)
	}
	return u
}

// addWfMember adds memberID to owner's org with the default "User" role,
// which does not hold can_manage_workflows.
func addWfMember(t *testing.T, db *storage.Database, logger *utility.Logger, owner wfUser, memberID string) {
	t.Helper()
	var userRole models.OrgRole
	if err := db.Postgresql.Where("name = ?", models.OrgRoleNameUser).First(&userRole).Error; err != nil {
		t.Fatalf("load User role: %v", err)
	}
	orgCtrl := organisation.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	r.POST("/api/v1/organisations/:org_id/users", middleware.Authorize(db.Postgresql), orgCtrl.AddMemberToOrganisation)
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(models.OrgUserCreateRequest{UserID: memberID, RoleID: userRole.ID})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/users", owner.org), &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+owner.token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Fatalf("add member: %d %s", rr.Code, rr.Body.String())
	}
}

// insertWorkflow stores a workflow row for orgID directly, skipping agent setup.
func insertWorkflow(t *testing.T, db *storage.Database, orgID, userID string) string {
	t.Helper()
	wf := models.Workflow{ID: utility.GenerateUUID(), Name: "authz test workflow", OrgId: orgID, UserId: userID}
	if err := db.Postgresql.Create(&wf).Error; err != nil {
		t.Fatalf("insert workflow: %v", err)
	}
	return wf.ID
}

// newWfRouter mounts the workflow routes the way pkg/router/workflow.go does.
func newWfRouter(db *storage.Database, logger *utility.Logger) *gin.Engine {
	ctrl := workflow.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	org := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	org.DELETE("/:org_id/workflows/:workflow_id", ctrl.DeleteWorkflow)
	cw := r.Group("/api/v1/channel-workflows", middleware.Authorize(db.Postgresql))
	cw.POST("", ctrl.AddWorkflowToChannel)
	cw.DELETE("/:workflow_id/channels/:channel_id", ctrl.RemoveWorkflowFromChannel)
	pub := r.Group("/api/v1/workflows")
	pub.GET("/:workflow_id", ctrl.GetGeneralMarketPlaceWorkflowByID)
	return r
}

// doWf sends a JSON request; an empty token sends no Authorization header.
func doWf(r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}
