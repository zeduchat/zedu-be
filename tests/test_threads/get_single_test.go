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

	threads := models.Threads{
		ID:         utility.GenerateUUID(),
		ChannelsID: channel.ID,
		Status:     "pending",
	}

	userChan := models.UserChannels{
		UserID:     adminUser.ID,
		ChannelsID: channel.ID,
		Username:   adminUser.Name,
	}

	db.Create(&adminUser)
	db.Create(&org)
	db.Create(&channel)
	db.Create(&threads)
	db.Create(&userChan)

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

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *threadController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/%s/channels/%s", threads.ID, channel.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		router, _ := setup()

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/%s/channels/%s", threads.ID, channel.ID), nil)
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token is invalid!")
	})

}
