package dm

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateGroupDMChannel(req models.GroupDMChannelsRequest, base *storage.Database, logger *utility.Logger) (*models.GroupDMChannelsResponse, int, error) {
	var dmchans models.DmChannels

	dmchans.ChatType = req.ChatType
	dmchans.ChannelType = "group_dm"
	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId
	dmchans.ChannelId = utility.GenerateUUID()
	dmchans.ID = utility.GenerateUUID()

	req.Participants = append(req.Participants, req.UserId)

	participants := utility.ReturnUniqueIDs(req.Participants)

	req.Participants = participants

	resp, statusCode, err := dmchans.CreateGroupDMChannel(base.Postgresql, req)
	if err != nil {
		return nil, statusCode, err
	}

	if statusCode == http.StatusCreated {
		var usernames []string
		for _, participantID := range participants {
			var user models.User
			userDetails, userErr := user.GetUserByID(base.Postgresql, participantID)
			if userErr == nil {
				if userDetails.Profile.UserName == "" {
					userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
				}
				usernames = append(usernames, "@"+userDetails.Profile.UserName)
			}
		}

		if len(usernames) > 0 {
			var content string
			if len(usernames) == 1 {
				content = fmt.Sprintf("%s joined the group", usernames[0])
			} else if len(usernames) == 2 {
				content = fmt.Sprintf("%s and %s joined the group", usernames[0], usernames[1])
			} else {
				lastUser := usernames[len(usernames)-1]
				otherUsers := strings.Join(usernames[:len(usernames)-1], ", ")
				content = fmt.Sprintf("%s, and %s joined the group", otherUsers, lastUser)
			}

			systemMsg := models.CreateThreadMsgReq{
				Content:    content,
				Type:       "system",
				UserId:     participants[0],
				ChannelsID: resp.ChannelId,
				OrgId:      req.OrgId,
				ThreadId:   utility.GenerateUUID(),
			}

			_, _, saveErr := CreateThreadDmMessage(systemMsg, base, logger, request.ExternalRequest{Logger: logger})
			if saveErr != nil {
				logger.Error("failed to save system message for group DM channel %s", resp.ChannelId)
			} else {
				logger.Info("Added system message for group DM creation")
			}
		}
	}

	return &resp, statusCode, nil
}

func LeaveGroupDMChannel(req models.DmChannelsRequest, base *storage.Database, logger *utility.Logger) (int, error) {
	var dmchans models.DmChannels

	dmchans.ChannelId = req.ChannelId
	dmchans.UserId = req.UserId

	statusCode, err := dmchans.LeaveGroupDMChannel(base.Postgresql)
	if err != nil {
		return statusCode, err
	}

	var user models.User
	userDetails, userErr := user.GetUserByID(base.Postgresql, req.UserId)
	if userErr == nil {
		if userDetails.Profile.UserName == "" {
			userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
		}

		systemMsg := models.CreateThreadMsgReq{
			Content:    fmt.Sprintf("@%s left the group", userDetails.Profile.UserName),
			Type:       "system",
			UserId:     req.UserId,
			ChannelsID: req.ChannelId,
			OrgId:      req.OrgId,
			ThreadId:   utility.GenerateUUID(),
		}

		_, _, saveErr := CreateThreadDmMessage(systemMsg, base, logger, request.ExternalRequest{Logger: logger})
		if saveErr != nil {
			logger.Error("failed to save system message for group DM channel %s", req.ChannelId)
		} else {
			logger.Info("Added system message for user leaving group DM")
		}
	}

	return statusCode, nil
}

func JoinGroupDMChannel(req models.DmChannelsRequest, base *storage.Database, logger *utility.Logger) (*models.GroupDMChannelsResponse, int, error) {
	var dmchans models.DmChannels

	dmchans.ChannelId = req.ChannelId
	dmchans.UserId = req.UserId
	dmchans.OrgId = req.OrgId

	resp, statusCode, err := dmchans.JoinGroupDMChannel(base.Postgresql)
	if err != nil {
		return nil, statusCode, err
	}

	var user models.User
	userDetails, userErr := user.GetUserByID(base.Postgresql, req.UserId)
	if userErr == nil {
		if userDetails.Profile.UserName == "" {
			userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
		}

		systemMsg := models.CreateThreadMsgReq{
			Content:    fmt.Sprintf("@%s joined the group", userDetails.Profile.UserName),
			Type:       "system",
			UserId:     req.UserId,
			ChannelsID: req.ChannelId,
			OrgId:      req.OrgId,
			ThreadId:   utility.GenerateUUID(),
		}

		_, _, saveErr := CreateThreadDmMessage(systemMsg, base, logger, request.ExternalRequest{Logger: logger})
		if saveErr != nil {
			logger.Error("failed to save system message for group DM channel %s", req.ChannelId)
		} else {
			logger.Info("Added system message for user joining group DM")
		}
	}

	return &resp, statusCode, nil
}

