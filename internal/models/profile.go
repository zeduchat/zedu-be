package models

import (
	"time"

	"gorm.io/gorm"
)

type Profile struct {
	ID        string         `gorm:"type:uuid;primary_key" json:"id"`
	FirstName string         `gorm:"column:first_name; type:text; not null" json:"first_name"`
	LastName  string         `gorm:"column:last_name; type:text;not null" json:"last_name"`
	FullName  string         `gorm:"column:full_name; type:text;" json:"full_name"`
	UserName  string         `gorm:"column:user_name; type:text;" json:"user_name"`
	Phone     string         `gorm:"type:varchar(255)" json:"phone"`
	AvatarURL string         `gorm:"type:varchar(255)" json:"avatar_url"`
	Userid    string         `gorm:"column:userid; type:uuid;" json:"user_id"`
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
    Email     *string `json:"email,omitempty"`
    Phone     *string `json:"phone,omitempty"`
    FullName  *string `json:"full_name,omitempty"`
    UserName  *string `json:"user_name,omitempty"`
    AvatarURL *string `json:"avatar_url,omitempty"`
}




func (profile *Profile) UpdateProfileFields(db *gorm.DB, req UpdateUserProfileRequest, profileId string) error {
	updates := map[string]interface{}{}
	if req.FullName != nil {
		updates["full_name"] = *req.FullName
	}
	if req.UserName != nil {
		updates["user_name"] = *req.UserName
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.AvatarURL != nil {
		updates["avatar_url"] = *req.AvatarURL
	}

	if len(updates) > 0 {
		if err := db.Model(profile).Where("id = ?", profileId).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

func PrepareResponseData(req UpdateUserProfileRequest) map[string]interface{} {
	responseData := map[string]interface{}{}
	if req.Email != nil {
		responseData["email"] = *req.Email
	}
	if req.FullName != nil {
		responseData["full_name"] = *req.FullName
	}
	if req.UserName != nil {
		responseData["user_name"] = *req.UserName
	}
	if req.Phone != nil {
		responseData["phone"] = *req.Phone
	}
	if req.AvatarURL != nil {
		responseData["avatar_url"] = *req.AvatarURL
	}
	return responseData
}