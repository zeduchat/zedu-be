package test_message

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	dmCtrl "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestDmFavourites(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create users
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("fav_user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Fav",
		LastName:    "User1",
		Password:    "password",
		UserName:    fmt.Sprintf("fav_user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("fav_user2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Fav",
		LastName:    "User2",
		Password:    "password",
		UserName:    fmt.Sprintf("fav_user2_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
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

	r := gin.Default()
	tst.SignupUser(t, r, authController, user1SignUpData, false)
	tst.SignupUser(t, gin.Default(), authController, user2SignUpData, false)
	token1 := tst.GetLoginToken(t, r, authController, loginData1)

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

	// Create a DM channel
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
	if err := db.Postgresql.Create(&dmChannel).Error; err != nil {
		t.Fatalf("Failed to create DM channel: %v", err)
	}

	// Create reverse DM channel
	dmChannel2 := models.DmChannels{
		ID:            utility.GenerateUUID(),
		UserId:        user2.ID,
		ChannelId:     dmChannelID,
		OrgId:         org.ID,
		ParticipantId: &user1.ID,
		ChatType:      "user",
		ChannelType:   "dm",
	}
	if err := db.Postgresql.Create(&dmChannel2).Error; err != nil {
		t.Fatalf("Failed to create reverse DM channel: %v", err)
	}

	// Create a group DM channel
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
	if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
		t.Fatalf("Failed to create group DM channel: %v", err)
	}

	// Create participants
	for _, user := range []models.User{user1, user2} {
		participant := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupDMChannelID,
			UserId:    user.ID,
			OrgId:     org.ID,
		}
		if err := db.Postgresql.Create(&participant).Error; err != nil {
			t.Fatalf("Failed to create channel participant: %v", err)
		}
	}

	extReq := request.ExternalRequest{Logger: logger, Test: true}
	controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

	t.Run("Toggle favourite - Add DM to favourites", func(t *testing.T) {
		r := gin.Default()
		r.POST("/api/v1/organisations/:org_id/dms/:channel_id/favourite", middleware.Authorize(db.Postgresql), controller.ToggleFavourite)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/favourite", org.ID, dmChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["message"] != "Added to favourites" {
			t.Errorf("Expected 'Added to favourites', got: %v", response["message"])
		}

		// Check data contains is_favourite = true
		data := response["data"].(map[string]interface{})
		if data["is_favourite"] != true {
			t.Errorf("Expected is_favourite=true in response data")
		}

		t.Log("✅ Successfully added DM to favourites via toggle")
	})

	t.Run("Toggle favourite - Add group DM to favourites", func(t *testing.T) {
		r := gin.Default()
		r.POST("/api/v1/organisations/:org_id/dms/:channel_id/favourite", middleware.Authorize(db.Postgresql), controller.ToggleFavourite)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/favourite", org.ID, groupDMChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		t.Log("✅ Successfully added group DM to favourites via toggle")
	})

	t.Run("Get favourite DMs", func(t *testing.T) {
		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/favourites", middleware.Authorize(db.Postgresql), controller.GetFavouriteDms)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/favourites", org.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)

		data, ok := response["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data array in response")
		}

		if len(data) != 2 {
			t.Errorf("Expected 2 favourite DMs, got %d", len(data))
		}

		// Verify all have is_favourite = true
		for i, dm := range data {
			dmMap := dm.(map[string]interface{})
			if isFav, ok := dmMap["is_favourite"].(bool); !ok || !isFav {
				t.Errorf("DM %d should have is_favourite=true", i)
			}
		}

		t.Logf("✅ Got %d favourite DMs", len(data))
	})

	t.Run("Toggle favourite - Remove from favourites", func(t *testing.T) {
		r := gin.Default()
		r.POST("/api/v1/organisations/:org_id/dms/:channel_id/favourite", middleware.Authorize(db.Postgresql), controller.ToggleFavourite)

		// Toggle again to remove
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/favourite", org.ID, dmChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["message"] != "Removed from favourites" {
			t.Errorf("Expected 'Removed from favourites', got: %v", response["message"])
		}

		// Check data contains is_favourite = false
		data := response["data"].(map[string]interface{})
		if data["is_favourite"] != false {
			t.Errorf("Expected is_favourite=false in response data")
		}

		t.Log("✅ Successfully removed from favourites via toggle")
	})

	t.Run("Get favourites after removal", func(t *testing.T) {
		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/favourites", middleware.Authorize(db.Postgresql), controller.GetFavouriteDms)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/favourites", org.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)

		data, _ := response["data"].([]interface{})
		if len(data) != 1 {
			t.Errorf("Expected 1 favourite DM after removal, got %d", len(data))
		}

		t.Log("✅ Correctly shows 1 favourite after removal")
	})

	t.Run("Toggle favourite - Re-add after removal", func(t *testing.T) {
		r := gin.Default()
		r.POST("/api/v1/organisations/:org_id/dms/:channel_id/favourite", middleware.Authorize(db.Postgresql), controller.ToggleFavourite)

		// Toggle again to add back
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/favourite", org.ID, dmChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["message"] != "Added to favourites" {
			t.Errorf("Expected 'Added to favourites', got: %v", response["message"])
		}

		t.Log("✅ Successfully re-added to favourites via toggle")
	})

	t.Run("GetDmChannels reflects favourite status", func(t *testing.T) {
		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms", middleware.Authorize(db.Postgresql), controller.GetDmChannels)

		checkFavStatus := func(expectFav bool) {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms", org.ID), nil)
			req.Header.Set("Authorization", "Bearer "+token1)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
			}

			var response map[string]interface{}
			json.Unmarshal(rr.Body.Bytes(), &response)

			data, ok := response["data"].([]interface{})
			if !ok {
				t.Fatalf("Expected data array in response")
			}

			found := false
			for _, item := range data {
				channel := item.(map[string]interface{})
				if channel["channel_id"] == dmChannelID {
					found = true
					isFav, _ := channel["is_favourite"].(bool)
					if isFav != expectFav {
						t.Errorf("For channel %s, expected is_favourite=%v, got %v", dmChannelID, expectFav, isFav)
					}
					break
				}
			}
			if !found {
				t.Errorf("Channel %s not found in GetDmChannels response", dmChannelID)
			}
		}

		t.Log("Checking if is_favourite=true after adding to favourites...")
		checkFavStatus(true)

		t.Log("Removing from favourites...")
		rPost := gin.Default()
		rPost.POST("/api/v1/organisations/:org_id/dms/:channel_id/favourite", middleware.Authorize(db.Postgresql), controller.ToggleFavourite)
		reqPost, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/favourite", org.ID, dmChannelID), nil)
		reqPost.Header.Set("Authorization", "Bearer "+token1)
		rrPost := httptest.NewRecorder()
		rPost.ServeHTTP(rrPost, reqPost)
		if rrPost.Code != http.StatusOK {
			t.Fatalf("Failed to remove from favourites: %v", rrPost.Body.String())
		}
	})

	t.Run("GetDmParticipants reflects favourite status", func(t *testing.T) {
		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/participants/:channel_id", middleware.Authorize(db.Postgresql), controller.GetDmParticipants)

		// Helper to check favourite status in GetDmParticipants response
		checkParticipantsFavStatus := func(expectFav bool) {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/participants/%s", org.ID, dmChannelID), nil)
			req.Header.Set("Authorization", "Bearer "+token1)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
			}

			var response map[string]interface{}
			json.Unmarshal(rr.Body.Bytes(), &response)

			data, ok := response["data"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected data object in response")
			}

			isFav, ok := data["is_favourite"].(bool)
			if !ok {
				t.Fatalf("is_favourite field missing in response")
			}

			if isFav != expectFav {
				t.Errorf("Expected is_favourite=%v in GetDmParticipants, got %v", expectFav, isFav)
			}
		}

		t.Log("Adding to favourites to test GetDmParticipants...")
		rPost := gin.Default()
		rPost.POST("/api/v1/organisations/:org_id/dms/:channel_id/favourite", middleware.Authorize(db.Postgresql), controller.ToggleFavourite)
		reqPost, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/favourite", org.ID, dmChannelID), nil)
		reqPost.Header.Set("Authorization", "Bearer "+token1)
		rrPost := httptest.NewRecorder()
		rPost.ServeHTTP(rrPost, reqPost)
		if rrPost.Code != http.StatusOK {
			t.Fatalf("Failed to add to favourites: %v", rrPost.Body.String())
		}

		t.Log("Checking if is_favourite=true in GetDmParticipants...")
		checkParticipantsFavStatus(true)

		t.Log("Removing from favourites to test GetDmParticipants...")
		reqPostRemove, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/favourite", org.ID, dmChannelID), nil)
		reqPostRemove.Header.Set("Authorization", "Bearer "+token1)
		rrPostRemove := httptest.NewRecorder()
		rPost.ServeHTTP(rrPostRemove, reqPostRemove)
		if rrPostRemove.Code != http.StatusOK {
			t.Fatalf("Failed to remove from favourites: %v", rrPostRemove.Body.String())
		}

		t.Log("Checking if is_favourite=false in GetDmParticipants...")
		checkParticipantsFavStatus(false)
	})
}
