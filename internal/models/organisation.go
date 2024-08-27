package models

import (
	"errors"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type Organisation struct {
	ID                 string `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name               string `gorm:"type:varchar(255);not null" json:"name"`
	Description        string `gorm:"type:text" json:"description"`
	Email              string `gorm:"type:varchar(255);not null" json:"email"`
	Type               string `gorm:"type:varchar(255)" json:"type"`
	Location           string `gorm:"type:varchar(255)" json:"location"`
	Country            string `gorm:"type:varchar(255)" json:"country"`
	OwnerID            string `gorm:"type:uuid;" json:"owner_id"`
	LogoURL            string `gorm:"type:varchar(255)" json:"logo_url"`
	ChannelssCount     int64  `gorm:"-" json:"channels_count"`
	TotalMessagesCount int64  `gorm:"-" json:"total_messages_count"`

	OrgRoles []OrgRole `gorm:"foreignKey:OrganisationID" json:"org_roles"`
	Users    []User    `gorm:"many2many:user_organisations;foreignKey:ID;joinForeignKey:organisation_id;References:ID;joinReferences:user_id"`

	CreatedAt time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Channels []Channels `gorm:"foreignKey:OrganisationID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"channels"`
}

type CreateOrgRequestModel struct {
	Name        string `json:"name" validate:"required,min=2,max=255"`
	Description string `json:"description" `
	Email       string `json:"email" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Location    string `json:"location" validate:"required"`
	Country     string `json:"country" validate:"required"`
	LogoURL     string `json:"logo_url" `
}

type UpdateOrgRequestModel struct {
	Name        string `json:"name"`
	Description string `json:"description" `
	Email       string `json:"email"`
	Type        string `json:"type"`
	Location    string `json:"location"`
	Country     string `json:"country"`
	LogoURL     string `json:"logo_url"`
}

type UserInOrgResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	AvatarURL   string    `json:"profile_url"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type AddUserToOrgRequestModel struct {
	UserId string `json:"user_id" validate:"required"`
}

type ChannelResp []struct {
	Channels
	ThreadCount int64 `json:"thread_count"`
}

func (c *Organisation) CreateOrganisation(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &c)
	if err != nil {
		return err
	}
	return nil
}

func (c *Organisation) Delete(db *gorm.DB, orgId string) error {

	if err := db.Exec(
		"DELETE FROM user_organisations WHERE organisation_id = ?",
		orgId).Error; err != nil {
		return err
	}

	if err := db.Where("organisation_id = ?", orgId).Delete(&OrgUserManagement{}).Error; err != nil {
		return err
	}

	err := postgresql.DeleteRecordFromDb(db, &c)
	if err != nil {
		return err
	}
	return nil
}

func (c *Organisation) Update(db *gorm.DB) (*Organisation, error) {
	result, err := postgresql.SaveAllFields(db, &c)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("failed to update organisation")
	}

	return c, nil
}

func (o *Organisation) GetOrgByID(db *gorm.DB, orgID string) (Organisation, error) {
	var org Organisation

	exists := postgresql.CheckExists(db, &org, "id = ?", orgID)
	if !exists {
		return org, errors.New("organisation not found")
	}

	query := db.Where("id = ?", orgID)
	query = postgresql.PreloadEntities(query, &org, "OrgRoles", "OrgRoles.Permissions")

	if err := query.First(&org).Error; err != nil {
		return org, err
	}

	channelsCount, err := o.CountOrganisationChannelss(db, orgID)
	if err != nil {
		return org, err
	}

	org.ChannelssCount = channelsCount

	return org, nil
}

func (o *Organisation) GetAllChannelssInOrganisation(db *gorm.DB, orgID string) (ChannelResp, error) {
	var (
		channels Channels
		thread   Threads
		chanResp ChannelResp
	)

	exists := postgresql.CheckExists(db, &o, "id = ?", orgID)
	if !exists {
		return chanResp, errors.New("organisation does not exist")
	}

	threadCountSubquery := db.Model(&thread).Select("count(*)").
		Where("threads.channels_id = channels.id").
		Where("threads.type = 'thread'")

	if err := db.Model(&channels).
		Select("channels.id, channels.name, channels.organisation_id, (?) AS thread_count",
			threadCountSubquery).
		Where("channels.organisation_id = ?", orgID).
		Scan(&chanResp).Error; err != nil {
		return nil, errors.New("error fetching channels")
	}

	return chanResp, nil
}

