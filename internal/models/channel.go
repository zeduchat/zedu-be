package models

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type Channels struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"channels_id"`
	Name           string    `gorm:"column:name; type:text; not null" json:"name"`
	Description    string    `gorm:"column:description; type:text; not null" json:"description"`
	Topic          string    `gorm:"column:topic; type:text; not null;default:'present'" json:"topic"`
	OrganisationID string    `gorm:"column:organisation_id; type:uuid;index" json:"organisation_id"`
	OwnerId        string    `gorm:"column:owner_id; type:uuid;index" json:"owner_id"`
	OwnerName      string    `gorm:"-" json:"owner_name"`
	Users          []User    `gorm:"many2many:user_channels;" json:"users,omitempty"`
	UserCount      int64     `gorm:"-" json:"user_count,omitempty"`
	MessageCount   int64     `gorm:"-" json:"-"`
	Archived       bool      `gorm:"column:archived;null; default:false" json:"archived,omitempty"`
	GroupID        *string   `gorm:"column:group_id; type:uuid;index;" json:"-"`
	IsPrivate      bool      `gorm:"column:is_private;default:false" json:"is_private"`
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt      time.Time `gorm:"column: deleted_at; not null; autoDeleteTime" json:"-"`
}

type UserChannels struct {
	ChannelsID   string                 `gorm:"type:uuid;primaryKey;not null" json:"channels_id"`
	UserID       string                 `gorm:"type:uuid;primaryKey;not null" json:"user_id"`
	Username     string                 `gorm:"column:username; type:varchar(100)" json:"username"`
	CreatedAt    time.Time              `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	ThreadCount  int64                  `gorm:"column:thread_count;default:0" json:"thread_count"`
	LastThreadId string                 `gorm:"column:last_thread_id" json:"last_thread_id"`
	LastReadAt   time.Time              `gorm:"column:last_read_at" json:"last_read_at"`
	MentionCount int64                  `gorm:"column:mention_count;default:0" json:"mention_count"`
	DeletedAt    time.Time              `gorm:"index" json:"deleted_at"`
	Preferences  NotificationPreference `gorm:"type:jsonb;not null;default:'{}'" json:"preferences"`
	OrgId        string                 `gorm:"-" json:"-"`
}

