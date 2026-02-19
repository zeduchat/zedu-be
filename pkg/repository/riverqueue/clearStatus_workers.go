package riverqueueBg

import (
	"context"
	"runtime/debug"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"github.com/riverqueue/river"
	"gorm.io/gorm"
)

type ClearUserStatusWorker struct {
	logger *utility.Logger
	db     *gorm.DB
	river.WorkerDefaults[models.ClearUserStatusJobArgs]
}

func (w *ClearUserStatusWorker) Work(ctx context.Context, job *river.Job[models.ClearUserStatusJobArgs]) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("Recovered in ClearUserStatusWorker Work from panic: %v", r)
			debug.PrintStack()
		}
	}()

	w.logger.Info("Processing ClearUserStatusJob for UserID: %s", job.Args.UserID)

	updates := map[string]any{
		"text":           "",
		"icon":           "",
		"status_timeout": "",
		"river_job_id":   nil,
	}

	if err := w.db.Model(&models.Profile{}).
		Where("userid = ?", job.Args.UserID).
		Updates(updates).Error; err != nil {
		w.logger.Error("Failed to clear status for user %s: %v", job.Args.UserID, err)
		return err
	}

	w.logger.Info("Successfully cleared status for user %s", job.Args.UserID)
	return nil
}
