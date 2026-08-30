package test_search

import (
	"encoding/json"
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
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestSearchScopeAndPagination(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("searchuser1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "SearchUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("searchuser1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("searchuser2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "SearchUser",
		LastName:    "Two",
		Password:    "password",
		UserName:    fmt.Sprintf("searchuser2_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}

	loginData2 := models.LoginRequestModel{
		Email:    user2SignUpData.Email,
		Password: user2SignUpData.Password,
	}

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	authRouter := gin.Default()
	tst.SignupUser(t, authRouter, authController, user1SignUpData, false)

	authRouter2 := gin.Default()
	tst.SignupUser(t, authRouter2, authController, user2SignUpData, false)

	loginRouter := gin.Default()
	token1 := tst.GetLoginToken(t, loginRouter, authController, loginData1)

	loginRouter2 := gin.Default()
	token2 := tst.GetLoginToken(t, loginRouter2, authController, loginData2)

	var user1, user2 models.User
	if err := db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error; err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}
	if err := db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error; err != nil {
		t.Fatalf("Failed to get user2: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	var role models.OrgRole
	if err := db.Postgresql.First(&role).Error; err != nil {
		t.Fatalf("Failed to fetch any OrgRole: %v", err)
	}

	// Make user2 join org
	userOrg := models.OrgUserManagement{
		UserID:         user2.ID,
		OrganisationID: org.ID,
		RoleID:         role.ID,
		Status:         "active",
	}
	if err := db.Postgresql.Create(&userOrg).Error; err != nil {
		t.Fatalf("Failed to add user2 to organization: %v", err)
	}

	r := SetupSearchRouter(db, logger)

	t.Run("Access control - regular channels", func(t *testing.T) {
		// Channel 1 Public, user 1 and user 2 are members
		channel1ID := utility.GenerateUUID()
		channel1 := models.Channels{
			ID:             channel1ID,
			Name:           "Public Channel",
			OrganisationID: org.ID,
			OwnerId:        user1.ID,
		}
		db.Postgresql.Create(&channel1)

		userChannel1_1 := models.UserChannels{
			ChannelsID: channel1ID,
			UserID:     user1.ID,
		}
		db.Postgresql.Create(&userChannel1_1)

		userChannel1_2 := models.UserChannels{
			ChannelsID: channel1ID,
			UserID:     user2.ID,
		}
		db.Postgresql.Create(&userChannel1_2)

		// Channel 2 Private, user 1 is member, user 2 is NOT
		channel2ID := utility.GenerateUUID()
		channel2 := models.Channels{
			ID:             channel2ID,
			Name:           "Private Channel",
			OrganisationID: org.ID,
			OwnerId:        user1.ID,
			IsPrivate:      true,
		}
		db.Postgresql.Create(&channel2)

		userChannel2_1 := models.UserChannels{
			ChannelsID: channel2ID,
			UserID:     user1.ID,
		}
		db.Postgresql.Create(&userChannel2_1)

		// Create threads containing messages in both channels
		threadId1 := utility.GenerateUUID()
		thread1 := map[string]any{
			"thread_id":    threadId1,
			"channels_id":  channel1ID,
			"channel_name": "Public Channel",
			"user_id":      user1.ID,
			"user_name":    user1SignUpData.UserName,
			"org_id":       org.ID,
			"message":      "Hello from the public channel keywordPublicChannel123",
			"created_at":   time.Now().Format(time.RFC3339),
			"updated_at":   time.Now().Format(time.RFC3339),
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId1, thread1, logger)

		threadId2 := utility.GenerateUUID()
		thread2 := map[string]any{
			"thread_id":    threadId2,
			"channels_id":  channel2ID,
			"channel_name": "Private Channel",
			"user_id":      user1.ID,
			"org_id":       org.ID,
			"message":      "Secret message in private channel keywordPublicChannel123",
			"created_at":   time.Now().Format(time.RFC3339),
			"updated_at":   time.Now().Format(time.RFC3339),
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId2, thread2, logger)

		time.Sleep(2 * time.Second) // wait for ES index refresh

		req1, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=keywordPublicChannel123", org.ID), nil)
		req1.Header.Set("Authorization", "Bearer "+token1)
		rr1 := httptest.NewRecorder()
		r.ServeHTTP(rr1, req1)

		if rr1.Code != http.StatusOK {
			t.Errorf("Expected status 200 for User 1, got %d", rr1.Code)
		}

		var resp1 map[string]any
		json.Unmarshal(rr1.Body.Bytes(), &resp1)
		data1 := resp1["data"].([]interface{})
		if len(data1) != 2 {
			t.Errorf("User 1 should see 2 results, got %d", len(data1))
		}

		req2, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=keywordPublicChannel123", org.ID), nil)
		req2.Header.Set("Authorization", "Bearer "+token2)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Errorf("Expected status 200 for User 2, got %d", rr2.Code)
		}

		var resp2 map[string]any
		json.Unmarshal(rr2.Body.Bytes(), &resp2)
		data2 := resp2["data"].([]interface{})
		if len(data2) != 1 {
			t.Errorf("User 2 should see 1 result, got %d", len(data2))
		} else {
			resObj := data2[0].(map[string]any)
			channelInfo := resObj["channel"].(map[string]any)
			if channelInfo["channel_id"] != channel1ID {
				t.Errorf("User 2 should only see public channel result")
			}
			if channelInfo["channel_name"] != channel1.Name {
				t.Errorf("User 2 should see correct channel name. Expected '%s', got '%v'", channel1.Name, channelInfo["channel_name"])
			}
		}
	})

	t.Run("DM search", func(t *testing.T) {
		dmChannelID := utility.GenerateUUID()
		participantID := user2.ID
		dmChannel := models.DmChannels{
			ID:            utility.GenerateUUID(),
			UserId:        user1.ID,
			ChannelId:     dmChannelID,
			OrgId:         org.ID,
			ParticipantId: &participantID,
			ChatType:      "user",
			ChannelType:   "dm",
		}
		db.Postgresql.Create(&dmChannel)

		dmChannel2 := models.DmChannels{
			ID:            utility.GenerateUUID(),
			UserId:        user2.ID,
			ChannelId:     dmChannelID,
			OrgId:         org.ID,
			ParticipantId: &user1.ID,
			ChatType:      "user",
			ChannelType:   "dm",
		}
		db.Postgresql.Create(&dmChannel2)

		threadId := utility.GenerateUUID()
		thread := map[string]any{
			"thread_id":   threadId,
			"channels_id": dmChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Hello this is a direct message uniqueKeyword",
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId, thread, logger)

		time.Sleep(2 * time.Second) // wait for ES index refresh

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=uniqueKeyword", org.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		if len(data) != 1 {
			t.Errorf("User 2 should see 1 result from DM, got %d", len(data))
		}
	})

	t.Run("Group DM search", func(t *testing.T) {
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

		groupDMChannel := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          user1.ID,
			ChannelId:       groupDMChannelID,
			OrgId:           org.ID,
			ParticipantHash: participantHash,
			ChatType:        "user",
			ChannelType:     "group_dm",
		}
		db.Postgresql.Create(&groupDMChannel)

		// user 1 & 2
		participant1 := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupDMChannelID,
			UserId:    user1.ID,
			OrgId:     org.ID,
		}
		db.Postgresql.Create(&participant1)

		participant2 := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupDMChannelID,
			UserId:    user2.ID,
			OrgId:     org.ID,
		}
		db.Postgresql.Create(&participant2)

		threadId := utility.GenerateUUID()
		thread := map[string]any{
			"thread_id":   threadId,
			"channels_id": groupDMChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Hello this is a group direct message uniqueGroupKeyword",
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId, thread, logger)

		time.Sleep(2 * time.Second) // wait for ES index refresh

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=uniqueGroupKeyword", org.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		if len(data) != 1 {
			t.Errorf("User 2 should see 1 result from Group DM, got %d", len(data))
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           "Pagination Channel",
			OrganisationID: org.ID,
			OwnerId:        user1.ID,
		}
		db.Postgresql.Create(&channel)

		userChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     user1.ID,
		}
		db.Postgresql.Create(&userChannel)

		for i := range 5 {
			threadId := utility.GenerateUUID()
			thread := map[string]any{
				"thread_id":   threadId,
				"channels_id": channelID,
				"user_id":     user1.ID,
				"org_id":      org.ID,
				"message":     fmt.Sprintf("Pagination message %d commonWord", i),
				"created_at":  time.Now().Format(time.RFC3339),
				"updated_at":  time.Now().Format(time.RFC3339),
			}
			elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId, thread, logger)
		}

		time.Sleep(2 * time.Second)

		req1, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=commonWord&page=1&limit=2", org.ID), nil)
		req1.Header.Set("Authorization", "Bearer "+token1)
		rr1 := httptest.NewRecorder()
		r.ServeHTTP(rr1, req1)

		if rr1.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr1.Code)
		}
		var resp1 map[string]any
		json.Unmarshal(rr1.Body.Bytes(), &resp1)
		data1 := resp1["data"].([]interface{})
		pagination1 := resp1["pagination"].(map[string]interface{})

		if len(data1) != 2 {
			t.Errorf("Expected 2 results per page, got %d", len(data1))
		}
		if pagination1["current_page"].(float64) != 1 {
			t.Errorf("Expected current_page=1")
		}

		// Fetch page 2 with limit 2
		req2, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=commonWord&page=2&limit=2", org.ID), nil)
		req2.Header.Set("Authorization", "Bearer "+token1)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)

		var resp2 map[string]any
		json.Unmarshal(rr2.Body.Bytes(), &resp2)
		data2 := resp2["data"].([]interface{})
		pagination2 := resp2["pagination"].(map[string]interface{})

		if len(data2) != 2 {
			t.Errorf("Expected 2 results per page on page 2, got %d", len(data2))
		}
		if pagination2["current_page"].(float64) != 2 {
			t.Errorf("Expected current_page=2")
		}

	})

	t.Run("Channel filter search (in:#channelName)", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channelName := "filter_test_channel"
		channel := models.Channels{
			ID:             channelID,
			Name:           channelName,
			OrganisationID: org.ID,
			IsPrivate:      false,
			OwnerId:        user1.ID,
		}
		db.Postgresql.Create(&channel)

		userChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     user1.ID,
		}
		db.Postgresql.Create(&userChannel)

		threadId := utility.GenerateUUID()
		thread := map[string]any{
			"thread_id":    threadId,
			"channels_id":  channelID,
			"channel_name": channelName,
			"user_id":      user1.ID,
			"org_id":       org.ID,
			"message":      "Testing specific channel filter keywordFilterMessage",
			"created_at":   time.Now().Format(time.RFC3339),
			"updated_at":   time.Now().Format(time.RFC3339),
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId, thread, logger)

		time.Sleep(2 * time.Second)

		queryParam := url.QueryEscape(fmt.Sprintf("keywordFilterMessage in:#%s", channelName))
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=%s", org.ID, queryParam), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		if len(data) != 1 {
			t.Errorf("User 1 should see 1 result from specific channel filter, got %d", len(data))
		}
	})

	t.Run("Filters search (from, in)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=keywordPublicChannel123+from%%3A%%22%s%%22+in%%3A%%22Public+Channel%%22", org.ID, user1SignUpData.UserName), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for from/in search, got %d", rr.Code)
		}

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if dataInt, ok := resp["data"]; ok && dataInt != nil {
			data := dataInt.([]interface{})
			if len(data) == 0 {
				t.Errorf("Expected > 0 results for from/in search, got 0. Response: %v", resp)
			}
		} else {
			t.Errorf("Member should see > 0 results but got null/none. Response: %v", resp)
		}
	})

	t.Run("Date filtering search (on, before, after)", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           "Date Filter Channel",
			OrganisationID: org.ID,
			OwnerId:        user1.ID,
		}
		db.Postgresql.Create(&channel)

		userChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     user1.ID,
		}
		db.Postgresql.Create(&userChannel)

		// Create messages on different dates
		threadId0 := utility.GenerateUUID()
		thread0 := map[string]any{
			"thread_id":   threadId0,
			"channels_id": channelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Date message 0 dateKeyword",
			"created_at":  "2023-01-01T12:00:00Z",
			"updated_at":  "2023-01-01T12:00:00Z",
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId0, thread0, logger)

		threadId1 := utility.GenerateUUID()
		thread1 := map[string]any{
			"thread_id":   threadId1,
			"channels_id": channelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Date message 1 dateKeyword",
			"created_at":  "2023-01-02T12:00:00Z",
			"updated_at":  "2023-01-02T12:00:00Z",
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId1, thread1, logger)

		threadId2 := utility.GenerateUUID()
		thread2 := map[string]any{
			"thread_id":   threadId2,
			"channels_id": channelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Date message 2 dateKeyword",
			"created_at":  "2023-01-03T12:00:00Z",
			"updated_at":  "2023-01-03T12:00:00Z",
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId2, thread2, logger)

		time.Sleep(2 * time.Second)

		// Test `on` filter using YYYY-MM-DD format
		queryParam := url.QueryEscape("dateKeyword on:2023-01-02")
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=%s", org.ID, queryParam), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		} else {
			var resp map[string]any
			json.Unmarshal(rr.Body.Bytes(), &resp)
			if dataInt, ok := resp["data"]; ok && dataInt != nil {
				data := dataInt.([]interface{})
				if len(data) != 1 {
					t.Errorf("User 1 should see exactly 1 result for 'on:2023-01-02', got %d", len(data))
				}
			} else {
				t.Errorf("data is null or missing")
			}
		}
	})

	t.Run("Exact and Sorting search (exact, sortby)", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           "Exact Filter Channel",
			OrganisationID: org.ID,
			OwnerId:        user1.ID,
		}
		db.Postgresql.Create(&channel)

		userChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     user1.ID,
		}
		db.Postgresql.Create(&userChannel)

		// Message 1 (oldest)
		threadId1 := utility.GenerateUUID()
		thread1 := map[string]any{
			"thread_id":   threadId1,
			"channels_id": channelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Please review the Project deadline today",
			"created_at":  time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId1, thread1, logger)

		// Message 2 (newest)
		threadId2 := utility.GenerateUUID()
		thread2 := map[string]any{
			"thread_id":   threadId2,
			"channels_id": channelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Here is the code for the Project deadline task",
			"created_at":  time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId2, thread2, logger)

		time.Sleep(2 * time.Second)

		// Search for exact text with sortby newest
		queryParam := url.QueryEscape(`exact:"Project deadline"`)
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=%s&sortby=newest", org.ID, queryParam), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		if len(data) != 2 {
			t.Errorf("User 1 should see exactly 2 results for exact filter, got %d", len(data))
		} else {
			// Because sortby=newest, thread2 should be first
			resObj := data[0].(map[string]any)
			threadInfo := resObj["thread"].(map[string]any)
			if threadInfo["id"] != threadId2 {
				t.Errorf("Expected newest thread to be first")
			}
		}

		// Search with sortby oldest
		req2, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=%s&sortby=oldest", org.ID, queryParam), nil)
		req2.Header.Set("Authorization", "Bearer "+token1)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)

		var resp2 map[string]any
		json.Unmarshal(rr2.Body.Bytes(), &resp2)
		data2 := resp2["data"].([]interface{})
		if len(data2) == 2 {
			resObj := data2[0].(map[string]any)
			threadInfo := resObj["thread"].(map[string]any)
			if threadInfo["id"] != threadId1 {
				t.Errorf("Expected oldest thread to be first")
			}
		}
	})

	t.Run("Exact phrase search", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           "Exact Phrase Channel",
			OrganisationID: org.ID,
			OwnerId:        user1.ID,
		}
		db.Postgresql.Create(&channel)

		userChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     user1.ID,
		}
		db.Postgresql.Create(&userChannel)

		threadId1 := utility.GenerateUUID()
		thread1 := map[string]any{
			"thread_id":   threadId1,
			"channels_id": channelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "quick brown fox jumps over the lazy dog",
			"created_at":  time.Now().Format(time.RFC3339),
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId1, thread1, logger)

		threadId2 := utility.GenerateUUID()
		thread2 := map[string]any{
			"thread_id":   threadId2,
			"channels_id": channelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "quick blue fox sleeps over the dog lazy",
			"created_at":  time.Now().Format(time.RFC3339),
		}
		elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId2, thread2, logger)

		time.Sleep(2 * time.Second)

		queryParam := url.QueryEscape("quick brown fox")
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/organisation/%s?query=%s", org.ID, queryParam), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		if len(data) != 1 {
			t.Errorf("Expected exactly 1 match for exact phrase 'quick brown fox', got %d", len(data))
		} else {
			resObj := data[0].(map[string]any)
			threadInfo := resObj["thread"].(map[string]any)
			if threadInfo["id"] != threadId1 {
				t.Errorf("Expected matching thread to be threadId1, got %v", threadInfo["id"])
			}
		}
	})
}

