package dm

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func CreateDmChannel(req models.DmChannelsRequest, extReq request.ExternalRequest, base *storage.Database, logger *utility.Logger, c *gin.Context) (*models.DmChannelsResponse, int, error) {

	var (
		dmchans models.DmChannels
		dmfetch models.DmChannels
	)

	exists := postgresql.CheckExists(base.Postgresql, &dmfetch, "user_id = ? AND participant_id = ? AND org_id = ?", req.UserId, req.ParticipantId, req.OrgId)

	if !exists {
		dmchans.ChannelId = utility.GenerateUUID()
		dmchans.ID = utility.GenerateUUID()
	} else {
		resp, err := dmfetch.GetDmChannelResponse(base.Postgresql, c)
		if err != nil {
			logger.Error("failed to get DM channel response for user %s and participant %s, err: %v", req.UserId, req.ParticipantId, err)
			return nil, http.StatusInternalServerError, err
		}
		return &resp, http.StatusOK, nil
	}
	dmchans.ChatType = req.ChatType
	dmchans.ChannelType = "dm"
	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId
	dmchans.ParticipantId = &req.ParticipantId

	var resp models.DmChannelsResponse
	var err error

	if req.ChatType == "bot" {
		resp, err = dmchans.CreateAgentDMChannel(base.Postgresql)
	} else {
		resp, err = dmchans.CreateDmChannel(base.Postgresql)
	}
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if !exists {
		if req.ChatType == "bot" {
			var orgAgent models.OrganisationIntegrations
			agentExists := postgresql.CheckExists(base.Postgresql, &orgAgent, "integration_id = ? AND org_id = ?", req.ParticipantId, req.OrgId)
			if agentExists {
				agentDetails, err := models.FetchDetailsFromAgentJSON(orgAgent)
				if err == nil {
					appName := utility.GetString(agentDetails, "app_name")
					if appName != "" {
						systemMsg := models.CreateThreadMsgReq{
							Content:    fmt.Sprintf("<p>started a conversation with <span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span> </p><p></p>", req.ParticipantId, appName, appName),
							Type:       "system",
							UserId:     "",
							AgentId:    req.ParticipantId,
							AgentName:  appName,
							ChannelsID: resp.ID,
							OrgId:      req.OrgId,
							ThreadId:   utility.GenerateUUID(),
							Mentions: []models.Mention{
								{ID: req.ParticipantId, Type: "bot"},
							},
						}

						_, _, saveErr := CreateThreadDmMessage(systemMsg, base, logger, request.ExternalRequest{Logger: logger})
						if saveErr != nil {
							logger.Error("failed to save system message for bot DM channel %s", resp.ID)
						} else {
							logger.Info("Added system message for bot DM channel creation")
						}
					}
				}
			}
		} else {
			var user models.User
			userDetails, userErr := user.GetUserByID(base.Postgresql, req.ParticipantId)
			if userErr == nil {
				if userDetails.Profile.UserName == "" {
					userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
				}

				systemMsg := models.CreateThreadMsgReq{
					Content:    fmt.Sprintf("<p>started a conversation with <span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span> </p><p></p>", userDetails.ID, userDetails.Profile.UserName, userDetails.Profile.UserName),
					Type:       "system",
					UserId:     req.UserId,
					ChannelsID: resp.ID,
					OrgId:      req.OrgId,
					ThreadId:   utility.GenerateUUID(),
					Mentions: []models.Mention{
						{ID: userDetails.ID, Type: "user"},
					},
				}

				_, _, saveErr := CreateThreadDmMessage(systemMsg, base, logger, request.ExternalRequest{Logger: logger})
				if saveErr != nil {
					logger.Error("failed to save system message for DM channel %s, err: %v", resp.ID, saveErr)
				} else {
					logger.Info("Added system message for DM channel creation")
				}
			}
		}
	}

	return &resp, http.StatusCreated, nil
}

