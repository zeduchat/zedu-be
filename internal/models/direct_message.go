package models

import (
	"cmp"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"

	"github.com/hngprojects/telex_be/utility"
)

type DmChannels struct {
	ID               string         `gorm:"type:uuid" json:"id"`
	UserId           string         `gorm:"type:uuid" json:"-"`
	ChannelId        string         `gorm:"type:uuid" json:"channel_id"`
	OrgId            string         `gorm:"type:uuid" json:"-"`
	ParticipantId    *string        `gorm:"type:uuid" json:"-"`
	ParticipantHash  string         `gorm:"type:string" json:"participant_hash"`
	ChatType         string         `gorm:"type:string;default:user" json:"chat_type"`  // user or bot
	ChannelType      string         `gorm:"type:string;default:dm" json:"channel_type"` // dm or group_dm
	CreatedAt        time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt        time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	ThreadCount      int64          `gorm:"column:thread_count;default:0" json:"thread_count"`
	LastThreadId     string         `gorm:"column:last_thread_id" json:"last_thread_id"`
	LastReadAt       time.Time      `gorm:"column:last_read_at" json:"last_read_at"`
	InteractedAt     time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"-"`
	GroupDescription string         `gorm:"column:group_description" json:"group_description"`
	VisibilityStatus *bool          `gorm:"column:visibility_status;default:true" json:"visibility_status"`
}

type UpdateDmVisibilityRequest struct {
	ChannelId        string `json:"-"`
	UserId           string `json:"-"`
	VisibilityStatus *bool  `json:"visibility_status" validate:"required"`
}


type DmChannelsResponse struct {
	ID               string        `json:"channel_id"`
	Name             string        `json:"username"`
	ParticipantId    string        `json:"participant_id"`
	AvatarUrl        string        `json:"avatar_url"`
	DefaultAvatarUrl string        `json:"default_avatar_url"`
	ParticipantEmail string        `json:"participant_email"`
	ChannelType      string        `json:"channel_type"`
	ThreadCount      int64         `json:"thread_count"`
	LastThreadId     string        `json:"last_thread_id"`
	LastReadAt       time.Time     `json:"last_read_at"`
	UserId           string        `json:"-"`
	PreviewMessage   string        `json:"preview_message"`
	PreviewThread    []Threads     `gorm:"-" json:"preview_thread"`
	Participants     []Participant `gorm:"-" json:"participants,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	IsFavourite      bool          `gorm:"-" json:"is_favourite"`
	IsSuggested      bool          `gorm:"-" json:"is_suggested"`
}

type DmChannelsRequest struct {
	ChatType      string `json:"chat_type" validate:"required,oneof=user bot"`
	ParticipantId string `json:"participant_id" validate:"required"`
	UserId        string `json:"user_id"`
	OrgId         string `json:"org_id"`
	ChannelId     string `json:"channel_id"`
}

type DmChannelMediaRequest struct {
	UserId    string `json:"user_id"`
	OrgId     string `json:"org_id"`
	ChannelId string `json:"channel_id"`
	MediaType string `json:"media_type"` // Optional: images, videos, documents, audio
}

type Participant struct {
	UserId            string   `json:"user_id"`
	Username          string   `json:"username"`
	Email             string   `json:"email"`
	Phone             string   `json:"phone"`
	FirstName         string   `json:"first_name"`
	LastName          string   `json:"last_name"`
	FullName          string   `json:"full_name"`
	DisplayName       string   `json:"display_name"`
	AvatarUrl         string   `json:"avatar_url"`
	DefaultAvatarUrl  string   `json:"default_avatar_url"`
	Title             string   `json:"title"`
	NamePronunciation string   `json:"name_pronunciation"`
	Timezone          string   `json:"timezone"`
	Icon              string   `json:"icon"`
	Text              string   `json:"text"`
	PauseNotification bool     `json:"pause_notification"`
	StatusTimeout     string   `json:"status_timeout"`
	WorkspaceID       string   `json:"workspace_id"`
	Track             string   `json:"track"`
	Links             []string `json:"links"`
	Online            bool     `json:"online"`
	IsActive          bool     `json:"is_active"`
	UserType          string   `json:"user_type"`
	IsAdmin           bool     `json:"is_admin"`
}

// NewParticipant builds a full Participant from a User (with Profile preloaded).
func NewParticipant(u User, isAdmin bool, userType string) Participant {
	username := u.Profile.UserName
	if username == "" {
		username = u.Name
	}
	return Participant{
		UserId:            u.ID,
		Username:          username,
		Email:             u.Email,
		Phone:             u.Profile.Phone,
		FirstName:         u.Profile.FirstName,
		LastName:          u.Profile.LastName,
		FullName:          u.Profile.FullName,
		DisplayName:       u.Profile.DisplayName,
		AvatarUrl:         u.Profile.AvatarURL,
		DefaultAvatarUrl:  avatar.GenerateDefaultAvatarURL(u.ID),
		Title:             u.Profile.Title,
		NamePronunciation: u.Profile.NamePronunciation,
		Timezone:          u.Profile.Timezone,
		Icon:              u.Profile.Icon,
		Text:              u.Profile.Text,
		PauseNotification: u.Profile.PauseNotification,
		StatusTimeout:     u.Profile.StatusTimeout,
		WorkspaceID:       u.Profile.WorkspaceID,
		Track:             u.Profile.Track,
		Links:             u.Profile.Links,
		Online:            u.Profile.Online,
		IsActive:          u.Profile.IsActive,
		UserType:          userType,
		IsAdmin:           isAdmin,
	}
}

type GroupInCommon struct {
	Name             string   `json:"name"`
	AvatarURL        string   `json:"avatar_url"`
	DefaultAvatarURL string   `json:"default_avatar_url"`
	ChannelID        string   `json:"channel_id"`
	Participants     []string `json:"participants"`
}

type DmParticipantsResponse struct {
	Type             string              `json:"type"` // bot, dm, or groupdm
	GroupDescription string              `json:"group_description"`
	Participants     []Participant       `json:"participants"`
	PreviewMedia     []FileMediaResponse `json:"preview_media"`
	CreatedAt        time.Time           `json:"created_at"`
	IsFavourite      bool                `json:"is_favourite"`
	GroupsInCommon   []GroupInCommon     `json:"groups_in_common,omitempty"`
}

