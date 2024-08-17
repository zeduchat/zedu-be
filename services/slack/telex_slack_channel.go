package slack

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateTelexSlackChannelMapping(db *gorm.DB, req models.TelexSlackChannelMappingReq, userId string, orgId string) (models.TelexSlackChannelMapping, error) {
	var telexSlackChannelMapping models.TelexSlackChannelMapping

	telexSlackChannelMapping = models.TelexSlackChannelMapping{
		ID:               utility.GenerateUUID(),
		UserID:           userId,
		OrganisationID:   orgId,
		TelexChannelName: req.TelexChannelName,
		SlackChannelName: req.SlackChannelName,
	}

	err := telexSlackChannelMapping.Create(db)

	if err != nil {
		return telexSlackChannelMapping, err
	}

	return telexSlackChannelMapping, nil
}
