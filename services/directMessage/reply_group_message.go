package dm

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func ReplyGroupChannelsMsg(req models.CreateMessageRequest, db *storage.Database,
	logger *utility.Logger) (*models.MessageDocument, int, error) {

	// Provision for bot group dms

	return SaveChannelsDmMsg(req, db, logger)
}