type GroupDescriptionRequest struct {
	ChannelId   string `json:"channel_id"`
	UserId      string `json:"user_id"`
	OrgId       string `json:"org_id"`
	Description string `json:"description" validate:"required,max=500"`
}

func FetchDetailsFromAgentJSON(agent OrganisationIntegrations) (map[string]any, error) {

	var data_r map[string]any

	data_r = map[string]any{
		"app_name":        agent.AppName,
		"app_logo":        agent.AppLogo,
		"app_description": agent.AppDescription,
		"version":         agent.Version,
		"is_paid":         agent.IsPaid,
		"is_approved":     agent.IsApproved,
		"provider":        agent.Provider,
		"prices":          agent.Prices,
		"agent":           true,
	}

	return data_r, nil
}

func buildDmResponse(dm *DmChannels, appName, appLogo string) DmChannelsResponse {
	participants := []Participant{
		{
			AvatarUrl: appLogo,
			Username:  appName,
			Email:     appName,
			UserType:  "bot",
			UserId:    *dm.ParticipantId,
		},
	}

	return DmChannelsResponse{
		ID:               dm.ChannelId,
		ParticipantId:    *dm.ParticipantId,
		ParticipantEmail: appName,
		AvatarUrl:        appLogo,
		Name:             appName,
		LastThreadId:     dm.LastThreadId,
		ThreadCount:      dm.ThreadCount,
		LastReadAt:       dm.LastReadAt,
		Participants:     participants,
	}
}

func (dm *DmChannels) CreateAgentDMChannel(db *gorm.DB) (DmChannelsResponse, error) {
	var orgAgent OrganisationIntegrations
	if !postgresql.CheckExists(db, &orgAgent, "org_id = ? AND integration_id = ?", dm.OrgId, dm.ParticipantId) {
		return DmChannelsResponse{}, fmt.Errorf("agent participant does not exist in organisation %v", dm.OrgId)
	}

	agentDetails, err := FetchDetailsFromAgentJSON(orgAgent)
	if err != nil {
		return DmChannelsResponse{}, fmt.Errorf("failed to fetch agent details: %w", err)
	}

	appName, appLogo := utility.GetString(agentDetails, "app_name"), utility.GetString(agentDetails, "app_logo")
	if appName == "" {
		return DmChannelsResponse{}, errors.New("missing required agent details (app_name, app_logo)")
	}

	if postgresql.CheckExists(db, &DmChannels{}, "user_id = ? AND participant_id = ? AND org_id = ?", dm.UserId, dm.ParticipantId, dm.OrgId) {
		return buildDmResponse(dm, appName, appLogo), nil
	}

	if err := postgresql.CreateOneRecord(db, &dm); err != nil {
		return DmChannelsResponse{}, fmt.Errorf("failed to create DM channel: %w", err)
	}

	return buildDmResponse(dm, appName, appLogo), nil
}

func (dm *DmChannels) CreateDmChannel(db *gorm.DB) (DmChannelsResponse, error) {
	var (
		user        User
		dmchanresp  DmChannelsResponse
		existDmchan DmChannels
	)

	userDetails, err := user.GetUserByID(db, *dm.ParticipantId)
	if err != nil {
		return dmchanresp, errors.New("participant does not exist")
	}

	participant := NewParticipant(userDetails, false, "user")

	exists := postgresql.CheckExists(db, &existDmchan, "user_id = ? AND participant_id = ? AND org_id = ?", dm.UserId, *dm.ParticipantId, dm.OrgId)
	if exists {
		participants := []Participant{participant}

		dmchanresp.AvatarUrl = userDetails.Profile.AvatarURL
		dmchanresp.DefaultAvatarUrl = avatar.GenerateDefaultAvatarURL(userDetails.ID)
		dmchanresp.Name = participant.Username
		dmchanresp.ID = existDmchan.ChannelId
		dmchanresp.ParticipantId = *dm.ParticipantId
		dmchanresp.ParticipantEmail = userDetails.Email
		dmchanresp.Participants = participants

		return dmchanresp, nil
	}

	err = postgresql.CreateOneRecord(db, &dm)
	if err != nil {
		return dmchanresp, err
	}

	participants := []Participant{participant}

	dmchanresp.AvatarUrl = userDetails.Profile.AvatarURL
	dmchanresp.DefaultAvatarUrl = avatar.GenerateDefaultAvatarURL(userDetails.ID)
	dmchanresp.Name = participant.Username
	dmchanresp.ID = dm.ChannelId
	dmchanresp.ParticipantId = *dm.ParticipantId
	dmchanresp.ParticipantEmail = userDetails.Email
	dmchanresp.Participants = participants

	return dmchanresp, nil
}

func (dm *DmChannels) DeleteDmChannel(db *gorm.DB, loggers ...*utility.Logger) error {
	var logger *utility.Logger
	if len(loggers) > 0 {
		logger = loggers[0]
	}

	if logger != nil {
		logger.Info("Starting DM channel deletion flow for channel ID: %s, user ID: %s", dm.ID, dm.UserId)
	}

	var (
		user User
	)

	_, err := user.GetUserByID(db, dm.UserId)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to fetch user %s during DM channel deletion: %v", dm.UserId, err)
		}
		return err
	}

	err = postgresql.DeleteSpecificRecord(
		db,
		&DmChannels{},
		"channel_id = ? AND user_id = ?",
		dm.ID,
		dm.UserId,
	)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to delete DM channel record %s from DB: %v", dm.ID, err)
		}
		return err
	}

	// Check if any other user still has an active entry for this DM channel
	otherDmExists := postgresql.CheckExists(db, &DmChannels{}, "channel_id = ? AND deleted_at IS NULL", dm.ID)
	otherPartExists := postgresql.CheckExists(db, &ChannelParticipant{}, "channel_id = ? AND deleted_at IS NULL", dm.ID)

	if otherDmExists || otherPartExists {
		if logger != nil {
			logger.Info("Other active participants still exist for channel ID %s. Preserving Elastic threads and messages.", dm.ID)
			logger.Info("Completed DM channel deletion flow for channel ID: %s, user ID: %s", dm.ID, dm.UserId)
		}
		return nil
	}

	// Background process to delete channel threads and messages from Elastic (since no remaining participants exist)
	channelID := dm.ID
	go func(cID string) {
		defer func() {
			if r := recover(); r != nil {
				if logger != nil {
					logger.Error("Recovered panic in background Elastic thread/message deletion for channel %s: %v", cID, r)
				}
			}
		}()

		if logger != nil {
			logger.Info("Starting background process: deleting channel threads and messages from Elastic for channel ID: %s", cID)
		}

		threadModel := Threads{ID: cID}
		if _, err := threadModel.ClearThreadsByChannelID(db); err != nil {
			if logger != nil {
				logger.Error("Failed background Elastic thread/message deletion for channel ID %s: %v", cID, err)
			}
		} else {
			if logger != nil {
				logger.Info("Completed background process: deleted channel threads and messages from Elastic for channel ID: %s", cID)
			}
		}
	}(channelID)

	if logger != nil {
		logger.Info("Completed DM channel deletion flow for channel ID: %s, user ID: %s", dm.ID, dm.UserId)
	}

	return nil
}



