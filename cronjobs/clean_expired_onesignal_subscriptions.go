package cronjobs

import (
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

func CleanExpiredOneSignalNotifications(extReq request.ExternalRequest, db storage.Database) {
	extReq.Logger.Info("Running cron job to clean up expired OneSignal notifications")
	err := models.DeleteExpiredNotifications(db.Postgresql)
	if err != nil {
		extReq.Logger.Error("Error cleaning up expired OneSignal notifications: %v", err)
	} else {
		extReq.Logger.Info("Successfully cleaned up expired OneSignal notifications")
	}
}
