package dm

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
)

// Reply message fn
func SaveChannelsDmMsg(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

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

	ch, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsId)

	if !ch || err != nil {
		return nil, http.StatusNotFound, errors.New("channel does not exist")
	}

	messageDoc := models.MessageDocument{
		ID:         utility.GenerateUUID(),
		Content:    req.Content,
		ChannelsID: req.ChannelsId,
		UserID:     req.UserId,
		ThreadID:   threadId,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
		AvatarURL:  profile.AvatarURL,
		Edited:     false,
		Username:   profile.UserName,
		FullName:   profile.FullName,
		Email:      user.Email,
	}

	err = messageDoc.CreateMessage(db, logger)

	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to save message, error: " + err.Error())
	}

	if err := thread.DetectAndAddMentions(messageDoc.ID, req.Content, db.Postgresql); err != nil {
		return &messageDoc, http.StatusBadRequest, err
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelsId,
		UserName:  profile.UserName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AvatarURL: profile.AvatarURL,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  req.ThreadId,
		Email:     user.Email,
		FullName:  profile.FullName,
		OrgId:     req.OrgId,
		UserId:    req.UserId,
	}

	err = centrifuge.BroadcastChannel(logger, threadId.String(), feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to threadId: %s, error: %v", threadId.String(), err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	err = centrifuge.BroadcastChannel(logger, channel.ParticipantId, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to particpant id: %s error: %v", channel.ParticipantId, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	return &messageDoc, http.StatusCreated, nil
}

func DeleteChannelsDmMsg(req models.EditMessageRequest) (*models.Message, int, error) {

	var message models.Message

	message.ID = req.MessageId

	if _, err := message.DeleteMessage(); err != nil {
		return nil, http.StatusBadRequest, err
	}

	return nil, http.StatusOK, nil
}

// Reply message fn
func AddChannelsDmMsg(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

	// Provision for bot dms

	return SaveChannelsDmMsg(req, db, logger)

}
