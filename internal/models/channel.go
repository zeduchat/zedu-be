package models

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"google.golang.org/appengine/log"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
)

type Channels struct {
	ID             string `gorm:"type:uuid;primary_key" json:"channels_id"`
	Name           string `gorm:"column:name; type:text; not null" json:"name"`
	Description    string `gorm:"column:description; type:text; not null" json:"description"`
	OrganisationID string `gorm:"column:organisation_id; type:uuid;index" json:"organisation_id"`
	OwnerId        string `gorm:"column:owner_id; type:uuid;index" json:"owner_id"`
	Users          []User `gorm:"many2many:user_channels;" json:"users"`
	UserCount      int64  `gorm:"-" json:"user_count"`
	MessageCount   int64  `gorm:"-" json:"message_count"`
	Archived       bool   `gorm:"column:archived;null; default:false" json:"archived"`
	// GroupID        sql.NullString `gorm:"column:group_id; type:uuid;index; null" json:"group_id"`
	GroupID *string `gorm:"column:group_id; type:uuid;index;" json:"group_id"`

	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt time.Time `gorm:"column: deleted_at; not null; autoDeleteTime" json:"deleted_at"`
	Threads   []Threads `gorm:"foreignKey:ChannelsID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"threads"`
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
	Description    string `json:"description"`
}

type GetChannelsRequest struct {
	Name string `json:"name" validate:"required"`
}

type GetChannelResp struct {
	Channels
	WebhookUrl string `json:"webhook_url"`
	Access     bool   `json:"access"`
}

type GetUserChannelResp []struct {
	Channels
	WebhookUrl  string `json:"webhook_url"`
	ThreadCount int64  `json:"thread_count"`
	Access      bool   `json:"access"`
}

type GetUserNotChannelResp []struct {
	Channels
	WebhookUrl  string `json:"webhook_url"`
	ThreadCount int64  `json:"thread_count"`
	Access      bool   `json:"access"`
}
type JoinChannelsRequest struct {
	Username   string `json:"username"`
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
type ChannelInfoResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserMsgProfile struct {
	FullName  string `json:"full_name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

type MessagesResp []struct {
	ID        string    `json:"id"`
	Edited    bool      `json:"edited"`
	Message   string    `json:"message"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserMsgProfile
}

type ChannelInfo struct {
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id"`
}

type AddMultipleMembersRequest struct {
	ChannelID string   `json:"channel_id" validate:"required"`
	UserIDs   []string `json:"user_ids" validate:"required"`
}

type ArchiveChannelRequest struct {
	Archived bool   `json:"archived"`
	UserId   string `json:"user_id" `
}

func (r *Channels) CreateChannels(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &r)
	if err != nil {
		return errors.New("could not create channel, invalid organisation id")
	}

	query := `
	INSERT INTO organisation_channels_integrations (id, org_id, integration_id, channel_id, is_active, created_at, updated_at)
	SELECT
		gen_random_uuid(), org_id, integration_id, ?, TRUE, NOW(), NOW()
	FROM organisation_integrations
	WHERE org_id = ?`

	if err := db.Exec(query, r.ID, r.OrganisationID).Error; err != nil {
		return fmt.Errorf("error inserting into OrganisationChannelsIntegrations: %v", err)
	}

	return nil
}