func GetDmChannels(req models.DmChannelsRequest, db *gorm.DB, c *gin.Context) ([]models.DmChannelsResponse, postgresql.PaginationResponse, int, error) {

	var dmchans models.DmChannels

	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId

	resp, pagResp, err := dmchans.GetDmChannels(db, c)
	if err != nil {
		return nil, pagResp, http.StatusInternalServerError, err
	}

	favouriteChannelIds, _ := models.GetUserFavouriteChannelIds(db, req.UserId, req.OrgId)
	favouriteMap := make(map[string]bool)
	for _, id := range favouriteChannelIds {
		favouriteMap[id] = true
	}

	for i := range resp {
		resp[i].IsFavourite = favouriteMap[resp[i].ID]
	}

	return resp, pagResp, http.StatusOK, err
}

func GetDmParticipants(req models.DmChannelsRequest, db *storage.Database, c *gin.Context, extReq request.ExternalRequest, rds *redis.Client) (models.DmParticipantsResponse, int, error) {
	var (
		user      models.User
		is_agent  bool = false
		orgAgent  models.OrganisationIntegrations
		dmchannel models.DmChannels
		favourite models.DmFavourite
	)

	resp := models.DmParticipantsResponse{
		Participants:   []models.Participant{},
		PreviewMedia:   []models.FileMediaResponse{},
		GroupsInCommon: []models.GroupInCommon{},
	}

	_, err := dmchannel.FetchChannelParticipant(db.Postgresql, req)
	if err != nil {
		return resp, http.StatusBadRequest, err
	}

	IsFavourite := favourite.IsFavouriteChannel(db.Postgresql, models.DmFavouriteRequest{
		UserId:    req.UserId,
		OrgId:     req.OrgId,
		ChannelId: dmchannel.ChannelId,
	})
	resp.IsFavourite = IsFavourite

	if dmchannel.ChatType == "bot" {
		//check if user is a bot
		is_agent = postgresql.CheckExists(db.Postgresql, &orgAgent, "integration_id = ? AND org_id = ?", dmchannel.ParticipantId, req.OrgId)
		if !is_agent {
			return resp, http.StatusNotFound, fmt.Errorf("user not found: %v", err)
		}

		agentDetails, err := models.FetchDetailsFromAgentJSON(orgAgent)
		if err != nil {
			return resp, http.StatusInternalServerError, fmt.Errorf("failed to fetch agent details: %w", err)
		}

		appName, appLogo := utility.GetString(agentDetails, "app_name"), utility.GetString(agentDetails, "app_logo")
		if appName == "" {
			return resp, http.StatusInternalServerError, errors.New("missing required agent details (app_name, app_logo)")
		}
		resp.Type = "bot"
		resp.Participants = append(resp.Participants, models.Participant{
			AvatarUrl: appLogo,
			Username:  appName,
			Email:     appName,
			UserType:  "bot",
			UserId:    *dmchannel.ParticipantId,
		})
		return resp, http.StatusOK, nil
	}

	switch dmchannel.ChannelType {
	case "group_dm":
		resp.Type = "groupdm"
		resp.GroupDescription = dmchannel.GroupDescription

		// Custom struct to hold the joined query results
		type ParticipantWithProfile struct {
			UserId    string
			Title     string
			AvatarURL string

			UserName string
			Email    string
			Online   bool
		}

		var participantsWithProfile []ParticipantWithProfile

		// Custom SQL query joining channel_participants with profiles and users
		err := db.Postgresql.Table("channel_participants cp").
			Select(`
				cp.user_id,
				COALESCE(p.title, '') as title,
				COALESCE(p.avatar_url, '') as avatar_url,
				COALESCE(p.user_name, SPLIT_PART(u.email, '@', 1)) as user_name,
				u.email,
				p.online
			`).
			Joins("JOIN users u ON u.id = cp.user_id").
			Joins("LEFT JOIN profiles p ON p.userid = cp.user_id").
			Where("cp.channel_id = ? AND cp.deleted_at IS NULL", dmchannel.ChannelId).
			Scan(&participantsWithProfile).Error

		if err != nil {
			return resp, http.StatusInternalServerError, fmt.Errorf("failed to fetch group DM participants: %w", err)
		}

		for _, part := range participantsWithProfile {
			var partUser models.User
			partUserDetails, err := partUser.GetUserByID(db.Postgresql, part.UserId)
			if err != nil {
				continue
			}
			resp.Participants = append(resp.Participants, models.NewParticipant(partUserDetails, part.UserId == dmchannel.UserId, "user"))
		}
	case "dm":
		resp.Type = "dm"

		userDetails, _ := user.GetUserByID(db.Postgresql, *dmchannel.ParticipantId)

		groupsInCommon, _ := models.GetGroupsInCommon(
			db.Postgresql,
			req.UserId,
			*dmchannel.ParticipantId,
			req.OrgId,
		)

		resp.GroupsInCommon = groupsInCommon

		resp.Participants = append(resp.Participants, models.NewParticipant(userDetails, false, "user"))

	}

	if db != nil {
		dmchannel.ChannelId = req.ChannelId
		previewMedia, _, err := dmchannel.GetPreviewMedia(db, 10)
		if err == nil {
			resp.PreviewMedia = previewMedia
		}
	}

	resp.CreatedAt = dmchannel.CreatedAt

	return resp, http.StatusOK, nil
}

