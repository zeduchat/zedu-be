package test_users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUpdateUserNotificationPreferences(t *testing.T) {

	_, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	adminUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Admin User",
		Email:    fmt.Sprintf("admin%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.SuperAdmin),
	}
	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}

	userID, _ := uuid.FromString(adminUser.ID)

	notificationPreferences := models.NotificationPreferences{
		ID:                      utility.GenerateUUID(),
		UserID:                  userID,
		NotifyAbout:             models.NotifyAllMessages,
		NotificationSchedule:    true,
		FromHour:                "08:00",
		ToHour:                  "20:00",
		NotificationMethodEmail: true,
	}

	db.Create(&adminUser)
	db.Create(&regularUser)
	db.Create(&notificationPreferences)

	setup := func() (*gin.Engine, *auth.Controller) {
		router, userController := SetupUsersTestRouter()
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}

		return router, &authController
	}

	t.Run("Successful Update User Notification Preferences for admin", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		updateData := models.NotificationPreferences{
			NotifyAbout:             models.NotifyDirectMentions,
			NotificationSchedule:    false,
			FromHour:                "09:00",
			ToHour:                  "21:00",
			NotificationMethodEmail: false,
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/notification-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "User notification preferences updated successfully")
	})

	t.Run("Unauthorized", func(t *testing.T) {
		router, _ := setup()

		updateData := models.NotificationPreferences{
			NotifyAbout:             models.NotifyDirectMentions,
			NotificationSchedule:    false,
			FromHour:                "09:00",
			ToHour:                  "21:00",
			NotificationMethodEmail: false,
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/notification-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token could not be found!")
	})

}
