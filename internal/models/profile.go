package models

import (
	"errors"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Profile struct {
	ID        string         `gorm:"type:uuid;primary_key" json:"profile_id"`
	FirstName string         `gorm:"column:first_name; type:text; not null" json:"first_name"`
	LastName  string         `gorm:"column:last_name; type:text;not null" json:"last_name"`
	FullName  string         `gorm:"column:full_name; type:text;" json:"full_name"`
	UserName  string         `gorm:"column:user_name; type:text;" json:"user_name"`
	Phone     string         `gorm:"type:varchar(255)" json:"phone"`
	AvatarURL string         `gorm:"type:varchar(255)" json:"avatar_url"`
	Userid    string         `gorm:"type:uuid;" json:"user_id"`
	CreatedAt time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProfileSummary struct {
	ID                string  `json:"id"`
	Email 		      string  `json:"email"`
	Phone             string  `json:"phone"`
	FirstName         string  `json:"first_name"`
	LastName          string  `json:"last_name"`
	FullName          string  `json:"full_name"`
	UserName          string  `json:"user_name"`
	AvatarURL	      string  `json:"avatar_url"`
	UserId         	  string  `json:"user_id"`	
	CreatedAt         string  `json:"created_at"`	
	UpdatedAt         string  `json:"updated_at"`	
	DeletedAt         string  `json:"deleted_at"`	
}

type UpdateUserProfileRequest struct {
	Email     string `json:"email"`
	Phone     string `json:"phone"`
    FullName  string `json:"full_name"`
    UserName  string `json:"user_name"`
    AvatarURL string `json:"avatar_url"`
}


func (j *Profile) UpdateProfileFields(db *gorm.DB, req UpdateUserProfileRequest, userId string) error {
	var userProfile Profile

	profileUpdates := Profile{
		FullName:  req.FullName,
		UserName:  req.UserName,
		Phone:     req.Phone,
		AvatarURL: req.AvatarURL,
	}

	query := "userid = ?"

	exist := postgresql.CheckExists(db, &userProfile, query, userId)
	if !exist {
		return errors.New("Profile does not exists")
	}

	result, err := postgresql.UpdateFields(db, &j, profileUpdates, query, userId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update user profile")
	}

	return nil
}