func (dm *DmChannels) UpsertGroupDescription(db *gorm.DB, req GroupDescriptionRequest) error {
	var dmChannel DmChannels

	exists := postgresql.CheckExists(db, &dmChannel, "channel_id = ? AND org_id = ?", req.ChannelId, req.OrgId)
	if !exists {
		return errors.New("channel does not exist")
	}

	if dmChannel.ChannelType != "group_dm" {
		return errors.New("group description can only be set for group DM channels")
	}

	var chanPart ChannelParticipant
	isParticipant := postgresql.CheckExists(db, &chanPart, "channel_id = ? AND user_id = ? AND deleted_at IS NULL", req.ChannelId, req.UserId)
	if !isParticipant {
		return errors.New("user is not a participant in this group DM")
	}

	result := db.Model(&DmChannels{}).
		Where("channel_id = ? AND org_id = ?", req.ChannelId, req.OrgId).
		Update("group_description", req.Description)

	if result.Error != nil {
		return fmt.Errorf("failed to update group description: %w", result.Error)
	}

	return nil
}

func (dm *DmChannels) GetDmChannelResponse(db *gorm.DB, c *gin.Context) (DmChannelsResponse, error) {
	var (
		user           User
		previewThread  = []Threads{}
		participants   = []Participant{}
		previewMessage string
	)

	threadCtx := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page=1&limit=50",
			},
		},
	}

	if c != nil && c.Request != nil {
		threadCtx.Request.Header = c.Request.Header
	}

	var thread Threads
	threads, _, _, threadErr := thread.GetAllThreadsByChannelID(threadCtx, db, dm.UserId, dm.ChannelId)
	if threadErr == nil && len(threads) > 0 {
		previewThread = threads
		previewMessage = BuildPreviewMessage(previewThread[0].Content, previewThread[0].Media)
	}

	switch dm.ChannelType {
	case "dm", "":
		userDetails, err := user.GetUserByID(db, *dm.ParticipantId, dm.OrgId)
		if err != nil {
			return DmChannelsResponse{}, err
		}

		participants = []Participant{NewParticipant(userDetails, false, "user")}

		return DmChannelsResponse{
			ID:               dm.ChannelId,
			Name:             participants[0].Username,
			AvatarUrl:        userDetails.Profile.AvatarURL,
			DefaultAvatarUrl: avatar.GenerateDefaultAvatarURL(userDetails.ID),
			ParticipantId:    *dm.ParticipantId,
			ParticipantEmail: userDetails.Email,
			ChannelType:      "dm",
			LastThreadId:     dm.LastThreadId,
			ThreadCount:      dm.ThreadCount,
			LastReadAt:       dm.LastReadAt,
			PreviewMessage:   previewMessage,
			PreviewThread:    previewThread,
			Participants:     participants,
			CreatedAt:        dm.CreatedAt,
		}, nil

	case "group_dm":
		type ParticipantWithProfile struct {
			UserId    string
			Title     string
			AvatarURL string
			UserName  string
			Email     string
			Online    bool
		}

		var participantsWithProfile []ParticipantWithProfile
		err := db.Table("channel_participants cp").
			Select(`
				cp.user_id,
				COALESCE(p.title, '') as title,
				COALESCE(p.avatar_url, '') as avatar_url,
				COALESCE(p.user_name, SPLIT_PART(u.email, '@', 1)) as user_name,
				u.email,
				p.online
			`).
			Joins("JOIN users u ON u.id = cp.user_id").
			Joins("LEFT JOIN profiles p ON p.userid = cp.user_id AND (p.organisation_id IS NULL OR p.organisation_id = ?)", dm.OrgId).
			Where("cp.channel_id = ? AND cp.deleted_at IS NULL", dm.ChannelId).
			Scan(&participantsWithProfile).Error

		if err != nil {
			return DmChannelsResponse{}, err
		}

		usernames := []string{}
		profilePic, defaultAvatar, email := "", "", ""

		for _, part := range participantsWithProfile {
			userDetails, err := user.GetUserByID(db, part.UserId, dm.OrgId)
			if err != nil {
				continue
			}

			p := NewParticipant(userDetails, part.UserId == dm.UserId, "user")
			usernames = append(usernames, p.Username)
			participants = append(participants, p)

			if profilePic == "" {
				profilePic = userDetails.Profile.AvatarURL
				defaultAvatar = avatar.GenerateDefaultAvatarURL(userDetails.ID)
				email = userDetails.Email
			}
		}

		sort.Strings(usernames)

		var chanParti ChannelParticipant
		postgresql.CheckExists(db, &chanParti, "channel_id = ? AND user_id = ?", dm.ChannelId, dm.UserId)

		return DmChannelsResponse{
			ID:               dm.ChannelId,
			Name:             strings.Join(usernames, ", "),
			AvatarUrl:        profilePic,
			DefaultAvatarUrl: defaultAvatar,
			ParticipantEmail: email,
			ChannelType:      "group_dm",
			LastThreadId:     chanParti.LastThreadId,
			ThreadCount:      chanParti.ThreadCount,
			LastReadAt:       chanParti.LastReadAt,
			PreviewMessage:   previewMessage,
			PreviewThread:    previewThread,
			Participants:     participants,
			CreatedAt:        dm.CreatedAt,
		}, nil
	}

	return DmChannelsResponse{}, errors.New("unsupported channel type")
}

