package test_threads

import (
	"bytes"
	"encoding/json"
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

func TestUpdateThread(t *testing.T) {
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
		IsPrivate:      false,
		OwnerId:        adminUser.ID,
		OrganisationID: org.ID,
	}

	threads := models.Threads{
		ID:           utility.GenerateUUID(),
		ChannelsID:   channel.ID,
		UserID:       adminUser.ID,
		ThreadStatus: "pending",
	}

	db.Create(&adminUser)
	db.Create(&org)
	db.Create(&channel)
	db.Create(&threads)

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

	t.Run("Successful Update Thread", func(t *testing.T) {
		router, threadController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *threadController, loginData)

		reqBody := models.UpdateThreadStatus{
			ThreadStatus: "closed",
		}
		reqBodyJSON, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/threads/%s", threads.ID), bytes.NewBuffer(reqBodyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Thread updated successfully")
	})

	t.Run("Bad Request - Bad Body", func(t *testing.T) {
		router, threadController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *threadController, loginData)

		invalidReqBody := map[string]interface{}{
			"status": 12345,
		}
		reqBodyJSON, _ := json.Marshal(invalidReqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/threads/%s", threads.ID), bytes.NewBuffer(reqBodyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Failed to parse request body")
	})
}
