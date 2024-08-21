package newsletter

import (
	"strings"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"gorm.io/gorm"
)

func NewsLetterSubscribe(newsletter *models.NewsLetter, db *gorm.DB, extReq request.ExternalRequest) error {

	newsletter.Email = strings.ToLower(newsletter.Email)

	if err := newsletter.CreateNewsLetter(db, newsletter.Email); err != nil {
		return err
	}

	msgReq := models.SendNewsletterSubscriptionMail{
		Email: newsletter.Email,
	}

	err := actions.AddNotificationToQueue(storage.DB.Redis, names.SendNewsletterMail, msgReq)
	if err != nil {
		extReq.Logger.Error("Failed to send newsletter subscription email:", err)
	}

	return nil
}
