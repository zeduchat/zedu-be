package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
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
}

type DmChannelsResponse struct {
	ID               string `json:"channel_id"`
	Name             string `json:"username"`
	ParticipantId    string `json:"participant_id"`
	AvatarUrl        string `json:"avatar_url"`
	ParticipantEmail string `json:"participant_email"`
	ChannelType      string `json:"channel_type"`
}

type DmChannelsRequest struct {
	ChatType      string `json:"chat_type" validate:"required,oneof=user bot"`
	ParticipantId string `json:"participant_id" validate:"required"`
	UserId        string `json:"user_id"`
	OrgId         string `json:"org_id"`
	ChannelId     string `json:"channel_id"`
}

func FetchDetailsFromAgentJSON(extReq request.ExternalRequest, agent OrganisationIntegrations, redisClient *redis.Client) (map[string]interface{}, error) {
	var response interface{}
	var data_r map[string]interface{}
	agentJSONURL := agent.JSONUrl
	redisKey := fmt.Sprintf("agent_json_%s", agentJSONURL)

	if agent.AppName == "" {

		cachedData, err := rd.RedisGet(redisClient, redisKey)
		if err == nil && len(cachedData) > 0 {
			var cachedResult interface{}

			if err := json.Unmarshal(cachedData, &cachedResult); err != nil {
				rd.RedisDelete(redisClient, redisKey)
				return nil, fmt.Errorf("failed to unmarshal cached data: %v", err)
			}

			data_r, ok := cachedResult.(map[string]interface{})
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

		response_data := response.(map[string]interface{})
		content, ok := response_data["data"].(map[string]interface{})
		if !ok {
			return nil, errors.New("could not fetch data from agent json")
		}

		data_r, ok := content["descriptions"].(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid agent details format")
		}

		data_r["bot"] = content["bot"]

		err = ValidateAgentData(data_r)
		if err != nil {
			return nil, fmt.Errorf("invalid agent json data: %v", err)
		}
	} else {

		data_r = map[string]interface{}{
			"app_name":        agent.AppName,
			"app_logo":        agent.AppLogo,
			"app_description": agent.AppDescription,
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

	if postgresql.CheckExists(db, &DmChannels{}, "user_id = ? AND participant_id = ?", dm.UserId, dm.ParticipantId) {
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

	var user User

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

	return nil
}

func (dm *DmChannels) GetDmChannels(db *gorm.DB, c *gin.Context) ([]DmChannelsResponse, postgresql.PaginationResponse, error) {
	var (
		user     User
		chanPart []ChannelParticipant
	)

	dmchans := []DmChannels{}
	dmChansResp := []DmChannelsResponse{}

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

	// Use SelectAllFromDbOrderByPaginated with the modified query
	paginationResp, err := postgresql.SelectAllFromDbOrderByPaginated(
		db, // No JOIN needed since we're using a subquery
		"created_at",
		"desc",
		pagination,
		&dmchans,
		queryString,
		dm.OrgId,
		"user",
		dm.UserId,
		dm.UserId,
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

			dmChansResp = append(dmChansResp, DmChannelsResponse{
				ID:               dmchan.ChannelId,
				Name:             strings.Join(usernames, ", "),
				AvatarUrl:        profilePic,
				ParticipantEmail: email,
				ChannelType:      "group_dm",
			})

		}

		slices.SortFunc(dmChansResp, func(a, b DmChannelsResponse) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})

	}

	return dmChansResp, paginationResp, nil
}

func (r *DmChannels) CheckChannelExists(db *gorm.DB, channelID string) (bool, error) {

	exists := postgresql.CheckExists(db, &r, "channel_id = ?", channelID)

	if !exists {
		return exists, errors.New("channel does not exist")
	}

	return exists, nil
}

func (r *DmChannels) FetchChannelParticipant(db *gorm.DB, req DmChannelsRequest) (bool, error) {

	exists := postgresql.CheckExists(db, &r, "channel_id = ? AND user_id = ?", req.ChannelId, req.UserId)

	if !exists {
		return exists, errors.New("channel does not exist")
	}

	return exists, nil
}
