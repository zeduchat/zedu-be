package models

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type DmChannels struct {
	ID              string         `gorm:"type:uuid" json:"id"`
	UserId          string         `gorm:"type:uuid" json:"-"`
	ChannelId       string         `gorm:"type:uuid" json:"channel_id"`
	OrgId           string         `gorm:"type:uuid" json:"-"`
	ParticipantId   *string        `gorm:"type:uuid" json:"-"`
	ParticipantHash string         `gorm:"type:string" json:"participant_hash"`
	ChatType        string         `gorm:"type:string" json:"chat_type"`    // user or bot
	ChannelType     string         `gorm:"type:string" json:"channel_type"` // dm or group_dm
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

func FetchDetailsFromAgentJSON(extReq request.ExternalRequest, agentJSONURL string) (map[string]interface{}, error) {
	data := map[string]string{"url": agentJSONURL}
	var response interface{}
	var err error

	for i := 0; i < 2; i++ {
		response, err = extReq.SendExternalRequest(request.AgentJsonContent, data)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(2<<i) * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("could not fetch agent json: %v", err)
	}

	response_data := response.(map[string]interface{})
	data_r, ok := response_data["data"].(map[string]interface{})
	if !ok {
		return nil, errors.New("could not fetch data from agent json")
	}

	err = ValidateAgentData(data_r)
	if err != nil {
		return nil, fmt.Errorf("invalid agent json data: %v", err)
	}

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

func (dm *DmChannels) CreateAgentDMChannel(extReq request.ExternalRequest, db *gorm.DB) (DmChannelsResponse, error) {
	var orgAgent OrganisationIntegrations
	if !postgresql.CheckExists(db, &orgAgent, "org_id = ? AND integration_id = ?", dm.OrgId, dm.ParticipantId) {
		return DmChannelsResponse{}, fmt.Errorf("agent participant does not exist in organisation %v", dm.OrgId)
	}

	agentDetails, err := FetchDetailsFromAgentJSON(extReq, orgAgent.JSONUrl)
	if err != nil {
		return DmChannelsResponse{}, fmt.Errorf("failed to fetch agent details: %w", err)
	}

	agentDescription, ok := agentDetails["descriptions"].(map[string]interface{})
	if !ok {
		return DmChannelsResponse{}, errors.New("invalid agent details format")
	}

	appName, appLogo := utility.GetString(agentDescription, "app_name"), utility.GetString(agentDescription, "app_logo")
	if appName == "" || appLogo == "" {
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

	exists := postgresql.CheckExists(db, &existDmchan, "user_id = ? AND participant_id = ?", dm.UserId, *dm.ParticipantId)
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

	paginationResp, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&dmchans,
		"org_id = ? AND user_id = ? AND chat_type = ?",
		dm.OrgId,
		dm.UserId,
		"user",
	)
	if err != nil {
		return nil, paginationResp, err
	}

	for _, dmchan := range dmchans {
		if dmchan.ChannelType == "dm" {
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
