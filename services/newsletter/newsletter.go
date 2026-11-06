package newsletter

import (
	"errors"
	"strings"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

var ErrInvalidUnsubscribeToken = errors.New("invalid or expired unsubscribe link")

// NewsLetterUnsubscribe removes the subscriber when the token matches the email.
// It succeeds when the address is not on the list, so the answer does not reveal
// who is subscribed.
func NewsLetterUnsubscribe(req *models.UnsubscribeNewsLetterRequest, db *gorm.DB) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if !utility.ValidNewsletterUnsubscribeToken(email, req.Token, config.GetConfig().Server.Secret) {
		return ErrInvalidUnsubscribeToken
	}

	var newsletter models.NewsLetter
	return newsletter.DeleteNewsLetterByEmail(db, email)
}

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