func (dm *DmChannels) GetDmChannels(db *gorm.DB, c *gin.Context) ([]DmChannelsResponse, postgresql.PaginationResponse, error) {
	var (
		orderBy string
		order   string
		args    []any
	)

	dmchans := []DmChannels{}
	dmChansResp := []DmChannelsResponse{}
	recentDm := c.Query("recent_dm") == "true"
	search := c.Query("search") // this search enters username, firstname, lastname or part of email
	limit := 10

	pagination := postgresql.GetPagination(c)

	// Define the query string to fetch DmChannels where the user is an active participant
	queryString := `
        dm_channels.org_id = ? AND dm_channels.chat_type = ? AND dm_channels.deleted_at IS NULL
        AND (
            -- For DMs: user_id matches the logged-in user
        	((dm_channels.channel_type IS NULL OR dm_channels.channel_type = 'dm') AND dm_channels.user_id = ?)
            OR
            -- For Group DMs: user is in channel_participants
            (dm_channels.channel_type = 'group_dm' AND EXISTS (
                SELECT 1 FROM channel_participants 
                WHERE channel_participants.channel_id = dm_channels.channel_id 
                AND channel_participants.user_id = ? 
                AND channel_participants.deleted_at IS NULL
            ))
        )
    `

	args = []any{dm.OrgId, "user", dm.UserId, dm.UserId}

	if search != "" {
		searchTerm := "%" + search + "%"
		queryString += `
            AND EXISTS (
                SELECT 1 FROM users u
                LEFT JOIN profiles p ON p.userid = u.id AND (p.organisation_id IS NULL OR p.organisation_id = dm_channels.org_id)
                WHERE (
                    (u.id = dm_channels.participant_id)
                    OR
                    EXISTS (
                        SELECT 1 FROM channel_participants cp 
                        WHERE cp.channel_id = dm_channels.channel_id 
                        AND cp.user_id = u.id
                    )
                )
                AND u.id != ?
                AND (
                    u.email ILIKE ? OR 
                    u.name ILIKE ? OR 
                    p.user_name ILIKE ? OR 
                    p.first_name ILIKE ? OR 
                    p.last_name ILIKE ?
                )
            )
        `
		args = append(args, dm.UserId, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if recentDm {
		queryString += `
			AND dm_channels.interacted_at >= NOW() - INTERVAL '10 days'
		`
		orderBy = "interacted_at"
		order = "desc"

		// Override pagination to fetch top 10 most recent only
		pagination.Limit = limit
	} else {
		orderBy = "created_at"
		order = "desc"
	}

	paginationResp, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		orderBy,
		order,
		pagination,
		&dmchans,
		queryString,
		args...,
	)

	if err != nil {
		return nil, paginationResp, err
	}

	for _, dmchan := range dmchans {
		resp, err := dmchan.GetDmChannelResponse(db, c)
		if err != nil {
			return nil, paginationResp, err
		}

		if len(resp.PreviewThread) == 1 && resp.PreviewThread[0].Type == "system" {
			continue
		}

		dmChansResp = append(dmChansResp, resp)
	}

	slices.SortStableFunc(dmChansResp, func(a, b DmChannelsResponse) int {
		if a.ThreadCount != b.ThreadCount {
			return cmp.Compare(b.ThreadCount, a.ThreadCount)
		}
		if a.ThreadCount == 0 {
			if len(a.PreviewThread) > 0 && len(b.PreviewThread) > 0 {
				return b.PreviewThread[0].CreatedAt.Compare(a.PreviewThread[0].CreatedAt)
			}
			if len(a.PreviewThread) > 0 {
				return -1
			}
			if len(b.PreviewThread) > 0 {
				return 1
			}
			return b.CreatedAt.Compare(a.CreatedAt)
		}
		return 0
	})

	if len(dmChansResp) < 10 {
		topUsers, err := dm.GetTopNUsersResponse(db, 20)
		if err == nil {
			existingParticipantIDs := make(map[string]bool)
			for _, resp := range dmChansResp {
				existingParticipantIDs[resp.ParticipantId] = true
			}

			for _, topUser := range topUsers {
				if len(dmChansResp) >= 10 {
					break
				}
				if !existingParticipantIDs[topUser.ParticipantId] {
					dmChansResp = append(dmChansResp, topUser)
				}
			}

			paginationResp.TotalItems = int64(len(dmChansResp))
			paginationResp.TotalPagesCount = 1
			paginationResp.CurrentPage = 1
			paginationResp.PageCount = len(dmChansResp)
		}
	}

	return dmChansResp, paginationResp, nil
}

func (dm *DmChannels) GetVisibleDmChannels(db *gorm.DB, c *gin.Context) ([]DmChannelsResponse, postgresql.PaginationResponse, error) {
	var (
		orderBy string
		order   string
		args    []any
	)

	dmchans := []DmChannels{}
	dmChansResp := []DmChannelsResponse{}
	recentDm := c.Query("recent_dm") == "true"
	search := c.Query("search")
	limit := 10

	pagination := postgresql.GetPagination(c)

	queryString := `
        dm_channels.org_id = ? AND dm_channels.chat_type = ? AND dm_channels.deleted_at IS NULL
        AND (dm_channels.visibility_status IS NULL OR dm_channels.visibility_status = TRUE)
        AND (
            -- For DMs: user_id matches the logged-in user
        	((dm_channels.channel_type IS NULL OR dm_channels.channel_type = 'dm') AND dm_channels.user_id = ?)
            OR
            -- For Group DMs: user is in channel_participants
            (dm_channels.channel_type = 'group_dm' AND EXISTS (
                SELECT 1 FROM channel_participants 
                WHERE channel_participants.channel_id = dm_channels.channel_id 
                AND channel_participants.user_id = ? 
                AND channel_participants.deleted_at IS NULL
            ))
        )
    `

	args = []any{dm.OrgId, "user", dm.UserId, dm.UserId}

	if search != "" {
		searchTerm := "%" + search + "%"
		queryString += `
            AND EXISTS (
                SELECT 1 FROM users u
                LEFT JOIN profiles p ON p.userid = u.id AND (p.organisation_id IS NULL OR p.organisation_id = dm_channels.org_id)
                WHERE (
                    (u.id = dm_channels.participant_id)
                    OR
                    EXISTS (
                        SELECT 1 FROM channel_participants cp 
                        WHERE cp.channel_id = dm_channels.channel_id 
                        AND cp.user_id = u.id
                    )
                )
                AND u.id != ?
                AND (
                    u.email ILIKE ? OR 
                    u.name ILIKE ? OR 
                    p.user_name ILIKE ? OR 
                    p.first_name ILIKE ? OR 
                    p.last_name ILIKE ?
                )
            )
        `
		args = append(args, dm.UserId, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if recentDm {
		queryString += `
			AND dm_channels.interacted_at >= NOW() - INTERVAL '10 days'
		`
		orderBy = "interacted_at"
		order = "desc"
		pagination.Limit = limit
	} else {
		orderBy = "created_at"
		order = "desc"
	}

	paginationResp, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		orderBy,
		order,
		pagination,
		&dmchans,
		queryString,
		args...,
	)

	if err != nil {
		return nil, paginationResp, err
	}

	for _, dmchan := range dmchans {
		resp, err := dmchan.GetDmChannelResponse(db, c)
		if err != nil {
			continue
		}
		dmChansResp = append(dmChansResp, resp)
	}


	return dmChansResp, paginationResp, nil
}

func (dm *DmChannels) UpdateDmVisibility(db *gorm.DB, req UpdateDmVisibilityRequest) error {
	var count int64
	db.Model(&DmChannels{}).Where("channel_id = ? AND user_id = ?", req.ChannelId, req.UserId).Count(&count)
	if count == 0 {
		return errors.New("DM channel not found for user")
	}

	result := db.Model(&DmChannels{}).
		Where("channel_id = ? AND user_id = ?", req.ChannelId, req.UserId).
		Update("visibility_status", *req.VisibilityStatus)

	if result.Error != nil {
		return result.Error
	}

	return nil
}


func (dm *DmChannels) GetTopNUsersResponse(db *gorm.DB, limit int) ([]DmChannelsResponse, error) {
	var (
		users       []User
		dmChansResp []DmChannelsResponse
	)
	err := db.Table("users").
		Joins("JOIN user_organisations ON user_organisations.user_id = users.id").
		Preload("Profile").
		Where("user_organisations.organisation_id = ? AND users.id != ?", dm.OrgId, dm.UserId).
		Order("users.created_at DESC").
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		participant := NewParticipant(u, false, "user")
		dmChansResp = append(dmChansResp, DmChannelsResponse{
			Name:             participant.Username,
			AvatarUrl:        u.Profile.AvatarURL,
			DefaultAvatarUrl: avatar.GenerateDefaultAvatarURL(u.ID),
			ParticipantId:    u.ID,
			ParticipantEmail: u.Email,
			ChannelType:      "dm",
			IsSuggested:      true,
			Participants:     []Participant{participant},
			PreviewMessage:   "Chat with this user",
			LastReadAt:       time.Now(),
			CreatedAt:        time.Now(),
		})
	}
	return dmChansResp, nil
}

func (r *DmChannels) CheckChannelExists(db *gorm.DB, channelID, userId string) (bool, error) {

	exists := postgresql.CheckExists(db, &r, "channel_id = ?", channelID)

	if !exists {
		return exists, errors.New("channel does not exist")
	}

	if r.ChannelType == "dm" && userId != "" {
		dmChan := DmChannels{}
		exists := postgresql.CheckExists(db, &dmChan, "channel_id = ? AND user_id = ?", channelID, userId)
		if !exists {
			return exists, errors.New("channel does not exist")
		}
		*r = dmChan
	}

	return exists, nil
}

func (r *DmChannels) FetchChannelParticipant(db *gorm.DB, req DmChannelsRequest) (bool, error) {

	exists := postgresql.CheckExists(db, &r, "channel_id = ? AND user_id = ?", req.ChannelId, req.UserId)
	if !exists {
		exists = postgresql.CheckExists(db, &r, "channel_id = ? AND channel_type = ?", req.ChannelId, "group_dm")
	}

	if !exists {
		return exists, errors.New("channel does not exist")
	}

	return exists, nil
}

func (r *DmChannels) FetchDmChannelInfo(db *gorm.DB) (DmChannelsResponse, error) {

	chanInfo := map[string]func() (DmChannelsResponse, error){
		"dm": func() (DmChannelsResponse, error) {
			res := DmChannelsResponse{}
			var (
				user   User
				dmChan DmChannels
			)

			exists := postgresql.CheckExists(db, &dmChan, "channel_id = ? AND user_id = ?", r.ChannelId, r.UserId)
			if !exists {
				return res, errors.New("channel does not exist")
			}

			userDetails, err := user.GetUserByID(db, *dmChan.ParticipantId, r.OrgId)
			if err != nil {
				return res, err
			}

			participant := NewParticipant(userDetails, false, "user")

			res = DmChannelsResponse{
				ID:               r.ChannelId,
				Name:             participant.Username,
				AvatarUrl:        userDetails.Profile.AvatarURL,
				DefaultAvatarUrl: avatar.GenerateDefaultAvatarURL(userDetails.ID),
				ParticipantId:    *dmChan.ParticipantId,
				ParticipantEmail: userDetails.Email,
				ChannelType:      "dm",
				LastThreadId:     dmChan.LastThreadId,
				ThreadCount:      dmChan.ThreadCount,
				LastReadAt:       dmChan.LastReadAt,
				Participants:     []Participant{participant},
			}

			return res, nil

		},
		"group_dm": func() (DmChannelsResponse, error) {
			res := DmChannelsResponse{}
			var (
				chanPart []ChannelParticipant
				user     User
			)

			err := postgresql.SelectAllFromDb(db, "", &chanPart, "channel_id = ?", r.ChannelId)
			if err != nil {
				return res, fmt.Errorf("failed to get participants for group DM channel %s", r.ChannelId)
			}

			usernames := []string{}
			profilePic := ""
			defaultAvatar := ""
			email := ""
			participants := []Participant{}

			for _, part := range chanPart {
				userDetails, err := user.GetUserByID(db, part.UserId, r.OrgId)
				if err != nil {
					return res, err
				}

				p := NewParticipant(userDetails, part.UserId == r.UserId, "user")
				usernames = append(usernames, p.Username)

				participants = append(participants, p)

				if profilePic == "" {
					profilePic = userDetails.Profile.AvatarURL
				}

				if defaultAvatar == "" {
					defaultAvatar = avatar.GenerateDefaultAvatarURL(userDetails.ID)
				}

				if email == "" {
					email = userDetails.Email
				}
			}

			sort.Strings(usernames)

			var chanParti ChannelParticipant

			exist := postgresql.CheckExists(db, &chanParti, "channel_id = ? AND user_id = ?", r.ChannelId, r.UserId)
			if !exist {
				return res, fmt.Errorf("user not found in channel")
			}

			res = DmChannelsResponse{
				ID:               r.ChannelId,
				Name:             strings.Join(usernames, ", "),
				AvatarUrl:        profilePic,
				DefaultAvatarUrl: defaultAvatar,
				ParticipantEmail: email,
				ChannelType:      "group_dm",
				LastThreadId:     chanParti.LastThreadId,
				ThreadCount:      chanParti.ThreadCount,
				LastReadAt:       chanParti.LastReadAt,
				Participants:     participants,
			}

			return res, nil
		},
	}

	return chanInfo[r.ChannelType]()
}

func (r *DmChannels) GetUserChannelsUnreadThread(base *storage.Database) ([]DmChannelsResponse, error) {

	var (
		chanResp []DmChannelsResponse
		db       = base.Postgresql
	)

	chanInfo := map[string]func() ([]DmChannelsResponse, error){

		"dm": func() ([]DmChannelsResponse, error) {
			var user User
			previewThread := []Threads{}

			if err := db.Model(&DmChannels{}).
				Select("dm_channels.*, dm_channels.channel_id as id, COALESCE(p.user_name, SPLIT_PART(u.email, '@', 1)) as name, p.avatar_url as avatar_url, u.email as participant_email").
				Joins("JOIN profiles AS p ON p.userid = dm_channels.user_id AND (p.organisation_id IS NULL OR p.organisation_id = dm_channels.org_id)").
				Joins("JOIN users AS u ON u.id = dm_channels.user_id").
				Where("dm_channels.channel_id = ? AND dm_channels.user_id != ?", r.ChannelId, r.UserId).
				Order("dm_channels.created_at").
				Omit("preview_thread").
				Scan(&chanResp).Error; err != nil {
				return nil, errors.New("error fetching channels")
			}

			threadCtx := &gin.Context{
				Request: &http.Request{
					URL: &url.URL{
						RawQuery: "page=1&limit=50",
					},
				},
			}

			var thread Threads
			threads, _, _, threadErr := thread.GetAllThreadsByChannelID(threadCtx, db, r.UserId, r.ChannelId)
			if threadErr == nil && len(threads) > 0 {
				previewThread = threads
			}

			for i := range chanResp {
				userDetails, err := user.GetUserByID(db, chanResp[i].UserId, r.OrgId)
				if err != nil {
					continue
				}

				participants := []Participant{NewParticipant(userDetails, false, "user")}

				previewMessage := ""
				if len(previewThread) > 0 {
					previewMessage = BuildPreviewMessage(previewThread[0].Content, previewThread[0].Media)
				}

				chanResp[i].PreviewMessage = previewMessage
				chanResp[i].PreviewThread = previewThread
				chanResp[i].Participants = participants
				chanResp[i].DefaultAvatarUrl = avatar.GenerateDefaultAvatarURL(userDetails.ID)
			}

			return chanResp, nil
		},

		"group_dm": func() ([]DmChannelsResponse, error) {
			var chanResp []DmChannelsResponse
			var user User
			previewThread := []Threads{}

			var userProfiles []struct {
				UserName  string
				AvatarURL string
				UserId    string
			}

			profJoinCond := "LEFT JOIN profiles p ON p.userid = u.id"
			var profJoinArgs []interface{}
			if r.OrgId != "" {
				profJoinCond += " AND (p.organisation_id IS NULL OR p.organisation_id = ?)"
				profJoinArgs = append(profJoinArgs, r.OrgId)
			}

			if err := db.Table("channel_participants AS cp").
				Select("COALESCE(p.user_name, SPLIT_PART(u.email, '@', 1)) AS user_name, p.avatar_url, u.id as user_id").
				Joins("JOIN users u ON u.id = cp.user_id").
				Joins(profJoinCond, profJoinArgs...).
				Where("cp.channel_id = ?", r.ChannelId).
				Scan(&userProfiles).Error; err != nil {
				return nil, errors.New("error fetching participant profiles")
			}

			usernames := []string{}
			profilePic := ""
			defaultAvatar := ""
			participants := []Participant{}
			for _, prof := range userProfiles {
				usernames = append(usernames, prof.UserName)
				if profilePic == "" {
					profilePic = prof.AvatarURL
				}

				userDetails, err := user.GetUserByID(db, prof.UserId, r.OrgId)
				if err == nil {
					if defaultAvatar == "" {
						defaultAvatar = avatar.GenerateDefaultAvatarURL(userDetails.ID)
					}

					participants = append(participants, NewParticipant(userDetails, false, "user"))
				}
			}
			sort.Strings(usernames)
			userNames := strings.Join(usernames, ", ")

			if err := db.
				Table("channel_participants AS cp").
				Select(`
			cp.channel_id AS id,
			? AS name,
			? AS avatar_url,
			u.email AS participant_email,
			cp.last_thread_id,
			cp.thread_count,
			cp.last_read_at,
			cp.user_id,
			'group_dm' AS channel_type
		`, userNames, profilePic).
				Joins("JOIN users u ON u.id = cp.user_id").
				Where("cp.channel_id = ? AND cp.user_id != ?", r.ChannelId, r.UserId).
				Order("cp.created_at").
				Omit("PreviewThread").
				Scan(&chanResp).Error; err != nil {
				return nil, fmt.Errorf("error fetching participants for channel %s: %w", r.ChannelId, err)
			}

			threadCtx := &gin.Context{
				Request: &http.Request{
					URL: &url.URL{
						RawQuery: "page=1&limit=50",
					},
				},
			}

			var thread Threads
			threads, _, _, threadErr := thread.GetAllThreadsByChannelID(threadCtx, db, r.UserId, r.ChannelId)
			if threadErr == nil && len(threads) > 0 {
				previewThread = threads
			}

			for i := range chanResp {
				previewMessage := ""
				if len(previewThread) > 0 {
					previewMessage = BuildPreviewMessage(previewThread[0].Content, previewThread[0].Media)
				}

				chanResp[i].PreviewMessage = previewMessage
				chanResp[i].PreviewThread = previewThread
				chanResp[i].Participants = participants
				chanResp[i].DefaultAvatarUrl = defaultAvatar
			}

			return chanResp, nil
		},
	}

	return chanInfo[r.ChannelType]()
}

func (c *DmChannels) UpdateLastRead(db *gorm.DB, req UpdateLastRead, mu *sync.Mutex, logger *utility.Logger) bool {

	mu.Lock()
	defer mu.Unlock()

	updateMap := map[string]func() bool{
		"dm": func() bool {
			var uc DmChannels

			query := "channel_id = ? AND user_id = ?"

			exists := postgresql.CheckExists(db, &uc, query, c.ChannelId, c.UserId)

			if exists && uc.LastThreadId == req.LastThreadId {
				return false
			}

			updateFields := map[string]any{
				"last_thread_id": req.LastThreadId,
				"last_read_at":   req.LastReadAt,
				"thread_count":   0,
			}

			result := db.Model(&DmChannels{}).
				Where(query, c.ChannelId, c.UserId).
				Updates(updateFields)

			if result.Error != nil {
				logger.Error("an error occurend while updating user last read: %v", result.Error)
				return false
			}

			logger.Info("user dm last read updated successfully")
			return true
		},
		"group_dm": func() bool {
			var uc ChannelParticipant

			query := "channel_id = ? AND user_id = ?"

			exists := postgresql.CheckExists(db, &uc, query, c.ChannelId, c.UserId)

			if exists && uc.LastThreadId == req.LastThreadId {
				return false
			}

			updateFields := map[string]any{
				"last_thread_id": req.LastThreadId,
				"last_read_at":   req.LastReadAt,
				"thread_count":   0,
			}

			result := db.Model(&ChannelParticipant{}).
				Where(query, c.ChannelId, c.UserId).
				Updates(updateFields)

			if result.Error != nil {
				logger.Error("an error occurend while updating user last read: %v", result.Error)
				return false
			}

			logger.Info("user group dm last read updated successfully")
			return true
		},
	}
	return updateMap[c.ChannelType]()
}

func (r *DmChannels) UpdateUnReadCount(db *gorm.DB, mu *sync.Mutex, logger *utility.Logger) {

	mu.Lock()
	defer mu.Unlock()

	updateFunc := map[string]func(){

		"dm": func() {
			query := "channel_id = ? AND user_id != ?"

			updateFields := map[string]any{
				"thread_count": gorm.Expr("thread_count + 1"),
			}

			result := db.Model(&DmChannels{}).
				Where(query, r.ChannelId, r.UserId).
				Updates(updateFields)

			if result.Error != nil {
				logger.Error("an error occurred while updating dm channel counts: %v", result.Error)
				return
			}

			logger.Info("user channels counts updated successfully")
		},
		"group_dm": func() {
			query := "channel_id = ? AND user_id != ?"

			updateFields := map[string]any{
				"thread_count": gorm.Expr("thread_count + 1"),
			}

			result := db.Model(&ChannelParticipant{}).
				Where(query, r.ChannelId, r.UserId).
				Updates(updateFields)

			if result.Error != nil {
				logger.Error("an error occurred while updating group dm channel counts: %v", result.Error)
				return
			}

			logger.Info("user channels counts updated successfully")

		},
	}

	updateFunc[r.ChannelType]()
}

func (c *DmChannels) SendChannelUnReadUpdate(mu *sync.Mutex, logger *utility.Logger, updateType UnReadUpdate) {

	mu.Lock()
	defer mu.Unlock()

	if updateType == Read {
		dmResp, err := c.FetchDmChannelInfo(storage.DB.Postgresql)

		if err != nil {
			logger.Error("Error updating dm Section: %v, with userId: %s, channelId: %s", err, c.UserId, c.ChannelId)
			return
		}

		notification := Notification[UnReadThreadChange]
		notification.SectionType = DmChannelsSection
		notification.Content = dmResp
		notification.NotificationId = utility.GenerateUUID()

		err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, c.UserId), notification)
		if err != nil {
			logger.Error("Error Publishing to channelid: %s, with userid: %s error: %v", c.ChannelId, c.UserId, err.Error())
			return
		}
	}

	if updateType == NewThread {

		if c.ChatType == "bot" {
			db := storage.DB.Postgresql
			var agentResp = AgentResp{}

			err := db.Table("organisation_integrations oi").
				Select(`
	                	oi.integration_id as id,
	                	oi.is_active,
	                	oi.app_name as name,
	                	oi.tone,
	                	oi.app_logo as avatar,
	                	oi.title,
	                	oi.app_description as description,
	                	oi.visibility,
	                	oi.stars,
	                	COALESCE(dc.thread_count, 0) as thread_count,
	                	COALESCE(dc.last_thread_id, '') as last_thread_id,
	                	COALESCE(dc.last_read_at, '0001-01-01'::timestamp) as last_read_at,
						dc.user_id
	                `).
				Joins(`
	                	LEFT JOIN dm_channels dc
	                	ON dc.participant_id = oi.integration_id
	                	AND dc.channel_id = ?
	                `, c.ChannelId).
				Where("oi.org_id = ? AND oi.integration_id = ?", c.OrgId, c.ParticipantId).
				First(&agentResp).Error

			if err != nil {
				logger.Error("Failed to send update count for bot channel, error %v: ", err)
			}

			notification := Notification[UnReadThreadChange]
			notification.SectionType = AgentChannelsSection
			notification.Content = agentResp
			notification.NotificationId = utility.GenerateUUID()

			err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, agentResp.UserId), notification)
			if err != nil {
				logger.Error(fmt.Sprintf("Error Publishing to bot dm with channelid: %s, with userid: %s error: %v", c.ChannelId, agentResp.UserId, err.Error()))
				return
			}

			logger.Info("user last read sent successfully")

			return
		}

		res, err := c.GetUserChannelsUnreadThread(storage.DB)

		if err != nil {
			logger.Error("Bulk update failed: %v", err)
			return
		}

		if len(res) < 1 {
			logger.Error("Empty channel response")
			return
		}

		for _, update := range res {
			notification := Notification[UnReadThreadChange]
			notification.SectionType = DmChannelsSection
			notification.Content = update
			notification.NotificationId = utility.GenerateUUID()

			err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, update.UserId), notification)
			if err != nil {
				logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, with userid: %s error: %v", c.ChannelId, update.UserId, err.Error()))
				return
			}
		}
	}

	logger.Info("user last read sent successfully")
}

