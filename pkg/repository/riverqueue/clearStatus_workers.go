package riverqueueBg

import (
	"context"
	"runtime/debug"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/utility"
	"github.com/riverqueue/river"
	"gorm.io/gorm"
)

type ClearUserStatusWorker struct {
	logger *utility.Logger
	db     *gorm.DB
	river.WorkerDefaults[models.ClearUserStatusJobArgs]
}

func NewClearUserStatusWorker(logger *utility.Logger, db *gorm.DB) *ClearUserStatusWorker {
	return &ClearUserStatusWorker{
		logger: logger,
		db:     db,
	}
}

func (w *ClearUserStatusWorker) Work(ctx context.Context, job *river.Job[models.ClearUserStatusJobArgs]) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("Recovered in ClearUserStatusWorker Work from panic: %v", r)
			debug.PrintStack()
		}
	}()

	w.logger.Info("Processing ClearUserStatusJob for UserID: %s", job.Args.UserID)

	var profile models.Profile
	var err error
	if job.Args.OrgID != "" {
		profile, err = profile.GetOrCreateProfileForOrg(w.db, job.Args.UserID, job.Args.OrgID, w.logger)
	} else {
		err = w.db.Where("userid = ?", job.Args.UserID).First(&profile).Error
	}
	if err != nil {
		w.logger.Error("Failed to fetch profile for user %s (org: %s): %v", job.Args.UserID, job.Args.OrgID, err)
		return err
	}

	updates := map[string]any{
		"text":           "",
		"icon":           "",
		"status_timeout": "",
		"river_job_id":   nil,
	}

	if err := w.db.Model(&profile).Updates(updates).Error; err != nil {
		w.logger.Error("Failed to clear status for user %s: %v", job.Args.UserID, err)
		return err
	}

	notification := models.Notification[models.ProfileStatusUpdated]
	notification.SectionType = models.ChannelsSection
	notification.NotificationId = utility.GenerateUUID()
	notification.ModificationDetails = &models.ModificationDetails{
		UserId: job.Args.UserID,
	}
	notification.Content = struct {
		UserID string            `json:"user_id"`
		Status models.UserStatus `json:"status"`
	}{
		UserID: job.Args.UserID,
		Status: models.UserStatus{
			Text:       "",
			Emoji:      "",
			Expiry:     0,
			Visibility: "public",
			Online:     profile.Online,
		},
	}

	channelID := job.Args.OrgID
	if err := centrifuge.PublishChannel(w.logger, channelID, notification); err != nil {
		w.logger.Error("Failed to publish status cleared event", "error", err, "channel_id", channelID)
	}

	w.logger.Info("Successfully cleared status for user %s and published to centrifugo", job.Args.UserID)
	return nil
}
