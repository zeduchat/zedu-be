package cronjobs

import (
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/campaign"
)

func CleanCampaignEvents(extReq request.ExternalRequest, db storage.Database) {
	if err := campaign.PurgeExpired(db.Postgresql); err != nil {
		extReq.Logger.Error("campaign retention failed", err)
	}
}
