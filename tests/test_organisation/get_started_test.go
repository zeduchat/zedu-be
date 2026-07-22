package test_organisation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetStarted_UserProfilesFix(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	v := validator.New()
	db := storage.Connection()

	uid := utility.GenerateUUID()
	ownerSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("owner%v@qa.team", uid),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Owner",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("owner_%v", uid),
	}
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, ownerSignUp, false)
	token := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: ownerSignUp.Email, Password: ownerSignUp.Password})

	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger}
	createOrg := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("getstarted_org_%v", uid),
		Description: "org for get started test",
		Email:       ownerSignUp.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrg, token)

	memberEmail := fmt.Sprintf("bambo%v@qa.team", uid)
	memberSignUp := models.CreateUserRequestModel{
		Email:       memberEmail,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Bambo",
		LastName:    "Test",
		Password:    "password",
		UserName:    "",
	}
	tst.SignupUser(t, r, authCtrl, memberSignUp, false)
	memberUserID := tst.GetUserIDFromToken(t, tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: memberSignUp.Email, Password: memberSignUp.Password}), db)

	var defaultRole models.OrgRole
	_ = db.Postgresql.Where("name = ?", models.OrgRoleNameUser).First(&defaultRole).Error

	// Add member to organisation
	orgMgt := models.OrgUserManagement{
		UserID:         memberUserID,
		OrganisationID: orgID,
		RoleID:         defaultRole.ID,
		Status:         "active",
	}
	err := orgMgt.CreateOrgUserManagement(db.Postgresql)
	assert.NoError(t, err)

	testApp := gin.Default()
	testApp.GET("/api/v1/organisations/:org_id/get-started", middleware.Authorize(db.Postgresql), orgCtrl.GetStarted)

	t.Run("Returns user profiles with fallback name from email and generated default avatar", func(t *testing.T) {
		reqURI := fmt.Sprintf("/api/v1/organisations/%s/get-started", orgID)
		req, _ := http.NewRequest(http.MethodGet, reqURI, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		testApp.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp map[string]any
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		data, ok := resp["data"].(map[string]any)
		assert.True(t, ok)

		profiles, ok := data["org_users_profile"].([]any)
		assert.True(t, ok)
		assert.True(t, len(profiles) > 0, "org_users_profile should not be empty")

		for _, p := range profiles {
			profMap, isMap := p.(map[string]any)
			assert.True(t, isMap)

			name, _ := profMap["name"].(string)
			avatarURL, _ := profMap["avatar_url"].(string)

			assert.NotEmpty(t, name, "Name should never be empty in org_users_profile")
			assert.NotEmpty(t, avatarURL, "Avatar URL should never be empty in org_users_profile")
		}

		channels, ok := data["org_channels"].([]any)
		assert.True(t, ok)
		for _, ch := range channels {
			chMap, isMap := ch.(map[string]any)
			assert.True(t, isMap)
			memberAvatars, ok := chMap["member_avatars"].([]any)
			assert.True(t, ok)
			for _, av := range memberAvatars {
				avStr, _ := av.(string)
				assert.NotEmpty(t, avStr, "member_avatars should never contain empty string entries")
			}
		}
	})
}