type UserChannelHistory struct {
	ID                  string         `gorm:"type:uuid;primary_key" json:"id"`
	UserID              string         `gorm:"column:user_id;type:uuid;not null;index" json:"user_id"`
	OrganisationID      string         `gorm:"column:organisation_id;type:uuid;not null;index" json:"organisation_id"`
	ChannelIDs          pq.StringArray `gorm:"column:channel_ids;type:uuid[];not null" json:"channel_ids"`
	BanishedToChannelID *string        `gorm:"column:banished_to_channel_id;type:uuid" json:"banished_to_channel_id,omitempty"`
	Action              string         `gorm:"column:action;type:varchar(20);not null;index" json:"action"` // banished, removed, restored
	CreatedAt           time.Time      `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt           *time.Time     `gorm:"index" json:"deleted_at,omitempty"`
}

type UpdateLastRead struct {
	LastThreadId string    `json:"last_thread_id,omitempty"`
	LastReadAt   time.Time `json:"last_read_at,omitempty"`
	ThreadCount  int64     `json:"thread_count"`
}
type CreateChannelsRequest struct {
	OrganisationID string `json:"organisation_id" validate:"required"`
	Username       string `json:"username" validate:"required"`
	Name           string `json:"name" validate:"required"`
	IsPrivate      bool   `json:"is_private"`
	Description    string `json:"description"`
	UserId         string `json:"user_id"`
	Topic          string `json:"topic"`
}

type GetChannelsRequest struct {
	Name string `json:"name" validate:"required"`
}

type ActiveBuzzInfo struct {
	BuzzID           string    `json:"buzz_id"`
	HostID           string    `json:"host_id"`
	HostName         string    `json:"host_name"`
	ParticipantCount int       `json:"participant_count"`
	StartedAt        time.Time `json:"started_at"`
}

type GetChannelResp struct {
	Channels
	OwnerName    string              `json:"owner_name"`
	OwnerEmail   string              `json:"owner_email"`
	WebhookUrl   string              `json:"webhook_url"`
	Access       bool                `json:"access"`
	ActiveBuzz   *ActiveBuzzInfo     `json:"active_buzz,omitempty"`
	PreviewMedia []FileMediaResponse `json:"preview_media"`
	CreatedAt    time.Time           `json:"created_at"`
	Participants []Participant       `json:"participants"`
}

type GetUserChannelResp []struct {
	Channels
	WebhookUrl     string          `json:"webhook_url,omitempty"`
	ThreadCount    int64           `json:"thread_count"`
	Access         bool            `json:"access"`
	MentionCount   int64           `json:"mention_count"`
	LastThreadId   string          `json:"last_thread_id"`
	MemberAvatars  []string        `gorm:"-" json:"member_avatars"`
	MembersCount   int             `json:"members_count"`
	LastPostTime   string          `json:"last_post_time"`
	UnreadCount    int64           `json:"unread_count"`
	ChannelSlug    string          `json:"channel_slug"`
	ActiveBuzz     *ActiveBuzzInfo `gorm:"-" json:"active_buzz,omitempty"`
	PreviewThread  []Threads       `gorm:"-" json:"preview_thread"`
	PreviewMessage string          `json:"preview_message"`
	LastReadAt     time.Time       `json:"last_read_at"`
}

type GetUserChannelsUnReadResp []struct {
	Channels
	UserId       string `json:"-"`
	ThreadCount  int64  `json:"thread_count"`
	Access       bool   `json:"access"`
	MentionCount int64  `json:"mention_count"`
	LastThreadId string `json:"last_thread_id"`
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
	Topic       string `json:"topic"`
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
	FullName         string `json:"full_name"`
	AvatarURL        string `json:"avatar_url"`
	DefaultAvatarURL string `json:"default_avatar_url"`
	Email            string `json:"email"`
}

type MessagesResp []struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
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
	UserIDs   []string `json:"user_ids" validate:"required,dive,required,uuid"`
	UserID    string   `json:"-"`
}

type RemoveMultipleMembersRequest struct {
	ChannelID string   `json:"channel_id" validate:"required"`
	UserIDs   []string `json:"user_ids" validate:"required,dive,required,uuid"`
	UserID    string   `json:"-"`
}

type ArchiveChannelRequest struct {
	Archived bool   `json:"archived"`
	UserId   string `json:"user_id" `
}

type MentionMessage struct {
	Mention []Mention
	Content any
}

type UnReadUpdate string

var Read UnReadUpdate = "read"
var NewThread UnReadUpdate = "new_thread"

func (r *Channels) CreateChannel(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &r)
	if err != nil {
		return errors.New("could not create channel an error occured: " + err.Error())
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

func (r *Channels) FetchChannelByID(db *storage.Database, channelId string) error {

	exist := postgresql.CheckExists(db.Postgresql, &r, "id = ?", channelId)
	if !exist {
		return errors.New("channel not found")
	}

	return nil
}

func (r *Channels) GetChannelByID(db *storage.Database, chanReq ChannelInfo) (GetChannelResp, error) {
	var (
		channel  Channels
		chanResp GetChannelResp
		ur       UserChannels
		webhook  Webhook
		owner    User
	)

	access := postgresql.CheckExists(db.Postgresql, &ur, "channels_id = ? AND user_id = ?", chanReq.ChannelID, chanReq.UserID)

	err, _ := postgresql.SelectOneFromDb(db.Postgresql.Preload("Users.Profile"), &channel, "id = ?", chanReq.ChannelID)
	if err != nil {
		return chanResp, fmt.Errorf("could not get channel by id: %v", err)
	}

	count, err := ur.CountChannelsUsers(db.Postgresql, chanReq.ChannelID)
	if err != nil {
		return chanResp, errors.New("could not get channel users count")
	}

	channel.UserCount = count
	webhook, err = webhook.GetChannelWebhook(db.Postgresql, chanReq)
	if err != nil {
		return chanResp, errors.New("could not get channel webhook")
	}

	// get owner name and email
	err, _ = postgresql.SelectOneFromDb(db.Postgresql, &owner, "id = ?", channel.OwnerId)
	if err != nil {
		return chanResp, errors.New("could not get channel owner")
	}

	// check for active buzz in this channel with host and participant details
	var activeBuzzInfo *ActiveBuzzInfo
	var buzzData struct {
		BuzzID           string
		HostID           string
		HostName         string
		ParticipantCount int
		StartedAt        time.Time
	}

	err = db.Postgresql.Raw(`
		SELECT b.id as buzz_id, b.host_id, u.name as host_name, 
		       b.buzz_start_time as started_at,
		       COUNT(bp.id) as participant_count
		FROM buzzs b
		JOIN users u ON b.host_id = u.id
		LEFT JOIN buzz_participants bp ON b.id = bp.buzz_id AND bp.status = ?
		WHERE b.channel_id = ? AND b.status = ? AND b.is_live_status = ?
		GROUP BY b.id, b.host_id, u.name, b.buzz_start_time
		LIMIT 1
	`, BuzzParticipantStatusActive, chanReq.ChannelID, BuzzStatusActive, true).Scan(&buzzData).Error

	if err == nil && buzzData.BuzzID != "" {
		activeBuzzInfo = &ActiveBuzzInfo{
			BuzzID:           buzzData.BuzzID,
			HostID:           buzzData.HostID,
			HostName:         buzzData.HostName,
			ParticipantCount: buzzData.ParticipantCount,
			StartedAt:        buzzData.StartedAt,
		}
	}

	var previewMedia []FileMediaResponse
	if db != nil {
		pm, _, err := channel.GetPreviewMedia(db, 10)
		if err == nil {
			previewMedia = pm
		}
	}

	var participants []Participant
	if db != nil {
		var channelUsers []User
		err := db.Postgresql.Joins("JOIN user_channels ON user_channels.user_id = users.id").
			Where("user_channels.channels_id = ?", channel.ID).
			Preload("Profile").
			Find(&channelUsers).Error

		if err == nil {
			for _, u := range channelUsers {
				isAdmin := u.ID == channel.OwnerId
				participants = append(participants, NewParticipant(u, isAdmin, "user"))
			}
		}
	}

	chanResp = GetChannelResp{
		Channels:     channel,
		OwnerName:    owner.Name,
		OwnerEmail:   owner.Email,
		WebhookUrl:   webhook.WebhookUrl,
		Access:       access,
		ActiveBuzz:   activeBuzzInfo,
		PreviewMedia: previewMedia,
		CreatedAt:    channel.CreatedAt,
		Participants: participants,
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
		Joins("left join profiles on profiles.userid = messages.user_id AND (profiles.organisation_id IS NULL OR profiles.organisation_id = ?)", userChannels.OrgId).
		Joins("left join users on users.id = messages.user_id").
		Where("messages.channels_id = ?", channelID).
		Scan(&messagesResp).Error
	if err != nil {
		return messagesResp, err
	}

	for i, msg := range messagesResp {
		messagesResp[i].DefaultAvatarURL = avatar.GenerateDefaultAvatarURL(msg.UserID)
	}

	return messagesResp, nil
}

func (r *Channels) AddUserToChannel(db *gorm.DB, req JoinChannelsRequest) (Channels, error) {

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

	if channel.IsPrivate && req.UserID != channel.OwnerId {
		return channel, errors.New("permission denied, channel type is private")
	}

	var userChannels UserChannels
	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	if exist {
		return channel, errors.New("user already in channel")
	}

	if req.Username == "" {
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
		userChannel  UserChannels
		userChanList []UserChannels
		validUserIds = []string{}
	)

	exists := postgresql.CheckExists(db, &r, "id = ?", channelID)
	if !exists {
		return validUserIds, errors.New("channel does not exist")
	}

	inviterExist := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", channelID, req.UserID)

	if r.IsPrivate && !inviterExist {
		return validUserIds, errors.New("invitation denied, inviter not in channel")
	}

	for _, user := range users {
		if user == req.UserID {
			continue
		}
		var userChannels UserChannels

		exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, user)
		if !exist {
			newUserChannels := UserChannels{
				ChannelsID: channelID,
				UserID:     user,
				Username:   userChannels.Username,
			}
			validUserIds = append(validUserIds, user)
			userChanList = append(userChanList, newUserChannels)
		}
	}

	err := postgresql.CreateMultipleRecords(db, userChanList, len(userChanList))
	if err != nil {
		return validUserIds, fmt.Errorf("could not add users to channel: %v", err)
	}

	return validUserIds, nil
}

func (r *Channels) RemoveMultipleUsersFromChannel(db *gorm.DB, req RemoveMultipleMembersRequest) ([]string, error) {
	var (
		channelID    = req.ChannelID
		validUserIds []string
		missingUsers []string
	)

	exists := postgresql.CheckExists(db, &r, "id = ?", channelID)
	if !exists {
		return validUserIds, errors.New("channel does not exist")
	}

	// Ensure remover is part of the channel to avoid arbitrary removals.
	removerExists := postgresql.CheckExists(db, &UserChannels{}, "channels_id = ? AND user_id = ?", channelID, req.UserID)
	if !removerExists {
		return validUserIds, errors.New("removal denied, remover not in channel")
	}

	for _, user := range req.UserIDs {
		var userChannels UserChannels
		exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, user)
		if !exist {
			missingUsers = append(missingUsers, user)
			continue
		}

		if err := postgresql.DeleteRecordFromDb(db, &userChannels); err != nil {
			return validUserIds, fmt.Errorf("could not remove user %s from channel: %v", user, err)
		}
		validUserIds = append(validUserIds, user)
	}

	if len(validUserIds) == 0 {
		return validUserIds, errors.New("no provided users are members of the channel")
	}

	if len(missingUsers) > 0 {
		return validUserIds, fmt.Errorf("users not in channel: %s", strings.Join(missingUsers, ", "))
	}

	return validUserIds, nil
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

func (c *Channels) Delete(db *gorm.DB) error {

	err := c.RemoveChannelResources(db)
	if err != nil {
		return fmt.Errorf("error removing channel resources: %v", err)
	}

	err = postgresql.DeleteRecordFromDb(db, &c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Channels) RemoveChannelResources(db *gorm.DB) error {
	var (
		userChannels UserChannels
		orgChanInt   OrganisationChannelsIntegrations
		thread       Threads
		webhook      Webhook
		chanworkflow ChannelWorkflow
	)

	err := db.Model(&userChannels).Where("channels_id = ?", c.ID).Delete(&userChannels).Error
	if err != nil {
		return errors.New("error removing users in channel")
	}

	err = postgresql.DeleteSpecificRecord(db, &orgChanInt, "channel_id = ?", c.ID)
	if err != nil {
		return errors.New("error removing channel from organisation channels integration")
	}

	//removing all associated threads tied to the channel
	thread.ChannelsID = c.ID
	if _, err := thread.ClearThreadsByChannelID(db); err != nil {
		return fmt.Errorf("failed to delete group DM channel threads: %v", err)
	}

	//remove all webhooks associated with the channel
	if err := webhook.DeleteChannelWebhook(db, c.ID); err != nil {
		return fmt.Errorf("failed to delete channel webhook: %v", err)
	}

	//remove all workflows associated with the channel
	chanworkflow.ChannelID = c.ID
	if err := chanworkflow.DeleteChannelWorkflows(db); err != nil {
		return fmt.Errorf("failed to delete channel workflows: %v", err)
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

func (r *Channels) UpdateChannels(db *gorm.DB, req UpdateChannelsRequest, userId string) (Channels, int, error) {
	var (
		channel Channels
		org     Organisation
	)

	err := db.Where("id = ?", r.ID).First(&channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return channel, http.StatusNotFound, errors.New("channel does not exist")
		}
		return channel, http.StatusInternalServerError, err
	}

	org.ID = channel.OrganisationID

	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = req.Name
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	if req.Topic != "" {
		updates["topic"] = req.Topic
	}

	if len(updates) == 0 {
		return Channels{}, http.StatusBadRequest, errors.New("no fields to update")
	}

	result := db.Model(&channel).Where("id = ?", r.ID).Updates(updates)
	if result.RowsAffected == 0 {
		return Channels{}, http.StatusInternalServerError, errors.New("failed to update channel")
	}

	updatedChannels := Channels{}
	err = db.First(&updatedChannels, "id = ?", r.ID).Error
	if err != nil {
		return Channels{}, http.StatusInternalServerError, err
	}

	return updatedChannels, http.StatusOK, nil
}

type ChannelMediaThreadResponse struct {
	ThreadID  string    `json:"thread_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	Media     []File    `json:"media"`
}

