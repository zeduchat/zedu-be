package dm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
)

// Reply message fn (in dm / group_dm)
func ReplyChannelDMMessage(req models.CreateMessageRequest, db *storage.Database, logger *utility.Logger) (*models.MessageDocument, int, error) {
	var (
		profile models.Profile
		user    models.User
		channel models.DmChannels
	)

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	err = profile.GetProfileByUserId(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user")
	}

	ch, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsId, req.UserId)
	if !ch || err != nil {
		return nil, http.StatusNotFound, errors.New("channel does not exist")
	}

	messageDoc := models.MessageDocument{
		ID:             utility.GenerateUUID(),
		Content:        req.Content,
		ChannelsID:     req.ChannelsId,
		UserID:         req.UserId,
		ThreadID:       threadId,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		AvatarURL:      profile.AvatarURL,
		Edited:         false,
		Username:       profile.UserName,
		FullName:       profile.FullName,
		Email:          user.Email,
		UserType:       "user",
		OrganisationID: channel.OrgId,
		Mentions:       req.Mentions,
		Media:          req.Media,
	}

	updateResp, err := messageDoc.CreateMessage(db, logger)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to save message, error: " + err.Error())
	}

	if err := thread.DetectAndAddMentions(messageDoc.ID, req.Content, db.Postgresql); err != nil {
		return &messageDoc, http.StatusBadRequest, err
	}

	username := utility.ThisOrThat(profile.UserName, utility.ThisOrThat(profile.FullName, user.Email))

	feed := models.FeedMessageRequest{
		ChannelID:   req.ChannelsId,
		UserName:    profile.UserName,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   messageDoc.UpdatedAt.String(),
		AvatarURL:   profile.AvatarURL,
		Type:        "message",
		Content:     req.Content,
		ThreadId:    req.ThreadId,
		Email:       user.Email,
		FullName:    profile.FullName,
		OrgId:       req.OrgId,
		UserType:    "user",
		UserId:      req.UserId,
		Media:       req.Media,
		Id:          messageDoc.ID,
		ChannelName: username,
	}

	err = centrifuge.PublishChannel(logger, threadId.String(), feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to threadId: %s, error: %v", threadId.String(), err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish webhook data: " + err.Error())
	}

	dataByte, _ := json.Marshal(feed)

	notifRec := models.PushNotificationRecord{
		ChannelType:  models.DMChannel,
		Data:         string(dataByte),
		Sent:         false,
		ChannelId:    req.ChannelsId,
		Section:      models.ReplySection,
		UpdateChange: updateResp,
		Type:         models.NewMessage,
	}

	err = actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec)

	if err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", req.ChannelsId, req.OrgId, err.Error())
	}

	logger.Info("added notification to queue for channel %s", req.ChannelsId)

	return &messageDoc, http.StatusCreated, nil
}

// Reply message fn
func AddChannelsDmMsg(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

	// Provision for bot dms

	return ReplyChannelDMMessage(req, db, logger)

}
