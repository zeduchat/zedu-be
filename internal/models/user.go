package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type User struct {
	ID                          string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name                        string         `gorm:"column:name; type:varchar(255)" json:"name"`
	Email                       string         `gorm:"column:email; type:varchar(255)" json:"email"`
	IsVerified                  bool           `gorm:"column:is_verified; type:bool" json:"is_verified"`
	Deactivated                 bool           `gorm:"column:deactivated; type:bool" json:"deactivated"`
	IsActive                    bool           `gorm:"column:is_active; type:bool; default:false" json:"is_active"`
	IsOnboarded                 bool           `gorm:"column:is_onboarded; type:bool" json:"is_onboarded"`
	ProfileUpdated              bool           `gorm:"column:profile_updated; type:bool" json:"profile_updated"`
	CurrentOrg                  uuid.UUID      `gorm:"column:current_org;null; type:uuid" json:"current_org"`
	OneSignalSubscriptionID     string         `gorm:"column:onesignal_subscription_id; type:varchar(255); null" json:"onesignal_subscription_id"`
	Profile                     Profile        `gorm:"foreignKey:Userid;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"profile"`
	Channelss                   []Channels     `gorm:"many2many:user_channels;" json:"channels"`
	Organisations               []Organisation `gorm:"many2many:user_organisations;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"organisations"`
	OrgRoleID                   *string        `gorm:"type:varchar(100);null;index" json:"org_role_id"`
	UserRoleID                  *string        `gorm:"type:varchar(100);null;index" json:"user_role_id"`
	OrgRole                     OrgRole        `gorm:"foreignKey:OrgRoleID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"org_role"`
	Password                    string         `gorm:"column:password; type:text; not null" json:"-"`
	CreatedAt                   time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt                   time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	LastLogInAt                 *time.Time     `gorm:"column:last_log_in_at;null" json:"last_log_in_at"`
	LastActivityAt              *time.Time     `gorm:"column:last_activity_at;null" json:"last_activity_at"`
	LastActivityStreakStartedAt *time.Time     `gorm:"column:last_activity_started_at;null" json:"-"`
	DeletedAt                   gorm.DeletedAt `gorm:"index" json:"-"`
	Role                        int            `gorm:"column:role" json:"role"`
}

type CreateUserRequestModel struct {
	Email       string `json:"email" validate:"required"`
	Password    string `json:"password" validate:"required"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name" `
	UserName    string `json:"username"`
	PhoneNumber string `json:"phone_number"`
	IsOnboarded bool   `json:"is_onboarded"`
	AvatarUrl   string `json:"avatar_url"`
}

type UpdateUserRequestModel struct {
	FirstName   string `json:"first_name" validate:"required"`
	LastName    string `json:"last_name" validate:"required"`
	UserName    string `json:"username" validate:"required"`
	PhoneNumber string `json:"phone_number"`
}

type LoginRequestModel struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LogoutReqModel struct {
	AccessUuid string
	Platform   string
	UserId     string
}

type SwitchUserOrgReqeust struct {
	CurrentOrg string `json:"current_org" validate:"required"`
}

type SwitchUserRoleRequest struct {
	UserId string `json:"user_id" validate:"required,uuid"`
	OrgId  string `json:"org_id" validate:"required,uuid"`
	RoleId string `json:"role_id" validate:"required,uuid"`
}

type OneSignalSubscriptionIDRequest struct {
	SubscriptionID string `json:"subscription_id" binding:"required"`
	Platform       string `json:"platform"`
	OrgID          string `json:"org_id"`
}

type UserMentionResponse struct {
	Username         string `json:"username"`
	FullName         string `json:"fullname"`
	FirstName        string `json:"firstname"`
	LastName         string `json:"lastname"`
	AvatarURL        string `json:"avatar_url"`
	DefaultAvatarURL string `json:"default_avatar_url"`
	DisplayName      string `json:"display_name"`
	StatusText       string `json:"status_text"`
	UserID           string `json:"userid"`
	OnlineStatus     bool   `json:"online_status"`
}


