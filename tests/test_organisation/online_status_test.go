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

func TestUserOnlineStatusInOrg(t *testing.T) {
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

	// Fetch the user to update their profile
	userEmail := fmt.Sprintf("testuser%v@qa.team", currUUID)
	var user models.User
	if err := db.Postgresql.Preload("Profile").Where("email = ?", userEmail).First(&user).Error; err != nil {
		t.Fatalf("Failed to fetch user: %v", err)
	}

	// Register route once
	r.Group("/api/v1", middleware.Authorize(db.Postgresql)).GET("/organisations/:org_id/users", orgController.GetUsersBotsInOrganisation)

	// Helper function to make request and check online status
	checkOnlineStatus := func(expectedStatus bool) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/users", orgID), nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		var data map[string]any
		if err := json.NewDecoder(rr.Body).Decode(&data); err != nil {
			t.Fatalf("Failed to decode response body: %v", err)
		}

		usersList, ok := data["data"].([]any)
		if !ok {
			t.Fatalf("Expected data['data'] to be a list, got %T: %v", data["data"], data["data"])
		}

		found := false
		for _, u := range usersList {
			userMap := u.(map[string]any)
			if userMap["email"] == userEmail {
				found = true
				online, ok := userMap["online"].(bool)
				if !ok {
					t.Errorf("Expected 'online' field to be boolean, got %T", userMap["online"])
				}
				if online != expectedStatus {
					t.Errorf("Expected online status to be %v, got %v", expectedStatus, online)
				}
			}
		}
		if !found {
			t.Errorf("User with email %s not found in response", userEmail)
		}
	}

	// Test 1: Set Online = true
	if err := db.Postgresql.Model(&models.Profile{}).Where("userid = ?", user.ID).Update("online", true).Error; err != nil {
		t.Fatalf("Failed to update profile online status: %v", err)
	}
	checkOnlineStatus(true)

	// Test 2: Set Online = false
	if err := db.Postgresql.Model(&models.Profile{}).Where("userid = ?", user.ID).Update("online", false).Error; err != nil {
		t.Fatalf("Failed to update profile online status: %v", err)
	}
	checkOnlineStatus(false)
}