func (c *Channels) GetChannelMedia(db *storage.Database, ctx *gin.Context, mediaType string, orgID ...string) ([]ChannelMediaThreadResponse, postgresql.PaginationResponse, error) {
	pagination := postgresql.GetPagination(ctx)

	from := (pagination.Page - 1) * pagination.Limit
	size := pagination.Limit

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"match": map[string]any{
							"channels_id": c.ID,
						},
					},
				},
				"filter": []map[string]any{
					{
						"nested": map[string]any{
							"path": "media",
							"query": map[string]any{
								"exists": map[string]any{
									"field": "media.id",
								},
							},
						},
					},
				},
			},
		},
		"_source": []string{"thread_id", "media", "user_id", "created_at"},
		"from":    from,
		"size":    size,
		"sort": []map[string]any{
			{
				"created_at": map[string]any{
					"order": "desc",
				},
			},
		},
	}

	var threadData any
	err := elastic.SelectAll(db.Elastic, ThreadIndexName, query, &threadData)
	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	// Parse response
	threadDataMap, ok := threadData.(map[string]any)
	if !ok {
		return []ChannelMediaThreadResponse{}, postgresql.PaginationResponse{}, nil
	}

	hits, ok := threadDataMap["hits"].(map[string]any)
	if !ok {
		return []ChannelMediaThreadResponse{}, postgresql.PaginationResponse{}, nil
	}

	var totalHits int64
	if totalMap, ok := hits["total"].(map[string]any); ok {
		if val, ok := totalMap["value"].(float64); ok {
			totalHits = int64(val)
		}
	} else if val, ok := hits["total"].(float64); ok {
		totalHits = int64(val)
	}

	hitsArray, ok := hits["hits"].([]any)
	if !ok || len(hitsArray) == 0 {
		pagResp := postgresql.PaginationResponse{
			CurrentPage:     pagination.Page,
			TotalPagesCount: 0,
			PageCount:       0,
			TotalItems:      totalHits,
		}
		return []ChannelMediaThreadResponse{}, pagResp, nil
	}

	var targetOrg string
	if len(orgID) > 0 {
		targetOrg = orgID[0]
	}

	var profModel Profile
	var result []ChannelMediaThreadResponse
	for _, hit := range hitsArray {
		hitMap, ok := hit.(map[string]any)
		if !ok {
			continue
		}

		source, ok := hitMap["_source"].(map[string]any)
		if !ok {
			continue
		}

		threadID := utility.GetString(source, "thread_id")
		threadUserID := utility.GetString(source, "user_id")
		threadCreatedAtStr := utility.GetString(source, "created_at")

		var username string
		var avatarURL string

		if threadUserID != "" && threadUserID != "WEBHOOK" && db != nil && db.Postgresql != nil {
			if p, err := profModel.GetOrCreateProfileForOrg(db.Postgresql, threadUserID, targetOrg); err == nil {
				username = p.UserName
				avatarURL = p.AvatarURL
			}
		}

		if avatarURL == "" && threadUserID != "" {
			avatarURL = avatar.GenerateDefaultAvatarURL(threadUserID)
		}

		var createdAt time.Time
		if threadCreatedAtStr != "" {
			createdAt, _ = time.Parse(time.RFC3339, threadCreatedAtStr)
		}

		mediaArray, ok := source["media"].([]any)
		if !ok {
			continue
		}

		var mediaFiles []File
		for _, m := range mediaArray {
			mediaMap, ok := m.(map[string]any)
			if !ok {
				continue
			}

			file := File{
				ID:        utility.GetString(mediaMap, "id"),
				FileName:  utility.GetString(mediaMap, "file_name"),
				FileType:  utility.GetString(mediaMap, "file_type"),
				MimeType:  utility.GetString(mediaMap, "mime_type"),
				FileLink:  utility.GetString(mediaMap, "file_link"),
				UserID:    threadUserID,
				CreatedAt: createdAt,
			}

			// Apply type filter if specified
			if mediaType != "" {
				if !MatchesMediaType(file.MimeType, mediaType) {
					continue
				}
			}

			mediaFiles = append(mediaFiles, file)
		}

		if len(mediaFiles) > 0 {
			result = append(result, ChannelMediaThreadResponse{
				ThreadID:  threadID,
				UserID:    threadUserID,
				Username:  username,
				AvatarURL: avatarURL,
				CreatedAt: createdAt,
				Media:     mediaFiles,
			})
		}
	}

	totalPages := int(math.Ceil(float64(totalHits) / float64(pagination.Limit)))
	if totalPages == 0 && totalHits > 0 {
		totalPages = 1
	}

	pagResp := postgresql.PaginationResponse{
		CurrentPage:     pagination.Page,
		TotalPagesCount: totalPages,
		PageCount:       len(result),
		TotalItems:      totalHits,
	}

	return result, pagResp, nil
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