func (u *User) AddUserToOrganisation(db *gorm.DB, user any, orgs []any) error {
	err := db.Model(user).Association("Organisations").Append(orgs...)
	if err != nil {
		return err
	}

	var userID string
	switch v := user.(type) {
	case *User:
		userID = v.ID
	case User:
		userID = v.ID
	case string:
		userID = v
	}

	if userID != "" {
		var profModel Profile
		for _, o := range orgs {
			var orgID string
			switch v := o.(type) {
			case *Organisation:
				orgID = v.ID
			case Organisation:
				orgID = v.ID
			case string:
				orgID = v
			}
			if orgID != "" {
				_, _ = profModel.GetOrCreateProfileForOrg(db, userID, orgID)
			}
		}
	}

	return nil
}

func (u *User) RemoveUserFromOrganisation(db *gorm.DB, user any, orgs []any) error {
	err := db.Model(user).Association("Organisations").Delete(orgs...)
	if err != nil {
		return err
	}

	var userID string
	switch v := user.(type) {
	case *User:
		userID = v.ID
	case User:
		userID = v.ID
	case string:
		userID = v
	}

	if userID != "" {
		for _, o := range orgs {
			var orgID string
			switch v := o.(type) {
			case *Organisation:
				orgID = v.ID
			case Organisation:
				orgID = v.ID
			case string:
				orgID = v
			}
			if orgID != "" {
				_ = db.Where("userid = ? AND organisation_id = ?", userID, orgID).Delete(&Profile{}).Error
			}
		}
	}

	return nil
}

func (u *User) GetUserByID(db *gorm.DB, userID string, orgID ...string) (User, error) {
	var user User

	err, _ := postgresql.SelectOneFromDb(db, &user, "id = ?", userID)
	if err != nil {
		return User{}, fmt.Errorf("user not found: %w", err)
	}

	targetOrg := ""
	if len(orgID) > 0 && orgID[0] != "" {
		targetOrg = orgID[0]
	} else if user.CurrentOrg.String() != "" && user.CurrentOrg.String() != "00000000-0000-0000-0000-000000000000" {
		targetOrg = user.CurrentOrg.String()
	}

	var profModel Profile
	prof, err := profModel.GetOrCreateProfileForOrg(db, userID, targetOrg)
	if err == nil {
		user.Profile = prof
	}

	return user, nil
}

func (u *User) GetUserByEmail(db *gorm.DB, userEmail string) (User, error) {
	var user User

	query := db.Where("email = ?", userEmail)
	query = postgresql.PreloadEntities(query, &user, "Profile", "Organisations")

	if err := query.First(&user).Error; err != nil {
		return user, fmt.Errorf("user with email %s not found: %w", userEmail, err)
	}
	return user, nil
}

func (u *User) CreateUser(db *gorm.DB) error {

	if u.OrgRoleID != nil && *u.OrgRoleID == "" {
		u.OrgRoleID = nil
	}

	err := postgresql.CreateOneRecord(db, &u)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) Update(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &u)
	return err
}

func (u *User) DeleteAUser(db *gorm.DB) error {

	err := postgresql.DeleteRecordFromDb(db, u)

	if err != nil {
		return err
	}

	return nil
}

func (u *User) GetProfileID(db *gorm.DB, userID string) (string, error) {
	var user User

	query := db.Where("id = ?", userID)
	query = postgresql.PreloadEntities(query, &user, "Profile")

	if err := query.First(&user).Error; err != nil {
		return user.Profile.ID, err
	}

	return user.Profile.ID, nil
}

func (u *User) GetOneSignalSubscriptionID(db *gorm.DB, userID string) (string, error) {
	var user User

	query := db.Where("id = ?", userID)

	if err := query.First(&user).Error; err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	return user.OneSignalSubscriptionID, nil
}

func (u *User) GetUserWithProfile(db *gorm.DB, userID string) (User, error) {
	var user User

	query := db.Where("id = ?", userID)
	query = postgresql.PreloadEntities(query, &user, "Profile")

	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}

