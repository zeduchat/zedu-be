package test_organisation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetLoadingMetrics_WithInvitationToken(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	v := validator.New()
	db := storage.Connection()

	uid := utility.GenerateUUID()
	userSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("inviter%v@qa.team", uid),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "owner",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("owner_%v", uid),
	}
	loginReq := models.LoginRequestModel{Email: userSignUp.Email, Password: userSignUp.Password}
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, userSignUp, false)
	token := tst.GetLoginToken(t, r, authCtrl, loginReq)

	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger}
	createOrg := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("org%v", uid),
		Description: "load org metrics org",
		Email:       userSignUp.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrg, token)

	existingEmail := fmt.Sprintf("existing%v@qa.team", uid)
	existingUserSignUp := models.CreateUserRequestModel{
		Email:       existingEmail,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "existing",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("existing_%v", uid),
	}
	tst.SignupUser(t, r, authCtrl, existingUserSignUp, false)

	existingInviteToken, _ := utility.GenerateInvitationToken()
	existingInvite := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          existingEmail,
		Token:          existingInviteToken,
		Status:         "invited",
		Role:           "00000000-0000-0000-0000-000000000000",
		OrganisationID: orgID,
		InvitedBy:      tst.GetUserIDFromToken(t, token, db),
		IsTelexUser:    true,
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, db.Postgresql.Create(&existingInvite).Error)

	newEmail := fmt.Sprintf("newuser%v@qa.team", uid)
	newInviteToken, _ := utility.GenerateInvitationToken()
	newInvite := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          newEmail,
		Token:          newInviteToken,
		Status:         "invited",
		Role:           "00000000-0000-0000-0000-000000000000",
		OrganisationID: orgID,
		InvitedBy:      tst.GetUserIDFromToken(t, token, db),
		IsTelexUser:    false,
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, db.Postgresql.Create(&newInvite).Error)

	testApp := gin.Default()
	testApp.GET("/api/v1/organisations/:org_id/load-org-info", orgCtrl.GetLoadingMetrics)

	t.Run("Existing Telex User returns is_new_user = false", func(t *testing.T) {
		reqURI := fmt.Sprintf("/api/v1/organisations/%s/load-org-info?invitation_token=%s", orgID, existingInviteToken)
		req, _ := http.NewRequest(http.MethodGet, reqURI, nil)
		rr := httptest.NewRecorder()
		testApp.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp map[string]any
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		data, ok := resp["data"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, false, data["is_new_user"])
	})

	t.Run("Non-existent Telex User returns is_new_user = true", func(t *testing.T) {
		reqURI := fmt.Sprintf("/api/v1/organisations/%s/load-org-info?invitation_token=%s", orgID, newInviteToken)
		req, _ := http.NewRequest(http.MethodGet, reqURI, nil)
		rr := httptest.NewRecorder()
		testApp.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp map[string]any
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		data, ok := resp["data"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, true, data["is_new_user"])
	})
}
