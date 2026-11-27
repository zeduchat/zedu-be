package test_organisation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	orgsvc "github.com/hngprojects/telex_be/services/organisation"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestSearchUsersInOrganisationAccess checks that only members of :org_id can
// search its user directory (#1431).
func TestSearchUsersInOrganisationAccess(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	userController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{
		Logger: logger,
		Test:   true,
	}}
	orgController := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	uuidA := utility.GenerateUUID()
	rA := gin.Default()
	orgA, tokenA := initialise(uuidA, t, rA, db, userController, orgController, true)
	require.NotEmpty(t, orgA)

	uuidB := utility.GenerateUUID()
	rB := gin.Default()
	orgB, tokenB := initialise(uuidB, t, rB, db, userController, orgController, true)
	require.NotEmpty(t, orgB)

	// Member M is added to org A by its owner.
	rA.Group("/api/v1", middleware.Authorize(db.Postgresql)).
		POST("/organisations/:org_id/users", orgController.AddMemberToOrganisation)
	mSignup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("member_%v@qa.team", uuidA),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "member",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("member_%v", uuidA),
	}
	tst.SignupUser(t, rA, userController, mSignup, true)
	var member models.User
	require.NoError(t, db.Postgresql.Where("email = ?", mSignup.Email).First(&member).Error)
	var defaultRole models.OrgRole
	require.NoError(t, db.Postgresql.Where("name = ?", "User").First(&defaultRole).Error)
	var b bytes.Buffer
	require.NoError(t, json.NewEncoder(&b).Encode(models.OrgUserCreateRequest{UserID: member.ID, RoleID: defaultRole.ID}))
	addReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/users", orgA), &b)
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+tokenA)
	addRR := httptest.NewRecorder()
	rA.ServeHTTP(addRR, addReq)
	require.True(t, addRR.Code == http.StatusOK || addRR.Code == http.StatusCreated, "failed to add member: %s", addRR.Body.String())
	tokenM := tst.GetLoginToken(t, rA, userController, models.LoginRequestModel{Email: mSignup.Email, Password: mSignup.Password})

	rA.Group("/api/v1", middleware.Authorize(db.Postgresql)).
		GET("/organisations/:org_id/users/search", middleware.RequireOrgMember(db.Postgresql), orgController.SearchUsersInOrganisation)

	ownerAEmail := fmt.Sprintf("testuser%v@qa.team", uuidA)

	cases := []struct {
		name     string
		orgID    string
		token    string
		wantCode int
	}{
		{"owner searches own org", orgA, tokenA, http.StatusOK},
		{"added member searches org", orgA, tokenM, http.StatusOK},
		{"other org owner searches org A", orgA, tokenB, http.StatusForbidden},
		{"other org owner searches random org", utility.GenerateUUID(), tokenB, http.StatusForbidden},
		{"malformed org id", "not-a-uuid", tokenA, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/users/search?query=test", tc.orgID), nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			rr := httptest.NewRecorder()
			rA.ServeHTTP(rr, req)
			require.Equal(t, tc.wantCode, rr.Code, rr.Body.String())

			if tc.wantCode == http.StatusOK {
				var resp struct {
					Data []models.UserInOrgResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
				found := false
				for _, u := range resp.Data {
					if u.Email == ownerAEmail {
						found = true
					}
				}
				assert.True(t, found, "expected org A owner in results")
			} else {
				assert.NotContains(t, rr.Body.String(), ownerAEmail)
			}
		})
	}

	// The service refuses non-members even when the middleware is not wired.
	var ownerB models.User
	require.NoError(t, db.Postgresql.Where("email = ?", fmt.Sprintf("testuser%v@qa.team", uuidB)).First(&ownerB).Error)
	users, code, err := orgsvc.SearchUsersInOrganisation(db.Postgresql, orgA, ownerB.ID, "test")
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, code)
	assert.Empty(t, users)
}