func AddParticipantsToGroupDM(req models.AddParticipantsRequest, base *storage.Database, logger *utility.Logger) (*models.AddParticipantsResponse, int, error) {
	var dmchans models.DmChannels

	dmchans.ChannelId = req.ChannelId

	resp, statusCode, err := dmchans.AddParticipantsToGroupDM(base.Postgresql, req)
	if err != nil {
		return nil, statusCode, err
	}

	if statusCode == http.StatusOK && resp.AddedCount > 0 {
		var usernames []string
		for _, participantID := range req.UserIds {
			var user models.User
			userDetails, userErr := user.GetUserByID(base.Postgresql, participantID)
			if userErr == nil {
				if userDetails.Profile.UserName == "" {
					userDetails.Profile.UserName = strings.Split(userDetails.Email, "@")[0]
				}
				usernames = append(usernames, "@"+userDetails.Profile.UserName)
			}
		}

		if len(usernames) > 0 {
			var content string
			if len(usernames) == 1 {
				content = fmt.Sprintf("%s joined the group", usernames[0])
			} else if len(usernames) == 2 {
				content = fmt.Sprintf("%s and %s joined the group", usernames[0], usernames[1])
			} else {
				lastUser := usernames[len(usernames)-1]
				otherUsers := strings.Join(usernames[:len(usernames)-1], ", ")
				content = fmt.Sprintf("%s, and %s joined the group", otherUsers, lastUser)
			}

			systemMsg := models.CreateThreadMsgReq{
				Content:    content,
				Type:       "system",
				UserId:     req.UserIds[0],
				ChannelsID: req.ChannelId,
				OrgId:      req.OrgId,
				ThreadId:   utility.GenerateUUID(),
			}

			_, _, saveErr := CreateThreadDmMessage(systemMsg, base, logger, request.ExternalRequest{Logger: logger})
			if saveErr != nil {
				logger.Error("failed to save system message for group DM channel %s", req.ChannelId)
			} else {
				logger.Info("Added system message for %d new participants in group DM", resp.AddedCount)
			}
		}
	}

	return &resp, statusCode, nil
}

func GetGroupDMChannels(req models.GroupDMChannelsRequest, db *gorm.DB, c *gin.Context) ([]models.GroupDMChannelsResponse, postgresql.PaginationResponse, int, error) {

	var dmchans models.DmChannels

	dmchans.OrgId = req.OrgId
	dmchans.UserId = req.UserId

	resp, pagResp, err := dmchans.GetGroupDMChannels(db, c)

	if err != nil {
		return nil, pagResp, http.StatusInternalServerError, err
	}

	return resp, pagResp, http.StatusOK, err
}

func GetUserGroupDMs(req models.GroupDMChannelsRequest, db *gorm.DB, c *gin.Context) (gin.H, int, error) {

	var (
		userProfile models.Profile
		user        models.User

		resp gin.H
	)

	user, err := user.GetUserByID(db, req.UserId)

	if err != nil {
		return resp, http.StatusNotFound, fmt.Errorf("user does not exist")
	}

	err = userProfile.GetProfileByUserId(db, req.UserId)
	if err != nil {
		return resp, http.StatusInternalServerError, err
	}

	resp = gin.H{
		"avatar_url":         userProfile.AvatarURL,
		"default_avatar_url": userProfile.DefaultAvatarURL,
		"username":           userProfile.UserName,
		"email":              user.Email,
		"user_type":          "user",
		"user_id":            req.UserId,
		"title":              userProfile.Title,
		"online":             userProfile.Online,
	}

	if resp["username"] == "" {
		resp["username"] = strings.Split(user.Email, "@")[0]
	}

	return resp, http.StatusOK, err
}
