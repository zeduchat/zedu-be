package models

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type Profile struct {
	ID                string         `gorm:"type:uuid;primary_key" json:"profile_id"`
	FirstName         string         `gorm:"column:first_name; type:text; not null" json:"first_name"`
	LastName          string         `gorm:"column:last_name; type:text;not null" json:"last_name"`
	FullName          string         `gorm:"column:full_name; type:text;" json:"full_name"`
	UserName          string         `gorm:"column:user_name; type:text;" json:"username"`
	Phone             string         `gorm:"type:varchar(255)" json:"phone"`
	AvatarURL         string         `gorm:"type:varchar(255)" json:"avatar_url"`
	Userid            string         `gorm:"type:uuid;" json:"user_id"`
	CreatedAt         time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	DisplayName       string         `gorm:"type:varchar(255)" json:"display_name"`
	Status            string         `gorm:"type:varchar(255)" json:"status"`
	Title             string         `gorm:"type:varchar(255)" json:"title"`
	NamePronunciation string         `gorm:"type:varchar(255)" json:"name_pronunciation"`
	Timezone          string         `gorm:"type:varchar(255)" json:"timezone"`
}

type ProfileSummary struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	FullName          string `json:"full_name"`
	UserName          string `json:"username"`
	AvatarURL         string `json:"avatar_url"`
	UserId            string `json:"user_id"`
	Deactivated       bool   `json:"deactivated"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	DeletedAt         string `json:"deleted_at"`
	ProfileUpdated    bool   `json:"profile_updated"`
	IsOnboarded       bool   `json:"is_onboarded"`
	DisplayName       string `json:"display_name"`
	Title             string `json:"title"`
	NamePronunciation string `json:"name_pronounciation"`
	Timezone          string `json:"timezone"`
	Status            string `json:"status"`
}

type UpdateUserProfileRequest struct {
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	FullName          string `json:"full_name"`
	UserName          string `json:"username"`
	AvatarURL         string `json:"avatar_url"`
	DisplayName       string `json:"display_name"`
	Title             string `json:"title"`
	NamePronunciation string `json:"name_pronounciation"`
	Timezone          string `json:"timezone"`
}

type UpdateProfileStatus struct {
	Status string `json:"status"`
	UserId string
}

func (j *Profile) UpdateProfileFields(db *gorm.DB, req UpdateUserProfileRequest, userId string) error {
	var userProfile Profile

	profileUpdates := Profile{
		FullName:          req.FullName,
		UserName:          req.UserName,
		Phone:             req.Phone,
		AvatarURL:         req.AvatarURL,
		DisplayName:       req.DisplayName,
		Title:             req.Title,
		NamePronunciation: req.NamePronunciation,
		Timezone:          req.Timezone,
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

func (j *Profile) UpdateProfileStatus(db *gorm.DB, req UpdateProfileStatus) error {
	var userProfile Profile

	profileUpdates := Profile{
		Status: req.Status,
	}

	query := "userid = ?"

	exist := postgresql.CheckExists(db, &userProfile, query, req.UserId)
	if !exist {
		return errors.New("Profile does not exists")
	}

	result, err := postgresql.UpdateFields(db, &j, profileUpdates, query, req.UserId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update user profile")
	}

	return nil
}

func (p *Profile) GetUserByUsername(db *gorm.DB, userName string) (Profile, error) {
	var user Profile

	query := db.Where("user_name = ?", userName)

	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}

func (p *Profile) SetProfileImageToEmpty(db *gorm.DB, userId string) error {
	var userProfile Profile

	exists := postgresql.CheckExists(db, &userProfile, "userid = ?", userId)

	if !exists {
		return errors.New("profile does not exist")
	}

	result := db.Model(&Profile{}).Where("userid = ?", userId).Update("avatar_url", "")

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update avatar URL")
	}

	return nil
}

func (p *Profile) GetProfileByUserId(db *gorm.DB, userId string) error {

	query := db.Where("userid = ?", userId)

	if err := query.First(&p).Error; err != nil {
		return err
	}

	return nil
}
