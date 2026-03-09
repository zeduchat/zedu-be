package models

import "time"

type BroadcastNotificationRequest struct {
	Title       string     `json:"title" validate:"required"`
	Message     string     `json:"message" validate:"required"`
	AvatarUrl   string     `json:"avatar_url"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

type BroadcastNotificationResponse struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Message            string     `json:"message"`
	TotalUsersTargeted int        `json:"total_users_targeted"`
	SuccessfullySent   int        `json:"successfully_sent"`
	CreatedAt          time.Time  `json:"created_at"`
	ScheduledAt        *time.Time `json:"scheduled_at,omitempty"`
}

type BroadcastNotificationLog struct {
	ID                 string     `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	AdminID            string     `gorm:"type:uuid;not null;index" json:"admin_id"`
	AdminEmail         string     `gorm:"type:varchar(255)" json:"admin_email"`
	Title              string     `gorm:"type:varchar(255);not null" json:"title"`
	Message            string     `gorm:"type:text;not null" json:"message"`
	AvatarUrl          string     `gorm:"type:text" json:"avatar_url"`
	TotalUsersTargeted int        `gorm:"not null" json:"total_users_targeted"`
	SuccessfullySent   int        `gorm:"not null" json:"successfully_sent"`
	FailedCount        int        `gorm:"not null" json:"failed_count"`
	ScheduledAt        *time.Time `gorm:"type:timestamp;null" json:"scheduled_at"`
	IPAddress          string     `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent          string     `gorm:"type:text" json:"user_agent"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null;autoCreateTime;index" json:"created_at"`
}