func (c *Channels) CheckChannelExistsInOrg(db *gorm.DB, channelID, organisationID string) bool {
	var channel Channels
	exists := postgresql.CheckExists(db, &channel, "id = ? AND organisation_id = ?", channelID, organisationID)
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

func (r *Channels) GetChannelsByID(db *gorm.DB, chanReq ChannelInfo) (GetChannelResp, error) {
	var (
		channel  Channels
		chanResp GetChannelResp
		ur       UserChannels
		webhook  Webhook
	)

	access := postgresql.CheckExists(db, &ur, "channels_id = ? AND user_id = ?", chanReq.ChannelID, chanReq.UserID)

	err, _ := postgresql.SelectOneFromDb(db.Preload("Users"), &channel, "id = ?", chanReq.ChannelID)
	if err != nil {
		return chanResp, errors.New("channel not found")
	}

	count, err := ur.CountChannelsUsers(db, chanReq.ChannelID)
	if err != nil {
		return chanResp, errors.New("could not get channel users count")
	}

	channel.UserCount = count
	webhook, err = webhook.GetChannelWebhook(db, chanReq)

	if err != nil {
		return chanResp, errors.New("could not get channel webhook")
	}

	chanResp = GetChannelResp{
		channel,
		webhook.WebhookUrl,
		access,
	}

	return chanResp, nil
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

func (r *Channels) GetChannelsMessages(db *gorm.DB, userID, channelID string) (MessagesResp, error) {

	var (
		userChannels UserChannels
		messagesResp MessagesResp
	)

	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	if !exist {
		return messagesResp, errors.New("user not in channel")
	}

	var err = db.Table("messages").
		Select("messages.*, profiles.full_name, profiles.user_name, profiles.avatar_url, users.email").
		Joins("left join profiles on profiles.userid = messages.user_id").
		Joins("left join users on users.id = messages.user_id").
		Where("messages.channels_id = ?", channelID).
		Scan(&messagesResp).Error
	if err != nil {
		return messagesResp, err
	}

	return messagesResp, nil
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

	if  req.Username == "" {
		req.Username = user.Email
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

func (c *Channels) ArchiveChannel(db *gorm.DB, channelId string, req ArchiveChannelRequest) (bool, error) {
	var channel Channels

	exists := postgresql.CheckExists(db, &channel, "id = ?", channelId)
	if !exists {
		return req.Archived, errors.New("channel does not exist")
	}

	if req.UserId == channel.OwnerId {
		return req.Archived, errors.New("unauthorized, only channel owner can perform this operation")
	}

	err := db.Raw("SELECT id, COALESCE(archived, false) as archived FROM channels WHERE id = ?", channelId).Scan(&channel).Error
	if err != nil {
		return req.Archived, errors.New("could not fetch current channel state")
	}

	if channel.Archived == req.Archived {
		return req.Archived, errors.New("channel is already in the requested state")
	}

	err = db.Model(&channel).Where("id = ?", channelId).Update("archived", req.Archived).Error
	if err != nil {
		return req.Archived, errors.New("could not update the archived status of the channel")
	}

	if req.Archived {
		err = db.Model(&channel).Where("id = ?", channelId).Update("group_id", nil).Error
		if err != nil {
			return req.Archived, errors.New("could not remove channel from group")
		}
	}

	return req.Archived, nil
}

func (r *Channels) AddMultipleUsersToChannel(db *gorm.DB, req AddMultipleMembersRequest) ([]string, error) {
	var (
		users        = req.UserIDs
		channelID    = req.ChannelID
		userChannels UserChannels
		userChanList []UserChannels
		addError     []string
	)

	exists := postgresql.CheckExists(db, &r, "id = ?", channelID)
	if !exists {
		return addError, errors.New("channel does not exist")
	}

	if len(users) > 10 {
		return addError, errors.New("maximum of 10 users can be added")
	}

	for _, user := range users {

		exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, user)
		if exist {
			addError = append(addError, fmt.Sprintf("%s already in the channel", userChannels.Username))
			continue
		}

		userChannels = UserChannels{
			ChannelsID: channelID,
			UserID:     user,
			Username:   userChannels.Username,
		}

		userChanList = append(userChanList, userChannels)
	}

	if len(userChanList) == 0 {
		return addError, errors.New("no user added to channel. All users already in channel")
	}

	err := postgresql.CreateMultipleRecords(db, userChanList, len(userChanList))
	if err != nil {
		return addError, errors.New("could not add user to channel")
	}

	return addError, nil
}

func (r *Channels) GetArchivedChannels(db *gorm.DB, ids map[string]string) ([]Channels, error) {
	var channels []Channels

	err := postgresql.SelectAllFromDb(db, "", &channels, "organisation_id = ? AND archived = ?", ids["organisation_id"], true)
	if err != nil {
		return channels, errors.New("could not get archived channels")
	}

	return channels, nil
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

func (r *Channels) CheckChannelExists(db *gorm.DB, channelID string) (bool, error) {

	exists := postgresql.CheckExists(db, &r, "id = ?", channelID)
	if !exists {
		return exists, errors.New("channel does not exist")
	}

	return exists, nil
}

func (uc *UserChannels) GetUserChannels(base *storage.Database, userId, orgID string) (GetUserChannelResp, error) {

	var (
		channels []Channels
		org      Organisation
		chanResp GetUserChannelResp
		c        context.Context
		es       = base.Elastic
		db       = base.Postgresql
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", orgID)
	if !exists {
		return chanResp, errors.New("organisation does not exist")
	}

	if err := db.Model(&[]Channels{}).
		Select("channels.id, channels.name, channels.description, channels.organisation_id, channels.owner_id, channels.archived, channels.group_id, channels.created_at").
		Joins("join user_channels on channels.id = user_channels.channels_id").
		Where("channels.organisation_id = ?", orgID).
		Where("user_channels.user_id = ?", userId).
		Order("channels.created_at").
		Scan(&channels).Error; err != nil {
		return nil, errors.New("error fetching channels")
	}

	getThreadCountFromElastic := func(es *elasticsearch.Client, channelID string) int {
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []map[string]interface{}{
						{
							"term": map[string]interface{}{
								"channels_id.keyword": channelID,
							},
						},
						{
							"term": map[string]interface{}{
								"type.keyword": "thread",
							},
						},
					},
				},
			},
			"size": 0,
			"aggs": map[string]interface{}{
				"thread_count": map[string]interface{}{
					"value_count": map[string]interface{}{
						"field": "thread_id.keyword",
					},
				},
			},
		}

		var countInfo any
		err := elastic.SelectAll(es, "threads", query, &countInfo)
		if err != nil {
			log.Errorf(c, "error fetching thread count from elastic: %v", err)
			return 0
		}

		count := countInfo.(map[string]interface{})["aggregations"].(map[string]interface{})["thread_count"].(map[string]interface{})["value"].(float64)

		return int(count)

	}

	for _, channel := range channels {
		count := getThreadCountFromElastic(es, channel.ID)
		chanResp = append(chanResp, struct {
			Channels
			WebhookUrl  string `json:"webhook_url"`
			ThreadCount int64  `json:"thread_count"`
			Access      bool   `json:"access"`
		}{
			Channels:    channel,
			WebhookUrl:  "",
			ThreadCount: int64(count),
			Access:      true,
		})
	}

	return chanResp, nil
}

func (uc *UserChannels) GetUserNotInChannels(db *gorm.DB, userId, orgId string) (GetUserNotChannelResp, error) {
	var (
		org      Organisation
		chanResp GetUserNotChannelResp
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", orgId)
	if !exists {
		return chanResp, errors.New("organisation does not exist")
	}

	err := db.Table("channels").
		Select("channels.id, channels.name, channels.description, channels.created_at, channels.archived, 'false' AS access").
		Where("channels.id NOT IN (SELECT user_channels.channels_id FROM user_channels WHERE user_channels.user_id = ?)", userId).
		Where("channels.organisation_id = ?", orgId).
		Order("channels.created_at").
		Scan(&chanResp).Error

	if err != nil {
		return chanResp, errors.New("could not get channels user is not part of")
	}
	return chanResp, nil
}
