package test_onesignal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/cronjobs"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	orgCtrl "github.com/hngprojects/telex_be/pkg/controller/organisation"
	userCtrl "github.com/hngprojects/telex_be/pkg/controller/user"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestOrgOneSignalNotifications(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := storage.Connection()
	logger := tst.Setup()
	validatorInstance := validator.New()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	userController := &userCtrl.Controller{
		Db:        db,
		Validator: validatorInstance,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	organisationController := &orgCtrl.Controller{
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
		protected.PUT("/users/onesignal-subscription-id", userController.UpdateOneSignalSubscriptionID)
		protected.GET("/organisations/:org_id/users/onesignal", organisationController.GetOneSignalNotifications)
	}

	testUser := CreateTestUser(t, db)
	token := generateTestToken(t, db, testUser.ID)

	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:      orgID,
		Name:    "Test Org OneSignal",
		OwnerID: testUser.ID,
	}
	require.NoError(t, db.Postgresql.Create(&org).Error)

	var role models.OrgRole
	db.Postgresql.Where("name = ?", "User").First(&role)
	if role.ID == "" {
		db.Postgresql.First(&role)
	}

	membership := models.OrgUserManagement{
		UserID:         testUser.ID,
		OrganisationID: orgID,
		RoleID:         role.ID,
		Status:         "active",
	}
	require.NoError(t, db.Postgresql.Create(&membership).Error)

	type UserOrganisation struct {
		UserID         string `gorm:"column:user_id"`
		OrganisationID string `gorm:"column:organisation_id"`
	}
	db.Postgresql.Table("user_organisations").Create(&UserOrganisation{
		UserID:         testUser.ID,
		OrganisationID: orgID,
	})

	if err := db.Postgresql.AutoMigrate(&models.OneSignalNotification{}); err != nil {
		t.Fatalf("Failed to auto migrate OneSignalNotification: %v", err)
	}
	db.Postgresql.Exec("DELETE FROM onesignal_notifications")

	t.Run("GET empty list when no notifications exist", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/organisations/"+orgID+"/users/onesignal", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.Equal(t, "success", response["status"].(string))
		data := response["data"].(map[string]interface{})
		notifications := data["notifications"].([]interface{})
		assert.Len(t, notifications, 0)
	})

	t.Run("GET active notifications and pagination", func(t *testing.T) {
		notif := models.OneSignalNotification{
			ID:                      utility.GenerateUUID(),
			UserID:                  testUser.ID,
			OrgID:                   &orgID,
			OneSignalNotificationID: "notif-123",
			Title:                   "Test Notification",
			Message:                 "Test Message",
			Status:                  models.OneSignalNotificationStatusPending,
			SentAt:                  time.Now(),
		}
		require.NoError(t, db.Postgresql.Create(&notif).Error)

		req := httptest.NewRequest("GET", "/api/v1/organisations/"+orgID+"/users/onesignal?page=1&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		data := response["data"].(map[string]interface{})
		notifications := data["notifications"].([]interface{})
		assert.Len(t, notifications, 1)

		notifData := notifications[0].(map[string]interface{})
		assert.Equal(t, "notif-123", notifData["onesignal_notification_id"].(string))
		assert.Equal(t, "Test Notification", notifData["title"].(string))
		assert.Equal(t, "Test Message", notifData["message"].(string))
	})

	t.Run("GET filters out expired notifications", func(t *testing.T) {
		db.Postgresql.Exec("DELETE FROM onesignal_notifications")

		expiredNotif := models.OneSignalNotification{
			ID:                      utility.GenerateUUID(),
			UserID:                  testUser.ID,
			OrgID:                   &orgID,
			OneSignalNotificationID: "expired-notif",
			Title:                   "Expired",
			Message:                 "Expired",
			Status:                  models.OneSignalNotificationStatusPending,
			SentAt:                  time.Now(),
		}
		require.NoError(t, db.Postgresql.Create(&expiredNotif).Error)

		// Set created_at to 2 months and 1 day ago to simulate expiration
		err := db.Postgresql.Model(&models.OneSignalNotification{}).
			Where("id = ?", expiredNotif.ID).
			Update("created_at", time.Now().AddDate(0, -2, -1)).Error
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/v1/organisations/"+orgID+"/users/onesignal", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		data := response["data"].(map[string]interface{})
		notifications := data["notifications"].([]interface{})
		assert.Len(t, notifications, 0)
	})

	t.Run("Weekly cron job cleans up expired notifications", func(t *testing.T) {
		db.Postgresql.Exec("DELETE FROM onesignal_notifications")

		expiredNotif := models.OneSignalNotification{
			ID:                      utility.GenerateUUID(),
			UserID:                  testUser.ID,
			OrgID:                   &orgID,
			OneSignalNotificationID: "expired-notif",
			Title:                   "Expired",
			Message:                 "Expired",
			Status:                  models.OneSignalNotificationStatusPending,
			SentAt:                  time.Now(),
		}
		require.NoError(t, db.Postgresql.Create(&expiredNotif).Error)

		// Set created_at to 2 months and 1 day ago to simulate expiration
		err := db.Postgresql.Model(&models.OneSignalNotification{}).
			Where("id = ?", expiredNotif.ID).
			Update("created_at", time.Now().AddDate(0, -2, -1)).Error
		require.NoError(t, err)

		activeNotif := models.OneSignalNotification{
			ID:                      utility.GenerateUUID(),
			UserID:                  testUser.ID,
			OrgID:                   &orgID,
			OneSignalNotificationID: "active-notif",
			Title:                   "Active",
			Message:                 "Active",
			Status:                  models.OneSignalNotificationStatusPending,
			SentAt:                  time.Now(),
		}
		require.NoError(t, db.Postgresql.Create(&activeNotif).Error)

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		cronjobs.CleanExpiredOneSignalNotifications(extReq, *db)

		var count int64
		db.Postgresql.Model(&models.OneSignalNotification{}).Where("onesignal_notification_id = ?", "expired-notif").Count(&count)
		assert.Equal(t, int64(0), count)

		db.Postgresql.Model(&models.OneSignalNotification{}).Where("onesignal_notification_id = ?", "active-notif").Count(&count)
		assert.Equal(t, int64(1), count)
	})
}
