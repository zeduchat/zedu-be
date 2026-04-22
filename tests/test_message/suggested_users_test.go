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
	"github.com/stretchr/testify/assert"
)

func TestGetDmChannels_SuggestedUsers(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("suggest_user_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Suggest",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("suggest_user_%v", currUUID),
	}

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
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
	tst.SignupUser(t, r, authController, userSignUpData, false)
	token := tst.GetLoginToken(t, r, authController, loginData)

	var user models.User
	if err := db.Postgresql.Where("email = ?", userSignUpData.Email).First(&user).Error; err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	for i := 1; i <= 15; i++ {
		otherUserID := utility.GenerateUUID()
		otherUser := models.User{
			ID:    otherUserID,
			Name:  fmt.Sprintf("Other User %d", i),
			Email: fmt.Sprintf("other_%d_%v@qa.team", i, currUUID),
		}
		if err := db.Postgresql.Create(&otherUser).Error; err != nil {
			t.Fatalf("Failed to create other user %d: %v", i, err)
		}

		if err := db.Postgresql.Table("user_organisations").Create(map[string]any{
			"user_id":         otherUserID,
			"organisation_id": org.ID,
		}).Error; err != nil {
			t.Fatalf("Failed to add user %d to org: %v", i, err)
		}

		if err := db.Postgresql.Create(&models.Profile{
			ID:       utility.GenerateUUID(),
			Userid:   otherUserID,
			UserName: fmt.Sprintf("otheruser%d", i),
		}).Error; err != nil {
			t.Fatalf("Failed to create profile for user %d: %v", i, err)
		}
	}

	extReq := request.ExternalRequest{Logger: logger, Test: true}
	controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

	r = gin.Default()
	r.GET("/api/v1/organisations/:org_id/dms", middleware.Authorize(db.Postgresql), controller.GetDmChannels)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms", org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	data, ok := response["data"].([]any)
	assert.True(t, ok, "Response should have data field")
	assert.Equal(t, 10, len(data), "Should return exactly 10 suggested users")

	for _, item := range data {
		channel := item.(map[string]any)
		assert.Equal(t, true, channel["is_suggested"], "User should be marked as suggested")
		assert.Equal(t, "dm", channel["channel_type"], "Channel type should be dm")
	}

	pagination, ok := response["pagination"].(map[string]any)
	assert.True(t, ok, "Response should have pagination field")
	assert.Equal(t, float64(10), pagination["total_items"], "Total items in pagination should be 10")
}
