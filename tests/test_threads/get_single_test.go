package test_threads

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetUserSingleThreads(t *testing.T) {
	_, threadController := SetupThreadsTestRouter()
	db := threadController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	adminUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Admin User",
		Email:    fmt.Sprintf("admin%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.SuperAdmin),
	}

	org := models.Organisation{
		ID:      utility.GenerateUUID(),
		Name:    fmt.Sprintf("Org comp%v", currUUID),
		Email:   fmt.Sprintf("orgtest%v@qa.team", currUUID),
		OwnerID: adminUser.ID,
	}

	channel := models.Channels{
		ID:             utility.GenerateUUID(),
		Name:           "General",
		Description:    "General discussion channel",
		OwnerId:        adminUser.ID,
		OrganisationID: org.ID,
	}

	userChan := models.UserChannels{
		UserID:     adminUser.ID,
		ChannelsID: channel.ID,
		Username:   adminUser.Name,
	}

	db.Create(&adminUser)
	db.Create(&org)
	db.Create(&channel)
	db.Create(&userChan)

	// Helper function to create and validate a thread in ElasticDB
	createAndValidateThread := func(t *testing.T, controller *auth.Controller) models.ThreadDocument {
		threadDoc := models.ThreadDocument{
			ID:             utility.GenerateUUID(),
			ChannelsID:     channel.ID,
			OrganisationID: org.ID,
			Username:       adminUser.Name,
			Content:        "Test thread content",
			UserId:         adminUser.ID,
			Type:           "thread",
			FullName:       adminUser.Name,
			Email:          adminUser.Email,
			AvatarURL:      "",
			UserType:       "user",
		}

		// Create the thread in ElasticDB
		err := threadDoc.CreateThread(controller.Db, controller.Logger)
		if err != nil {
			t.Fatalf("Failed to create thread in ElasticDB: %v", err)
		}

		// Validate the thread was created by retrieving it
		var retrievedThread models.ThreadDocument
		err = retrievedThread.GetThreadById(threadDoc.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve created thread from ElasticDB: %v", err)
		}

		// Validate thread fields
		if retrievedThread.ID != threadDoc.ID {
			t.Errorf("Expected thread ID %s, got %s", threadDoc.ID, retrievedThread.ID)
		}
		if retrievedThread.ChannelsID != channel.ID {
			t.Errorf("Expected channel ID %s, got %s", channel.ID, retrievedThread.ChannelsID)
		}
		if retrievedThread.UserId != adminUser.ID {
			t.Errorf("Expected user ID %s, got %s", adminUser.ID, retrievedThread.UserId)
		}
		if retrievedThread.Content != threadDoc.Content {
			t.Errorf("Expected content %s, got %s", threadDoc.Content, retrievedThread.Content)
		}

		return threadDoc
	}

	setup := func() (*gin.Engine, *auth.Controller) {
		router, threadController := SetupThreadsTestRouter()
		authcontroller := auth.Controller{
			Db:        threadController.Db,
			Validator: threadController.Validator,
			Logger:    threadController.Logger,
			ExtReq:    threadController.ExtReq,
		}

		return router, &authcontroller
	}

	t.Run("Successful Get User Single Thread", func(t *testing.T) {
		router, threadController := setup()

		// Create and validate thread in ElasticDB
		thread := createAndValidateThread(t, threadController)

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *threadController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/%s/channels/%s", thread.ID, channel.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		router, threadController := setup()

		// Create and validate thread in ElasticDB
		thread := createAndValidateThread(t, threadController)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/%s/channels/%s", thread.ID, channel.ID), nil)
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token is invalid!")
	})

}