func (r *DmChannels) UpdateInteractionAt(db *gorm.DB) error {
	result := db.Model(&DmChannels{}).
		Where("channel_id = ?", r.ChannelId).
		Updates(map[string]any{
			"interacted_at":     time.Now(),
			"visibility_status": true,
		})


	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("no channel found with the given channelId")
	}

	return nil
}

func (r *DmChannels) GetLastMessageByChannelId(db *gorm.DB, channelId string) string {
	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"channels_id": channelId,
			},
		},
		"size": 1,
		"sort": []map[string]any{
			{
				"created_at": map[string]any{
					"order": "desc",
				},
			},
		},
	}

	var threadData any
	err := elastic.SelectAll(storage.DB.Elastic, ThreadIndexName, query, &threadData)
	if err != nil {
		return ""
	}

	threadDataMap, ok := threadData.(map[string]any)
	if !ok {
		return ""
	}

	hits, ok := threadDataMap["hits"].(map[string]any)
	if !ok {
		return ""
	}

	hitsArray, ok := hits["hits"].([]any)
	if !ok || len(hitsArray) == 0 {
		return ""
	}

	firstHit, ok := hitsArray[0].(map[string]any)
	if !ok {
		return ""
	}

	source, ok := firstHit["_source"].(map[string]any)
	if !ok {
		return ""
	}

	content, ok := source["message"].(string)
	if !ok {
		return ""
	}

	return content
}

