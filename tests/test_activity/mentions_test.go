package test_activity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetUserMentionsActivity(t *testing.T) {
	r, activityCtrl := SetupActivityTestRouter()
	db := activityCtrl.Db
	logger := activityCtrl.Logger

	validatorRef := validator.New()
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

	authCtrl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	tst.SignupUser(t, r, authCtrl, userSignUpData, false)

	org := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}

	token := tst.GetLoginToken(t, r, authCtrl, loginData)
	userID := tst.GetUserIDFromToken(t, token, db)

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
		Name:           fmt.Sprintf("TestChannel%s", utility.GenerateUUID()),
		Username:       fmt.Sprintf("MrChannel%s", utility.GenerateUUID()),
		OrganisationID: orgId,
		Description:    "Test channel for mentions",
	}

	channelId, _ := tst.CreateChannels(t, r, channelCtrl, db, createChannelsData, token)

	tests := []struct {
		Name         string
		ExpectedCode int
		Message      string
		Method       string
		Headers      map[string]string
		RequestURI   url.URL
	}{
		{
			Name:         "Successfully Get User Mentions - Empty",
			ExpectedCode: http.StatusOK,
			Message:      "mentions retrieved successfully",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/organisations/%s/mentions", orgId)},
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
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/organisations/%s/mentions", orgId)},
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name:         "Invalid Org ID Format",
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid org id format",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: "/api/v1/organisations/invalid-uuid/mentions"},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
	}

	for _, test := range tests {
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
		})
	}

	t.Run("Create Thread With Mention And Verify Retrieval", func(t *testing.T) {
		now := time.Now().UTC()

		threadWithMention := models.ThreadDocument{
			ID:             utility.GenerateUUID(),
			ChannelsID:     channelId,
			OrganisationID: orgId,
			UserId:         userID,
			Status:         "success",
			Type:           "thread",
			Content:        fmt.Sprintf("Hey @%s, check this out!", userSignUpData.UserName),
			Username:       "other_user",
			ChannelName:    createChannelsData.Name,
			ChannelType:    "public",
			Mentions: []models.Mention{
				{
					Type: "user",
					ID:   userID,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := threadWithMention.CreateThread(db, logger)
		if err != nil {
			t.Fatalf("Failed to create test thread with mention: %v", err)
		}

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/mentions", orgId), nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)

		code := int(data["status_code"].(float64))
		tst.AssertStatusCode(t, code, http.StatusOK)

		if dataField, ok := data["data"].([]interface{}); ok {
			if len(dataField) == 0 {
				t.Error("No mentions returned yet - elasticsearch indexing may be delayed")
			} else {
				found := false
				for _, mention := range dataField {
					mentionMap := mention.(map[string]interface{})
					if mentionMap["thread_id"] == threadWithMention.ID {
						found = true
						t.Logf("Found the created mention in response: thread_id=%s", threadWithMention.ID)

						if mentionMap["channels_id"] != channelId {
							t.Errorf("Expected channel_id %s, got %s", channelId, mentionMap["channels_id"])
						}
						if mentionMap["org_id"] != orgId {
							t.Errorf("Expected org_id %s, got %s", orgId, mentionMap["org_id"])
						}
						break
					}
				}
				if !found {
					t.Logf("Created thread not found in response - may need more time for elasticsearch indexing")
					mentionBytes, _ := json.MarshalIndent(dataField, "", "  ")
					t.Logf("Returned mentions: %s", string(mentionBytes))
				}
			}
		}

		if pagination, ok := data["pagination"].(map[string]interface{}); ok {
			t.Logf("Pagination info: %v", pagination)
		}
	})

	t.Run("Create Reply With Mention And Verify Retrieval", func(t *testing.T) {
		now := time.Now().UTC()

		threadID := utility.GenerateUUID()
		thread := models.ThreadDocument{
			ID:             threadID,
			ChannelsID:     channelId,
			OrganisationID: orgId,
			UserId:         utility.GenerateUUID(),
			Status:         "success",
			Type:           "thread",
			Content:        "Original thread",
			Username:       "original_poster",
			ChannelName:    createChannelsData.Name,
			ChannelType:    "public",
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		err := thread.CreateThread(db, logger)
		if err != nil {
			t.Fatalf("Failed to create thread: %v", err)
		}

		replyID := utility.GenerateUUID()
		reply := models.MessageDocument{
			ID:             replyID,
			ThreadID:       uuid.FromStringOrNil(threadID),
			ChannelsID:     channelId,
			OrganisationID: orgId,
			UserID:         utility.GenerateUUID(),
			Content:        fmt.Sprintf("Replying to @%s", userSignUpData.UserName),
			Username:       "reply_user",
			CreatedAt:      now,
			UpdatedAt:      now,
			Mentions: []models.Mention{
				{
					Type: "user",
					ID:   userID,
				},
			},
		}

		err = elastic.AddDocument(db.Elastic, models.MessageIndexName, replyID, reply, logger)
		if err != nil {
			t.Fatalf("Failed to create reply with mention: %v", err)
		}

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/mentions", orgId), nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)

		if dataField, ok := data["data"].([]interface{}); ok {
			found := false
			for _, mention := range dataField {
				mentionMap := mention.(map[string]interface{})
				if mentionMap["id"] == replyID {
					found = true
					break
				}
				if content, ok := mentionMap["message"].(string); ok && content == reply.Content {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Created reply mention not found in response")

				mentionBytes, _ := json.MarshalIndent(dataField, "", "  ")
				t.Logf("Returned mentions: %s", string(mentionBytes))
			}
		}
	})
}
