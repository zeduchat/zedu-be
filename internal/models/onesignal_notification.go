package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type OneSignalNotificationStatus string

const (
	OneSignalNotificationStatusPending   OneSignalNotificationStatus = "pending"
	OneSignalNotificationStatusDelivered OneSignalNotificationStatus = "delivered"
	OneSignalNotificationStatusRead      OneSignalNotificationStatus = "read"
	OneSignalNotificationStatusFailed    OneSignalNotificationStatus = "failed"
)

type OneSignalNotification struct {
	ID                      string                      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OneSignalNotificationID string                      `gorm:"column:onesignal_notification_id;not null;index" json:"onesignal_notification_id"`
	UserID                  string                      `gorm:"column:user_id;not null;index" json:"user_id"`
	OrgID                   *string                     `gorm:"column:org_id;type:uuid;index" json:"org_id,omitempty"`
	Title                   string                      `gorm:"column:title;not null;size:500" json:"title"`
	Message                 string                      `gorm:"column:message;not null;type:text" json:"message"`
	Payload                 map[string]interface{}      `gorm:"column:payload;type:jsonb;serializer:json" json:"payload,omitempty"`
	AvatarURL               string                      `gorm:"column:avatar_url;size:500" json:"avatar_url,omitempty"`
	Status                  OneSignalNotificationStatus `gorm:"column:status;not null;default:'pending';index" json:"status"`
	SentAt                  time.Time                   `gorm:"column:sent_at;not null;default:now();index" json:"sent_at"`
	DeliveredAt             *time.Time                  `gorm:"column:delivered_at" json:"delivered_at,omitempty"`
	ReadAt                  *time.Time                  `gorm:"column:read_at" json:"read_at,omitempty"`
	CreatedAt               time.Time                   `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt               time.Time                   `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	DeletedAt               gorm.DeletedAt              `gorm:"index" json:"-"`
}

func (OneSignalNotification) TableName() string {
	return "onesignal_notifications"
}

type OneSignalNotificationPaginationResponse struct {
	Notifications []OneSignalNotification `json:"notifications"`
	Page          int                     `json:"page"`
	PageSize      int                     `json:"page_size"`
	TotalItems    int64                   `json:"total_items"`
	TotalPages    int                     `json:"total_pages"`
}

func (n *OneSignalNotification) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &n)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

func (n *OneSignalNotification) Update(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &n)
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}
	return nil
}

func (n *OneSignalNotification) MarkAsDelivered(db *gorm.DB) error {
	now := time.Now()
	n.Status = OneSignalNotificationStatusDelivered
	n.DeliveredAt = &now

	updates := map[string]interface{}{
		"status":       n.Status,
		"delivered_at": n.DeliveredAt,
	}

	_, err := postgresql.UpdateFields(db, &n, updates, "id = ?", n.ID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as delivered: %w", err)
	}
	return nil
}

func (n *OneSignalNotification) MarkAsRead(db *gorm.DB) error {
	now := time.Now()
	n.Status = OneSignalNotificationStatusRead
	n.ReadAt = &now

	updates := map[string]interface{}{
		"status":  n.Status,
		"read_at": n.ReadAt,
	}

	_, err := postgresql.UpdateFields(db, &n, updates, "id = ?", n.ID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	return nil
}

func (n *OneSignalNotification) MarkAsFailed(db *gorm.DB) error {
	n.Status = OneSignalNotificationStatusFailed

	updates := map[string]interface{}{
		"status": n.Status,
	}

	_, err := postgresql.UpdateFields(db, &n, updates, "id = ?", n.ID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as failed: %w", err)
	}
	return nil
}

func GetNotificationsByUser(db *gorm.DB, userID string, page, pageSize int, status string, startDate, endDate *time.Time) ([]OneSignalNotification, int64, error) {
	var notifications []OneSignalNotification

	whereClause := "user_id = ?"
	args := []interface{}{userID}

	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	if startDate != nil {
		whereClause += " AND sent_at >= ?"
		args = append(args, *startDate)
	}

	if endDate != nil {
		whereClause += " AND sent_at <= ?"
		args = append(args, *endDate)
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"sent_at",
		"DESC",
		postgresql.Pagination{
			Page:  page,
			Limit: pageSize,
		},
		&notifications,
		whereClause,
		args...,
	)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch notifications: %w", err)
	}

	return notifications, paginationResponse.TotalItems, nil
}

func GetNotificationsByUserAndOrg(db *gorm.DB, userID, orgID string, page, pageSize int) ([]OneSignalNotification, postgresql.PaginationResponse, error) {
	var notifications []OneSignalNotification
	var total int64

	query := db.Model(&OneSignalNotification{}).
		Where("user_id = ? AND org_id = ? AND created_at > ?", userID, orgID, time.Now().AddDate(0, -2, 0))

	if err := query.Count(&total).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("sent_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&notifications).Error
	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	paginationResponse := postgresql.PaginationResponse{
		CurrentPage:     page,
		PageCount:       len(notifications),
		TotalPagesCount: totalPages,
		TotalItems:      total,
	}

	return notifications, paginationResponse, nil
}

func DeleteExpiredNotifications(db *gorm.DB) error {
	return db.Unscoped().Where("created_at <= ?", time.Now().AddDate(0, -2, 0)).Delete(&OneSignalNotification{}).Error
}