func (uc *UserChannels) GetUserChannels(base *storage.Database, ids IDS) (GetUserChannelResp, error) {

	var (
		db     = base.Postgresql
		search = ids.Search
	)
	chanResp := make(GetUserChannelResp, 0)

	query := db.Model(&Channels{}).
		Select("channels.id, channels.name, channels.description, channels.organisation_id, channels.is_private, channels.owner_id, channels.archived, channels.group_id, channels.created_at, uc.mention_count, uc.thread_count, uc.last_thread_id, 'true' AS access").
		Joins("JOIN user_channels AS uc ON channels.id = uc.channels_id").
		Where("channels.organisation_id = ? AND uc.user_id = ? AND channels.archived = FALSE", ids.OrganisationID, ids.UserID)

	if search != "" {
		query = query.Where("channels.name ILIKE ?", "%"+search+"%")
	}

	if err := query.Order("channels.created_at DESC").Scan(&chanResp).Error; err != nil {
		return nil, errors.New("error fetching channels")
	}

	for i := range chanResp {
		var (
			avatars      []string
			totalMembers int64
			lastReadAt   *time.Time
		)
		previewThread := []Threads{}

		threadCtx := &gin.Context{
			Request: &http.Request{
				URL: &url.URL{
					RawQuery: "page=1&limit=50",
				},
			},
		}

		var thread Threads
		threads, _, _, threadErr := thread.GetAllThreadsByChannelID(threadCtx, db, ids.UserID, chanResp[i].ID)
		if threadErr == nil && len(threads) > 0 {
			previewThread = threads
		}

		if err := db.Table("user_channels").
			Select("profiles.avatar_url").
			Joins("JOIN channels ON channels.id = user_channels.channels_id").
			Joins("JOIN profiles ON profiles.userid = user_channels.user_id AND (profiles.organisation_id IS NULL OR profiles.organisation_id = channels.organisation_id)").
			Where("user_channels.channels_id = ? AND profiles.avatar_url != ''", chanResp[i].ID).
			Limit(8).
			Pluck("profiles.avatar_url", &avatars).Error; err != nil {
			return nil, fmt.Errorf("error fetching member avatars: %w", err)
		}

		if err := db.Table("user_channels").
			Where("channels_id = ?", chanResp[i].ID).
			Count(&totalMembers).Error; err != nil {
			return nil, fmt.Errorf("error counting members: %w", err)
		}

		membersLeft := int(totalMembers) - len(avatars)
		if membersLeft <= 0 {
			membersLeft = int(totalMembers)
		}

		err := db.Table("user_channels").
			Where("channels_id = ? AND user_id = ?", chanResp[i].ID, ids.UserID).
			Pluck("last_read_at", &lastReadAt).Error
		if err != nil {
			lastReadAt = &time.Time{}
			return nil, fmt.Errorf("error fetching last read at: %w", err)
		}

		unreadCount, err := GetChannelUnreadCount(base, chanResp[i].ID, ids.UserID, *lastReadAt)
		if err != nil {
			chanResp[i].UnreadCount = 0
		} else {
			chanResp[i].UnreadCount = unreadCount
		}

		lastPostTime, err := FetchLastMessageTime(base, chanResp[i].ID)
		if err != nil {
			chanResp[i].LastPostTime = "Last post unavailable"
		} else if lastPostTime.IsZero() {
			chanResp[i].LastPostTime = "No posts yet"
		} else {
			duration := time.Since(lastPostTime)

			switch {
			case duration < time.Minute:
				chanResp[i].LastPostTime = "Just now"
			case duration < time.Hour:
				minutes := int(duration.Minutes())
				if minutes == 1 {
					chanResp[i].LastPostTime = "1 minute ago"
				} else {
					chanResp[i].LastPostTime = fmt.Sprintf("%d minutes ago", minutes)
				}
			case duration < 24*time.Hour:
				hours := int(duration.Hours())
				if hours == 1 {
					chanResp[i].LastPostTime = "1 hour ago"
				} else {
					chanResp[i].LastPostTime = fmt.Sprintf("%d hourrs ago", hours)
				}
			case duration < 7*24*time.Hour:
				days := int(duration.Hours() / 24)
				if days == 1 {
					chanResp[i].LastPostTime = "1 day ago"
				} else {
					chanResp[i].LastPostTime = fmt.Sprintf("%d days ago", days)
				}
			default:
				chanResp[i].LastPostTime = "Long time ago"
			}
		}

		chanResp[i].MemberAvatars = avatars
		chanResp[i].MembersCount = membersLeft
		chanResp[i].LastReadAt = *lastReadAt
		if previewThread != nil {
			chanResp[i].PreviewThread = previewThread
		} else {
			chanResp[i].PreviewThread = []Threads{}
		}
		previewMessage := ""
		if len(previewThread) > 0 {
			previewMessage = BuildPreviewMessage(previewThread[0].Content, previewThread[0].Media)
		}
		chanResp[i].PreviewMessage = previewMessage
		parts := strings.Split(chanResp[i].ID, "-")
		lastPart := parts[len(parts)-1]
		chanResp[i].ChannelSlug = fmt.Sprintf("%s-%s", slug.Make(chanResp[i].Name), lastPart)
	}

	sort.Slice(chanResp, func(i, j int) bool {
		if chanResp[i].UnreadCount != chanResp[j].UnreadCount {
			return chanResp[i].UnreadCount > chanResp[j].UnreadCount
		}

		var iTime, jTime time.Time
		iHasThread := chanResp[i].PreviewThread != nil && len(chanResp[i].PreviewThread) > 0
		jHasThread := chanResp[j].PreviewThread != nil && len(chanResp[j].PreviewThread) > 0

		if iHasThread {
			iTime = chanResp[i].PreviewThread[0].CreatedAt
		}

		if jHasThread {
			jTime = chanResp[j].PreviewThread[0].CreatedAt
		}

		if iHasThread && jHasThread {
			return iTime.After(jTime)
		}

		if iHasThread {
			return true
		}

		if jHasThread {
			return false
		}

		return chanResp[i].CreatedAt.After(chanResp[j].CreatedAt)
	})

	// Batch fetch active buzzes for all channels to avoid N+1 queries
	if len(chanResp) > 0 {
		channelIDs := make([]string, len(chanResp))
		for i := range chanResp {
			channelIDs[i] = chanResp[i].ID
		}

		var activeBuzzes []struct {
			BuzzID           string
			ChannelID        string
			HostID           string
			HostName         string
			ParticipantCount int
			StartedAt        time.Time
		}

		err := db.Raw(`
			SELECT b.id as buzz_id, b.channel_id, b.host_id, u.name as host_name, 
			       b.buzz_start_time as started_at,
			       COUNT(bp.id) as participant_count
			FROM buzzs b
			JOIN users u ON b.host_id = u.id
			LEFT JOIN buzz_participants bp ON b.id = bp.buzz_id AND bp.status = ?
			WHERE b.channel_id IN ? AND b.status = ? AND b.is_live_status = ?
			GROUP BY b.id, b.channel_id, b.host_id, u.name, b.buzz_start_time
		`, BuzzParticipantStatusActive, channelIDs, BuzzStatusActive, true).Scan(&activeBuzzes).Error

		if err == nil {
			// Create a map for quick lookup
			buzzMap := make(map[string]*ActiveBuzzInfo)
			for _, buzz := range activeBuzzes {
				buzzMap[buzz.ChannelID] = &ActiveBuzzInfo{
					BuzzID:           buzz.BuzzID,
					HostID:           buzz.HostID,
					HostName:         buzz.HostName,
					ParticipantCount: buzz.ParticipantCount,
					StartedAt:        buzz.StartedAt,
				}
			}

			// Assign active buzz info to channels
			for i := range chanResp {
				if buzzInfo, exists := buzzMap[chanResp[i].ID]; exists {
					chanResp[i].ActiveBuzz = buzzInfo
				}
			}
		}
	}

	return chanResp, nil
}

