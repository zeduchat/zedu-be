package test_organisation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetAllChannelsWithNullLastReadAt(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	orgController := organisation.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
	}

	userReq := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("user_null_lastread_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "NullRead",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("nullread_%v", currUUID),
	}

	r := gin.Default()
	tst.SignupUser(t, r, authController, userReq, false)
	token := tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: userReq.Email, Password: userReq.Password})

	var user models.User
	if err := db.Postgresql.Where("email = ?", userReq.Email).First(&user).Error; err != nil {
		t.Fatalf("Failed to fetch user: %v", err)
	}

	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgController, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("Org NullRead %v", currUUID),
		Email:       userReq.Email,
		Description: "Org for testing null last read at",
		Type:        "type1",
		Location:    "loc",
		Country:     "country",
	}, token)

	chanID := utility.GenerateUUID()
	channel := models.Channels{
		ID:             chanID,
		Name:           fmt.Sprintf("chan-nullread-%v", currUUID),
		Description:    "Test channel with null last_read_at",
		OrganisationID: orgID,
		OwnerId:        user.ID,
		CreatedAt:      time.Now(),
	}
	if err := db.Postgresql.Create(&channel).Error; err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	db.Postgresql.Exec("INSERT INTO user_channels (channels_id, user_id, username, created_at, last_read_at) VALUES (?, ?, ?, ?, NULL)", chanID, user.ID, userReq.UserName, time.Now())

	orgUrl := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	{
		orgUrl.GET("/organisations/:org_id/channels", orgController.GetAllChannelssInOrganisation)
	}

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/channels", orgID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	var resp struct {
		Data []struct {
			ID           string `json:"channels_id"`
			LastPostTime string `json:"last_post_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	found := false
	for _, ch := range resp.Data {
		if ch.ID == chanID {
			found = true
			if ch.LastPostTime == "Last post unavailable" {
				t.Errorf("Expected LastPostTime not to be 'Last post unavailable', got %s", ch.LastPostTime)
			}
		}
	}

	if !found {
		t.Errorf("Channel %s was not returned in response", chanID)
	}
}
