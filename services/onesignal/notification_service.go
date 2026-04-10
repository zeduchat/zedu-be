package onesignal

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
)

func SaveNotification(db *gorm.DB, onesignalID, userID, title, message, avatarURL string, payload map[string]interface{}) (*models.OneSignalNotification, int, error) {
	notification := &models.OneSignalNotification{
		OneSignalNotificationID: onesignalID,
		UserID:                  userID,
		Title:                   title,
		Message:                 message,
		Payload:                 payload,
		AvatarURL:               avatarURL,
		Status:                  models.OneSignalNotificationStatusPending,
		SentAt:                  time.Now(),
	}

	err := notification.Create(db)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to save notification: %w", err)
	}

	return notification, http.StatusCreated, nil
}

func SaveBatchNotifications(db *gorm.DB, onesignalID string, userIDs []string, title, message, avatarURL string, payload map[string]interface{}) (int, int, error) {
	successCount := 0
	concatErr := ""

	for _, userID := range userIDs {
		notification := &models.OneSignalNotification{
			OneSignalNotificationID: onesignalID,
			UserID:                  userID,
			Title:                   title,
			Message:                 message,
			Payload:                 payload,
			AvatarURL:               avatarURL,
			Status:                  models.OneSignalNotificationStatusPending,
			SentAt:                  time.Now(),
		}

		err := notification.Create(db)
		if err != nil {
			concatErr += err.Error() + ", "
			continue
		}
		successCount++
	}

	if successCount == 0 {
		return 0, http.StatusInternalServerError, fmt.Errorf("failed to save any notifications: %s", concatErr)
	}

	return successCount, http.StatusOK, nil
}

func GetUserNotifications(db *gorm.DB, c *gin.Context) (*models.OneSignalNotificationPaginationResponse, int, error) {
	userID, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("authentication required: %w", err)
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid user ID type")
	}

	page := c.GetInt("page")
	if page < 1 {
		page = 1
	}

	pageSize := c.GetInt("page_size")
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := c.Query("status")

	var startDate, endDate *time.Time
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, startDateStr)
		if err == nil {
			startDate = &parsedTime
		}
	}

	if endDateStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, endDateStr)
		if err == nil {
			endDate = &parsedTime
		}
	}

	notifications, total, err := models.GetNotificationsByUser(db, userIDStr, page, pageSize, status, startDate, endDate)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch notifications: %w", err)
	}

	totalPages := int(total) / int(pageSize)
	if int(total)%int(pageSize) > 0 {
		totalPages++
	}

	return &models.OneSignalNotificationPaginationResponse{
		Notifications: notifications,
		Page:          page,
		PageSize:      pageSize,
		TotalItems:    total,
		TotalPages:    totalPages,
	}, http.StatusOK, nil
}

func UpdateNotificationStatus(db *gorm.DB, onesignalID string, status models.OneSignalNotificationStatus) error {
	var notification models.OneSignalNotification
	result := db.Model(&notification).Where("onesignal_notification_id = ? AND deleted_at IS NULL", onesignalID).First(&notification)
	if result.Error != nil {
		return fmt.Errorf("notification not found: %w", result.Error)
	}

	notification.Status = status

	if status == models.OneSignalNotificationStatusDelivered {
		now := time.Now()
		notification.DeliveredAt = &now
	}

	err := notification.Update(db)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	return nil
}

func MarkNotificationAsRead(db *gorm.DB, notificationID, userID string) (string, int, error) {
	var notification models.OneSignalNotification
	result := db.Model(&notification).Where("id = ? AND user_id = ? AND deleted_at IS NULL", notificationID, userID).First(&notification)
	if result.Error != nil {
		return "", http.StatusNotFound, fmt.Errorf("notification not found: %w", result.Error)
	}

	err := notification.MarkAsRead(db)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return notification.ID, http.StatusOK, nil
}
