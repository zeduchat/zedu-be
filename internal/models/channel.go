package models

import (
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
)

type Channels struct {
	ID          string `gorm:"type:uuid;primary_key" json:"channels_id"`
	Name        string `gorm:"column:name;unique type:text; not null" json:"name"`
	Description string `gorm:"column:description; type:text; not null" json:"description"`

	OrganisationID string    `gorm:"column:organisation_id; type:uuid;index" json:"organisation_id"`
	OwnerId        string    `gorm:"column:owner_id; type:uuid;index" json:"owner_id"`
	Users          []User    `gorm:"many2many:user_channels;" json:"users"`
	UserCount      int64     `gorm:"-" json:"user_count"`
	MessageCount   int64     `gorm:"-" json:"message_count"`
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt      time.Time `gorm:"column: deleted_at; not null; autoDeleteTime" json:"deleted_at"`
	Threads        []Threads `gorm:"foreignKey:ChannelsID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"threads"`
}

type UserChannels struct {
	ChannelsID string    `gorm:"type:uuid;primaryKey;not null" json:"channels_id"`
	UserID     string    `gorm:"type:uuid;primaryKey;not null" json:"user_id"`
	Username   string    `gorm:"column:username; type:varchar(100)" json:"username"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	DeletedAt  time.Time `gorm:"index" json:"deleted_at"`
}

type CreateChannelsRequest struct {
	OrganisationID string `json:"organisation_id" validate:"required"`
	Username       string `json:"username" validate:"required"`
	Name           string `json:"name" validate:"required"`
	Description    string `json:"description" validate:"required"`
}

type GetChannelsRequest struct {
	Name string `json:"name" validate:"required"`
}

type JoinChannelsRequest struct {
	Username   string `json:"username" validate:"required"`
	ChannelsID string `json:"channels_id" `
	UserID     string `json:"user_id" `
}

type UpdateChannelsRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateChannelsUserNameReq struct {
	Username string `json:"username" validate:"required"`
}

func (r *Channels) CreateChannels(db *gorm.DB, typesenseDb *typesense.Client) error {
	err := postgresql.CreateOneRecord(db, r)
	if err != nil {
		return errors.New("could not create channel, invalid organisation id")
	}

	fields := []api.Field{
		{Name: "id", Type: "string"},
		{Name: "type", Type: "string"},
		{Name: "channels_id", Type: "string"},
		{Name: "thread_id", Type: "string"},
		{Name: "event_name", Type: "string"},
		{Name: "username", Type: "string"},
		{Name: "action_type", Type: "string"},
		{Name: "status", Type: "string"},
		{Name: "content", Type: "string"},
		{Name: "created_at", Type: "int64"},
	}

	err = tydb.CreateCollection(typesenseDb, r.ID, fields)
	if err != nil {
		return errors.New("could not create channel collection in Typesense")
	}

	return nil
}

func (c *Channels) CheckChannelExistsInOrg(db *gorm.DB, channelID, organisationID string) bool {
	var channel Channels
	exists := postgresql.CheckExists(db, &channel, "channels_id = ? AND organisation_id = ?", channelID, organisationID)
	return exists
}

func (r *Channels) GetChannelsUsersByID(db *gorm.DB, channelID string) ([]User, error) {
	var users []User

	err := postgresql.SelectUsersFromDb(
		db.Where("channels_id = ?", channelID),
		"",
		&users,
		"channels_id = ?",
		channelID,
	)

	postgresql.SelectAllFromDb(db, "", &users, "channels_id = ?", channelID)

	if err != nil {
		return users, errors.New("could not get users in channel")
	}

	return users, nil
}

func (ch *Channels) GetUsersInChannel(c *gin.Context, db *gorm.DB, channelId string) ([]User, postgresql.PaginationResponse, error) {
    var users []User
    pagination := postgresql.GetPagination(c)

    offset := (pagination.Page - 1) * pagination.Limit

    if err := db.Preload("Profile").
        Joins("JOIN user_channels ON user_channels.user_id = users.id").
        Where("user_channels.channels_id = ?", channelId).
        Offset(offset).
        Limit(pagination.Limit).
        Find(&users).Error; err != nil {
        return nil, postgresql.PaginationResponse{}, err
    }

    var totalUsers int64
    if err := db.Table("users").
        Joins("JOIN user_channels ON user_channels.user_id = users.id").
        Where("user_channels.channels_id = ?", channelId).
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

func (r *Channels) GetChannelsByName(db *gorm.DB, name string) ([]Channels, error) {
	var (
		channels []Channels
		ur       UserChannels
	)

	exists := postgresql.CheckExists(db, &channels, "name= ?", name)
	if !exists {
		return channels, errors.New("channel not found")
	}

	err := postgresql.SelectAllFromDb(db, "", &channels, "name= ?", name)
	if err != nil {
		return channels, errors.New("could not get channel by name")
	}

	for i, channel := range channels {
		count, _ := ur.CountChannelsUsers(db, channel.ID)
		channels[i].UserCount = count
	}

	return channels, nil
}

func (r *Channels) GetChannelsByID(db *gorm.DB, channelID string) (Channels, error) {
	var (
		channel Channels
		ur      UserChannels
	)

	err, _ := postgresql.SelectOneFromDb(db.Preload("Users"), &channel, "id = ?", channelID)
	if err != nil {
		return channel, errors.New("channel not found")
	}

	count, err := ur.CountChannelsUsers(db, channelID)
	if err != nil {
		return channel, errors.New("could not get channel users count")
	}

	channel.UserCount = count

	return channel, nil
}

func (u *UserChannels) CountChannelsUsers(db *gorm.DB, channelID string) (int64, error) {
	var count int64
	err := db.Model(&UserChannels{}).Where("channels_id = ?", channelID).Count(&count).Error
	if err != nil {
		return 0, errors.New("could not count users in channel")
	}
	return count, nil
}

func (r *Channels) CountChannelsMessages(db *gorm.DB, channelID string) (int64, error) {
	var (
		count   int64
		message Message
	)
	err := db.Model(&message).Where("channels_id = ?", channelID).Count(&count).Error
	if err != nil {
		return 0, errors.New("could not count messages in channel")
	}
	return count, nil
}

func (r *Channels) CountTeamChannelss(db *gorm.DB, teamId string) (int64, error) {
	var rs []Channels

	err := postgresql.SelectAllFromDb(db, "", &rs, "team_id = ?", teamId)
	if err != nil {
		return 0, errors.New("error counting channels in a team")
	}
	return int64(len(rs)), nil
}

func (r *Channels) GetChannelsMessages(db *gorm.DB, userID, channelID string) ([]Message, error) {

	var (
		messages     []Message
		userChannels UserChannels
	)

	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	if !exist {
		return messages, errors.New("user not in channel")
	}

	err := postgresql.SelectAllFromDb(
		db.Where("channels_id = ?", channelID),
		"",
		&messages,
		"channels_id = ?",
		channelID,
	)
	if err != nil {
		return messages, err
	}

	return messages, nil
}

func (r *Channels) AddUserToChannels(db *gorm.DB, req JoinChannelsRequest) (Channels, error) {

	var (
		user      User
		channel   Channels
		userID    = req.UserID
		channelID = req.ChannelsID
	)

	exists := postgresql.CheckExists(db, &user, "id = ?", userID)
	if !exists {
		return channel, errors.New("user does not exist")
	}

	exists = postgresql.CheckExists(db, &channel, "id = ?", channelID)
	if !exists {
		return channel, errors.New("channel does not exist")
	}

	var userChannels UserChannels
	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	if exist {
		return channel, errors.New("user already in channel")
	}

	userChannels = UserChannels{
		ChannelsID: channelID,
		UserID:     userID,
		Username:   req.Username,
	}

	err := postgresql.CreateOneRecord(db, &userChannels)
	if err != nil {
		return channel, errors.New("could not add user to channel")
	}

	return channel, nil
}


func (r *Channels) RemoveUserFromChannels(db *gorm.DB, channelID, userID string) error {
	var userChannels UserChannels

	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	if !exist {
		return errors.New("user not in channel")
	}

	err, _ := postgresql.SelectOneFromDb(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	if err != nil {
		return errors.New("could not get user in channel")
	}

	err = postgresql.DeleteRecordFromDb(db, &userChannels)
	if err != nil {
		return errors.New("could not remove user from channel")
	}
	return nil
}

func (r *UserChannels) UpdateUsername(db *gorm.DB, req UpdateChannelsUserNameReq, channelId, userId string) error {

	var userChannels UserChannels

	query := "channels_id = ? AND user_id = ?"

	exist := postgresql.CheckExists(db, &userChannels, query, channelId, userId)
	if !exist {
		return errors.New("user not in channel")
	}

	result, err := postgresql.UpdateFields(db, &r, req, query, channelId, userId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update username")
	}

	return nil
}

func (c *Channels) Delete(db *gorm.DB, typesenseDb *typesense.Client) error {

	err := db.Model(UserChannels{}).Where("channels_id = ?", c.ID).Delete(UserChannels{}).Error

	if err != nil {
		return errors.New("error removing users in channel")
	}

	err = tydb.DeleteCollection(typesenseDb, c.ID)
	if err != nil {
		return errors.New("could not delete channel collection in Typesense")
	}

	err = postgresql.DeleteRecordFromDb(db, &c)
	if err != nil {
		return err
	}

	return nil
}

func (c *UserChannels) UserInChannels(db *gorm.DB, channelID, userID string) error {

	var userChannels UserChannels

	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	if !exist {
		return errors.New("user not in channel")
	}

	return nil
}

func (r *Channels) UpdateChannels(db *gorm.DB, req UpdateChannelsRequest, channelID string, userId string) (Channels, int, error) {
	var channel Channels

	exists := postgresql.CheckExists(db, &channel, "id = ?", channelID)
	if !exists {
		return Channels{}, http.StatusNotFound, errors.New("channel does not exist")
	}

	if channel.OwnerId != userId {
		return Channels{}, http.StatusUnauthorized, errors.New("user not authorized")
	}

	result, err := postgresql.UpdateFields(db, &channel, req, "id = ?", channelID)
	if err != nil {
		return Channels{}, http.StatusInternalServerError, nil
	}

	if result.RowsAffected == 0 {
		return Channels{}, http.StatusInternalServerError, errors.New("failed to update channel")
	}

	updatedChannels := Channels{}
	err = db.First(&updatedChannels, "id = ?", channelID).Error
	if err != nil {
		return Channels{}, http.StatusInternalServerError, err
	}
	return updatedChannels, http.StatusOK, nil
}

func (r *UserChannels) CheckUser(db *gorm.DB, userID, channelID string) (bool, string) {

	var (
		userChannels UserChannels
	)

	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	if !exist {
		return false, "user not in channel"
	}

	return true, "user in channel"
}

func (r *Channels) SearchChannelssByName(db *gorm.DB, c *gin.Context, name string) ([]Channels, postgresql.PaginationResponse, error) {
	var channels []Channels

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db.Preload("Users"),
		"created_at",
		"desc",
		pagination,
		&channels,
		"name LIKE ?",
		"%"+name+"%",
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	for i, channel := range channels {
		var userChannels UserChannels
		count, _ := userChannels.CountChannelsUsers(db, channel.ID)
		channels[i].UserCount = count
	}

	return channels, paginationResponse, nil
}
