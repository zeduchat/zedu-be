package test_organisation

import (
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
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestRoleFiltering(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	user := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	org := organisation.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
	}

	r := gin.Default()
	orgID, token := initialise(currUUID, t, r, db, user, org, true)

	var adminRole, userRole, guestRole models.OrgRole

	err := db.Postgresql.Where("name = ? AND (organisation_id = ? OR is_default = ?)", "Administrator", orgID, true).First(&adminRole).Error
	if err != nil {
		t.Fatalf("Failed to find Administrator role: %v", err)
	}

	err = db.Postgresql.Where("name = ? AND (organisation_id = ? OR is_default = ?)", "User", orgID, true).First(&userRole).Error
	if err != nil {
		t.Fatalf("Failed to find User role: %v", err)
	}

	err = db.Postgresql.Where("name = ? AND (organisation_id = ? OR is_default = ?)", "Guest", orgID, true).First(&guestRole).Error
	if err != nil {
		t.Fatalf("Failed to find Guest role: %v", err)
	}

	user2UUID := utility.GenerateUUID()
	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser2_%v@qa.team", user2UUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test2",
		LastName:    "user2",
		Password:    "password",
		UserName:    fmt.Sprintf("admin_user_%v", user2UUID),
	}
	tst.SignupUser(t, gin.Default(), user, user2SignUpData, true)

	user3UUID := utility.GenerateUUID()
	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser3_%v@qa.team", user3UUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test3",
		LastName:    "user3",
		Password:    "password",
		UserName:    fmt.Sprintf("guest_user_%v", user3UUID),
	}
	tst.SignupUser(t, gin.Default(), user, user3SignUpData, true)

	var user2, user3 models.User
	err = db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error
	if err != nil {
		t.Fatalf("Failed to find user2: %v", err)

	}

	err = db.Postgresql.Where("email = ?", user3SignUpData.Email).First(&user3).Error
	if err != nil {
		t.Fatalf("Failed to find user3: %v", err)
	}

	orgUserMgt2 := models.OrgUserManagement{
		UserID:         user2.ID,
		OrganisationID: orgID,
		RoleID:         adminRole.ID,
		Status:         "active",
	}
	err = db.Postgresql.Create(&orgUserMgt2).Error
	if err != nil {
		t.Fatalf("Failed to add user2 to org: %v", err)
	}

	orgUserMgt3 := models.OrgUserManagement{
		UserID:         user3.ID,
		OrganisationID: orgID,
		RoleID:         guestRole.ID,
		Status:         "active",
	}
	err = db.Postgresql.Create(&orgUserMgt3).Error
	if err != nil {
		t.Fatalf("Failed to add user3 to org: %v", err)
	}

	err = db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", user2.ID, orgID).Error
	if err != nil {
		t.Fatalf("Failed to add user2 to user_organisations: %v", err)
	}

	err = db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", user3.ID, orgID).Error
	if err != nil {
		t.Fatalf("Failed to add user3 to user_organisations: %v", err)
	}

	var profModel models.Profile
	_, _ = profModel.GetOrCreateProfileForOrg(db.Postgresql, user2.ID, orgID)
	_, _ = profModel.GetOrCreateProfileForOrg(db.Postgresql, user3.ID, orgID)

	tests := []struct {
		Name         string
		Role         string
		Search       string
		IncludeBots  string
		ExpectedCode int
		Message      string
		ValidateFunc func(*testing.T, []models.UserInOrgResponse)
	}{
		{
			Name:         "Filter by exact role name - Administrator",
			Role:         "Administrator",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one Administrator user")
					return
				}
				for _, u := range users {
					if u.Role != "Administrator" {
						t.Errorf("Expected role 'Administrator', got '%s'", u.Role)
					}
				}
			},
		},
		{
			Name:         "Filter by exact role name - User",
			Role:         "User",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				for _, u := range users {
					if u.Role != "User" {
						t.Errorf("Expected role 'User', got '%s'", u.Role)
					}
				}
			},
		},
		{
			Name:         "Filter by exact role name - Guest",
			Role:         "Guest",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one Guest user")
					return
				}
				for _, u := range users {
					if u.Role != "Guest" {
						t.Errorf("Expected role 'Guest', got '%s'", u.Role)
					}
				}
			},
		},
		{
			Name:         "Filter by partial role name - 'admin' matches Administrator",
			Role:         "admin",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one user with role containing 'admin'")
					return
				}
				for _, u := range users {
					if u.Role != "Administrator" {
						t.Errorf("Expected role 'Administrator', got '%s'", u.Role)
					}
				}
			},
		},
		{
			Name:         "Filter by empty role - returns all users",
			Role:         "",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) < 3 {
					t.Errorf("Expected at least 3 users (all roles), got %d", len(users))
				}
			},
		},
		{
			Name:         "Filter by non-existent role",
			Role:         "NonExistentRole",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) != 0 {
					t.Errorf("Expected 0 users for non-existent role, got %d", len(users))
				}
			},
		},
		{
			Name:         "Case insensitivity - 'ADMINISTRATOR' matches Administrator",
			Role:         "ADMINISTRATOR",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one Administrator user")
					return
				}
				for _, u := range users {
					if u.Role != "Administrator" {
						t.Errorf("Expected role 'Administrator', got '%s'", u.Role)
					}
				}
			},
		},
		{
			Name:         "Case insensitivity - 'user' matches User",
			Role:         "user",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				for _, u := range users {
					if u.Role != "User" {
						t.Errorf("Expected role 'User', got '%s'", u.Role)
					}
				}
			},
		},
		{
			Name:         "Role filter with bots included",
			Role:         "bot",
			Search:       "",
			IncludeBots:  "true",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				for _, u := range users {
					if u.EntityType == "bot" && u.Role != "bot" {
						t.Errorf("Expected bot role to be 'bot', got '%s'", u.Role)
					}
				}
			},
		},
		{
			Name:         "Partial match - 'ist' matches 'Administrator'",
			Role:         "ist",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one user with role containing 'ist'")
					return
				}
				for _, u := range users {
					if u.Role != "Administrator" {
						t.Errorf("Expected role 'Administrator', got '%s'", u.Role)
					}
				}
			},
		},
		{
			Name:         "SQL injection attempt - role filter should be safe",
			Role:         "' OR '1'='1",
			Search:       "",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) != 0 {
					t.Errorf("SQL injection attempt should return 0 users, got %d", len(users))
				}
			},
		},
		{
			Name:         "Filter by username - exact match",
			Role:         "",
			Search:       user2SignUpData.UserName,
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one user matching the username")
					return
				}
				found := false
				for _, u := range users {
					if u.UserName == user2SignUpData.UserName || u.Name == user2SignUpData.UserName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find user with username '%s'", user2SignUpData.UserName)
				}
			},
		},
		{
			Name:         "Filter by username - partial match",
			Role:         "",
			Search:       "admin_user",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one user with username containing 'admin_user'")
					return
				}
			},
		},
		{
			Name:         "Filter by username - case insensitive",
			Role:         "",
			Search:       "ADMIN_USER",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one user with username containing 'admin_user' (case insensitive)")
					return
				}
			},
		},
		{
			Name:         "Filter by username and role combined",
			Role:         "Administrator",
			Search:       "admin_user",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one Administrator user with username containing 'admin_user'")
					return
				}
				for _, u := range users {
					if u.Role != "Administrator" {
						t.Errorf("Expected role 'Administrator', got '%s'", u.Role)
					}
				}
			},
		},
		{
			Name:         "Filter by username - no match",
			Role:         "",
			Search:       "nonexistent_username_xyz",
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) != 0 {
					t.Errorf("Expected 0 users for non-existent username, got %d", len(users))
				}
			},
		},
		{
			Name:         "Filter by email address",
			Role:         "",
			Search:       user3SignUpData.Email,
			IncludeBots:  "false",
			ExpectedCode: http.StatusOK,
			Message:      "users and bots retrieved successfully",
			ValidateFunc: func(t *testing.T, users []models.UserInOrgResponse) {
				if len(users) == 0 {
					t.Error("Expected at least one user matching the email")
					return
				}
				found := false
				for _, u := range users {
					if u.Email == user3SignUpData.Email {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find user with email '%s'", user3SignUpData.Email)
				}
			},
		},
	}

	for _, test := range tests {
		r := gin.Default()

		orgUrl := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
		{
			orgUrl.GET("/organisations/:org_id/users", org.GetUsersBotsInOrganisation)
		}

		t.Run(test.Name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/organisations/%s/users?role=%s&search=%s&include_bots=%s",
				orgID, test.Role, test.Search, test.IncludeBots)

			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)

			data := tst.ParseResponse(rr)

			code := int(data["status_code"].(float64))
			tst.AssertStatusCode(t, code, test.ExpectedCode)

			if test.Message != "" {
				message := data["message"]
				if message != nil {
					tst.AssertResponseMessage(t, message.(string), test.Message)
				} else {
					tst.AssertResponseMessage(t, "", test.Message)
				}
			}

			if test.ValidateFunc != nil && data["data"] != nil {
				usersData, ok := data["data"].([]interface{})
				if !ok {
					t.Fatalf("Expected data to be an array, got %T", data["data"])
				}

				var users []models.UserInOrgResponse
				jsonData, _ := json.Marshal(usersData)
				json.Unmarshal(jsonData, &users)

				test.ValidateFunc(t, users)
			}
		})
	}
}