func (uc *UserChannels) GetUserNotInChannels(db *gorm.DB, ids IDS) (GetUserNotChannelResp, error) {
	var (
		org      Organisation
		chanResp GetUserNotChannelResp
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", ids.OrganisationID)
	if !exists {
		return chanResp, errors.New("organisation does not exist")
	}

	err := db.Table("channels").
		Select("channels.id, channels.name, channels.description, channels.created_at, channels.archived, 'false' AS access").
		Where("channels.id NOT IN (SELECT user_channels.channels_id FROM user_channels WHERE user_channels.user_id = ?)", ids.UserID).
		Where("channels.organisation_id = ? AND channels.is_private !=  ?", ids.OrganisationID, "true").
		Order("channels.created_at").
		Scan(&chanResp).Error

	if err != nil {
		return chanResp, errors.New("could not get channels user is not part of")
	}
	return chanResp, nil
}

func (ch *Channels) FetchChannelUsers(db *gorm.DB, channelId, userId string) ([]string, error) {
	var users []string

	if err := db.Table("user_channels").
		Select("user_channels.*").
		Where("user_channels.channels_id = ? AND user_channels.user_id != ?", channelId, userId).
		Pluck("user_id", &users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (c *UserChannels) UpdateLastRead(db *gorm.DB, req UpdateLastRead, mu *sync.Mutex, logger *utility.Logger) bool {

	mu.Lock()
	defer mu.Unlock()

	var uc UserChannels

	query := "channels_id = ? AND user_id = ?"

	exists := postgresql.CheckExists(db, &uc, query, c.ChannelsID, c.UserID)

	if !exists {
		return false
	}

	updateFields := map[string]any{
		"last_thread_id": req.LastThreadId,
		"last_read_at":   req.LastReadAt,
		"mention_count":  0,
		"thread_count":   0,
	}

	result := db.Model(&UserChannels{}).
		Where(query, c.ChannelsID, c.UserID).
		Updates(updateFields)

	if result.Error != nil {
		logger.Error("an error occurend while updating user last read: %v", result.Error)
		return false
	}

	logger.Info("user last read updated successfully")
	return true

}

func (c *UserChannels) UpdateUnReadCount(db *gorm.DB, mu *sync.Mutex, logger *utility.Logger) {

	mu.Lock()
	defer mu.Unlock()

	query := "channels_id = ? AND user_id != ?"

	updateFields := map[string]any{
		"thread_count": gorm.Expr("thread_count + 1"),
	}

	result := db.Model(&UserChannels{}).
		Where(query, c.ChannelsID, c.UserID).
		Updates(updateFields)

	if result.Error != nil {
		logger.Error("an error occurred while updating user channel counts: %v", result.Error)
		return
	}

	logger.Info("user channels counts updated successfully")
}

func (c *UserChannels) ProcessMentions(db *gorm.DB, req []Mention, mu *sync.Mutex, logger *utility.Logger) {

	mu.Lock()
	defer mu.Unlock()

	IdCount := map[string]int{}
	channelMention := false

	for _, mention := range req {
		if mention.ID == "00000000-0000-0000-0000-000000000000" {
			channelMention = true
			break
		}
		if mention.Type == "user" {
			IdCount[mention.ID]++
		}
	}

	if len(IdCount) == 0 && !channelMention {
		logger.Info("No mentions to update")
		return
	}

	query := ""
	args := []any{}

	if !channelMention {

		var userIDs []string
		caseStmt := "CASE user_id"
		for userID, count := range IdCount {
			userIDs = append(userIDs, fmt.Sprintf("'%s'", userID))
			caseStmt += fmt.Sprintf(" WHEN '%s' THEN mention_count + %d", userID, count)
		}
		caseStmt += " END"

		// Build the query
		query = fmt.Sprintf(`
		UPDATE user_channels
		SET mention_count = %s
		WHERE channels_id = ? AND user_id IN (%s)
	`, caseStmt, strings.Join(userIDs, ","))

		args = append(args, c.ChannelsID)

	} else {
		caseStmt := "mention_count + 1"
		query = fmt.Sprintf(`
		UPDATE user_channels
		SET mention_count = %s
		WHERE channels_id = ? AND user_id != ?
	`, caseStmt)
		args = append(args, c.ChannelsID, c.UserID)
	}

	if err := db.Exec(query, args...).Error; err != nil {
		logger.Error("Bulk update failed: %v", err)
		return
	}

	logger.Info("Mentions processed and mention_count updated successfully")
}

func (uc *UserChannels) GetUserChannel(base *storage.Database, userId, channel_id string) (GetUserChannelResp, error) {

	var (
		chanResp GetUserChannelResp
		db       = base.Postgresql
	)

	if err := db.Model(&Channels{}).
		Select("channels.id, channels.name, channels.description, channels.organisation_id, channels.owner_id,  channels.is_private, channels.group_id, channels.created_at, uc.mention_count, uc.thread_count, uc.last_thread_id, 'true' AS access").
		Joins("JOIN user_channels AS uc ON channels.id = uc.channels_id").
		Where("channels.id = ? AND uc.user_id = ?", channel_id, userId).
		Order("channels.created_at").
		Scan(&chanResp).Error; err != nil {
		return nil, errors.New("error fetching channels")
	}

	return chanResp, nil
}

func (uc *UserChannels) GetUserChannelsUnreadThread(base *storage.Database, userId, channel_id string) (GetUserChannelsUnReadResp, error) {

	var (
		chanResp GetUserChannelsUnReadResp
		db       = base.Postgresql
	)

	if err := db.Model(&Channels{}).
		Select("channels.id, channels.name, channels.description, channels.organisation_id, channels.is_private, channels.owner_id, channels.group_id, channels.created_at, uc.mention_count, uc.thread_count, uc.last_thread_id, 'true' AS access, uc.user_id").
		Joins("JOIN user_channels AS uc ON channels.id = uc.channels_id").
		Where("channels.id = ? AND (uc.user_id != ? OR uc.mention_count > 0)", channel_id, userId).
		Order("channels.created_at").
		Scan(&chanResp).Error; err != nil {
		return nil, errors.New("error fetching channels")
	}

	return chanResp, nil
}

func (c *UserChannels) SendChannelUnReadUpdate(mu *sync.Mutex, logger *utility.Logger, updateType UnReadUpdate, mentionMsg MentionMessage) {

	mu.Lock()
	defer mu.Unlock()

	if updateType == Read {

		res, err := c.GetUserChannel(storage.DB, c.UserID, c.ChannelsID)

		if err != nil {
			logger.Error("Bulk update failed: %v", err)
			return
		}

		if len(res) < 1 {
			logger.Error("Empty channel response")
			return
		}

		notification := Notification[UnReadThreadChange]
		notification.SectionType = ChannelsSection
		notification.Content = res[0]
		notification.NotificationId = utility.GenerateUUID()

		err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, c.UserID), notification)
		if err != nil {
			logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, with userid: %s error: %v", c.ChannelsID, c.UserID, err.Error()))
			return
		}

		triggerNotif := Notification[TriggerNotification]
		triggerNotif.SectionType = ChannelsSection
		triggerNotif.ModificationDetails = &ModificationDetails{
			OrgId:     c.OrgId,
			ChannelId: c.ChannelsID,
			UserId:    c.UserID,
		}
		triggerNotif.Content = TriggerNotificationPayload{
			TriggerAction:   "refresh",
			TargetComponent: "sidebar",
		}
		triggerNotif.NotificationId = utility.GenerateUUID()

		centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, c.UserID), triggerNotif)
	}

	if updateType == NewThread {

		userMention := map[string]bool{}
		channelMention := false
		mentionNotification := Notification[ChannelMention]

		for _, mention := range mentionMsg.Mention {
			if mention.ID == "00000000-0000-0000-0000-000000000000" {
				channelMention = true
				break
			}
			if mention.Type == "user" {
				userMention[mention.ID] = true
			}
		}

		res, err := c.GetUserChannelsUnreadThread(storage.DB, c.UserID, c.ChannelsID)

		if err != nil {
			logger.Error("Bulk update failed: %v", err)
			return
		}

		if len(res) < 1 {
			logger.Error("Empty channel response")
			return
		}

		if len(userMention) > 0 || channelMention {
			mentionNotification = Notification[ChannelMention]
			mentionNotification.Content = mentionMsg.Content
			mentionNotification.NotificationId = utility.GenerateUUID()
		}

		for _, update := range res {
			notification := Notification[UnReadThreadChange]
			notification.SectionType = ChannelsSection
			notification.Content = update
			notification.NotificationId = utility.GenerateUUID()

			err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, update.UserId), notification)
			if err != nil {
				logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, with userid: %s error: %v", c.ChannelsID, update.UserId, err.Error()))
				return
			}

			if channelMention || userMention[update.UserId] {
				err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, update.UserId), mentionNotification)
				if err != nil {
					logger.Error(fmt.Sprintf("Error mention message to Publishing to channelid: %s, with userid: %s error: %v", c.ChannelsID, update.UserId, err.Error()))
					return
				}
				logger.Info("Published in-channel mention to user : %s", update.UserId)
			}
		}
	}

	logger.Info("user last read updated successfully")
}

