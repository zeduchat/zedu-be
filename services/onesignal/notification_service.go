package onesignal

import (
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"gorm.io/gorm"
)

type OneSignalNotificationService struct {
	db *gorm.DB
}

func NewOneSignalNotificationService(db *gorm.DB) *OneSignalNotificationService {
	return &OneSignalNotificationService{db: db}
}

func (s *OneSignalNotificationService) SaveNotification(onesignalID, userID, title, message, avatarURL string, payload map[string]interface{}) (*models.OneSignalNotification, error) {
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

	err := notification.Create(s.db)
	if err != nil {
		return nil, err
	}

	return notification, nil
}

func (s *OneSignalNotificationService) SaveBatchNotifications(onesignalID string, userIDs []string, title, message, avatarURL string, payload map[string]interface{}) error {
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

		err := notification.Create(s.db)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *OneSignalNotificationService) GetUserNotifications(userID string, page, pageSize int, status string, startDate, endDate *time.Time) (*models.OneSignalNotificationPaginationResponse, error) {
	notifications, total, err := models.GetNotificationsByUser(s.db, userID, page, pageSize, status, startDate, endDate)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &models.OneSignalNotificationPaginationResponse{
		Notifications: notifications,
		Page:          page,
		PageSize:      pageSize,
		TotalItems:    total,
		TotalPages:    totalPages,
	}, nil
}

func (s *OneSignalNotificationService) UpdateNotificationStatus(onesignalID string, status models.OneSignalNotificationStatus) error {
	var notification models.OneSignalNotification
	err := s.db.Where("onesignal_notification_id = ? AND deleted_at IS NULL", onesignalID).First(&notification).Error
	if err != nil {
		return err
	}

	notification.Status = status

	if status == models.OneSignalNotificationStatusDelivered {
		now := time.Now()
		notification.DeliveredAt = &now
	}

	return notification.Update(s.db)
}

func (s *OneSignalNotificationService) MarkNotificationAsRead(notificationID string) error {
	var notification models.OneSignalNotification
	err := s.db.Where("id = ? AND deleted_at IS NULL", notificationID).First(&notification).Error
	if err != nil {
		return err
	}

	return notification.MarkAsRead(s.db)
}
