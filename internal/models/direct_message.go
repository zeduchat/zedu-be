package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
)

type DmChannels struct {
	ID              string         `gorm:"type:uuid" json:"id"`
	UserId          string         `gorm:"type:uuid" json:"-"`
	ChannelId       string         `gorm:"type:uuid" json:"channel_id"`
	OrgId           string         `gorm:"type:uuid" json:"-"`
	ParticipantId   *string        `gorm:"type:uuid" json:"-"`
	ParticipantHash string         `gorm:"type:string" json:"participant_hash"`
	ChatType        string         `gorm:"type:string;default:user" json:"chat_type"`  // user or bot
	ChannelType     string         `gorm:"type:string;default:dm" json:"channel_type"` // dm or group_dm
	CreatedAt       time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt       time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	ThreadCount     int64          `gorm:"column:thread_count;default:0" json:"thread_count"`
	LastThreadId    string         `gorm:"column:last_thread_id" json:"last_thread_id"`
	LastReadAt      time.Time      `gorm:"column:last_read_at" json:"last_read_at"`
	InteractedAt    time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"-"`
}

type DmChannelsResponse struct {
	ID               string    `json:"channel_id"`
	Name             string    `json:"username"`
	ParticipantId    string    `json:"participant_id"`
	AvatarUrl        string    `json:"avatar_url"`
	ParticipantEmail string    `json:"participant_email"`
	ChannelType      string    `json:"channel_type"`
	ThreadCount      int64     `json:"thread_count"`
	LastThreadId     string    `json:"last_thread_id"`
	LastReadAt       time.Time `json:"last_read_at"`
	UserId           string    `json:"-"`
}

type DmChannelsRequest struct {
	ChatType      string `json:"chat_type" validate:"required,oneof=user bot"`
	ParticipantId string `json:"participant_id" validate:"required"`
	UserId        string `json:"user_id"`
	OrgId         string `json:"org_id"`
	ChannelId     string `json:"channel_id"`
}

func FetchDetailsFromAgentJSON(extReq request.ExternalRequest, agent OrganisationIntegrations, redisClient *redis.Client) (map[string]any, error) {
	var response any
	var data_r map[string]any
	agentJSONURL := agent.JSONUrl
	redisKey := fmt.Sprintf("agent_json_%s", agentJSONURL)

	if agent.AppName == "" {

		cachedData, err := rd.RedisGet(redisClient, redisKey)
		if err == nil && len(cachedData) > 0 {
			var cachedResult any

			if err := json.Unmarshal(cachedData, &cachedResult); err != nil {
				rd.RedisDelete(redisClient, redisKey)
				return nil, fmt.Errorf("failed to unmarshal cached data: %v", err)
			}

			data_r, ok := cachedResult.(map[string]any)
			if !ok {
				rd.RedisDelete(redisClient, redisKey)
				return nil, errors.New("cached data is not in the expected format")
			}

			return data_r, nil
		}

		data := map[string]string{"url": agentJSONURL}

		for i := 0; i < 2; i++ {
			response, err = extReq.SendExternalRequest(request.AgentJsonContent, data)
			if err == nil {
				break
			}
			time.Sleep(time.Duration(2<<i) * time.Second) // exponential backoff
		}
		if err != nil {
			return nil, fmt.Errorf("could not fetch agent json: %v", err)
		}

		response_data := response.(map[string]any)
		content, ok := response_data["data"].(map[string]any)
		if !ok {
			return nil, errors.New("could not fetch data from agent json")
		}

		data_r, ok := content["descriptions"].(map[string]any)
		if !ok {
			return nil, errors.New("invalid agent details format")
		}

		data_r["bot"] = content["bot"]

		err = ValidateAgentData(data_r)
		if err != nil {
			return nil, fmt.Errorf("invalid agent json data: %v", err)
		}
	} else {

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
	}

	rd.RedisSet(redisClient, redisKey, data_r, 12*time.Hour)

	return data_r, nil
}

func buildDmResponse(dm *DmChannels, appName, appLogo string) DmChannelsResponse {
	return DmChannelsResponse{
		ID:               dm.ChannelId,
		ParticipantId:    *dm.ParticipantId,
		ParticipantEmail: appName,
		AvatarUrl:        appLogo,
		Name:             appName,
		LastThreadId:     dm.LastThreadId,
		ThreadCount:      dm.ThreadCount,
		LastReadAt:       dm.LastReadAt,
	}
}

func (dm *DmChannels) CreateAgentDMChannel(extReq request.ExternalRequest, db *gorm.DB, rds *redis.Client) (DmChannelsResponse, error) {
	var orgAgent OrganisationIntegrations
	if !postgresql.CheckExists(db, &orgAgent, "org_id = ? AND integration_id = ?", dm.OrgId, dm.ParticipantId) {
		return DmChannelsResponse{}, fmt.Errorf("agent participant does not exist in organisation %v", dm.OrgId)
	}

	agentDetails, err := FetchDetailsFromAgentJSON(extReq, orgAgent, rds)
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

	if userDetails.Profile.UserName == "" {
		userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
	}

	exists := postgresql.CheckExists(db, &existDmchan, "user_id = ? AND participant_id = ? AND org_id = ?", dm.UserId, *dm.ParticipantId, dm.OrgId)
	if exists {
		dmchanresp.AvatarUrl = userDetails.Profile.AvatarURL
		dmchanresp.Name = userDetails.Profile.UserName
		dmchanresp.ID = existDmchan.ChannelId
		dmchanresp.ParticipantId = *dm.ParticipantId
		dmchanresp.ParticipantEmail = userDetails.Email

		return dmchanresp, nil
	}

	err = postgresql.CreateOneRecord(db, &dm)
	if err != nil {
		return dmchanresp, err
	}

	dmchanresp.AvatarUrl = userDetails.Profile.AvatarURL
	dmchanresp.Name = userDetails.Profile.UserName
	dmchanresp.ID = dm.ChannelId
	dmchanresp.ParticipantId = *dm.ParticipantId
	dmchanresp.ParticipantEmail = userDetails.Email

	return dmchanresp, nil
}

func (dm *DmChannels) DeleteDmChannel(db *gorm.DB) error {

	var (
		user         User
		savedMessage SavedMessage
	)

	_, err := user.GetUserByID(db, dm.UserId)
	if err != nil {
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
		return err
	}

	if err := savedMessage.DeleteSavedMessagesByChannelID(db, dm.ID, dm.UserId); err != nil {
		return err
	}

	return nil
}

func (dm *DmChannels) GetDmChannels(db *gorm.DB, c *gin.Context) ([]DmChannelsResponse, postgresql.PaginationResponse, error) {
	var (
		user     User
		chanPart []ChannelParticipant
		orderBy  string
		order    string
		args     []any
	)

	dmchans := []DmChannels{}
	dmChansResp := []DmChannelsResponse{}
	recentDm := c.Query("recent_dm") == "true"
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

	if recentDm {
		queryString += `
			AND dm_channels.interacted_at >= NOW() - INTERVAL '10 days'
		`
		orderBy = "interacted_at"
		order = "desc"
		args = []any{dm.OrgId, "user", dm.UserId, dm.UserId}

		// Override pagination to fetch top 10 most recent only
		pagination.Limit = limit
	} else {
		orderBy = "created_at"
		order = "desc"
		args = []any{dm.OrgId, "user", dm.UserId, dm.UserId}
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
		if dmchan.ChannelType == "dm" || dmchan.ChannelType == "" {
			userDetails, err := user.GetUserByID(db, *dmchan.ParticipantId)
			if err != nil {
				return nil, paginationResp, err
			}

			if userDetails.Profile.UserName == "" {
				userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
			}

			dmChansResp = append(dmChansResp, DmChannelsResponse{
				ID:               dmchan.ChannelId,
				Name:             userDetails.Profile.UserName,
				AvatarUrl:        userDetails.Profile.AvatarURL,
				ParticipantId:    *dmchan.ParticipantId,
				ParticipantEmail: userDetails.Email,
				ChannelType:      "dm",
				LastThreadId:     dmchan.LastThreadId,
				ThreadCount:      dmchan.ThreadCount,
				LastReadAt:       dmchan.LastReadAt,
			})
		} else if dmchan.ChannelType == "group_dm" {

			err = postgresql.SelectAllFromDb(db, "", &chanPart, "channel_id = ?", dmchan.ChannelId)
			if err != nil {
				return nil, paginationResp, fmt.Errorf("failed to get participants for group DM channel %s", dmchan.ChannelId)
			}

			usernames := []string{}
			profilePic := ""
			email := ""

			for _, part := range chanPart {
				userDetails, err := user.GetUserByID(db, part.UserId)
				if err != nil {
					return nil, paginationResp, err
				}

				if userDetails.Profile.UserName == "" {
					userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
				}

				usernames = append(usernames, userDetails.Profile.UserName)

				if profilePic == "" {
					profilePic = userDetails.Profile.AvatarURL
				}

				if email == "" {
					email = userDetails.Email
				}
			}

			sort.Strings(usernames)

			var chanParti ChannelParticipant

			exist := postgresql.CheckExists(db, &chanParti, "channel_id = ? AND user_id = ?", dmchan.ChannelId, dm.UserId)
			if !exist {
				return nil, paginationResp, fmt.Errorf("user not found in channel")
			}

			dmChansResp = append(dmChansResp, DmChannelsResponse{
				ID:               dmchan.ChannelId,
				Name:             strings.Join(usernames, ", "),
				AvatarUrl:        profilePic,
				ParticipantEmail: email,
				ChannelType:      "group_dm",
				LastThreadId:     chanParti.LastThreadId,
				ThreadCount:      chanParti.ThreadCount,
				LastReadAt:       chanParti.LastReadAt,
			})

		}

		slices.SortFunc(dmChansResp, func(a, b DmChannelsResponse) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})

	}

	return dmChansResp, paginationResp, nil
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

			userDetails, err := user.GetUserByID(db, *dmChan.ParticipantId)
			if err != nil {
				return res, err
			}

			if userDetails.Profile.UserName == "" {
				userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
			}

			res = DmChannelsResponse{
				ID:               r.ChannelId,
				Name:             userDetails.Profile.UserName,
				AvatarUrl:        userDetails.Profile.AvatarURL,
				ParticipantId:    *dmChan.ParticipantId,
				ParticipantEmail: userDetails.Email,
				ChannelType:      "dm",
				LastThreadId:     dmChan.LastThreadId,
				ThreadCount:      dmChan.ThreadCount,
				LastReadAt:       dmChan.LastReadAt,
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
			email := ""

			for _, part := range chanPart {
				userDetails, err := user.GetUserByID(db, part.UserId)
				if err != nil {
					return res, err
				}

				if userDetails.Profile.UserName == "" {
					userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
				}

				usernames = append(usernames, userDetails.Profile.UserName)

				if profilePic == "" {
					profilePic = userDetails.Profile.AvatarURL
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
				ParticipantEmail: email,
				ChannelType:      "group_dm",
				LastThreadId:     chanParti.LastThreadId,
				ThreadCount:      chanParti.ThreadCount,
				LastReadAt:       chanParti.LastReadAt,
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
			if err := db.Model(&DmChannels{}).
				Select("dm_channels.*, dm_channels.channel_id as id, COALESCE(p.user_name, SPLIT_PART(u.email, '@', 1)) as name, p.avatar_url as avatar_url, u.email as participant_email").
				Joins("JOIN profiles AS p ON p.userid = dm_channels.user_id").
				Joins("JOIN users AS u ON u.id = dm_channels.user_id").
				Where("dm_channels.channel_id = ? AND dm_channels.user_id != ?", r.ChannelId, r.UserId).
				Order("dm_channels.created_at").
				Scan(&chanResp).Error; err != nil {
				return nil, errors.New("error fetching channels")
			}

			return chanResp, nil
		},

		"group_dm": func() ([]DmChannelsResponse, error) {
			var chanResp []DmChannelsResponse

			var userProfiles []struct {
				UserName  string
				AvatarURL string
			}

			if err := db.Table("channel_participants AS cp").
				Select("COALESCE(p.user_name, SPLIT_PART(u.email, '@', 1)) AS user_name, p.avatar_url").
				Joins("JOIN users u ON u.id = cp.user_id").
				Joins("LEFT JOIN profiles p ON p.userid = u.id").
				Where("cp.channel_id = ?", r.ChannelId).
				Scan(&userProfiles).Error; err != nil {
				return nil, errors.New("error fetching participant profiles")
			}

			usernames := []string{}
			profilePic := ""
			for _, prof := range userProfiles {
				usernames = append(usernames, prof.UserName)
				if profilePic == "" {
					profilePic = prof.AvatarURL
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
				Scan(&chanResp).Error; err != nil {
				return nil, fmt.Errorf("error fetching participants for channel %s: %w", r.ChannelId, err)
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

		err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, c.UserId), notification)
		if err != nil {
			logger.Error("Error Publishing to channelid: %s, with userid: %s error: %v", c.ChannelId, c.UserId, err.Error())
			return
		}
	}

	if updateType == NewThread {

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

			err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, update.UserId), notification)
			if err != nil {
				logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, with userid: %s error: %v", c.ChannelId, update.UserId, err.Error()))
				return
			}
		}
	}

	logger.Info("user last read updated successfully")
}

func (r *DmChannels) UpdateInteractionAt(db *gorm.DB) error {

	result := db.Model(&DmChannels{}).
		Where("channel_id = ?", r.ChannelId).
		Update("interacted_at", time.Now())

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("no channel found with the given channelId")
	}

	return nil
}
