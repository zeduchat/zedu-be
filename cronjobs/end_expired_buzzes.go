package cronjobs

import (
	"errors"
	"time"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	buzzservice "github.com/hngprojects/telex_be/services/buzz"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func EndExpiredBuzzes(extReq request.ExternalRequest, db storage.Database) {
	now := time.Now().UTC()

	var expiredBuzzes []models.Buzz
	if err := db.Postgresql.Where(
		"status = ? AND is_live_status = ? AND buzz_end_time IS NOT NULL AND buzz_end_time < ?",
		models.BuzzStatusActive, true, now,
	).Find(&expiredBuzzes).Error; err != nil {
		extReq.Logger.Error("failed to fetch expired buzzes: %v", err)
		return
	}

	for _, buzz := range expiredBuzzes {
		if err := endBuzzBySystem(extReq, db, &buzz); err != nil {
			extReq.Logger.Error("failed to auto-end expired buzz %s: %v", buzz.ID, err)
			continue
		}
		extReq.Logger.Info("auto-ended expired buzz %s", buzz.ID)
	}
}

func endBuzzBySystem(extReq request.ExternalRequest, db storage.Database, buzz *models.Buzz) error {
	now := time.Now().UTC()

	err := db.Postgresql.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&buzz).Updates(map[string]interface{}{
			"status":         models.BuzzStatusEnded,
			"is_live_status": false,
			"updated_at":     now,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.BuzzParticipant{}).
			Where("buzz_id = ? AND status = ?", buzz.ID, models.BuzzParticipantStatusActive).
			Updates(map[string]interface{}{
				"status":  models.BuzzParticipantStatusLeft,
				"left_at": now,
			}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return errors.New("failed to end buzz in database")
	}

	if err := models.ExpireInvitationsForBuzz(db.Postgresql, buzz.ID); err != nil {
		extReq.Logger.Error("failed to expire invitations for buzz %s: %v", buzz.ID, err)
	}

	eventPayload := models.BuzzEventPayload{
		Event:          string(models.BuzzEnded),
		BuzzID:         buzz.ID,
		ChannelID:      buzz.ChannelID,
		HostID:         buzz.HostID,
		ParticipantIDs: []string{},
		CreatedAt:      now,
		Status:         models.BuzzStatusEnded,
	}

	notification := models.Notification[models.BuzzEnded]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(extReq.Logger, buzz.ChannelID, notification); err != nil {
		extReq.Logger.Error("failed to publish auto-ended buzz event: %v", err)
	}

	if err := buzzservice.CreateBuzzSystemMessage(&db, extReq.Logger, buzz, buzz.HostID, "ended"); err != nil {
		extReq.Logger.Error("failed to create system message for buzz end: %v", err)
	}

	return nil
}
