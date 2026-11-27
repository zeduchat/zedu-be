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

// Ziki #2345: org user listings must return the profile handle (profiles.user_name)
// in the "username" field instead of an empty string or a SQL error.
func TestOrgUsersReturnPopulatedUsername(t *testing.T) {
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
	handle := fmt.Sprintf("handle_%v", memberUUID[:8])
	memberData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("handle_%v@qa.team", memberUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "handle",
		LastName:    "member",
		Password:    "password",
		UserName:    handle,
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
	}).Error; err != nil {
		t.Fatalf("failed to add member to org: %v", err)
	}
	if err := db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?) ON CONFLICT DO NOTHING", member.ID, orgID).Error; err != nil {
		t.Fatalf("failed to add user_organisations row: %v", err)
	}
	// Make sure the member has a profile in this org (signup binds the first
	// profile to the member's own org), then give every profile the handle.
	var profModel models.Profile
	if _, err := profModel.GetOrCreateProfileForOrg(db.Postgresql, member.ID, orgID); err != nil {
		t.Fatalf("failed to resolve member org profile: %v", err)
	}
	if err := db.Postgresql.Model(&models.Profile{}).Where("userid = ?", member.ID).Update("user_name", handle).Error; err != nil {
		t.Fatalf("failed to set profile handle: %v", err)
	}

	t.Run("GET /organisations/:org_id/users returns username", func(t *testing.T) {
		r.GET("/api/v1/organisations/:org_id/users", middleware.Authorize(db.Postgresql), org.GetUsersBotsInOrganisation)
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/users?search=%s", orgID, handle), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		users, _ := tst.ParseResponse(rr)["data"].([]interface{})
		if len(users) == 0 {
			t.Fatalf("expected the member in the response, got none")
		}
		for _, u := range users {
			row := u.(map[string]interface{})
			if row["id"] != member.ID {
				continue
			}
			if row["username"] != handle {
				t.Errorf("expected username %q, got %q", handle, row["username"])
			}
		}
	})

	t.Run("GetUsersAndBotsInOrganisation returns username", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest(http.MethodGet, "/?page=1&limit=100", nil)

		var o models.Organisation
		users, _, err := o.GetUsersAndBotsInOrganisation(c, db.Postgresql, orgID)
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		found := false
		for _, u := range users {
			if u.ID != member.ID {
				continue
			}
			found = true
			if u.UserName != handle {
				t.Errorf("expected username %q, got %q", handle, u.UserName)
			}
		}
		if !found {
			t.Fatalf("member not returned by GetUsersAndBotsInOrganisation")
		}
	})
}
