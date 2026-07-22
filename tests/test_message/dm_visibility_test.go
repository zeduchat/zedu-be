package test_message

import (
	"bytes"
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

func TestDmChannelVisibilityFlow(t *testing.T) {
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
		protected.GET("/organisations/:org_id/dms", directMessageController.GetDmChannels)
		protected.GET("/organisations/:org_id/dms/visible", directMessageController.GetVisibleDmChannels)
		protected.PATCH("/dms/:channel_id/visibility", directMessageController.UpdateDmVisibility)
	}

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("visuser1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "VisUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("visuser1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("visuser2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "VisUser",
		LastName:    "Two",
		Password:    "password",
		UserName:    fmt.Sprintf("visuser2_%v", currUUID),
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
	trueVal := true
	dmChan1 := models.DmChannels{
		ID:               utility.GenerateUUID(),
		ChannelId:        dmChannelID,
		UserId:           user1.ID,
		ParticipantId:    &user2.ID,
		OrgId:            org.ID,
		ChatType:         "user",
		ChannelType:      "dm",
		VisibilityStatus: &trueVal,
	}
	require.NoError(t, db.Postgresql.Create(&dmChan1).Error)

	dmChan2 := models.DmChannels{
		ID:               utility.GenerateUUID(),
		ChannelId:        dmChannelID,
		UserId:           user2.ID,
		ParticipantId:    &user1.ID,
		OrgId:            org.ID,
		ChatType:         "user",
		ChannelType:      "dm",
		VisibilityStatus: &trueVal,
	}
	require.NoError(t, db.Postgresql.Create(&dmChan2).Error)

	t.Run("Initially visible in GET /organisations/:org_id/dms/visible", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/organisations/"+org.ID+"/dms/visible", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		data := response["data"].([]interface{})
		assert.NotEmpty(t, data)
	})

	t.Run("User 1 sets DM channel visibility to false", func(t *testing.T) {
		payload := map[string]bool{"visibility_status": false}
		jsonBody, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/api/v1/dms/"+dmChannelID+"/visibility", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.Equal(t, "DM channel visibility status updated successfully", response["message"].(string))
	})

	t.Run("Verify User 1 no longer sees channel in GET /dms/visible, but User 2 still sees it", func(t *testing.T) {
		// User 1 check
		req1 := httptest.NewRequest("GET", "/api/v1/organisations/"+org.ID+"/dms/visible", nil)
		req1.Header.Set("Authorization", "Bearer "+token1)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)
		var response1 map[string]interface{}
		require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &response1))
		data1 := response1["data"].([]interface{})
		for _, item := range data1 {
			ch := item.(map[string]interface{})
			assert.NotEqual(t, dmChannelID, ch["channel_id"])
		}

		// User 2 check
		req2 := httptest.NewRequest("GET", "/api/v1/organisations/"+org.ID+"/dms/visible", nil)
		req2.Header.Set("Authorization", "Bearer "+token2)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
		var response2 map[string]interface{}
		require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &response2))
		data2 := response2["data"].([]interface{})
		assert.NotEmpty(t, data2)
	})

	t.Run("Sending a new message in DM channel restores visibility_status to true for User 1", func(t *testing.T) {
		dmModel := models.DmChannels{ChannelId: dmChannelID}
		err := dmModel.UpdateInteractionAt(db.Postgresql)
		require.NoError(t, err)

		req1 := httptest.NewRequest("GET", "/api/v1/organisations/"+org.ID+"/dms/visible", nil)
		req1.Header.Set("Authorization", "Bearer "+token1)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)
		var response1 map[string]interface{}
		require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &response1))
		data1 := response1["data"].([]interface{})
		found := false
		for _, item := range data1 {
			ch := item.(map[string]interface{})
			if ch["channel_id"] == dmChannelID {
				found = true
				break
			}
		}
		assert.True(t, found, "DM channel should be restored to visible after new message")
	})
}
