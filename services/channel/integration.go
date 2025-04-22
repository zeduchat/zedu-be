package channel

import (
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func AddIntegrationChannel(db *gorm.DB, req models.AddIntegrationChannel) (models.IntegrationChannel, int, error) {

	intchan := models.IntegrationChannel{
		ID:             utility.GenerateUUID(),
		IntegrationID:  req.IntegrationModifierID,
		IntChannelID:   req.IntChannelID,
		IntChannelName: req.IntChannelName,
		ChannelID:      req.ChannelID,
		OutputID:       req.IntegrationOutputID,
	}

	res, code, err := intchan.CreateIntegrationChan(db, req.IntegrationOutputID)

	return res, code, err
}

func DeleteChannelIntegration(db *gorm.DB, req models.IntegrationChannelReq) (int, error) {
	var intchan models.IntegrationChannel

	code, err := intchan.DeleteChannelIntegration(db, req)

	return code, err
}

func GetChannelIntegration(db *gorm.DB, channel_id, int_id string) ([]models.IntegrationOutput, int, error) {

	intchan := models.IntegrationChannel{
		ChannelID:     channel_id,
		IntegrationID: int_id,
	}

	res, code, err := intchan.GetIntegrationChannels(db)

	return res, code, err
}
