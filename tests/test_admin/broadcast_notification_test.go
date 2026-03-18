package test_admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/admin"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestBroadcastNotification(t *testing.T) {
	r, _, _, db := SetupAdminTestRouter()
	superAdminID, superAdminToken := CreateSuperAdminAndGetTokenWithID(t, r, db)

	user1ID := utility.GenerateUUID()
	user2ID := utility.GenerateUUID()
	user3ID := utility.GenerateUUID()

	users := []models.User{
		{
			ID:                      user1ID,
			Name:                    "Test User 1",
			Email:                   "user1@test.com",
			OneSignalSubscriptionID: "onesignal_user1",
			IsActive:                true,
		},
		{
			ID:                      user2ID,
			Name:                    "Test User 2",
			Email:                   "user2@test.com",
			OneSignalSubscriptionID: "onesignal_user2",
			IsActive:                true,
		},
		{
			ID:                      user3ID,
			Name:                    "Test User 3",
			Email:                   "user3@test.com",
			OneSignalSubscriptionID: "",
			IsActive:                true,
		},
	}
	for _, user := range users {
		db.Postgresql.Create(&user)
	}

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, superAdminID, []string{})
		for _, user := range users {
			db.Postgresql.Exec("DELETE FROM users WHERE id = ?", user.ID)
		}
		db.Postgresql.Exec("DELETE FROM broadcast_notification_logs WHERE admin_id = ?", superAdminID)
		db.Postgresql.Exec("DELETE FROM audit_logs WHERE actor_id = ? AND action = ?", superAdminID, models.ActionBroadcastNotificationCreated)
	})

	t.Run("Broadcast Notification - Success", func(t *testing.T) {
		payload := models.BroadcastNotificationRequest{
			Title:     "Test Broadcast",
			Message:   "This is a test broadcast notification",
			AvatarUrl: "https://example.com/icon.png",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/notifications/broadcast", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		response := tst.ParseResponse(rr)
		assert.Equal(t, "success", response["status"])
		assert.Equal(t, "Broadcast notification sent successfully", response["message"])

		data := response["data"].(map[string]any)
		broadcastID := data["id"].(string)
		assert.NotEmpty(t, broadcastID)
		assert.Equal(t, payload.Title, data["title"])
		assert.Equal(t, payload.Message, data["message"])

		// Counts are 0 initially since processing is async
		totalTargeted := data["total_users_targeted"].(float64)
		successfullySent := data["successfully_sent"].(float64)
		assert.Equal(t, float64(0), totalTargeted)
		assert.Equal(t, float64(0), successfullySent)

		// Verify broadcast log was created with 'started' status
		var broadcastLog models.BroadcastNotificationLog
		err := db.Postgresql.Where("id = ?", broadcastID).First(&broadcastLog).Error
		assert.NoError(t, err)
		assert.Equal(t, "started", broadcastLog.Status)
	})

	t.Run("Broadcast Notification - Validation Error - Missing Title", func(t *testing.T) {
		// Use empty string so ShouldBindJSON passes, but validator rejects it
		payload := map[string]any{
			"title":      "",
			"message":    "This is a test broadcast notification",
			"avatar_url": "https://example.com/icon.png",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/notifications/broadcast", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		response := tst.ParseResponse(rr)
		assert.Equal(t, "error", response["status"])
		assert.Equal(t, "Validation failed", response["message"])
	})

	t.Run("Broadcast Notification - Validation Error - Missing Message", func(t *testing.T) {
		payload := map[string]any{
			"title":      "Test Broadcast",
			"message":    "",
			"avatar_url": "https://example.com/icon.png",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/notifications/broadcast", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		response := tst.ParseResponse(rr)
		assert.Equal(t, "error", response["status"])
		assert.Equal(t, "Validation failed", response["message"])
	})

	t.Run("Broadcast Notification - Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/notifications/broadcast", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		response := tst.ParseResponse(rr)
		assert.Equal(t, "error", response["status"])
		assert.Equal(t, "Failed to parse request body", response["message"])
	})
}

func TestBroadcastNotificationAuditLogs(t *testing.T) {
	r, _, _, db := SetupAdminTestRouter()
	superAdminID, superAdminToken := CreateSuperAdminAndGetTokenWithID(t, r, db)

	userID := utility.GenerateUUID()
	user := models.User{
		ID:                      userID,
		Name:                    "Audit Test User",
		Email:                   "audit@test.com",
		OneSignalSubscriptionID: "onesignal_audit",
		IsActive:                true,
	}
	db.Postgresql.Create(&user)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, superAdminID, []string{})
		db.Postgresql.Exec("DELETE FROM users WHERE id = ?", userID)
		db.Postgresql.Exec("DELETE FROM broadcast_notification_logs WHERE admin_id = ?", superAdminID)
		db.Postgresql.Exec("DELETE FROM audit_logs WHERE actor_id = ? AND action = ?", superAdminID, models.ActionBroadcastNotificationCreated)
	})

	t.Run("Broadcast Notification Creates Audit Log", func(t *testing.T) {
		payload := models.BroadcastNotificationRequest{
			Title:   "Audit Test Broadcast",
			Message: "Testing audit logging",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/notifications/broadcast", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var auditLogs []models.AuditLog
		db.Postgresql.Where(
			"actor_id = ? AND action = ?",
			superAdminID,
			models.ActionBroadcastNotificationCreated,
		).Find(&auditLogs)

		assert.Len(t, auditLogs, 1)
		auditLog := auditLogs[0]
		assert.Equal(t, models.ActionBroadcastNotificationCreated, auditLog.Action)
		assert.Equal(t, models.ResourceNotification, auditLog.ResourceType)
		assert.NotEmpty(t, auditLog.ResourceID)
		assert.Equal(t, "admin", auditLog.ActorRole)
		assert.True(t, auditLog.Success)
		assert.Contains(t, auditLog.Description, "sent broadcast notification")
		assert.NotEmpty(t, auditLog.IPAddress)
		assert.NotEmpty(t, auditLog.UserAgent)
	})

	t.Run("Broadcast Notification Creates Broadcast Log", func(t *testing.T) {
		payload := models.BroadcastNotificationRequest{
			Title:   "Broadcast Log Test",
			Message: "Testing broadcast logging",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/notifications/broadcast", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Scope by admin + unique title to isolate this test's log
		var broadcastLogs []models.BroadcastNotificationLog
		db.Postgresql.Where("admin_id = ? AND title = ?", superAdminID, payload.Title).Find(&broadcastLogs)

		assert.Len(t, broadcastLogs, 1)
		broadcastLog := broadcastLogs[0]
		assert.Equal(t, payload.Title, broadcastLog.Title)
		assert.Equal(t, payload.Message, broadcastLog.Message)
		assert.Equal(t, "started", broadcastLog.Status)     // Status should be 'started' initially
		assert.Equal(t, 0, broadcastLog.TotalUsersTargeted) // Will be updated by worker
		assert.Equal(t, 0, broadcastLog.SuccessfullySent)   // Will be updated by worker
		assert.NotEmpty(t, broadcastLog.IPAddress)
		assert.NotEmpty(t, broadcastLog.UserAgent)
	})
}

func TestBroadcastNotificationNoUsers(t *testing.T) {
	r, _, _, db := SetupAdminTestRouter()
	superAdminID, superAdminToken := CreateSuperAdminAndGetTokenWithID(t, r, db)

	db.Postgresql.Exec("UPDATE users SET is_active = false")

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, superAdminID, []string{})
		db.Postgresql.Exec("UPDATE users SET is_active = true")
		db.Postgresql.Exec("DELETE FROM broadcast_notification_logs WHERE admin_id = ?", superAdminID)
		db.Postgresql.Exec("DELETE FROM audit_logs WHERE actor_id = ? AND action = ?", superAdminID, models.ActionBroadcastNotificationCreated)
	})

	t.Run("Broadcast Notification With No Active Users", func(t *testing.T) {
		payload := models.BroadcastNotificationRequest{
			Title:   "No Users Broadcast",
			Message: "This should handle no users gracefully",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/notifications/broadcast", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		response := tst.ParseResponse(rr)
		assert.Equal(t, "success", response["status"])

		data := response["data"].(map[string]any)
		assert.Equal(t, float64(0), data["total_users_targeted"])
		assert.Equal(t, float64(0), data["successfully_sent"])
	})
}

