package test_slack

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetSlackAccessInfo(t *testing.T) {
	_, slackTelexController := SetupSlackTelexTestRouter()

	db := slackTelexController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
	}

	orgId := utility.GenerateUUID()

	slackTelex := models.SlackTelex{
		ID:             utility.GenerateUUID(),
		UserID:         regularUser.ID,
		OrganisationID: orgId,
		IntegrationID:  utility.GenerateUUID(),
		AccessToken:    "acesstoken-accestoken",
	}

	db.Create(&regularUser)
	db.Create(&slackTelex)

	setup := func() (*gin.Engine, *auth.Controller) {
		router, slackTelexController := SetupSlackTelexTestRouter()
		authController := auth.Controller{
			Db:        slackTelexController.Db,
			Validator: slackTelexController.Validator,
			Logger:    slackTelexController.Logger,
			ExtReq:    slackTelexController.ExtReq,
		}

		return router, &authController
	}

	router, authController := setup()

	loginData := models.LoginRequestModel{
		Email:    regularUser.Email,
		Password: "password",
	}

	token := tst.GetLoginToken(t, router, *authController, loginData)

	tests := []struct {
		Name         string
		OrgId        string
		ExpectedCode int
		Message      string
		Headers      map[string]string
	}{
		{
			Name:         "Successful Retrieval of Slack Information",
			OrgId:        slackTelex.OrganisationID,
			ExpectedCode: http.StatusOK,
			Message:      "slack access info fetched successfully",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			fmt.Println(orgId)
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/slack/organisations/%s", orgId), nil)

			for i, v := range test.Headers {
				req.Header.Set(i, v)
			}

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			tst.AssertStatusCode(t, resp.Code, test.ExpectedCode)
			response := tst.ParseResponse(resp)
			tst.AssertResponseMessage(t, response["message"].(string), test.Message)

		})
	}

}
