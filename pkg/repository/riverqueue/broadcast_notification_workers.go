package riverqueueBg

import (
	"context"
	"runtime/debug"

	"github.com/hngprojects/telex_be/internal/models"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/utility"
	"github.com/riverqueue/river"
	"gorm.io/gorm"
)

type BroadcastNotificationWorker struct {
	logger *utility.Logger
	db     *gorm.DB
	river.WorkerDefaults[models.BroadcastNotificationJobArgs]
}

func (w *BroadcastNotificationWorker) Work(ctx context.Context, job *river.Job[models.BroadcastNotificationJobArgs]) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("Recovered in BroadcastNotificationWorker Work from panic: %v", r)
			debug.PrintStack()
		}
	}()

	args := job.Args
	w.logger.Info("Processing BroadcastNotificationJob for BroadcastLogID: %s", args.BroadcastLogID)

	// Update status to processing
	if err := w.db.Model(&models.BroadcastNotificationLog{}).
		Where("id = ?", args.BroadcastLogID).
		Update("status", "processing").Error; err != nil {
		w.logger.Error("Failed to update broadcast log status to processing: %s", err.Error())
		return err
	}

	// Fetch users in batches
	batchSize := 1000
	offset := 0
	totalSent := 0
	totalFailed := 0

	for {
		var users []models.User
		if err := w.db.Where("is_active = ?", true).
			Select("id", "onesignal_subscription_id").
			Offset(offset).
			Limit(batchSize).
			Find(&users).Error; err != nil {
			w.logger.Error("Failed to fetch users batch: %s", err.Error())
			// Update status to failed
			w.db.Model(&models.BroadcastNotificationLog{}).
				Where("id = ?", args.BroadcastLogID).
				Updates(map[string]interface{}{
					"status":            "failed",
					"successfully_sent": totalSent,
					"failed_count":      totalFailed + len(users), // approximate
				})
			return err
		}

		if len(users) == 0 {
			break
		}

		// Build subscription IDs for this batch
		subscriptionIDs := make([]string, 0, len(users))
		for _, user := range users {
			if user.OneSignalSubscriptionID != "" {
				subscriptionIDs = append(subscriptionIDs, user.OneSignalSubscriptionID)
			}
		}

		if len(subscriptionIDs) > 0 {
			pushReq := models.PushRequest{
				Title:     args.Title,
				Message:   args.Message,
				AvatarUrl: args.AvatarUrl,
			}

			if err := push_notifications.PushOneSignalToUsersForBroadcast(pushReq, w.logger, w.db, subscriptionIDs); err != nil {
				w.logger.Error("Failed to send broadcast notification batch: %s", err.Error())
				totalFailed += len(subscriptionIDs)
			} else {
				totalSent += len(subscriptionIDs)
			}
		}

		offset += batchSize

		// Update progress
		w.db.Model(&models.BroadcastNotificationLog{}).
			Where("id = ?", args.BroadcastLogID).
			Updates(map[string]interface{}{
				"successfully_sent": totalSent,
				"failed_count":      totalFailed,
			})
	}

	// Update final status
	finalStatus := "completed"
	if totalFailed > 0 && totalSent == 0 {
		finalStatus = "failed"
	}

	if err := w.db.Model(&models.BroadcastNotificationLog{}).
		Where("id = ?", args.BroadcastLogID).
		Updates(map[string]interface{}{
			"status":               finalStatus,
			"successfully_sent":    totalSent,
			"failed_count":         totalFailed,
			"total_users_targeted": totalSent + totalFailed,
		}).Error; err != nil {
		w.logger.Error("Failed to update final broadcast log status: %s", err.Error())
		return err
	}

	w.logger.Info("Completed BroadcastNotificationJob for BroadcastLogID: %s, sent: %d, failed: %d", args.BroadcastLogID, totalSent, totalFailed)

	return nil
}
