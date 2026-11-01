package test_group

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	groupCtrl "github.com/hngprojects/telex_be/pkg/controller/group"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

type testUser struct {
	token string
	orgID string
}

// setupTestUser registers, logs in a fresh user and creates an organisation it owns,
// on its own engine so route registrations never collide between users.
func setupTestUser(t *testing.T, db *storage.Database, logger *utility.Logger, currUUID, tag string) testUser {
	t.Helper()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.New()

	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("%v%v@qa.team", tag, currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Test",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("%v%v", tag, currUUID),
	}
	tst.SignupUser(t, r, authCtl, userSignUpData, false)
	token := tst.GetLoginToken(t, r, authCtl, models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	})

	orgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("Org %v %v", tag, currUUID),
		Email:       userSignUpData.Email,
		Description: "organisation used for group membership tests",
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtl, orgData, token)

	return testUser{token: token, orgID: orgID}
}

func getGroup(t *testing.T, r *gin.Engine, token, orgID, groupID string) *httptest.ResponseRecorder {
	t.Helper()

	path := fmt.Sprintf("/api/v1/organisations/%s/groups/%s", orgID, groupID)
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func assertNoGroupData(t *testing.T, body, groupID, groupName string) {
	t.Helper()
	if strings.Contains(body, groupID) || strings.Contains(body, groupName) {
		t.Errorf("response leaked group data: %s", body)
	}
}

func TestGetGroupOrganisationMembership(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()
	validatorRef := validator.New()
	currUUID := utility.GenerateUUID()

	userA := setupTestUser(t, db, logger, currUUID, "amember")
	userB := setupTestUser(t, db, logger, currUUID, "bmember")

	groupCtl := groupCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.New()
	groupURL := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	groupURL.GET("/organisations/:org_id/groups/:group_id", groupCtl.GetGroup)
	groupURL.POST("/organisations/:org_id/groups", groupCtl.CreateGroup)

	// Seed a group inside userB's organisation through the API as its owner.
	createBody, _ := json.Marshal(models.CreateGroupRequest{
		Name:        fmt.Sprintf("secret-group-%v", currUUID[:8]),
		Description: "must only be visible to organisation members",
	})
	createReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/groups", userB.orgID), bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+userB.token)
	createRr := httptest.NewRecorder()
	r.ServeHTTP(createRr, createReq)
	if createRr.Code != http.StatusOK {
		t.Fatalf("failed to seed group: status %d, body: %s", createRr.Code, createRr.Body.String())
	}
	created := tst.ParseResponse(createRr)
	createdData, _ := created["data"].(map[string]any)
	groupID, _ := createdData["id"].(string)
	groupName, _ := createdData["name"].(string)
	if groupID == "" || groupName == "" {
		t.Fatalf("seeded group response missing id/name: %s", createRr.Body.String())
	}

	t.Run("org member gets the group", func(t *testing.T) {
		rr := getGroup(t, r, userB.token, userB.orgID, groupID)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)
		dataM, _ := data["data"].(map[string]any)
		if dataM["id"] != groupID {
			t.Errorf("expected group id %v, got %v", groupID, dataM["id"])
		}
	})

	t.Run("non member is forbidden", func(t *testing.T) {
		rr := getGroup(t, r, userA.token, userB.orgID, groupID)
		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
		assertNoGroupData(t, rr.Body.String(), groupID, groupName)
	})

	t.Run("non member is forbidden even with own org in path", func(t *testing.T) {
		rr := getGroup(t, r, userB.token, userA.orgID, groupID)
		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
		assertNoGroupData(t, rr.Body.String(), groupID, groupName)
	})

	t.Run("member cannot read a group from another org", func(t *testing.T) {
		rr := getGroup(t, r, userA.token, userA.orgID, groupID)
		if rr.Code == http.StatusOK {
			t.Errorf("expected failure status, got 200")
		}
		assertNoGroupData(t, rr.Body.String(), groupID, groupName)
	})

	t.Run("invalid org id is rejected", func(t *testing.T) {
		rr := getGroup(t, r, userA.token, "not-a-uuid", groupID)
		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
		assertNoGroupData(t, rr.Body.String(), groupID, groupName)
	})

	t.Run("invalid group id is rejected", func(t *testing.T) {
		rr := getGroup(t, r, userA.token, userA.orgID, "not-a-uuid")
		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
		assertNoGroupData(t, rr.Body.String(), groupID, groupName)
	})

	t.Run("missing token is unauthorized", func(t *testing.T) {
		rr := getGroup(t, r, "", userB.orgID, groupID)
		tst.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
		assertNoGroupData(t, rr.Body.String(), groupID, groupName)
	})
}
