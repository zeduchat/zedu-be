package test_message

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestDeleteDmChannelEndpoint(t *testing.T) {
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
		Email:       fmt.Sprintf("dmdeluser1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "DMDelUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("dmdeluser1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("dmdeluser2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "DMDelUser",
		LastName:    "Two",
		Password:    "password",
		UserName:    fmt.Sprintf("dmdeluser2_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}

	tst.SignupUser(t, r, authController, user1SignUpData, false)
	tst.SignupUser(t, gin.Default(), authController, user2SignUpData, false)
	token1 := tst.GetLoginToken(t, r, authController, loginData1)

	var user1, user2 models.User
	require.NoError(t, db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error)
	require.NoError(t, db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error)

	var org models.Organisation
	require.NoError(t, db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error)

	dmChannelID := utility.GenerateUUID()
	dmChan := models.DmChannels{
		ID:            utility.GenerateUUID(),
		ChannelId:     dmChannelID,
		UserId:        user1.ID,
		ParticipantId: &user2.ID,
		OrgId:         org.ID,
		ChatType:      "user",
		ChannelType:   "dm",
	}
	require.NoError(t, db.Postgresql.Create(&dmChan).Error)

	t.Run("DELETE DM Channel successfully", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/organisations/"+org.ID+"/dms/"+dmChannelID, nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.Equal(t, "success", response["status"].(string))
		assert.Equal(t, "Dm channel deleted successfully", response["message"].(string))

		// Verify record is deleted from DB
		var count int64
		db.Postgresql.Model(&models.DmChannels{}).Where("channel_id = ? AND user_id = ?", dmChannelID, user1.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}
