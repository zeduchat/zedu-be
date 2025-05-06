package notification_processor

import (
	"encoding/json"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
)

func AddNotificationToQueue(rdb *redis.Client, name names.NotificationName, data interface{}) error {
	dataByte, err := json.Marshal(data)
	if err != nil {
		return err
	}

	notificationRecord := models.PushNotificationRecord{
		Name: string(name),
		Data: string(dataByte),
		Sent: false,
	}

	err = notificationRecord.PushToQueue(rdb)
	if err != nil {
		return err
	}

	return nil
}

func ProcessNotification(db *gorm.DB, rec models.PushNotificationRecord, logger *utility.Logger) error {

	var err error

	if rec.Name == string(names.SendPushNotification) {

		err = DMNotification(db, logger, rec)
	} else if rec.Name == string(names.SendMassPushNotification) {

		err = ChannelNotification(db, logger, rec)
	}

	return err

}

func ChannelNotification(db *gorm.DB, logger *utility.Logger, rec models.PushNotificationRecord) error {

	channel_query := ""
	mention_query := ""

	return nil
}

func DMNotification(db *gorm.DB, logger *utility.Logger, rec models.PushNotificationRecord) error {
	return nil
}

func FetchUsersEmails(db *gorm.DB, logger *utility.Logger) {}
