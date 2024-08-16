package cronjobs

import (
	"strings"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
)

func SendNotifications(extReq request.ExternalRequest, db storage.Database) {
	notificationRecord := models.NotificationRecord{}

	res, err := notificationRecord.PopFromQueue(db.Redis)

	if err != nil {
		if strings.Contains(err.Error(), "could not pop from Redis queue: redis: nil") {
			return
		}
		extReq.Logger.Error("error getting notification records: ", err.Error())
		return
	}

	extReq.Logger.Info("Sending records found: ", res)

	err = actions.Send(extReq, db.Postgresql, db.Redis, &res)

	if err != nil {
		extReq.Logger.Error("error sending mail: ", err.Error())
		return
	}
}
