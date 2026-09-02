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
	serviceThread "github.com/hngprojects/telex_be/services/thread"
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

	// Create a second organisation to test boundary isolation
	createOrgData2 := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("TestTeam2%s", currUUID),
		Description: "Some Random description 2",
		Email:       fmt.Sprintf("testuser2%v@qa.team", currUUID),
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgId2, _, _ := tst.CreateOrganisation(t, gin.Default(), db, orgCtrl, createOrgData2, token)

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

	// Create a channel in the second organisation and add the user to it
	createChannelsData2 := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("TestChannels2%s", utility.GenerateUUID()),
		Username:       fmt.Sprintf("Mr2%sChannels", utility.GenerateUUID()),
		OrganisationID: orgId2,
		Description:    "Some Random description 2",
	}
	channelId2, _ := tst.CreateChannels(t, gin.Default(), channelCtrl, db, createChannelsData2, token)

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
		UserId:         otherUserID,
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

	threadOtherOrg := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channelId2,
		OrganisationID: orgId2,
		UserId:         userID,
		Status:         "success",
		Type:           "thread",
		Content:        "Thread from other org",
		Username:       userSignUpData.UserName,
		ChannelName:    createChannelsData2.Name,
		CreatedAt:      now.Add(-3 * time.Hour),
		UpdatedAt:      now.Add(-3 * time.Hour),
		Messages: []models.MessageDocument{
			{
				ID:         utility.GenerateUUID(),
				Content:    "Thread from other org",
				UserID:     userID,
				Username:   userSignUpData.UserName,
				ThreadID:   uuid.Nil,
				CreatedAt:  now.Add(-3 * time.Hour),
				ChannelsID: channelId2,
			},
		},
	}
	threadOtherOrg.Messages[0].ThreadID = uuid.FromStringOrNil(threadOtherOrg.ID)

	for _, td := range []models.ThreadDocument{thread1, thread2, thread3, mentionThread, threadOtherOrg} {
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
		{
			Name:         "User not part of organisation",
			ExpectedCode: http.StatusForbidden,
			Message:      "user is not a member of this organisation",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/threads/organisations/%s", orgId2)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + otherUserToken,
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

				dataObj, ok := dataField.(map[string]interface{})
				if !ok {
					t.Fatalf("Expected data to be map[string]interface{}, got %T", dataField)
				}

				threads, ok := dataObj["threads"].([]interface{})
				if !ok {
					t.Fatalf("Expected threads field to be an array, got %T", dataObj["threads"])
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
					if content == "Thread from other org" {
						t.Errorf("Leaked thread from other org: found thread with content 'Thread from other org'")
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

func TestGetOrgThreadsUnseenCountBeforeAndAfterRead(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	userAData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("usera_unseen%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 7999999999)),
		FirstName:   "User",
		LastName:    "A",
		Password:    "password",
		UserName:    fmt.Sprintf("user_a_unseen%v", currUUID),
	}
	userBData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("userb_unseen%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(8000000000, 8999999999)),
		FirstName:   "User",
		LastName:    "B",
		Password:    "password",
		UserName:    fmt.Sprintf("user_b_unseen%v", currUUID),
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

	rA := gin.Default()
	tst.SignupUser(t, rA, authCtrl, userAData, false)
	tst.SignupUser(t, gin.Default(), authCtrl, userBData, false)

	tokenA := tst.GetLoginToken(t, rA, authCtrl, models.LoginRequestModel{Email: userAData.Email, Password: userAData.Password})
	tokenB := tst.GetLoginToken(t, gin.Default(), authCtrl, models.LoginRequestModel{Email: userBData.Email, Password: userBData.Password})
	userIDA := tst.GetUserIDFromToken(t, tokenA, db)
	userIDB := tst.GetUserIDFromToken(t, tokenB, db)

	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	threadCtrl := thread.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	orgId, _, _ := tst.CreateOrganisation(t, rA, db, orgCtrl, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("UnseenOrg%s", currUUID),
		Description: "Unseen org test",
		Email:       userAData.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}, tokenA)

	channelId, _ := tst.CreateChannels(t, rA, channelCtrl, db, models.CreateChannelsRequest{
		Name:           "UnseenChannel",
		Username:       "UnseenChan",
		OrganisationID: orgId,
		Description:    "Channel for unseen test",
	}, tokenA)

	_ = db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?) ON CONFLICT DO NOTHING", userIDB, orgId).Error
	_ = db.Postgresql.Create(&models.UserChannels{ChannelsID: channelId, UserID: userIDB, Username: userBData.UserName, OrgId: orgId}).Error

	now := time.Now().UTC()
	thread1ID := utility.GenerateUUID()
	thread2ID := utility.GenerateUUID()

	thread1 := models.ThreadDocument{
		ID:             thread1ID,
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         userIDA,
		Status:         "success",
		Type:           "thread",
		Content:        "Thread 1 Root",
		Username:       userAData.UserName,
		ChannelName:    "UnseenChannel",
		CreatedAt:      now.Add(-2 * time.Hour),
		UpdatedAt:      now.Add(-2 * time.Hour),
		LastReply:      now.Add(-2 * time.Hour),
		Messages: []models.MessageDocument{
			{
				ID:         utility.GenerateUUID(),
				Content:    "Root message 1",
				UserID:     userIDA,
				Username:   userAData.UserName,
				ThreadID:   uuid.FromStringOrNil(thread1ID),
				CreatedAt:  now.Add(-2 * time.Hour),
				ChannelsID: channelId,
			},
		},
	}

	thread2 := models.ThreadDocument{
		ID:             thread2ID,
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         userIDA,
		Status:         "success",
		Type:           "thread",
		Content:        "Thread 2 Root",
		Username:       userAData.UserName,
		ChannelName:    "UnseenChannel",
		CreatedAt:      now.Add(-1 * time.Hour),
		UpdatedAt:      now.Add(-1 * time.Hour),
		LastReply:      now.Add(-1 * time.Hour),
		Messages: []models.MessageDocument{
			{
				ID:         utility.GenerateUUID(),
				Content:    "Root message 2",
				UserID:     userIDA,
				Username:   userAData.UserName,
				ThreadID:   uuid.FromStringOrNil(thread2ID),
				CreatedAt:  now.Add(-1 * time.Hour),
				ChannelsID: channelId,
			},
		},
	}

	_ = thread1.CreateThread(db, logger)
	_ = thread2.CreateThread(db, logger)
	_, _ = thread1.Messages[0].CreateMessage(db, logger)
	_, _ = thread2.Messages[0].CreateMessage(db, logger)

	_ = serviceThread.ProcessThreadUnseenForParticipants(db.Postgresql, logger, orgId, thread1ID, []string{userIDA, userIDB})
	_ = serviceThread.ProcessThreadUnseenForParticipants(db.Postgresql, logger, orgId, thread2ID, []string{userIDA, userIDB})

	r := gin.Default()
	r.GET("/api/v1/threads/organisations/:org_id", middleware.Authorize(db.Postgresql), threadCtrl.GetAllUserOrgThreads)
	r.GET("/api/v1/threads/:thread_id/channels/:channel_id", middleware.Authorize(db.Postgresql), threadCtrl.GetUserSingleThreads)

	reqBefore, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/organisations/%s", orgId), nil)
	reqBefore.Header.Set("Authorization", "Bearer "+tokenA)
	rrBefore := httptest.NewRecorder()
	r.ServeHTTP(rrBefore, reqBefore)

	tst.AssertStatusCode(t, rrBefore.Code, http.StatusOK)
	resBefore := tst.ParseResponse(rrBefore)
	dataBefore := resBefore["data"].(map[string]interface{})
	unseenBefore := int64(dataBefore["unseen_thread_count"].(float64))

	if unseenBefore != 2 {
		t.Errorf("Expected unseen_thread_count BEFORE reading replies to be 2, got %d", unseenBefore)
	}

	reqRead, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/%s/channels/%s", thread1ID, channelId), nil)
	reqRead.Header.Set("Authorization", "Bearer "+tokenA)
	rrRead := httptest.NewRecorder()
	r.ServeHTTP(rrRead, reqRead)
	tst.AssertStatusCode(t, rrRead.Code, http.StatusOK)

	time.Sleep(200 * time.Millisecond)

	reqAfter, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/organisations/%s", orgId), nil)
	reqAfter.Header.Set("Authorization", "Bearer "+tokenA)
	rrAfter := httptest.NewRecorder()
	r.ServeHTTP(rrAfter, reqAfter)

	tst.AssertStatusCode(t, rrAfter.Code, http.StatusOK)
	resAfter := tst.ParseResponse(rrAfter)
	dataAfter := resAfter["data"].(map[string]interface{})
	unseenAfter := int64(dataAfter["unseen_thread_count"].(float64))

	if unseenAfter != 1 {
		t.Errorf("Expected unseen_thread_count AFTER reading thread 1 replies to be 1, got %d", unseenAfter)
	}
}

func TestGetUserOrgThreads_SortedByLastReply(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	userAData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("usera_sort%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 7999999999)),
		FirstName:   "User",
		LastName:    "Sort",
		Password:    "password",
		UserName:    fmt.Sprintf("user_a_sort%v", currUUID),
	}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, userAData, false)
	token := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: userAData.Email, Password: userAData.Password})
	userID := tst.GetUserIDFromToken(t, token, db)

	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	threadCtrl := thread.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("SortOrg%s", currUUID),
		Description: "Sort org test",
		Email:       userAData.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}, token)

	channelId, _ := tst.CreateChannels(t, r, channelCtrl, db, models.CreateChannelsRequest{
		Name:           "SortChannel",
		Username:       "SortChan",
		OrganisationID: orgId,
		Description:    "Channel for sort test",
	}, token)

	now := time.Now().UTC()
	threadOldID := utility.GenerateUUID()
	threadNewID := utility.GenerateUUID()

	threadOld := models.ThreadDocument{
		ID:             threadOldID,
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         userID,
		Status:         "success",
		Type:           "thread",
		Content:        "Older Thread Root",
		Username:       userAData.UserName,
		ChannelName:    "SortChannel",
		CreatedAt:      now.Add(-5 * time.Hour),
		UpdatedAt:      now.Add(-5 * time.Hour),
		LastReply:      now.Add(-5 * time.Hour),
		Messages: []models.MessageDocument{
			{
				ID:         utility.GenerateUUID(),
				Content:    "Root message old",
				UserID:     userID,
				Username:   userAData.UserName,
				ThreadID:   uuid.FromStringOrNil(threadOldID),
				CreatedAt:  now.Add(-5 * time.Hour),
				ChannelsID: channelId,
			},
		},
	}

	threadNew := models.ThreadDocument{
		ID:             threadNewID,
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         userID,
		Status:         "success",
		Type:           "thread",
		Content:        "Newer Thread Root",
		Username:       userAData.UserName,
		ChannelName:    "SortChannel",
		CreatedAt:      now.Add(-1 * time.Hour),
		UpdatedAt:      now.Add(-1 * time.Hour),
		LastReply:      now.Add(-1 * time.Hour),
		Messages: []models.MessageDocument{
			{
				ID:         utility.GenerateUUID(),
				Content:    "Root message new",
				UserID:     userID,
				Username:   userAData.UserName,
				ThreadID:   uuid.FromStringOrNil(threadNewID),
				CreatedAt:  now.Add(-1 * time.Hour),
				ChannelsID: channelId,
			},
		},
	}

	_ = threadOld.CreateThread(db, logger)
	_ = threadNew.CreateThread(db, logger)
	_, _ = threadOld.Messages[0].CreateMessage(db, logger)
	_, _ = threadNew.Messages[0].CreateMessage(db, logger)

	// Add recent reply message to threadOld
	replyMsg := models.MessageDocument{
		ID:             utility.GenerateUUID(),
		Content:        "Brand new reply to older thread",
		UserID:         userID,
		Username:       userAData.UserName,
		ThreadID:       uuid.FromStringOrNil(threadOldID),
		CreatedAt:      now.Add(10 * time.Minute),
		UpdatedAt:      now.Add(10 * time.Minute),
		ChannelsID:     channelId,
		OrganisationID: orgId,
	}
	_, _ = replyMsg.CreateMessage(db, logger)

	time.Sleep(2 * time.Second)

	router := gin.Default()
	router.GET("/api/v1/threads/organisations/:org_id", middleware.Authorize(db.Postgresql), threadCtrl.GetAllUserOrgThreads)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/organisations/%s", orgId), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)
	res := tst.ParseResponse(rr)
	dataObj := res["data"].(map[string]interface{})
	threads := dataObj["threads"].([]interface{})

	if len(threads) < 2 {
		t.Fatalf("Expected at least 2 threads, got %d", len(threads))
	}

	firstGroup := threads[0].(map[string]interface{})
	firstMsgs := firstGroup["thread_messages"].([]interface{})
	firstThreadDoc := firstMsgs[0].(map[string]interface{})

	if firstThreadDoc["thread_id"] != threadOldID {
		t.Errorf("Expected oldest thread (%s) with newest reply to be returned first, got thread_id: %v", threadOldID, firstThreadDoc["thread_id"])
	}
}
