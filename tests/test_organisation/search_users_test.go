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
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestSearchUsersInOrganisationReturnsDeterministicUsername(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	userController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{
		Logger: logger,
		Test:   true,
	}}
	orgController := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()

	orgID, token := initialise(currUUID, t, r, db, userController, orgController, true)
	require.NotEmpty(t, orgID)

	var user models.User
	require.NoError(t, db.Postgresql.Where("email = ?", fmt.Sprintf("testuser%v@qa.team", currUUID)).First(&user).Error)
	require.NoError(t, db.Postgresql.Model(&models.User{}).Where("id = ?", user.ID).Update("name", "divine john").Error)

	require.NoError(t, db.Postgresql.Unscoped().Where("userid = ?", user.ID).Delete(&models.Profile{}).Error)

	blank := models.Profile{
		ID:        utility.GenerateUUID(),
		Userid:    user.ID,
		FirstName: "divine",
		LastName:  "john",
	}
	require.NoError(t, db.Postgresql.Create(&blank).Error)

	handle := "dicint"
	populated := models.Profile{
		ID:             utility.GenerateUUID(),
		Userid:         user.ID,
		OrganisationID: &orgID,
		FirstName:      "divine",
		LastName:       "john",
		UserName:       handle,
	}
	require.NoError(t, db.Postgresql.Create(&populated).Error)

	r.Group("/api/v1", middleware.Authorize(db.Postgresql)).
		GET("/organisations/:org_id/users/search", orgController.SearchUsersInOrganisation)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/users/search?query=divine", orgID), nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Data []models.UserInOrgResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, handle, resp.Data[0].UserName)
}

// TestSearchUsersInOrganisationWithSimilarUsernames tests that multiple users
// with similar usernames/names are all returned in search results.
func TestSearchUsersInOrganisationWithSimilarUsernames(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	userController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{
		Logger: logger,
		Test:   true,
	}}
	orgController := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()

	orgID, token := initialise(currUUID, t, r, db, userController, orgController, true)
	require.NotEmpty(t, orgID)

	// Register add member route for test
	r.Group("/api/v1", middleware.Authorize(db.Postgresql)).
		POST("/organisations/:org_id/users", orgController.AddMemberToOrganisation)

	// Get the owner user
	var ownerUser models.User
	require.NoError(t, db.Postgresql.Where("email = ?", fmt.Sprintf("testuser%v@qa.team", currUUID)).First(&ownerUser).Error)

	// Get default role ID for adding members
	var defaultRole models.OrgRole
	require.NoError(t, db.Postgresql.Where("name = ?", "User").First(&defaultRole).Error)

	// Create additional users with similar names
	userData := []struct {
		email     string
		name      string
		userName  string
		firstName string
		lastName  string
	}{
		{
			email:     fmt.Sprintf("user1_%v@qa.team", currUUID),
			name:      "john doe",
			userName:  "johndoe1",
			firstName: "john",
			lastName:  "doe",
		},
		{
			email:     fmt.Sprintf("user2_%v@qa.team", currUUID),
			name:      "john smith",
			userName:  "johnsmith2",
			firstName: "john",
			lastName:  "smith",
		},
		{
			email:     fmt.Sprintf("user3_%v@qa.team", currUUID),
			name:      "jane doe",
			userName:  "janedoe3",
			firstName: "jane",
			lastName:  "doe",
		},
	}

	var createdUsers []models.User
	for _, ud := range userData {
		signupData := models.CreateUserRequestModel{
			Email:       ud.email,
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   ud.firstName,
			LastName:    ud.lastName,
			Password:    "password",
			UserName:    ud.userName,
		}
		tst.SignupUser(t, r, userController, signupData, true)

		var u models.User
		require.NoError(t, db.Postgresql.Where("email = ?", ud.email).First(&u).Error)
		createdUsers = append(createdUsers, u)

		// Add user to organisation
		addMemberBody := models.OrgUserCreateRequest{
			UserID: u.ID,
			RoleID: defaultRole.ID,
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(addMemberBody)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/users", orgID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		require.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusCreated, "failed to add member: %s", rr.Body.String())

		// Delete any auto-created profile and create org-scoped profile with username for each user
		require.NoError(t, db.Postgresql.Unscoped().Where("userid = ? AND organisation_id = ?", u.ID, orgID).Delete(&models.Profile{}).Error)
		profile := models.Profile{
			ID:             utility.GenerateUUID(),
			Userid:         u.ID,
			OrganisationID: &orgID,
			FirstName:      ud.firstName,
			LastName:       ud.lastName,
			UserName:       ud.userName,
		}
		require.NoError(t, db.Postgresql.Create(&profile).Error)
	}

	// Search for "john" - should match user1 (john doe) and user2 (john smith)
	r.Group("/api/v1", middleware.Authorize(db.Postgresql)).
		GET("/organisations/:org_id/users/search", orgController.SearchUsersInOrganisation)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/users/search?query=john", orgID), nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Data []models.UserInOrgResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	// Should find at least 2 users (john doe and john smith)
	require.GreaterOrEqual(t, len(resp.Data), 2)

	// Verify the returned users have the correct usernames
	foundUsernames := make(map[string]bool)
	for _, u := range resp.Data {
		foundUsernames[u.UserName] = true
	}
	assert.True(t, foundUsernames["johndoe1"], "expected johndoe1 in results")
	assert.True(t, foundUsernames["johnsmith2"], "expected johnsmith2 in results")
}

