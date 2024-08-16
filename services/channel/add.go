package channel

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"
)

func AddChannelsMsg(req models.CreateMessageRequest, db *gorm.DB, typesenseDb *typesense.Client,
	logger *utility.Logger) (*models.Message, int, error) {

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	message := models.Message{
		ID:         utility.GenerateUUID(),
		Content:    req.Content,
		ChannelsID: req.ChannelsId,
		UserID:     req.UserId,
		ThreadID:   threadId,
	}

	if err := message.CreateMessage(db, typesenseDb); err != nil {
		return &message, http.StatusBadRequest, err
	}

	if err := thread.DetectAndAddMentions(message.ID, req.Content, db); err != nil {
		return &message, http.StatusBadRequest, err
	}

	err = centrifuge.BroadcastChannel(logger, req.ChannelsId, message)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error broadcasting message %s to channel %s: %v", message.Content, req.ChannelsId, err.Error()))
		return &message, http.StatusBadRequest, errors.New("failed to broadcast message")
	}

	return &message, http.StatusCreated, nil
}