// GetChannelMedia fetches all media files from threads in a DM channel
func (dm *DmChannels) GetChannelMedia(db *storage.Database, c *gin.Context, mediaType string) ([]File, postgresql.PaginationResponse, error) {
	pagination := postgresql.GetPagination(c)

	// Build the base query for threads in this channel that have media
	// Note: media is not a nested type in the index, so we use a simple exists query
	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id": dm.ChannelId,
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
		"size":    1000, // Get all threads with media first
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
		return nil, postgresql.PaginationResponse{}, fmt.Errorf("failed to fetch media: %w", err)
	}

	// Parse the response and collect all media files
	threadDataMap, ok := threadData.(map[string]any)
	if !ok {
		return []File{}, postgresql.PaginationResponse{}, nil
	}

	hits, ok := threadDataMap["hits"].(map[string]any)
	if !ok {
		return []File{}, postgresql.PaginationResponse{}, nil
	}

	hitsArray, ok := hits["hits"].([]any)
	if !ok || len(hitsArray) == 0 {
		return []File{}, postgresql.PaginationResponse{}, nil
	}

	// Collect all media from all threads
	var allMedia []File
	for _, hit := range hitsArray {
		hitMap, ok := hit.(map[string]any)
		if !ok {
			continue
		}

		source, ok := hitMap["_source"].(map[string]any)
		if !ok {
			continue
		}

		// Get thread-level metadata
		threadUserID := utility.GetString(source, "user_id")
		threadCreatedAt := utility.GetString(source, "created_at")

		mediaArray, ok := source["media"].([]any)
		if !ok {
			continue
		}

		for _, m := range mediaArray {
			mediaMap, ok := m.(map[string]any)
			if !ok {
				continue
			}

			// Parse created_at time
			var createdAt time.Time
			if threadCreatedAt != "" {
				createdAt, _ = time.Parse(time.RFC3339, threadCreatedAt)
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

			allMedia = append(allMedia, file)
		}
	}

	// Apply pagination
	totalItems := len(allMedia)
	start := (pagination.Page - 1) * pagination.Limit
	end := start + pagination.Limit

	if start > totalItems {
		start = totalItems
	}
	if end > totalItems {
		end = totalItems
	}

	paginatedMedia := allMedia[start:end]

	totalPages := (totalItems + pagination.Limit - 1) / pagination.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	pagResp := postgresql.PaginationResponse{
		CurrentPage:     pagination.Page,
		TotalPagesCount: totalPages,
		PageCount:       len(paginatedMedia),
		TotalItems:      int64(totalItems),
	}

	return paginatedMedia, pagResp, nil
}

