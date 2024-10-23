package channel

import (
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func AddIntegrationChannel(db *gorm.DB, req models.AddIntegrationChannel) (models.IntegrationChannel, error) {

	intchan := models.IntegrationChannel{
		ID:             utility.GenerateUUID(),
		IntegrationID:  req.IntegrationID,
		IntChannelID:   req.IntChannelID,
		IntChannelName: req.IntChannelName,
		ChannelID:      req.ChannelID,
	}

	res, err := intchan.CreateIntegrationChan(db)

	return res, err
}

func DeleteChannelIntegration(db *gorm.DB, req models.IntegrationChannelReq) (int, error) {
	intchan := models.IntegrationChannel{
		IntegrationID: req.IntegrationID,
		IntChannelID:  req.IntChannelID,
		ChannelID:     req.ChannelID,
	}

	code, err := intchan.DeleteChannelIntegration(db)

	return code, err
}

func GetChannelIntegration(db *gorm.DB, channel_id string) ([]models.IntegrationOutput, error) {

	intchan := models.IntegrationChannel{
		ChannelID: channel_id,
	}

	res, err := intchan.GetIntegrationChannels(db)

	return res, err
}