func (u *Organisation) GetOrganisationsByUserID(db *gorm.DB, userID string) ([]Organisation, error) {
	var (
		ErrNotFound   = errors.New("user not found")
		organisations = []Organisation{}
	)

	query := db.Model(&Organisation{}).
		Joins("INNER JOIN user_organisations uo ON organisations.id = uo.organisation_id").
		Where("uo.user_id = ?", userID)

	if err := query.Find(&organisations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return organisations, ErrNotFound
		}
		return organisations, err
	}
	if len(organisations) == 0 {
		return organisations, ErrNotFound
	}

	return organisations, nil
}

func (o *Organisation) GetUserOrganisations(db *gorm.DB, userID string) ([]Organisation, error) {
	var (
		orgs []Organisation
	)

	// Join the organisations table with the org_user_managements table to get the organisations the user belongs to
	err := db.Table("organisations AS org").
		Select("org.*").
		Joins("JOIN org_user_managements AS oum ON org.id = oum.organisation_id").
		Where("oum.user_id = ?", userID).
		Find(&orgs).Error

	if err != nil {
		return orgs, err
	}

	return orgs, nil
}

func (u *Organisation) GetOrganisationsByUserIDs(db *gorm.DB, userID, requesterID string) ([]Organisation, error) {

	var (
		ErrNotFound   = errors.New("user not in your organisation")
		organisations = []Organisation{}
	)

	var isOwner bool
	err := db.Model(&Organisation{}).
		Select("count(*) > 0").
		Where("owner_id = ?", requesterID).
		Find(&isOwner).
		Error
	if err != nil {
		return nil, err
	}

	if isOwner {

		query := db.Model(&Organisation{}).
			Joins("INNER JOIN user_organisations uo ON organisations.id = uo.organisation_id").
			Where("uo.user_id = ?", userID).
			Where("organisations.owner_id = ?", requesterID)
		if err := query.Find(&organisations).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {

				return organisations, ErrNotFound
			}
			return organisations, err
		}
		if len(organisations) == 0 {
			return organisations, ErrNotFound
		}
		return organisations, nil
	}

	query := db.Model(&Organisation{}).
		Joins("INNER JOIN user_organisations uo ON organisations.id = uo.organisation_id").
		Where("uo.user_id = ?", requesterID)
	if err := query.Find(&organisations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return organisations, ErrNotFound
		}
		return organisations, err
	}

	return organisations, nil
}

func (o *Organisation) GetUsersInOrganisation(c *gin.Context, db *gorm.DB, orgId string) ([]UserInOrgResponse, postgresql.PaginationResponse, error) {
	var users []UserInOrgResponse
	pagination := postgresql.GetPagination(c)

	offset := (pagination.Page - 1) * pagination.Limit

	if err := db.Table("users").
		Select("users.id, users.email, profiles.phone as phone, profiles.full_name as name, profiles.avatar_url as avatar_url, users.created_at ").
		Joins("JOIN user_organisations ON user_organisations.user_id = users.id").
		Joins("JOIN profiles ON profiles.userid = users.id").
		Where("user_organisations.organisation_id = ?", orgId).
		Offset(offset).
		Limit(pagination.Limit).
		Find(&users).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	var totalUsers int64
	if err := db.Table("users").
		Joins("JOIN user_organisations ON user_organisations.user_id = users.id").
		Joins("JOIN profiles ON profiles.userid = users.id").
		Where("user_organisations.organisation_id = ?", orgId).
		Count(&totalUsers).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	totalPages := int(math.Ceil(float64(totalUsers) / float64(pagination.Limit)))
	paginationResponse := postgresql.PaginationResponse{
		CurrentPage:     pagination.Page,
		PageCount:       pagination.Limit,
		TotalPagesCount: totalPages,
	}

	return users, paginationResponse, nil
}

