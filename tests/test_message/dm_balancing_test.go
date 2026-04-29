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

func TestGetDmChannelsBalancing(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create a user and an org
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("dmbalance_%v@qa.team", currUUID),
		FirstName:   "BalanceUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("dmbalance_%v", currUUID),
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
	
	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}
	token1 := tst.GetLoginToken(t, r, authController, loginData1)

	var user1 models.User
	db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1)

	var org models.Organisation
	db.Postgresql.Where("owner_id = ?", user1.ID).First(&org)

	// Create 15 more users in the same org
	for i := 0; i < 15; i++ {
		email := fmt.Sprintf("otheruser%d_%v@qa.team", i, currUUID)
		u := models.User{
			ID:       utility.GenerateUUID(),
			Email:    email,
			Name:     fmt.Sprintf("Other User %d", i),
			Password: "password",
		}
		db.Postgresql.Create(&u)
		db.Postgresql.Table("user_organisations").Create(map[string]interface{}{
			"user_id":         u.ID,
			"organisation_id": org.ID,
		})
		
		db.Postgresql.Model(&models.OrgUserManagement{}).Create(&models.OrgUserManagement{
			UserID:         u.ID,
			OrganisationID: org.ID,
			RoleID:         utility.GenerateUUID(),
			Status:         "active",
		})

		// Also create profile
		p := models.Profile{
			ID:       utility.GenerateUUID(),
			Userid:   u.ID,
			UserName: fmt.Sprintf("otheruser%d_%v", i, currUUID),
			FullName: fmt.Sprintf("Other User %d", i),
		}
		db.Postgresql.Create(&p)
	}

	// Now fetch DM channels. Since there are 0 DM channels, it should be balanced to 10 suggested users.
	dmController := dmCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
	}

	r.GET("/api/v1/organisations/:org_id/dms", middleware.Authorize(db.Postgresql), dmController.GetDmChannels)

	t.Run("DM Channels Balanced to 10", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms?page=1&limit=10", org.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		tst.AssertStatusCode(t, resp.Code, http.StatusOK)

		var response struct {
			Data []models.DmChannelsResponse `json:"data"`
			Pagination struct {
				TotalItems int64 `json:"total_items"`
			} `json:"pagination"`
		}
		json.Unmarshal(resp.Body.Bytes(), &response)

		if len(response.Data) != 10 {
			t.Errorf("Expected 10 balanced items, got %d", len(response.Data))
		}
		
		if response.Pagination.TotalItems != 10 {
			t.Errorf("Expected TotalItems to be 10, got %d", response.Pagination.TotalItems)
		}
	})
}
