package notification_processor

import (
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/utility"
)






func GetUserNotificationPrefence(db *gorm.DB, logger *utility.Logger){}

func FetchUsersEmails(db *gorm.DB, logger *utility.Logger) {}

func DequeNotifications(){}

func EnqueueNotifications(){}
