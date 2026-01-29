package cronjobs

import (
	"time"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

const (
	buzzWarningInterval = 5 * time.Minute
)

func CheckBuzzTimeWarnings(extReq request.ExternalRequest, db storage.Database) {
	now := time.Now().UTC()

	var buzzes []models.Buzz
	if err := db.Postgresql.Where("status = ? AND is_live_status = ? AND buzz_end_time IS NOT NULL",
		models.BuzzStatusActive, true).Find(&buzzes).Error; err != nil {
		extReq.Logger.Error("failed to fetch active buzzes: %v", err)
		return
	}

	for _, buzz := range buzzes {
		if buzz.BuzzEndTime == nil {
			continue
		}

		remaining := time.Until(*buzz.BuzzEndTime)

		warningMinutes := []int{60, 30, 10, 5, 1}
		for _, minutes := range warningMinutes {
			warningTime := (*buzz.BuzzEndTime).Add(-time.Duration(minutes) * time.Minute)

			if remaining > 0 &&
				remaining <= time.Duration(minutes)*time.Minute &&
				now.Sub(warningTime) < buzzWarningInterval/2 {

				sendTimeWarningNotification(extReq, &buzz, minutes)
				break
			}
		}
	}
}

func sendTimeWarningNotification(extReq request.ExternalRequest, buzz *models.Buzz, remainingMinutes int) {
	notification := models.Notification[models.BuzzTimeWarning]
	notification.SectionType = models.ChannelsSection
	notification.Content = models.BuzzTimeWarningPayload{
		Event:            string(models.BuzzTimeWarning),
		BuzzID:           buzz.ID,
		ChannelID:        buzz.ChannelID,
		HostID:           buzz.HostID,
		RemainingMinutes: remainingMinutes,
		EstimatedEndTime: *buzz.BuzzEndTime,
		Timestamp:        time.Now().UTC(),
	}
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	var publishChannel string
	if buzz.BuzzType == models.BuzzTypeOrganization {
		publishChannel = buzz.ID
	} else {
		publishChannel = buzz.ChannelID
	}

	if err := centrifuge.PublishChannel(extReq.Logger, publishChannel, notification); err != nil {
		extReq.Logger.Error("failed to send warning for buzz %s: %v", buzz.ID, err)
	}

	extReq.Logger.Info("sent %d minute warning for buzz %s",
		remainingMinutes, buzz.ID)
}