func TestBroadcastNotificationStatus(t *testing.T) {
	r, _, _, db := SetupAdminTestRouter()
	superAdminID, superAdminToken := CreateSuperAdminAndGetTokenWithID(t, r, db)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, superAdminID, []string{})
		db.Postgresql.Exec("DELETE FROM broadcast_notification_logs WHERE admin_id = ?", superAdminID)
		db.Postgresql.Exec("DELETE FROM audit_logs WHERE actor_id = ? AND action = ?", superAdminID, models.ActionBroadcastNotificationCreated)
	})

	t.Run("Get Broadcast Notification Status - Started", func(t *testing.T) {
		// Create a broadcast notification
		payload := models.BroadcastNotificationRequest{
			Title:   "Status Test Broadcast",
			Message: "Testing status field",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/notifications/broadcast", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		response := tst.ParseResponse(rr)
		data := response["data"].(map[string]any)
		broadcastID := data["id"].(string)

		// Verify the broadcast log has 'started' status
		var broadcastLog models.BroadcastNotificationLog
		err := db.Postgresql.Where("id = ?", broadcastID).First(&broadcastLog).Error
		assert.NoError(t, err)
		assert.Equal(t, "started", broadcastLog.Status)
		assert.Equal(t, payload.Title, broadcastLog.Title)
		assert.Equal(t, payload.Message, broadcastLog.Message)
	})

	t.Run("Get Broadcast Notification Status - Not Found", func(t *testing.T) {
		nonExistentID := utility.GenerateUUID()

		// Attempt to retrieve non-existent broadcast notification
		broadcastLog, err := admin.GetBroadcastNotificationStatus(db, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, broadcastLog)
	})

	t.Run("Broadcast Log Contains Admin Information", func(t *testing.T) {
		payload := models.BroadcastNotificationRequest{
			Title:   "Admin Info Test",
			Message: "Testing admin information is saved",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/notifications/broadcast", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var broadcastLogs []models.BroadcastNotificationLog
		db.Postgresql.Where("admin_id = ? AND title = ?", superAdminID, payload.Title).Find(&broadcastLogs)

		assert.Len(t, broadcastLogs, 1)
		log := broadcastLogs[0]
		assert.Equal(t, superAdminID, log.AdminID)
		assert.NotEmpty(t, log.AdminEmail)
		assert.Equal(t, "203.0.113.1", log.IPAddress)
		assert.NotEmpty(t, log.UserAgent)
	})
}
