package test_organisation

import (
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
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetOrgUsersExcludeDeactivated(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	user := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}
	org := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	orgID, token := initialise(currUUID, t, r, db, user, org, true)

	memberUUID := utility.GenerateUUID()
	memberData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("deactivated_%v@qa.team", memberUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "deactivated",
		LastName:    "member",
		Password:    "password",
		UserName:    fmt.Sprintf("deactivated_%v", memberUUID),
	}
	tst.SignupUser(t, gin.Default(), user, memberData, true)

	var member models.User
	if err := db.Postgresql.Where("email = ?", memberData.Email).First(&member).Error; err != nil {
		t.Fatalf("failed to find member: %v", err)
	}

	if err := db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         member.ID,
		OrganisationID: orgID,
		RoleID:         utility.GenerateUUID(),
		Status:         "active",
		IsDeactivated:  true,
	}).Error; err != nil {
		t.Fatalf("failed to add member to org: %v", err)
	}

	r.GET("/api/v1/organisations/:org_id/users", middleware.Authorize(db.Postgresql), org.GetUsersBotsInOrganisation)

	fetch := func(t *testing.T, query string) []interface{} {
		url := fmt.Sprintf("/api/v1/organisations/%s/users?search=%s%s", orgID, memberData.UserName, query)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		users, _ := tst.ParseResponse(rr)["data"].([]interface{})
		return users
	}

	t.Run("deactivated user returned by default", func(t *testing.T) {
		if users := fetch(t, ""); len(users) != 1 {
			t.Errorf("expected 1 user, got %d", len(users))
		}
	})

	t.Run("deactivated user excluded when exclude_deactivated=true", func(t *testing.T) {
		if users := fetch(t, "&exclude_deactivated=true"); len(users) != 0 {
			t.Errorf("expected 0 users, got %d", len(users))
		}
	})
}
