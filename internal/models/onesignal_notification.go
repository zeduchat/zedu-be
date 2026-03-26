package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OneSignalNotificationStatus string

const (
	OneSignalNotificationStatusPending   OneSignalNotificationStatus = "pending"
	OneSignalNotificationStatusDelivered OneSignalNotificationStatus = "delivered"
	OneSignalNotificationStatusRead      OneSignalNotificationStatus = "read"
	OneSignalNotificationStatusFailed    OneSignalNotificationStatus = "failed"
)

type OneSignalNotification struct {
	ID                      uuid.UUID                   `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OneSignalNotificationID string                      `json:"onesignal_notification_id" gorm:"column:onesignal_notification_id;not null;index"`
	UserID                  string                      `json:"user_id" gorm:"column:user_id;not null;index"`
	Title                   string                      `json:"title" gorm:"column:title;not null;size:500"`
	Message                 string                      `json:"message" gorm:"column:message;not null;type:text"`
	Payload                 map[string]interface{}      `json:"payload,omitempty" gorm:"column:payload;type:jsonb"`
	AvatarURL               string                      `json:"avatar_url,omitempty" gorm:"column:avatar_url;size:500"`
	Status                  OneSignalNotificationStatus `json:"status" gorm:"column:status;not null;default:'pending';index"`
	SentAt                  time.Time                   `json:"sent_at" gorm:"column:sent_at;not null;default:now();index"`
	DeliveredAt             *time.Time                  `json:"delivered_at,omitempty" gorm:"column:delivered_at"`
	ReadAt                  *time.Time                  `json:"read_at,omitempty" gorm:"column:read_at"`
	CreatedAt               time.Time                   `json:"created_at" gorm:"column:created_at;not null;default:now()"`
	UpdatedAt               time.Time                   `json:"updated_at" gorm:"column:updated_at;not null;default:now()"`
	DeletedAt               *time.Time                  `json:"deleted_at,omitempty" gorm:"column:deleted_at;index"`
}

type OneSignalNotificationPaginationResponse struct {
	Notifications []OneSignalNotification `json:"notifications"`
	Page          int                     `json:"page"`
	PageSize      int                     `json:"page_size"`
	TotalItems    int64                   `json:"total_items"`
	TotalPages    int                     `json:"total_pages"`
}

func (n *OneSignalNotification) Create(db *gorm.DB) error {
	return db.Create(n).Error
}

func (n *OneSignalNotification) Update(db *gorm.DB) error {
	return db.Save(n).Error
}

func (n *OneSignalNotification) MarkAsDelivered(db *gorm.DB) error {
	now := time.Now()
	n.Status = OneSignalNotificationStatusDelivered
	n.DeliveredAt = &now
	return db.Model(n).Updates(map[string]interface{}{
		"status":       n.Status,
		"delivered_at": n.DeliveredAt,
	}).Error
}

func (n *OneSignalNotification) MarkAsRead(db *gorm.DB) error {
	now := time.Now()
	n.Status = OneSignalNotificationStatusRead
	n.ReadAt = &now
	return db.Model(n).Updates(map[string]interface{}{
		"status":  n.Status,
		"read_at": n.ReadAt,
	}).Error
}

func (n *OneSignalNotification) MarkAsFailed(db *gorm.DB) error {
	n.Status = OneSignalNotificationStatusFailed
	return db.Model(n).Updates(map[string]interface{}{
		"status": n.Status,
	}).Error
}

func GetNotificationsByUser(db *gorm.DB, userID string, page, pageSize int, status string, startDate, endDate *time.Time) ([]OneSignalNotification, int64, error) {
	var notifications []OneSignalNotification
	var total int64

	query := db.Model(&OneSignalNotification{}).Where("user_id = ? AND deleted_at IS NULL", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if startDate != nil {
		query = query.Where("sent_at >= ?", *startDate)
	}

	if endDate != nil {
		query = query.Where("sent_at <= ?", *endDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("sent_at DESC").Limit(pageSize).Offset(offset).Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}
