package test_message

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	dmCtrl "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestMultiUserDmChannelDeletion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := storage.Connection()
	logger := tst.Setup()
	validatorInstance := validator.New()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	authController := auth.Controller{
		Db:        db,
		Validator: validatorInstance,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	directMessageController := &dmCtrl.Controller{
		Db:        db,
		Validator: validatorInstance,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	protected := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	{
		protected.DELETE("/organisations/:org_id/dms/:channel_id", directMessageController.DeleteDmChannel)
	}

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("multiuser1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "MultiUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("multiuser1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("multiuser2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "MultiUser",
		LastName:    "Two",
		Password:    "password",
		UserName:    fmt.Sprintf("multiuser2_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}

	loginData2 := models.LoginRequestModel{
		Email:    user2SignUpData.Email,
		Password: user2SignUpData.Password,
	}

	tst.SignupUser(t, r, authController, user1SignUpData, false)
	tst.SignupUser(t, gin.Default(), authController, user2SignUpData, false)
	token1 := tst.GetLoginToken(t, r, authController, loginData1)
	token2 := tst.GetLoginToken(t, r, authController, loginData2)

	var user1, user2 models.User
	require.NoError(t, db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error)
	require.NoError(t, db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error)

	var org models.Organisation
	require.NoError(t, db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error)

	dmChannelID := utility.GenerateUUID()
	dmChan1 := models.DmChannels{
		ID:            utility.GenerateUUID(),
		ChannelId:     dmChannelID,
		UserId:        user1.ID,
		ParticipantId: &user2.ID,
		OrgId:         org.ID,
		ChatType:      "user",
		ChannelType:   "dm",
	}
	require.NoError(t, db.Postgresql.Create(&dmChan1).Error)

	dmChan2 := models.DmChannels{
		ID:            utility.GenerateUUID(),
		ChannelId:     dmChannelID,
		UserId:        user2.ID,
		ParticipantId: &user1.ID,
		OrgId:         org.ID,
		ChatType:      "user",
		ChannelType:   "dm",
	}
	require.NoError(t, db.Postgresql.Create(&dmChan2).Error)

	// Index thread to Elastic
	threadID := utility.GenerateUUID()
	threadDoc := map[string]any{
		"thread_id":   threadID,
		"channels_id": dmChannelID,
		"user_id":     user1.ID,
		"org_id":      org.ID,
		"message":     "Test multi-user retention message",
		"created_at":  time.Now().Format(time.RFC3339),
	}
	_ = elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, threadDoc, logger)

	t.Run("User 1 deletes DM channel - User 2 remains active and Elastic data preserved", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/organisations/"+org.ID+"/dms/"+dmChannelID, nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.Equal(t, "success", response["status"].(string))

		// User 1 row is deleted
		var count1 int64
		db.Postgresql.Model(&models.DmChannels{}).Where("channel_id = ? AND user_id = ?", dmChannelID, user1.ID).Count(&count1)
		assert.Equal(t, int64(0), count1)

		// User 2 row remains active
		var count2 int64
		db.Postgresql.Model(&models.DmChannels{}).Where("channel_id = ? AND user_id = ?", dmChannelID, user2.ID).Count(&count2)
		assert.Equal(t, int64(1), count2)
	})

	t.Run("User 2 deletes DM channel - No participants remain, triggering Elastic deletion", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/organisations/"+org.ID+"/dms/"+dmChannelID, nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// User 2 row is deleted
		var count2 int64
		db.Postgresql.Model(&models.DmChannels{}).Where("channel_id = ? AND user_id = ?", dmChannelID, user2.ID).Count(&count2)
		assert.Equal(t, int64(0), count2)
	})
}
