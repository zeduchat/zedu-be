package notifications

import (
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

type NotificationObject struct {
	Notification *models.NotificationRecord
	ExtReq       request.ExternalRequest
	rdb          *redis.Client
	Db           *gorm.DB
}

func NewNotificationObject(extReq request.ExternalRequest, rdb *redis.Client, db *gorm.DB, notification *models.NotificationRecord) *NotificationObject {
	return &NotificationObject{
		ExtReq:       extReq,
		rdb:          rdb,
		Db:           db,
		Notification: notification,
	}
}

func ConvertToMapAndAddExtraData(data any, newData map[string]any) (map[string]any, error) {
	var (
		mapData map[string]any
	)

	mapData, err := utility.StructToMap(data)
	if err != nil {
		return mapData, err
	}

	for key, value := range newData {
		mapData[key] = value
	}

	return mapData, nil
}

func thisOrThatStr(this, that string) string {
	if this == "" {
		return that
	}
	return this
}
