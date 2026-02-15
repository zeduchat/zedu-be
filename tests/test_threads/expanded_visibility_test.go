package test_threads

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestGetUserOrgThreadsExpanded(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// 1. Setup 3 Users
	userData := []models.CreateUserRequestModel{
		{
			Email:       fmt.Sprintf("usera%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 7999999999)),
			FirstName:   "User",
			LastName:    "A",
			Password:    "password",
			UserName:    fmt.Sprintf("user_a%v", currUUID),
		},
		{
			Email:       fmt.Sprintf("userb%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(8000000000, 8999999999)),
			FirstName:   "User",
			LastName:    "B",
			Password:    "password",
			UserName:    fmt.Sprintf("user_b%v", currUUID),
		},
		{
			Email:       fmt.Sprintf("userc%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(9000000000, 9999999999)),
			FirstName:   "User",
			LastName:    "C",
			Password:    "password",
			UserName:    fmt.Sprintf("user_c%v", currUUID),
		},
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

	tokens := make([]string, 3)
	userIDs := make([]string, 3)

	for i, ud := range userData {
		r := gin.Default()
		tst.SignupUser(t, r, authCtrl, ud, false)
		loginData := models.LoginRequestModel{Email: ud.Email, Password: ud.Password}
		tokens[i] = tst.GetLoginToken(t, r, authCtrl, loginData)
		userIDs[i] = tst.GetUserIDFromToken(t, tokens[i], db)
	}

	r := gin.Default()
	userID_A, userID_B, userID_C := userIDs[0], userIDs[1], userIDs[2]
	token_A, token_B, token_C := tokens[0], tokens[1], tokens[2]

	// 2. Setup Org and Channel
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	threadCtrl := thread.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("TestTeam%s", currUUID),
		Description: "Multi-user test org",
		Email:       userData[0].Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}

	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, token_A)

	createChannelsData := models.CreateChannelsRequest{
		Name:           "General",
		Username:       "GeneralChannel",
		OrganisationID: orgId,
		Description:    "General discussion",
	}

	channelId, _ := tst.CreateChannels(t, r, channelCtrl, db, createChannelsData, token_A)

	for _, uid := range []string{userID_B, userID_C} {
		err := db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?) ON CONFLICT DO NOTHING", uid, orgId).Error
		if err != nil {
			t.Fatalf("Failed to add user %s to org: %v", uid, err)
		}
	}

	for _, i := range []int{1, 2} {
		uc := models.UserChannels{
			ChannelsID: channelId,
			UserID:     userIDs[i],
			Username:   userData[i].UserName,
		}
		err := db.Postgresql.Create(&uc).Error
		if err != nil {
			t.Fatalf("Failed to add user %s to channel: %v", userIDs[i], err)
		}
	}

	// 3. Create Threads for Scenarios
	now := time.Now().UTC()

	// Scenario 1: User A creates a thread.
	thread1 := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         userID_A,
		Status:         "success",
		Type:           "thread",
		Content:        "Thread by User A",
		Username:       userData[0].UserName,
		ChannelName:    "General",
		CreatedAt:      now.Add(-2 * time.Hour),
		UpdatedAt:      now.Add(-2 * time.Hour),
		LastReply:      now.Add(-2 * time.Hour),
		Messages: []models.MessageDocument{
			{
				ID:        utility.GenerateUUID(),
				Content:   "Hi from A",
				UserID:    userID_A,
				ThreadID:  uuid.Nil,
				CreatedAt: now.Add(-4 * time.Hour),
			},
		},
	}
	thread1.Messages[0].ThreadID = uuid.FromStringOrNil(thread1.ID)

	// Scenario 2: User A mentions User B.
	thread2 := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channelId,
		OrganisationID: orgId,
		UserId:         userID_A,
		Status:         "success",
		Type:           "thread",
		Content:        "Hey @UserB",
		Username:       userData[0].UserName,
		ChannelName:    "General",
		CreatedAt:      now.Add(-3 * time.Hour),
		UpdatedAt:      now.Add(-3 * time.Hour),
		Mentions: []models.Mention{
			{Type: "user", ID: userID_B},
		},
		Messages: []models.MessageDocument{
			{
				ID:        utility.GenerateUUID(),
				Content:   "Mentioning B",
				UserID:    userID_A,
				Username:  userData[0].UserName,
				ThreadID:  uuid.Nil,
				CreatedAt: now.Add(-3 * time.Hour),
			},
		},
	}
	thread2.Messages[0].ThreadID = uuid.FromStringOrNil(thread2.ID)

	// Scenario 3: User C comments on User A's thread (Thread 1).
	commentByC := models.MessageDocument{
		ID:        utility.GenerateUUID(),
		Content:   "User C commenting",
		UserID:    userID_C,
		Username:  userData[2].UserName,
		ThreadID:  uuid.FromStringOrNil(thread1.ID),
		CreatedAt: now.Add(-2 * time.Hour),
	}
	thread1.Messages = append(thread1.Messages, commentByC)

	// Scenario 4: User B mentions User C in a reply on User A's thread (Thread 1).
	replyByB := models.MessageDocument{
		ID:        utility.GenerateUUID(),
		Content:   "Hey @UserC here",
		UserID:    userID_B,
		Username:  userData[1].UserName,
		ThreadID:  uuid.FromStringOrNil(thread1.ID),
		CreatedAt: now.Add(-1 * time.Hour),
		Mentions: []models.Mention{
			{Type: "user", ID: userID_C},
		},
	}
	thread1.Messages = append(thread1.Messages, replyByB)
	// Add mention to thread doc too as it would be updated by the service
	thread1.Mentions = append(thread1.Mentions, models.Mention{Type: "user", ID: userID_C})

	// Save all to ES
	for _, td := range []models.ThreadDocument{thread1, thread2} {
		if err := td.CreateThread(db, logger); err != nil {
			t.Fatalf("Failed to create test thread: %v", err)
		}
	}

	time.Sleep(2 * time.Second)

	// 4. Test Runs for each User
	tests := []struct {
		Name          string
		Token         string
		ExpectedCount int
		ThreadIDs     []string
	}{
		{
			Name:          "User A visibility (all 3 threads)",
			Token:         token_A,
			ExpectedCount: 3,
			ThreadIDs:     []string{thread1.ID, thread2.ID},
		},
		{
			Name:          "User B visibility (both threads: mentioned in 2, participant in 1)",
			Token:         token_B,
			ExpectedCount: 2,
			ThreadIDs:     []string{thread1.ID, thread2.ID},
		},
		{
			Name:          "User C visibility (only thread 2: commented and mentioned)",
			Token:         token_C,
			ExpectedCount: 2,
			ThreadIDs:     []string{thread1.ID},
		},
	}

	for _, test := range tests {
		r := gin.Default()
		threadUrl := r.Group("/api/v1/threads", middleware.Authorize(db.Postgresql))
		{
			threadUrl.GET("/organisations/:org_id", threadCtrl.GetAllUserOrgThreads)
		}

		t.Run(test.Name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/organisations/%s", orgId), nil)
			req.Header.Set("Authorization", "Bearer "+test.Token)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			tst.AssertStatusCode(t, rr.Code, http.StatusOK)

			if rr.Code != http.StatusOK {
				return
			}

			data := tst.ParseResponse(rr)
			threads, ok := data["data"].([]interface{})
			if !ok {
				t.Fatalf("Expected data to be []interface{}, got %T", data["data"])
			}

			if len(threads) != test.ExpectedCount {
				t.Errorf("Expected %d threads, got %d", test.ExpectedCount, len(threads))
			}

			for _, expectedID := range test.ThreadIDs {
				found := false
				for _, threadGroup := range threads {
					tg := threadGroup.(map[string]interface{})
					msgs := tg["thread_messages"].([]interface{})
					if len(msgs) > 0 {
						if msgs[0].(map[string]interface{})["thread_id"].(string) == expectedID {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("Thread ID %s not found in user response", expectedID)
				}
			}
		})
	}
}
