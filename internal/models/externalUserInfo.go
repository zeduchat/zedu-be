package models

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/utility"
)

type UnauthenticatedUser struct {
	ID              string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserIdentifier  string    `gorm:"index;not null" json:"user_identifier"`
	UsageCount      int64     `gorm:"not null;default:0" json:"usage_count"`
	ChannelID       string    `gorm:"type:uuid" json:"channel_id"`
	LastAgentChatID string    `gorm:"type:uuid" json:"last_agent_chat_id"`
	SubtokenID      string    `gorm:"type:uuid" json:"subtoken_id"`
	CreatedAt       time.Time `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}

type UnauthReq struct {
	UserIdentifier  string `json:"user_identifier"`
	ChannelID       string `json:"channel_id"`
	SubtokenID      string `json:"subtoken_id"`
	Limit           int64
	LastAgentChatID string
}

type UserUsageInfoResponse struct {
	ChannelID     string `json:"channel_id"`
	SubtokenID    string `json:"subtoken_id"`
	UsageCount    int64  `json:"usage"`
	LimitExceeded bool   `json:"limit_exceeded"`
}

func (d *UnauthenticatedUser) IncrementUsage(db *gorm.DB, req UnauthReq) (int, error) {
	var user UnauthenticatedUser

	err := db.Where("user_identifier = ?", req.UserIdentifier).First(&user).Error
	if err != nil {
		return http.StatusBadRequest, err
	}

	if time.Since(user.UpdatedAt) >= 24*time.Hour {
		user.UsageCount = 0
	}

	if user.UsageCount >= req.Limit {
		return http.StatusOK, nil
	}

	user.UsageCount++
	user.LastAgentChatID = req.LastAgentChatID

	return http.StatusOK, db.Save(&user).Error
}

func (d *UnauthenticatedUser) IsUsageGreaterThan(db *gorm.DB, req UnauthReq, logger *utility.Logger) (bool, error) {
	var user UnauthenticatedUser
	err := db.Where("user_identifier = ?", req.UserIdentifier).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	if time.Since(user.UpdatedAt) >= 24*time.Hour {
		user.UsageCount = 0
		if saveErr := db.Save(&user).Error; saveErr != nil {
			logger.Error("An error occured while resetting user usage count, error :%v", saveErr)
			return false, saveErr
		}
		return false, nil
	}

	return user.UsageCount > req.Limit, nil
}

func (d *UnauthenticatedUser) VerifySubtoken(db *gorm.DB, req UnauthReq) (bool, error) {
	var user UnauthenticatedUser
	err := db.Where("user_identifier = ? AND subtoken_id = ?", req.UserIdentifier, req.SubtokenID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, errors.New("Invalid token, user have not setup")
	}

	if user.UsageCount >= req.Limit && time.Since(user.UpdatedAt) < 24*time.Hour {
		return false, errors.New("Usage limit reached, can't subscribe again, try again tomorrow")
	}

	return true, nil
}

func (d *UnauthenticatedUser) GetOrCreateUserInfo(db *gorm.DB, userIdentifier string) (*UserUsageInfoResponse, error) {
	var user UnauthenticatedUser

	err := db.Where("user_identifier = ?", userIdentifier).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			newUser := UnauthenticatedUser{
				UserIdentifier: userIdentifier,
				ChannelID:      utility.GenerateUUID(),
				SubtokenID:     utility.GenerateUUID(),
				UsageCount:     0,
			}
			if err := db.Create(&newUser).Error; err != nil {
				return nil, err
			}
			user = newUser
		} else {
			return nil, err
		}
	}

	return &UserUsageInfoResponse{
		ChannelID:     user.ChannelID,
		SubtokenID:    user.SubtokenID,
		UsageCount:    user.UsageCount,
		LimitExceeded: user.UsageCount > 5,
	}, nil
}
