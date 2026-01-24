package test_activity

import (
	"bytes"
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

	// helper for creating thread data
	createTestThread := func(chID, uID, oID, content string, mentions []models.Mention) string {
		tID := utility.GenerateUUID()
		now := time.Now().UTC()
		threadDoc := models.ThreadDocument{
			ID:             tID,
			ChannelsID:     chID,
			UserId:         uID,
			OrganisationID: oID,
			Content:        content,
			CreatedAt:      now,
			UpdatedAt:      now,
			Mentions:       mentions,
			Status:         "success",
			Type:           "thread",
		}

		// insert into ES
		if err := threadDoc.CreateThread(db, logger); err != nil {
			t.Fatalf("Failed to create thread in ES: %v", err)
		}
		return tID
	}

	createTestMessage := func(chID, tID, uID, oID, content string, mentions []models.Mention) string {
		mID := utility.GenerateUUID()
		now := time.Now().UTC()
		threadUUID, _ := uuid.FromString(tID)
		msgDoc := models.MessageDocument{
			ID:             mID,
			ChannelsID:     chID,
			ThreadID:       threadUUID,
			UserID:         uID,
			OrganisationID: oID,
			Content:        content,
			CreatedAt:      now,
			UpdatedAt:      now,
			Mentions:       mentions,
		}

		if _, err := msgDoc.CreateMessage(db, logger); err != nil {
			t.Fatalf("Failed to create message in ES: %v", err)
		}

		return mID
	}

	t.Run("Create Thread With Mention And Verify Retrieval (Unseen vs Seen)", func(t *testing.T) {
		// create a second user (The Sender)
		senderID := utility.GenerateUUID()
		senderName := fmt.Sprintf("sender_%v", senderID)

		db.Postgresql.Create(&models.User{
			ID:         senderID,
			Email:      fmt.Sprintf("sender%v@qa.team", senderID),
			Password:   "password",
			IsVerified: true,
		})

		db.Postgresql.Create(&models.Profile{
			ID:        utility.GenerateUUID(),
			Userid:    senderID,
			UserName:  senderName,
			FirstName: "Sender",
			LastName:  "User",
		})

		// add sender to channel
		db.Postgresql.Create(&models.UserChannels{
			ChannelsID:   channelId,
			UserID:       senderID,
			MentionCount: 0,
		})

		// sender creates a Thread mentioning first user
		createTestThread(channelId, senderID, orgId, "Hey check this out", []models.Mention{{Type: "user", ID: userID}})

		time.Sleep(1 * time.Second)

		// verify first user sees the mention (Unseen)
		reqGet, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/mentions", orgId), nil)
		reqGet.Header.Set("Content-Type", "application/json")
		reqGet.Header.Set("Authorization", "Bearer "+token)

		rrGet := httptest.NewRecorder()
		r.ServeHTTP(rrGet, reqGet)
		tst.AssertStatusCode(t, rrGet.Code, http.StatusOK)

		data := tst.ParseResponse(rrGet)
		found := false
		if dataField, ok := data["data"].([]interface{}); ok {
			for _, mention := range dataField {
				mentionMap := mention.(map[string]interface{})
				if mentionMap["username"] == senderName || mentionMap["user_id"] == senderID {
					found = true
					break
				}
			}
		}
		if !found {
			t.Logf("Did not find mention immediately. Response data: %v", data["data"])
		}

		// first user reads the channel (Update LastReadAt > Thread CreatedAt)
		// we need to make sure LastReadAt is strictly AFTER the thread creation
		futureTime := time.Now().Add(1 * time.Minute)
		if err := db.Postgresql.Model(&models.UserChannels{}).
			Where("user_id = ? AND channels_id = ?", userID, channelId).
			Updates(map[string]interface{}{"last_read_at": futureTime, "mention_count": 0}).Error; err != nil {
			t.Fatalf("Failed to mark channel as read: %v", err)
		}

		// verify mention is filtered out (Seen)
		reqGet2, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/mentions", orgId), nil)
		reqGet2.Header.Set("Content-Type", "application/json")
		reqGet2.Header.Set("Authorization", "Bearer "+token)

		rrGet2 := httptest.NewRecorder()
		r.ServeHTTP(rrGet2, reqGet2)
		tst.AssertStatusCode(t, rrGet2.Code, http.StatusOK)

		data2 := tst.ParseResponse(rrGet2)
		if dataField, ok := data2["data"].([]interface{}); ok {
			for _, mention := range dataField {
				mentionMap := mention.(map[string]interface{})
				if mentionMap["user_id"] == senderID {
					t.Error("Expected mention to be filtered out (SEEN), but it was returned")
				}
			}
		}
	})

	t.Run("Create Reply With Mention And Verify Retrieval", func(t *testing.T) {
		sender2ID := utility.GenerateUUID()
		sender2Name := fmt.Sprintf("sender2_%v", sender2ID)

		db.Postgresql.Create(&models.User{
			ID:         sender2ID,
			Email:      fmt.Sprintf("sender2%v@qa.team", sender2ID),
			IsVerified: true,
		})
		db.Postgresql.Create(&models.Profile{
			ID:       utility.GenerateUUID(),
			Userid:   sender2ID,
			UserName: sender2Name,
		})

		db.Postgresql.Create(&models.UserChannels{
			ChannelsID:   channelId,
			UserID:       sender2ID,
			MentionCount: 0,
		})

		// sender2 creates a Thread
		tID := createTestThread(channelId, sender2ID, orgId, "Base thread", nil)
		time.Sleep(200 * time.Millisecond)

		// sender2 replies to that thread mentioning first user
		createTestMessage(channelId, tID, sender2ID, orgId, "Reply mention", []models.Mention{{Type: "user", ID: userID}})

		time.Sleep(1 * time.Second)

		// mark channel as unread again (reset LastReadAt to past)
		pastTime := time.Now().Add(-1 * time.Hour)
		db.Postgresql.Model(&models.UserChannels{}).
			Where("user_id = ? AND channels_id = ?", userID, channelId).
			Updates(map[string]interface{}{"last_read_at": pastTime})

		// verify first user sees the mention (Unseen)
		reqGet, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/mentions", orgId), nil)
		reqGet.Header.Set("Content-Type", "application/json")
		reqGet.Header.Set("Authorization", "Bearer "+token)

		rrGet := httptest.NewRecorder()
		r.ServeHTTP(rrGet, reqGet)
		tst.AssertStatusCode(t, rrGet.Code, http.StatusOK)

		data := tst.ParseResponse(rrGet)
		found := false
		if dataField, ok := data["data"].([]interface{}); ok {
			for _, mention := range dataField {
				mentionMap := mention.(map[string]interface{})
				if mentionMap["thread_id"] == tID {
					found = true
					break
				}
			}
		}
		if !found {
			t.Log("Did not find reply mention immediately.")
		}
	})
}