func GetChannelUnreadCount(db *storage.Database, channelID, userID string, lastReadAt time.Time) (int64, error) {
	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id": channelID,
						},
					},
					{
						"range": map[string]any{
							"created_at": map[string]any{
								"gt": lastReadAt.Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
		"track_total_hits": true,
		"size":             0,
	}

	var result any
	err := elastic.SelectAll(db.Elastic, MessageIndexName, query, &result)
	if err != nil {
		return 0, err
	}

	hits, ok := result.(map[string]any)["hits"].(map[string]any)
	if !ok {
		return 0, nil
	}
	total, ok := hits["total"].(map[string]any)
	if !ok {
		return 0, nil
	}
	value, ok := total["value"].(float64)
	if !ok {
		return 0, nil
	}
	return int64(value), nil
}

func UpdateUserChannelLastRead(db *gorm.DB, channelID, userID string) error {
	return db.Model(&UserChannels{}).
		Where("channels_id = ? AND user_id = ?", channelID, userID).
		Update("last_read_at", time.Now()).Error
}

// GetPreviewMedia fetches the N most recent media files from threads in a channel
func (c *Channels) GetPreviewMedia(db *storage.Database, limit int) ([]FileMediaResponse, int, error) {
	if limit <= 0 {
		limit = 10
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id": c.ID,
						},
					},
				},
				"filter": []map[string]any{
					{
						"nested": map[string]any{
							"path": "media",
							"query": map[string]any{
								"exists": map[string]any{
									"field": "media.id",
								},
							},
						},
					},
				},
			},
		},
		"_source": []string{"media", "user_id", "created_at"},
		"size":    limit,
		"sort": []map[string]any{
			{
				"created_at": map[string]any{
					"order": "desc",
				},
			},
		},
	}

	var threadData any
	err := elastic.SelectAll(db.Elastic, ThreadIndexName, query, &threadData)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch preview media: %w", err)
	}

	threadDataMap, ok := threadData.(map[string]any)
	if !ok {
		return []FileMediaResponse{}, 0, nil
	}

	hits, ok := threadDataMap["hits"].(map[string]any)
	if !ok {
		return []FileMediaResponse{}, 0, nil
	}

	hitsArray, ok := hits["hits"].([]any)
	if !ok || len(hitsArray) == 0 {
		return []FileMediaResponse{}, 0, nil
	}

	var allMedia []FileMediaResponse
	for _, hit := range hitsArray {
		if len(allMedia) >= limit {
			break
		}

		hitMap, ok := hit.(map[string]any)
		if !ok {
			continue
		}

		source, ok := hitMap["_source"].(map[string]any)
		if !ok {
			continue
		}

		threadUserID := utility.GetString(source, "user_id")
		threadCreatedAt := utility.GetString(source, "created_at")

		mediaArray, ok := source["media"].([]any)
		if !ok {
			continue
		}

		for _, m := range mediaArray {
			if len(allMedia) >= limit {
				break
			}

			mediaMap, ok := m.(map[string]any)
			if !ok {
				continue
			}

			var createdAt time.Time
			if threadCreatedAt != "" {
				createdAt, _ = time.Parse(time.RFC3339, threadCreatedAt)
			}

			file := FileMediaResponse{
				ID:        utility.GetString(mediaMap, "id"),
				FileName:  utility.GetString(mediaMap, "file_name"),
				FileType:  utility.GetString(mediaMap, "file_type"),
				MimeType:  utility.GetString(mediaMap, "mime_type"),
				FileLink:  utility.GetString(mediaMap, "file_link"),
				UserID:    threadUserID,
				CreatedAt: createdAt,
			}

			allMedia = append(allMedia, file)
		}
	}

	return allMedia, len(allMedia), nil
}

