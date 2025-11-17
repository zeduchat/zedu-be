package waitlist

import (
	"strings"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
)

func WaitlistLetterSubscribe(req *models.WaitlistRequest, db *gorm.DB, extReq request.ExternalRequest) error {

	var waitlist models.Waitlist

	req.Email = strings.ToLower(req.Email)

	if err := waitlist.CreateWaitlist(db, req.Email); err != nil {
		return err
	}

	msgReq := models.SendWaitlistletterSubscriptionMail{
		Email: req.Email,
	}

	err := actions.AddNotificationToQueue(storage.DB.Redis, names.SendWaitListLetterMail, msgReq)
	if err != nil {
		extReq.Logger.Error("Failed to send newsletter subscription email:", err)
	}

	return nil
}
