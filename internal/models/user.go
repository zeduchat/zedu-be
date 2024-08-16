package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type User struct {
	ID                 string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name               string         `gorm:"column:name; type:varchar(255)" json:"name"`
	Email              string         `gorm:"column:email; type:varchar(255)" json:"email"`
	IsVerified         bool           `gorm:"column:is_verified; type:bool" json:"is_verified"`
	Deactivated        bool           `gorm:"column:deactivated; type:bool" json:"deactivated"`
	IsActive           bool           `gorm:"column:is_active; type:bool; default:false" json:"is_active"`
	IsOnboarded        bool           `gorm:"column:is_onboarded; type:bool" json:"is_onboarded"`
	CurrentOrg         uuid.UUID      `gorm:"column:current_org;null; type:uuid" json:"current_org"`
	SubscriptionPlanId string         `gorm:"column:subscription_plan_id; type:varchar(255)" json:"subscription_plan_id"`
	Profile            Profile        `gorm:"foreignKey:Userid;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"profile"`
	Channelss          []Channels     `gorm:"many2many:user_channels;" json:"channels"`
	Organisations      []Organisation `gorm:"many2many:user_organisations;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"organisations"`
	OrgRoleID          *string        `gorm:"type:varchar(100);null;index" json:"org_role_id"`
	UserRoleID         *string        `gorm:"type:varchar(100);null;index" json:"user_role_id"`
	OrgRole            OrgRole        `gorm:"foreignKey:OrgRoleID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"org_role"`
	Password           string         `gorm:"column:password; type:text; not null" json:"-"`
	CreatedAt          time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
	StripeCustomerID   string         `gorm:"column:stripe_customer_id; type:varchar(255)" json:"stripe_customer_id"`
	Role               int            `gorm:"column:role" json:"role"`
}

type CreateUserRequestModel struct {
	Email       string `json:"email" validate:"required"`
	Password    string `json:"password" validate:"required"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name" `
	UserName    string `json:"username"`
	PhoneNumber string `json:"phone_number"`
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

type SwitchUserOrgReqeust struct {
	CurrentOrg string `json:"current_org" validate:"required"`
}

func (u *User) AddUserToOrganisation(db *gorm.DB, user interface{}, orgs []interface{}) error {

	err := db.Model(user).Association("Organisations").Append(orgs...)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) RemoveUserFromOrganisation(db *gorm.DB, user interface{}, orgs []interface{}) error {

	err := db.Model(user).Association("Organisations").Delete(orgs...)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) GetUserByID(db *gorm.DB, userID string) (User, error) {
	var user User

	query := db.Where("id = ?", userID)
	query = postgresql.PreloadEntities(query, &user, "Profile", "Organisations")

	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}

func (u *User) GetUserByEmail(db *gorm.DB, userEmail string) (User, error) {
	var user User

	query := db.Where("email = ?", userEmail)
	query = postgresql.PreloadEntities(query, &user, "Profile", "Organisations")

	if err := query.First(&user).Error; err != nil {
		return user, err
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
		query = postgresql.PreloadEntities(query, &user, "Profile", "Products", "Organisations")

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
	query = postgresql.PreloadEntities(query, &user, "Profile", "Products", "Organisations")

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
	userUpdates := User{Deactivated: true}

	result, err := postgresql.UpdateFields(db, &user, userUpdates, "id = ?", userId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to deactivate user")
	}

	return nil
}