func (u *User) CheckUserIsAdmin(db *gorm.DB) bool {
	return u.Role == int(RoleIdentity.SuperAdmin)
}

func (u *User) GetUserByIDsAdmin(db *gorm.DB, userID, requesterID string) (User, error) {

	var (
		ErrNotFound = errors.New("user not found")
		user        = User{}
	)

	var isOwner bool
	err := db.Model(&Organisation{}).
		Select("count(*) > 0").
		Where("owner_id = ? AND id IN (SELECT organisation_id FROM user_organisations WHERE user_id = ?)", requesterID, userID).
		Find(&isOwner).
		Error
	if err != nil {
		return user, err
	}

	if isOwner {
		query := db.Model(&User{}).
			Joins("INNER JOIN user_organisations uo ON users.id = uo.user_id").
			Where("uo.organisation_id IN (SELECT organisation_id FROM user_organisations WHERE user_id = ?)", userID)
		query = postgresql.PreloadEntities(query, &user, "Profile", "Organisations")

		if err := query.First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return user, ErrNotFound
			}
			return user, err
		}
		return user, nil
	}

	query := db.Model(&User{}).
		Joins("INNER JOIN user_organisations uo ON users.id = uo.user_id").
		Where("users.id = ? AND users.id = ?", userID, requesterID)
	query = postgresql.PreloadEntities(query, &user, "Profile", "Organisations")

	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, ErrNotFound
		}
		return user, err
	}

	return user, nil
}

func (user *User) UpdateUserEmail(db *gorm.DB, req UpdateUserProfileRequest, userId string) error {

	userUpdates := User{Email: req.Email}

	result, err := postgresql.UpdateFields(db, &user, userUpdates, "id = ?", userId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update user profile")
	}

	return nil
}

func (user *User) DeactivateUser(db *gorm.DB, userId string) error {
	userUpdates := map[string]any{
		"deactivated": true,
	}

	result, err := postgresql.UpdateFields(db, &user, userUpdates, "id = ?", userId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to deactivate user")
	}

	return nil
}

func (user *User) ActivateOrgMember(db *gorm.DB, orgID string) error {
	result := db.Model(&OrgUserManagement{}).
		Where("user_id = ? AND organisation_id = ?", user.ID, orgID).
		Updates(map[string]any{
			"is_deactivated": false,
			"status":         "active",
		})

	if result.Error != nil {
		return fmt.Errorf("failed to activate member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("user does not exist in organisation")
	}
	return nil
}

func (user *User) DeactivateOrgMember(db *gorm.DB, orgID string) error {
	result := db.Model(&OrgUserManagement{}).
		Where("user_id = ? AND organisation_id = ?", user.ID, orgID).
		Updates(map[string]any{
			"is_deactivated": true,
			"status":         "inactive",
		})

	if result.Error != nil {
		return fmt.Errorf("failed to deactivate member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("user does not exist in organisation")
	}
	return nil
}

func (user *User) ActivateUser(db *gorm.DB, userId string) error {
	userUpdates := map[string]any{
		"deactivated": false,
	}

	result, err := postgresql.UpdateFields(db, &user, userUpdates, "id = ?", userId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to activate user")
	}

	return nil
}

func (u *User) CheckUserExists(db *gorm.DB, userID string) bool {
	return postgresql.CheckExists(db, u, "id = ?", userID)
}

func ValidateUserIDs(db *gorm.DB, orgID string, userIDs []string) ([]string, []string, error) {
	var validIDs []string
	if err := db.Table("users").
		Where("id IN ?", userIDs).
		Pluck("id", &validIDs).Error; err != nil {
		return nil, nil, fmt.Errorf("error validating users: %w", err)
	}

	validMap := make(map[string]bool)
	for _, id := range validIDs {
		validMap[id] = true
	}

	var invalid []string
	for _, id := range userIDs {
		if !validMap[id] {
			invalid = append(invalid, id)
		}
	}

	return validIDs, invalid, nil
}
