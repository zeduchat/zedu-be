package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/hngprojects/telex_be/utility"
)

type UserDeviceToken struct {
	ID        string    `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	UserId    string    `gorm:"column:user_id;type:uuid;not null;index" json:"user_id"`
	Token     string    `gorm:"column:token;type:text;not null;uniqueIndex" json:"token"`
	Platform  string    `gorm:"column:platform;type:varchar(20);not null" json:"platform"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (UserDeviceToken) TableName() string {
	return "user_device_tokens"
}

type DeleteFcmTokenRequest struct {
	FcmToken string `json:"fcm_token" validate:"required"`
}

func (token *UserDeviceToken) Upsert(db *gorm.DB) error {
	var existing UserDeviceToken
	err := db.Where("user_id = ? AND platform = ?", token.UserId, token.Platform).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if token.ID == "" {
			token.ID = utility.GenerateUUID()
		}
		return db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "token"}},
			DoUpdates: clause.AssignmentColumns([]string{"user_id", "platform", "updated_at"}),
		}).Create(token).Error
	}
	if err != nil {
		return err
	}

	token.ID = existing.ID
	return db.Model(&existing).Update("token", token.Token).Error
}

func ListDeviceTokensByUserIDs(db *gorm.DB, userIDs []string) ([]UserDeviceToken, error) {
	var tokens []UserDeviceToken
	if len(userIDs) == 0 {
		return tokens, nil
	}

	err := db.Where("user_id IN ? AND token <> ''", userIDs).Find(&tokens).Error
	return tokens, err
}

func DeleteDeviceToken(db *gorm.DB, userID, token string) error {
	return db.Where("user_id = ? AND token = ?", userID, token).Delete(&UserDeviceToken{}).Error
}

func DeleteDeviceTokens(db *gorm.DB, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	return db.Where("token IN ?", tokens).Delete(&UserDeviceToken{}).Error
}
