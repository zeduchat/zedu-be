package models

import (
	"errors"
	"math"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type Organisation struct {
	ID                 string `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name               string `gorm:"type:varchar(255);not null" json:"name"`
	Description        string `gorm:"type:text" json:"description"`
	Email              string `gorm:"type:varchar(255);unique" json:"email"`
	Type               string `gorm:"type:varchar(255)" json:"type"`
	Location           string `gorm:"type:varchar(255)" json:"location"`
	Country            string `gorm:"type:varchar(255)" json:"country"`
	OwnerID            string `gorm:"type:uuid;" json:"owner_id"`
	RoomsCount         int64  `gorm:"-" json:"rooms_count"`
	TotalMessagesCount int64  `gorm:"-" json:"total_messages_count"`

	OrgRoles []OrgRole `gorm:"foreignKey:OrganisationID" json:"org_roles"`
	Users    []User    `gorm:"many2many:user_organisations;foreignKey:ID;joinForeignKey:organisation_id;References:ID;joinReferences:user_id"`

	CreatedAt time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Rooms []Room `gorm:"foreignKey:OrganisationID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"rooms"`
}

type CreateOrgRequestModel struct {
	Name        string `json:"name" validate:"required,min=2,max=255"`
	Description string `json:"description" `
	Email       string `json:"email" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Location    string `json:"location" validate:"required"`
	Country     string `json:"country" validate:"required"`
}

type UpdateOrgRequestModel struct {
	Name        string `json:"name"`
	Description string `json:"description" `
	Email       string `json:"email"`
	Type        string `json:"type"`
	Location    string `json:"location"`
	Country     string `json:"country"`
}

type UserInOrgResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	Name        string `json:"name"`
}

type AddUserToOrgRequestModel struct {
	UserId string `json:"user_id" validate:"required"`
}

func (c *Organisation) CreateOrganisation(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &c)
	if err != nil {
		return err
	}
	return nil
}

func (c *Organisation) Delete(db *gorm.DB) error {
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

	roomsCount, err := o.CountOrganisationRooms(db, orgID)
	if err != nil {
		return org, err
	}

	org.RoomsCount = roomsCount

	return org, nil
}

func (o *Organisation) GetAllRoomsInOrganisation(db *gorm.DB, orgID string) ([]Room, map[string]interface{}, error) {
	var (
		rooms []Room
	)

	exists := postgresql.CheckExists(db, &o, "id = ?", orgID)
	if !exists {
		return rooms, map[string]interface{}{}, errors.New("organisation does not exist")
	}

	err := postgresql.SelectAllFromDb(db, "desc", &rooms, "organisation_id = ?", orgID)
	if err != nil {
		return rooms, map[string]interface{}{}, err
	}

	totalRoomCount := len(rooms)
	totalMessagesCount := int64(0)

	for _, room := range rooms {
		count, _ := room.CountRoomMessages(db, room.ID)
		totalMessagesCount += int64(count)
	}

	additionalInfo := map[string]interface{}{
		"rooms_count":         int64(totalRoomCount),
		"totalmessage_count": totalMessagesCount,
	}

	return rooms, additionalInfo, nil
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
		Select("users.id, users.email, profiles.phone as phone_number , users.name").
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
	org, err := o.GetOrgByID(db, orgId)
	if err != nil {
		return org, err
	}

	return org, nil
}

func (o *Organisation) CheckUserIsMemberOfOrg(userId string, orgId string, db *gorm.DB) (bool, error) {
	var user User

	_, err := o.GetOrgByID(db, orgId)
	if err != nil {
		return false, err
	}

	user, err = user.GetUserByID(db, userId)
	if err != nil {
		return false, err
	}

	for _, org := range user.Organisations {
		if org.ID == orgId {
			return true, nil
		}
	}

	return false, nil
}

func (o *Organisation) IsOwnerOfOrganisation(db *gorm.DB, requesterID, organisationID string) (bool, error) {
	var count int64
	err := db.Model(&Organisation{}).
		Where("id = ? AND owner_id = ?", organisationID, requesterID).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (o *Organisation) CountOrganisationRooms(db *gorm.DB, orgId string) (int64, error) {
	var rs []Room

	err := postgresql.SelectAllFromDb(db, "", &rs, "organisation_id = ?", orgId)
	if err != nil {
		return 0, errors.New("error counting rooms in an organisation")
	}
	return int64(len(rs)), nil
}