func (o *Organisation) CheckOrgExists(orgId string, db *gorm.DB) (Organisation, error) {
	var (
		org Organisation
	)

	exists := postgresql.CheckExists(db, &o, "id = ?", orgId)
	if !exists {
		return org, errors.New("organisation not found")
	}

	err, _ := postgresql.SelectOneFromDb(db, &org, "id = ?", orgId)
	if err != nil {
		return org, err
	}

	return org, nil
}

func (o *Organisation) CheckUserIsMemberOfOrg(userId string, orgId string, db *gorm.DB) (bool, error) {
	var count int64
	err := db.Table("user_organisations").
		Where("user_id = ? AND organisation_id = ?", userId, orgId).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (o *Organisation) IsOwnerOfOrganisation(db *gorm.DB, requesterID, organisationID string) (bool, error) {
	count, err := postgresql.CountSpecificRecords(
		db,
		&Organisation{},
		"id = ? AND owner_id = ?",
		organisationID,
		requesterID,
	)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (o *Organisation) CountOrganisationChannelss(db *gorm.DB, orgId string) (int64, error) {
	var rs []Channels

	err := postgresql.SelectAllFromDb(db, "", &rs, "organisation_id = ?", orgId)
	if err != nil {
		return 0, errors.New("error counting channels in an organisation")
	}
	return int64(len(rs)), nil
}

func (o *Organisation) GetOrganisationInvites(c *gin.Context, db *gorm.DB, userID, orgID string) ([]Invitation, postgresql.PaginationResponse, error) {
	var (
		invitations []Invitation
	)

	exists := postgresql.CheckExists(db, o, "id = ?", orgID)
	if !exists {
		return invitations, postgresql.PaginationResponse{}, errors.New("organisation not found")
	}

	exists = postgresql.CheckExists(db, &User{}, "id = ?", userID)
	if !exists {
		return invitations, postgresql.PaginationResponse{}, errors.New("user not found")
	}

	isowner, err := o.IsOwnerOfOrganisation(db, userID, orgID)
	if err != nil {
		return invitations, postgresql.PaginationResponse{}, err
	}
	if !isowner {
		return invitations, postgresql.PaginationResponse{}, errors.New("user is not the owner of the organisation")
	}

	pagination := postgresql.GetPagination(c)
	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"",
		"desc",
		pagination,
		&invitations,
		"organisation_id = ?",
		orgID,
	)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return invitations, paginationResponse, errors.New("channel not found")
		}
		return invitations, paginationResponse, err
	}
	return invitations, paginationResponse, nil
}

func (o *Organisation) GetOrganisationDetails(db *gorm.DB, orgID string) (Organisation, error) {
	var org Organisation

	err := db.Where("id = ?", orgID).First(&org).Error
	if err != nil {
		return org, err
	}

	channelsCount, err := o.CountOrganisationChannelss(db, orgID)
	if err != nil {
		return org, err
	}

	org.ChannelssCount = channelsCount

	return org, nil
}

type OrgMetricsResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	OwnerID   string `json:"owner_id"`
	OwnerName string `json:"owner_name"`
	Users     []User `json:"users"`
}

// get organisation by organisation id
func (o *Organisation) LoadOrganisationMetrics(db *gorm.DB, orgID string) (OrgMetricsResponse, error) {
	var org Organisation
	var ogm OrgMetricsResponse

	exists := postgresql.CheckExists(db, &org, "id = ?", orgID)
	if !exists {
		return ogm, errors.New("organisation not found")
	}

	err, _ := postgresql.SelectOneFromDb(db.Preload("Users"), &org, "id = ?", orgID)
	if err != nil {
		return ogm, err
	}

	//get the owner of the organisation
	var owner User

	err, _ = postgresql.SelectOneFromDb(db.Preload("Profile"), &owner, "id = ?", org.OwnerID)
	if err != nil {
		return ogm, err
	}

	response := OrgMetricsResponse{
		ID:        org.ID,
		Name:      org.Name,
		Email:     org.Email,
		OwnerID:   org.OwnerID,
		OwnerName: owner.Profile.FullName,
		Users:     org.Users,
	}

	return response, nil
}
