package thread

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/utility"
)

func ProcessThreadUnseenForParticipants(db *gorm.DB, logger *utility.Logger, orgID, threadID string, participantIDs []string) error {
	newlyUnseenUserIDs, err := models.MarkThreadUnseenForParticipants(db, orgID, threadID, participantIDs)
	if err != nil {
		return err
	}

	if len(newlyUnseenUserIDs) > 0 {
		userChannels := make([]string, len(newlyUnseenUserIDs))
		for i, uID := range newlyUnseenUserIDs {
			userChannels[i] = fmt.Sprintf("%s/%s", orgID, uID)
		}

		triggerNotif := models.Notification[models.TriggerNotification]
		triggerNotif.SectionType = models.OrgThreadsSection
		triggerNotif.ModificationDetails = &models.ModificationDetails{
			OrgId:    orgID,
			ThreadId: threadID,
		}
		triggerNotif.Content = models.TriggerNotificationPayload{
			TriggerAction:   models.RefreshTriggerAction,
			TargetComponent: models.ThreadsTargetComponent,
		}
		triggerNotif.NotificationId = utility.GenerateUUID()

		if batchErr := centrifuge.BatchBroadcastToChannel(logger, userChannels, triggerNotif); batchErr != nil {
			logger.Error("failed to batch broadcast org_threads trigger notification: %v", batchErr)
		}
	}

	return nil
}

func ProcessThreadSeenForUser(db *gorm.DB, logger *utility.Logger, orgID, userID, threadID string) error {
	wasUnseen, err := models.MarkThreadSeenForUser(db, userID, threadID)
	if err != nil {
		return err
	}

	if wasUnseen {
		triggerNotif := models.Notification[models.TriggerNotification]
		triggerNotif.SectionType = models.OrgThreadsSection
		triggerNotif.ModificationDetails = &models.ModificationDetails{
			OrgId:    orgID,
			UserId:   userID,
			ThreadId: threadID,
		}
		triggerNotif.Content = models.TriggerNotificationPayload{
			TriggerAction:   models.RefreshTriggerAction,
			TargetComponent: models.ThreadsTargetComponent,
		}
		triggerNotif.NotificationId = utility.GenerateUUID()

		userChannel := fmt.Sprintf("%s/%s", orgID, userID)
		if pubErr := centrifuge.PublishChannel(logger, userChannel, triggerNotif); pubErr != nil {
			logger.Error("failed to publish org_threads trigger notification to %s: %v", userChannel, pubErr)
		}
	}

	return nil
}
