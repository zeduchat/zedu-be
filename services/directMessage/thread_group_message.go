package dm

import (
	"net/http"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func CreateGroupThreadDMMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, int, error) {

	thread := models.ThreadDocument{
		UserId:     req.UserId,
		ChannelsID: req.ChannelsID,
	}

	dmchannel := models.DmChannels{}

	res, err := dmchannel.CheckChannelExists(db.Postgresql, req.ChannelsID, "")
	if !res || err != nil {
		return &thread, http.StatusBadRequest, err
	}

	return SaveThreadDmMessage(req, db, logger)
}