func (u *UserChannels) GetChannelsWithMentions(db *gorm.DB, userID string) (map[string]time.Time, error) {
	var results []struct {
		ChannelsID string
		LastReadAt time.Time
	}

	err := db.Model(&UserChannels{}).
		Select("channels_id, last_read_at").
		Where("user_id = ? AND mention_count > 0", userID).
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	channelMap := make(map[string]time.Time)
	for _, r := range results {
		channelMap[r.ChannelsID] = r.LastReadAt
	}
	return channelMap, nil
}

// UserChannelHistory methods

// SaveChannelHistory saves a record of user channel action (banished, removed, restored)
func (h *UserChannelHistory) SaveChannelHistory(db *gorm.DB) error {
	return postgresql.CreateOneRecord(db, &h)
}

// GetUserChannelHistory retrieves all channel history for a user
func (h *UserChannelHistory) GetUserChannelHistory(db *gorm.DB, userID string) ([]UserChannelHistory, error) {
	var history []UserChannelHistory
	err := db.Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		Find(&history).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get user channel history: %v", err)
	}
	return history, nil
}

// GetUserBanishedChannels retrieves channels the user was banished from but not yet restored
func (h *UserChannelHistory) GetUserBanishedChannels(db *gorm.DB, userID string) ([]UserChannelHistory, error) {
	var history []UserChannelHistory
	err := db.Where("user_id = ? AND action = ? AND deleted_at IS NULL", userID, "banished").
		Order("created_at DESC").
		Find(&history).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get user banished channels: %v", err)
	}
	return history, nil
}

