package test_teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/teams"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestTeamsEndpoints(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username%v", currUUID),
	}
	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	auth := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}
	team := teams.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()
	tst.SignupUser(t, r, auth, userSignUpData, false)

	token := tst.GetLoginToken(t, r, auth, loginData)

	createTeamData := models.CreateTeamRequest{
		Name:        fmt.Sprintf("TestTeam%s", utility.GenerateUUID()),
		Description: "Some Random description",
	}

	teamId, _ := tst.CreateTeam(t, r, team, db, createTeamData, token)

	tests := []struct {
		Name         string
		RequestBody  interface{}
		ExpectedCode int
		Message      string
		Method       string
		Headers      map[string]string
		RequestURI   url.URL
	}{
		{
			Name: "Create Team Action",
			RequestBody: models.CreateTeamRequest{
				Name:        "Test-Team" + utility.GenerateUUID(),
				Description: "This is a test team",
			},
			ExpectedCode: http.StatusCreated,
			Message:      "team created successfully",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: "/api/v1/teams/"},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Get Team Action",
			ExpectedCode: http.StatusOK,
			Message:      "team fetched successfully",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/teams/%s", teamId)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		}, {
			Name:         "Get All Rooms in a Team Action",
			ExpectedCode: http.StatusOK,
			Message:      "rooms fetched successfully",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/teams/rooms/%s", teamId)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Delete Team Action",
			ExpectedCode: http.StatusOK,
			Message:      "team deleted successfully",
			Method:       http.MethodDelete,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/teams/%s", teamId)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
	}

	for _, test := range tests {
		r := gin.Default()

		teamUrl := r.Group(fmt.Sprintf("%v", "/api/v1/teams"), middleware.Authorize(db.Postgresql))
		{
			teamUrl.POST("/", team.CreateTeam)
			teamUrl.GET("/rooms/:teamId", team.GetAllRoomsInTeam)
			teamUrl.GET("/:teamId", team.GetTeamByID)
			teamUrl.DELETE("/:teamId", team.DeleteTeam)
		}

		t.Run(test.Name, func(t *testing.T) {
			var b bytes.Buffer
			json.NewEncoder(&b).Encode(test.RequestBody)

			req, err := http.NewRequest(test.Method, test.RequestURI.String(), &b)
			if err != nil {
				t.Fatal(err)
			}

			for i, v := range test.Headers {
				req.Header.Set(i, v)
			}

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
		})
	}
}
