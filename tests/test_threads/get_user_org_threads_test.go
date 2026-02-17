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
	"github.com/gofrs/uuid"

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

func TestGetUserOrgThreads(t *testing.T) {
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

	otherUserSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser2%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test2",
		LastName:    "user2",
		Password:    "password",
		UserName:    fmt.Sprintf("test_other_username%v", currUUID),
	}

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	otherUserLoginData := models.LoginRequestModel{
		Email:    otherUserSignUpData.Email,
		Password: otherUserSignUpData.Password,
	}

	authCtrl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, userSignUpData, false)
	tst.SignupUser(t, gin.Default(), authCtrl, otherUserSignUpData, false)

	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	threadCtrl := thread.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	token := tst.GetLoginToken(t, r, authCtrl, loginData)
	otherUserToken := tst.GetLoginToken(t, gin.Default(), authCtrl, otherUserLoginData)
	userID := tst.GetUserIDFromToken(t, token, db)
	otherUserID := tst.GetUserIDFromToken(t, otherUserToken, db)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("TestTeam%s", currUUID),
		Description: "Some Random description",
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}

	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, token)

	createChannelsData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("TestChannels%s", utility.GenerateUUID()),
		Username:       fmt.Sprintf("Mr%sChannels", utility.GenerateUUID()),
		OrganisationID: orgId,
		Description:    "Some Random description",
	}

	channelId, _ := tst.CreateChannels(t, r, channelCtrl, db, createChannelsData, token)
	uc := models.UserChannels{
		ChannelsID: channelId,
		UserID:     otherUserID,
		Username:   otherUserSignUpData.UserName,
		OrgId:      orgId,
	}
	err := db.Postgresql.Create(&uc).Error
	if err != nil {
		t.Fatalf("Failed to add user %s to channel: %v", otherUserID, err)
	}

	now := time.Now().UTC()

	thread1 := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         userID,
		Status:         "success",
		Type:           "thread",
		Content:        "Oldest thread",
		Username:       userSignUpData.UserName,
		ChannelName:    createChannelsData.Name,
		CreatedAt:      now.Add(-2 * time.Hour),
		UpdatedAt:      now.Add(-2 * time.Hour),
		Messages: []models.MessageDocument{
			{
				ID:         utility.GenerateUUID(),
				Content:    "msg in oldest",
				UserID:     userID,
				Username:   userSignUpData.UserName,
				ThreadID:   uuid.Nil,
				CreatedAt:  now.Add(-2 * time.Hour),
				ChannelsID: channelId,
			},
		},
	}
	thread1.Messages[0].ThreadID = uuid.FromStringOrNil(thread1.ID)

	thread2 := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         userID,
		Status:         "success",
		Type:           "thread",
		Content:        "Middle thread",
		Username:       userSignUpData.UserName,
		ChannelName:    createChannelsData.Name,
		CreatedAt:      now.Add(-1 * time.Hour),
		UpdatedAt:      now.Add(-1 * time.Hour),
		Messages: []models.MessageDocument{
			{
				ID:         utility.GenerateUUID(),
				Content:    "msg in middle",
				UserID:     userID,
				Username:   userSignUpData.UserName,
				ThreadID:   uuid.Nil,
				CreatedAt:  now.Add(-1 * time.Hour),
				ChannelsID: channelId,
			},
		},
	}
	thread2.Messages[0].ThreadID = uuid.FromStringOrNil(thread2.ID)

	thread3 := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         userID,
		Status:         "success",
		Type:           "thread",
		Content:        "Newest thread",
		Username:       userSignUpData.UserName,
		ChannelName:    createChannelsData.Name,
		CreatedAt:      now,
		UpdatedAt:      now,
		Messages: []models.MessageDocument{
			{
				ID:         utility.GenerateUUID(),
				Content:    "msg in newest",
				UserID:     userID,
				Username:   userSignUpData.UserName,
				ThreadID:   uuid.Nil,
				CreatedAt:  now,
				ChannelsID: channelId,
			},
		},
	}
	thread3.Messages[0].ThreadID = uuid.FromStringOrNil(thread3.ID)

	mentionThread := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         "00000000-0000-0000-0000-000000000000",
		Status:         "success",
		Type:           "thread",
		Content:        "Thread with mention",
		Username:       "other_user",
		ChannelName:    createChannelsData.Name,
		CreatedAt:      now.Add(1 * time.Hour),
		UpdatedAt:      now.Add(1 * time.Hour),
		Mentions: []models.Mention{
			{
				Type: "user",
				ID:   userID,
			},
		},
		Messages: []models.MessageDocument{
			{
				ID:         utility.GenerateUUID(),
				Content:    "hey you were mentioned",
				UserID:     otherUserID,
				Username:   otherUserSignUpData.UserName,
				ThreadID:   uuid.Nil,
				CreatedAt:  now.Add(1 * time.Hour),
				ChannelsID: channelId,
			},
		},
	}
	mentionThread.Messages[0].ThreadID = uuid.FromStringOrNil(mentionThread.ID)

	for _, td := range []models.ThreadDocument{thread1, thread2, thread3, mentionThread} {
		if err := td.CreateThread(db, logger); err != nil {
			t.Fatalf("Failed to create test thread: %v", err)
		}
		for _, msg := range td.Messages {
			if _, err := msg.CreateMessage(db, logger); err != nil {
				t.Fatalf("Failed to create test message: %v", err)
			}
		}
	}

	time.Sleep(2 * time.Second)

	tests := []struct {
		Name         string
		ExpectedCode int
		Message      string
		Method       string
		Headers      map[string]string
		RequestURI   url.URL
	}{
		{
			Name:         "Successfully retrieve user org threads",
			ExpectedCode: http.StatusOK,
			Message:      "Data retrieved successfully",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/threads/organisations/%s", orgId)},
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
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/threads/organisations/%s", orgId)},
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name:         "Invalid Org ID format",
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid org id format",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: "/api/v1/threads/organisations/not-a-uuid"},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
	}

	for _, test := range tests {
		r := gin.Default()

		threadUrl := r.Group("/api/v1/threads", middleware.Authorize(db.Postgresql))
		{
			threadUrl.GET("/organisations/:org_id", threadCtrl.GetAllUserOrgThreads)
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
				if dataField == nil {
					t.Fatal("Expected data field in response, got nil")
				}

				threads, ok := dataField.([]interface{})
				if !ok {
					t.Fatal("Expected data to be an array")
				}

				t.Logf("Retrieved %d thread groups", len(threads))

				if len(threads) < 3 {
					t.Errorf("Expected at least 3 thread groups (user-created), got %d", len(threads))
				}

				foundMention := false
				for _, threadGroup := range threads {
					tg, ok := threadGroup.(map[string]interface{})
					if !ok {
						continue
					}
					threadMsgs, ok := tg["thread_messages"].([]interface{})
					if !ok || len(threadMsgs) == 0 {
						continue
					}
					firstThread := threadMsgs[0].(map[string]interface{})
					content := firstThread["message"].(string)
					if content == "Thread with mention" {
						foundMention = true
						t.Logf("✓ Mention thread found in response")
					}
				}

				if !foundMention {
					t.Errorf("Expected to find 'Thread with mention' (user mentioned via mentions field) in results")
				}

				if len(threads) >= 2 {
					allDescending := true
					for i := 0; i < len(threads)-1; i++ {
						currGroup := threads[i].(map[string]interface{})
						nextGroup := threads[i+1].(map[string]interface{})

						currMsgs := currGroup["thread_messages"].([]interface{})
						nextMsgs := nextGroup["thread_messages"].([]interface{})

						if len(currMsgs) == 0 || len(nextMsgs) == 0 {
							continue
						}

						currThread := currMsgs[0].(map[string]interface{})
						nextThread := nextMsgs[0].(map[string]interface{})

						currTime := currThread["created_at"].(string)
						nextTime := nextThread["created_at"].(string)

						currParsed, _ := time.Parse(time.RFC3339Nano, currTime)
						nextParsed, _ := time.Parse(time.RFC3339Nano, nextTime)

						if currParsed.Before(nextParsed) {
							allDescending = false
							t.Errorf("Thread at index %d (created_at: %s) is before thread at index %d (created_at: %s) — expected descending order",
								i, currTime, i+1, nextTime)
						}
					}

					if allDescending {
						t.Logf("✓ All threads are in descending created_at order")
					}
				}

				for _, threadGroup := range threads {
					tg, ok := threadGroup.(map[string]interface{})
					if !ok {
						continue
					}
					channelName, ok := tg["channel_name"].(string)
					if !ok || channelName == "" {
						t.Errorf("Expected non-empty channel_name in thread group response")
					}
				}

				for _, threadGroup := range threads {
					tg, ok := threadGroup.(map[string]interface{})
					if !ok {
						continue
					}
					participants, ok := tg["participants"].(string)
					if !ok {
						t.Errorf("Expected participants field to be a string")
					}
					if participants == "" {
						t.Errorf("Expected non-empty participants string")
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