func DeleteDmChannel(req models.DmChannelsRequest, db *gorm.DB, loggers ...*utility.Logger) (int, error) {
	var (
		dmchans models.DmChannels
		logger  *utility.Logger
	)
	if len(loggers) > 0 {
		logger = loggers[0]
	}

	dmchans.ID = req.ChannelId
	dmchans.UserId = req.UserId

	err := dmchans.DeleteDmChannel(db, logger)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}


func GetDmChannelMedia(req models.DmChannelMediaRequest, db *storage.Database, c *gin.Context) ([]models.File, postgresql.PaginationResponse, int, error) {
	var dmchan models.DmChannels

	// Verify user is a participant in the channel
	dmchan.ChannelId = req.ChannelId
	dmchan.UserId = req.UserId

	exists, err := dmchan.CheckChannelExists(db.Postgresql, req.ChannelId, req.UserId)
	if err != nil || !exists {
		return nil, postgresql.PaginationResponse{}, http.StatusNotFound, fmt.Errorf("channel not found or user is not a participant")
	}

	// Fetch media from Elasticsearch
	media, pagResp, err := dmchan.GetChannelMedia(db, c, req.MediaType)
	if err != nil {
		return nil, pagResp, http.StatusInternalServerError, err
	}

	return media, pagResp, http.StatusOK, nil
}

