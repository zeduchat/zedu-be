package models

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type NotifyOption struct {
	Option string
}

type AppearanceOption struct {
	Position string
}

var (
	NotifyAllMessages    = NotifyOption{Option: "all_new_messages"}
	NotifyDirectMentions = NotifyOption{Option: "direct_messages_mentions"}
	NotifyNothing        = NotifyOption{Option: "nothing"}
)

type NotificationPreferences struct {
	ID                      string         `gorm:"type:uuid;primaryKey;unique;" json:"id"`
	UserID                  uuid.UUID      `gorm:"type:uuid;" json:"user_id"`
	NotifyAbout             NotifyOption   `gorm:"embedded" json:"notify_about"`
	NotificationSchedule    bool           `gorm:"default:false" json:"notification_schedule"`
	FromHour                string         `gorm:"type:varchar(5);default:'00:00'" json:"from_hour"`
	ToHour                  string         `gorm:"type:varchar(5);default:'00:00'" json:"to_hour"`
	NotificationMethodEmail bool           `gorm:"default:false" json:"notification_method_email"`
	CreatedAt               time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt               gorm.DeletedAt `gorm:"index" json:"-"`
}

func (d *NotificationPreferences) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &d)

	if err != nil {
		return err
	}

	return nil
}

func (d *NotificationPreferences) Update(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &d)
	return err
}

func (d *NotificationPreferences) GetUserDataByID(db *gorm.DB, userID string) (NotificationPreferences, error) {
	var user NotificationPreferences

	query := db.Where("user_id::uuid = ?", userID)
	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	return user, nil

}