// TestSearchUsersInOrganisationOrdering tests that search results are returned
// in a deterministic order (by user ID as per the DISTINCT ON ordering).
func TestSearchUsersInOrganisationOrdering(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	userController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{
		Logger: logger,
		Test:   true,
	}}
	orgController := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()

	orgID, token := initialise(currUUID, t, r, db, userController, orgController, true)
	require.NotEmpty(t, orgID)

	// Register add member route for test
	r.Group("/api/v1", middleware.Authorize(db.Postgresql)).
		POST("/organisations/:org_id/users", orgController.AddMemberToOrganisation)

	// Get the owner user
	var ownerUser models.User
	require.NoError(t, db.Postgresql.Where("email = ?", fmt.Sprintf("testuser%v@qa.team", currUUID)).First(&ownerUser).Error)

	// Get default role ID
	var defaultRole models.OrgRole
	require.NoError(t, db.Postgresql.Where("name = ?", "User").First(&defaultRole).Error)

	// Create multiple users with distinct names to test ordering
	userData := []struct {
		email     string
		name      string
		userName  string
		firstName string
		lastName  string
	}{
		{
			email:     fmt.Sprintf("aaa_user_%v@qa.team", currUUID),
			name:      "aaa first",
			userName:  "aaa_first",
			firstName: "aaa",
			lastName:  "first",
		},
		{
			email:     fmt.Sprintf("bbb_user_%v@qa.team", currUUID),
			name:      "bbb second",
			userName:  "bbb_second",
			firstName: "bbb",
			lastName:  "second",
		},
		{
			email:     fmt.Sprintf("ccc_user_%v@qa.team", currUUID),
			name:      "ccc third",
			userName:  "ccc_third",
			firstName: "ccc",
			lastName:  "third",
		},
	}

	var createdUsers []models.User
	for _, ud := range userData {
		signupData := models.CreateUserRequestModel{
			Email:       ud.email,
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   ud.firstName,
			LastName:    ud.lastName,
			Password:    "password",
			UserName:    ud.userName,
		}
		tst.SignupUser(t, r, userController, signupData, true)

		var u models.User
		require.NoError(t, db.Postgresql.Where("email = ?", ud.email).First(&u).Error)
		createdUsers = append(createdUsers, u)

		// Add user to organisation
		addMemberBody := models.OrgUserCreateRequest{
			UserID: u.ID,
			RoleID: defaultRole.ID,
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(addMemberBody)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/users", orgID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		require.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusCreated, "failed to add member: %s", rr.Body.String())

		// Delete any auto-created profile and create org-scoped profile
		require.NoError(t, db.Postgresql.Unscoped().Where("userid = ? AND organisation_id = ?", u.ID, orgID).Delete(&models.Profile{}).Error)
		profile := models.Profile{
			ID:             utility.GenerateUUID(),
			Userid:         u.ID,
			OrganisationID: &orgID,
			FirstName:      ud.firstName,
			LastName:       ud.lastName,
			UserName:       ud.userName,
		}
		require.NoError(t, db.Postgresql.Create(&profile).Error)
	}

	// Search with a query that matches all three (search by "user" in email or name)
	r.Group("/api/v1", middleware.Authorize(db.Postgresql)).
		GET("/organisations/:org_id/users/search", orgController.SearchUsersInOrganisation)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/users/search?query=user", orgID), nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Data []models.UserInOrgResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	// Should find all 3 created users plus the owner (who has name "test user" from initialise)
	require.GreaterOrEqual(t, len(resp.Data), 3)

	// Verify results are ordered by user ID (ascending) as per DISTINCT ON ordering
	// The query uses ORDER BY users.id for DISTINCT ON, so results come back in users.id order
	var prevID string
	for _, u := range resp.Data {
		if prevID != "" {
			assert.True(t, prevID <= u.ID, "results should be ordered by user ID ascending: %s > %s", prevID, u.ID)
		}
		prevID = u.ID
	}
}