func UpsertGroupDescription(req models.GroupDescriptionRequest, db *gorm.DB) (int, error) {
	var dmchan models.DmChannels

	err := dmchan.UpsertGroupDescription(db, req)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func ToggleFavourite(req models.DmFavouriteRequest, db *gorm.DB) (bool, int, error) {
	var dmFav models.DmFavourite

	isFavourite := dmFav.IsFavouriteChannel(db, req)

	if isFavourite {
		err := dmFav.RemoveFromFavourites(db, req)
		if err != nil {
			return false, http.StatusInternalServerError, err
		}
		return false, http.StatusOK, nil
	}

	dmFav.ID = utility.GenerateUUID()
	err := dmFav.AddToFavourites(db, req)
	if err != nil {
		return false, http.StatusInternalServerError, err
	}
	return true, http.StatusOK, nil
}

func GetFavouriteDms(db *storage.Database, req models.DmChannelsRequest) ([]models.DmChannelsResponse, int, error) {
	favouriteChannelIds, err := models.GetUserFavouriteChannelIds(db.Postgresql, req.UserId, req.OrgId)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if len(favouriteChannelIds) == 0 {
		return []models.DmChannelsResponse{}, http.StatusOK, nil
	}

	// Get DM channels that match the favourite channel IDs
	var dmChannels []models.DmChannels
	err = db.Postgresql.Where("user_id = ? AND org_id = ? AND channel_id IN ?", req.UserId, req.OrgId, favouriteChannelIds).
		Find(&dmChannels).Error

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	var response []models.DmChannelsResponse
	var user models.User

	for _, dm := range dmChannels {
		resp := models.DmChannelsResponse{
			ID:           dm.ChannelId,
			ChannelType:  dm.ChannelType,
			CreatedAt:    dm.CreatedAt,
			IsFavourite:  true,
			ThreadCount:  dm.ThreadCount,
			LastThreadId: dm.LastThreadId,
			LastReadAt:   dm.LastReadAt,
		}

		switch dm.ChannelType {
		case "dm", "":
			if dm.ParticipantId != nil {
				userDetails, err := user.GetUserByID(db.Postgresql, *dm.ParticipantId)
				if err == nil {
					p := models.NewParticipant(userDetails, false, "user")

					resp.Name = p.Username
					resp.AvatarUrl = userDetails.Profile.AvatarURL
					resp.DefaultAvatarUrl = avatar.GenerateDefaultAvatarURL(userDetails.ID)
					resp.ParticipantId = *dm.ParticipantId
					resp.ParticipantEmail = userDetails.Email
					resp.Participants = []models.Participant{models.NewParticipant(userDetails, false, "user")}
				}
			}
		case "group_dm":
			type ParticipantWithProfile struct {
				UserId    string
				AvatarURL string

				UserName string
				Email    string
				Title    string
				Online   bool
			}

			var participantsWithProfile []ParticipantWithProfile
			err := db.Postgresql.Table("channel_participants cp").
				Select(`
					cp.user_id,
					COALESCE(p.avatar_url, '') as avatar_url,
					COALESCE(p.user_name, SPLIT_PART(u.email, '@', 1)) as user_name,
					u.email,
					COALESCE(p.title, '') as title,
					COALESCE(p.online, false) as online
				`).
				Joins("JOIN users u ON u.id = cp.user_id").
				Joins("LEFT JOIN profiles p ON p.userid = cp.user_id").
				Where("cp.channel_id = ? AND cp.deleted_at IS NULL", dm.ChannelId).
				Scan(&participantsWithProfile).Error

			if err == nil {
				usernames := []string{}
				profilePic := ""
				defaultProfilePic := ""
				email := ""
				var participants []models.Participant

				for _, part := range participantsWithProfile {
					var partUser models.User
					partUserDetails, userErr := partUser.GetUserByID(db.Postgresql, part.UserId)
					if userErr != nil {
						continue
					}
					p := models.NewParticipant(partUserDetails, part.UserId == dm.UserId, "user")
					usernames = append(usernames, p.Username)
					if profilePic == "" {
						profilePic = partUserDetails.Profile.AvatarURL
					}
					if defaultProfilePic == "" {
						defaultProfilePic = avatar.GenerateDefaultAvatarURL(partUserDetails.ID)
					}
					if email == "" {
						email = partUserDetails.Email
					}
					participants = append(participants, p)
				}

				resp.Name = strings.Join(usernames, ", ")
				resp.AvatarUrl = profilePic
				resp.DefaultAvatarUrl = defaultProfilePic
				resp.ParticipantEmail = email
				resp.Participants = participants
			}
		}

		response = append(response, resp)
	}

	return response, http.StatusOK, nil
}

func GetVisibleDmChannels(req models.DmChannels, db *storage.Database, c *gin.Context) ([]models.DmChannelsResponse, postgresql.PaginationResponse, int, error) {
	dmChansResp, paginationResp, err := req.GetVisibleDmChannels(db.Postgresql, c)
	if err != nil {
		return nil, paginationResp, http.StatusInternalServerError, err
	}
	return dmChansResp, paginationResp, http.StatusOK, nil
}

func UpdateDmVisibility(req models.UpdateDmVisibilityRequest, db *storage.Database) (int, error) {
	var dm models.DmChannels
	err := dm.UpdateDmVisibility(db.Postgresql, req)
	if err != nil {
		if err.Error() == "DM channel not found for user" {
			return http.StatusNotFound, err
		}
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

