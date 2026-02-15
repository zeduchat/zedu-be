package test_threads

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetUserRecentThreads(t *testing.T) {
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

	auth := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	tst.SignupUser(t, r, auth, userSignUpData, false)

	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	org := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	threadCtrl := thread.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	token := tst.GetLoginToken(t, r, auth, loginData)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("TestTeam%s", currUUID),
		Description: "Some Random description",
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}

	orgId, _, _ := tst.CreateOrganisation(t, r, db, org, createOrgData, token)

	createChannelsData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("TestChannels%s", utility.GenerateUUID()),
		Username:       fmt.Sprintf("Mr%sChannels", utility.GenerateUUID()),
		OrganisationID: orgId,
		Description:    "Some Random description",
	}

	channelId, _ := tst.CreateChannels(t, r, channelCtrl, db, createChannelsData, token)

	// Create some test threads with time delay to ensure proper ordering
	now := time.Now().UTC()

	thread1 := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         tst.GetUserIDFromToken(t, token, db),
		Status:         "success",
		Type:           "thread",
		Content:        "Test thread 1 (older)",
		Username:       "test_user",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := thread1.CreateThread(db, logger)
	if err != nil {
		t.Fatalf("Failed to create test thread 1: %v", err)
	}

	time.Sleep(3 * time.Second)

	now2 := time.Now().UTC()
	thread2 := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         tst.GetUserIDFromToken(t, token, db),
		Status:         "success",
		Type:           "thread",
		Content:        "Test thread 2 (newer)",
		Username:       "test_user",
		CreatedAt:      now2,
		UpdatedAt:      now2,
	}

	err = thread2.CreateThread(db, logger)
	if err != nil {
		t.Fatalf("Failed to create test thread 2: %v", err)
	}

	time.Sleep(1 * time.Second)

	tests := []struct {
		Name         string
		ExpectedCode int
		Message      string
		Method       string
		Headers      map[string]string
		RequestURI   url.URL
	}{
		{
			Name:         "Successfully Get Recent Threads",
			ExpectedCode: http.StatusOK,
			Message:      "Data retrieved successfully",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: "/api/v1/threads/recent"},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Unauthorized - No Token",
			ExpectedCode: http.StatusUnauthorized,
			Message:      "",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: "/api/v1/threads/recent"},
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, test := range tests {
		r := gin.Default()

		threadUrl := r.Group(fmt.Sprintf("%v", "/api/v1/threads"), middleware.Authorize(db.Postgresql))
		{
			threadUrl.GET("/recent", threadCtrl.GetUserRecentThreads)
		}

		t.Run(test.Name, func(t *testing.T) {
			var b bytes.Buffer

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

			if test.ExpectedCode == http.StatusOK {
				dataField := data["data"]
				if dataField != nil {

					threads, ok := dataField.([]interface{})
					if ok && len(threads) >= 2 {
						firstThread := threads[0].(map[string]interface{})
						secondThread := threads[1].(map[string]interface{})

						firstContent := firstThread["message"].(string)
						secondContent := secondThread["message"].(string)

						if firstContent != "Test thread 2 (newer)" {
							t.Errorf("Expected first thread to be 'Test thread 2 (newer)', got '%s'", firstContent)
						}

						if secondContent != "Test thread 1 (older)" {
							t.Errorf("Expected second thread to be 'Test thread 1 (older)', got '%s'", secondContent)
						}

						t.Logf("✓ Threads correctly ordered by updated_at descending")
					} else {
						t.Logf("Warning: Expected at least 2 threads, got %d", len(threads))
					}
				}

				pagination := data["pagination"]
				if pagination != nil {
					t.Logf("Pagination info: %v", pagination)
				}
			}
		})
	}
}