// GetPreviewMedia fetches the N most recent media files from threads in a DM channel
func (dm *DmChannels) GetPreviewMedia(db *storage.Database, limit int) ([]FileMediaResponse, int, error) {
	if limit <= 0 {
		limit = 10
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id": dm.ChannelId,
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

func (dm *DmChannels) GetDMsWithUnread(db *gorm.DB, userID string) (map[string]time.Time, error) {
	channelMap := make(map[string]time.Time)

	var dmResults []struct {
		ChannelId  string
		LastReadAt time.Time
	}

	err := db.Model(&DmChannels{}).
		Select("channel_id, last_read_at").
		Where("user_id = ? AND thread_count > 0", userID).
		Scan(&dmResults).Error
	if err != nil {
		return nil, err
	}
	for _, r := range dmResults {
		channelMap[r.ChannelId] = r.LastReadAt
	}

	// fetch group dms
	var groupResults []struct {
		ChannelId  string
		LastReadAt time.Time
	}
	err = db.Model(&ChannelParticipant{}).
		Select("channel_id, last_read_at").
		Where("user_id = ? AND thread_count > 0", userID).
		Scan(&groupResults).Error
	if err != nil {
		return nil, err
	}
	for _, r := range groupResults {
		channelMap[r.ChannelId] = r.LastReadAt
	}

	return channelMap, nil
}