// MarkChannelsAsRestored marks banished channels as restored by updating records that contain the specified channel IDs
func (h *UserChannelHistory) MarkChannelsAsRestored(db *gorm.DB, userID string, channelIDs []string) error {
	if len(channelIDs) == 0 {
		return nil
	}

	// Update existing banished records to restored where channel_ids array contains any of the specified IDs
	err := db.Model(&UserChannelHistory{}).
		Where("user_id = ? AND action = ? AND deleted_at IS NULL AND channel_ids && ?", userID, "banished", pq.Array(channelIDs)).
		Update("action", "restored").Error
	if err != nil {
		return fmt.Errorf("failed to mark channels as restored: %v", err)
	}

	return nil
}

// DeleteChannelHistory soft deletes channel history records containing the specified channel IDs
func (h *UserChannelHistory) DeleteChannelHistory(db *gorm.DB, userID string, channelIDs []string) error {
	if len(channelIDs) == 0 {
		return nil
	}

	// Use PostgreSQL array overlap operator (&&) to find records containing any of the channel IDs
	err := db.Model(&UserChannelHistory{}).
		Where("user_id = ? AND channel_ids && ?", userID, pq.Array(channelIDs)).
		Update("deleted_at", time.Now()).Error
	if err != nil {
		return fmt.Errorf("failed to delete channel history: %v", err)
	}

	return nil
}

// GetBanishedChannelIDs extracts all unique channel IDs from a user's banished history
func (h *UserChannelHistory) GetBanishedChannelIDs(db *gorm.DB, userID string) ([]string, error) {
	var historyRecords []UserChannelHistory
	err := db.Where("user_id = ? AND action = ? AND deleted_at IS NULL", userID, "banished").
		Select("channel_ids").
		Find(&historyRecords).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get banished channel IDs: %v", err)
	}

	// Use a map to deduplicate channel IDs
	channelMap := make(map[string]bool)
	for _, record := range historyRecords {
		for _, channelID := range record.ChannelIDs {
			channelMap[channelID] = true
		}
	}

	// Convert map keys to slice
	uniqueChannelIDs := make([]string, 0, len(channelMap))
	for channelID := range channelMap {
		uniqueChannelIDs = append(uniqueChannelIDs, channelID)
	}

	return uniqueChannelIDs, nil
}
