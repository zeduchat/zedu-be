package actions

import (
	"encoding/json"

	"github.com/go-redis/redis/v8"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions/names"
)

func AddNotificationToQueue(rdb *redis.Client, name names.NotificationName, data any) error {
	if storage.DB != nil && storage.DB.IsTest {
		return nil
	}
	dataByte, err := json.Marshal(data)
	if err != nil {
		return err
	}

	notificationRecord := models.NotificationRecord{
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

func AddPushNotificationToQueue(rdb *redis.Client, notificationRecord models.PushNotificationRecord) error {
	if storage.DB != nil && storage.DB.IsTest {
		return nil
	}

	err := notificationRecord.PushToQueue(rdb)

	if err != nil {
		return err
	}

	return nil
}
