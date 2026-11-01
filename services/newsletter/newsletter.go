package newsletter

import (
	"errors"
	"strings"
	"time"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

var (
	ErrInvalidUnsubscribeToken = errors.New("invalid or expired unsubscribe link")
	ErrInvalidConfirmToken     = errors.New("invalid or expired confirmation link")
)

// confirmationResendInterval stops someone from using repeat sign-ups to
// flood an inbox with confirmation emails.
const confirmationResendInterval = 10 * time.Minute

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

func queueNewsletterMail(name names.NotificationName, msg any, extReq request.ExternalRequest) {
	if err := actions.AddNotificationToQueue(storage.DB.Redis, name, msg); err != nil {
		extReq.Logger.Error("Failed to queue newsletter email:", err)
	}
}

// NewsLetterSubscribe stores the address as pending and emails a confirmation
// link. The caller gets the same answer whether or not the address is already on
// the list, so a signup cannot be used to find out who is subscribed.
func NewsLetterSubscribe(newsletter *models.NewsLetter, db *gorm.DB, extReq request.ExternalRequest) error {
	email := strings.ToLower(strings.TrimSpace(newsletter.Email))

	var existing models.NewsLetter
	if postgresql.CheckExists(db, &existing, "email = ?", email) {
		if existing.Pending && time.Since(existing.UpdatedAt) > confirmationResendInterval {
			if err := db.Model(&models.NewsLetter{}).Where("email = ?", email).Update("pending", true).Error; err != nil {
				return err
			}
			queueNewsletterMail(names.SendNewsletterConfirmMail, models.SendNewsletterConfirmationMail{Email: email}, extReq)
		}
		return nil
	}

	newsletter.Email = email
	newsletter.Pending = true

	if err := newsletter.CreateNewsLetter(db, email); err != nil {
		// Another request stored the same address first.
		if err.Error() == "email already subscribed" {
			return nil
		}
		return err
	}

	queueNewsletterMail(names.SendNewsletterConfirmMail, models.SendNewsletterConfirmationMail{Email: email}, extReq)
	return nil
}

// NewsLetterConfirm marks a pending subscriber as confirmed when the token
// matches the email, then sends the welcome email once. A repeated click or an
// unknown address succeeds without doing anything.
func NewsLetterConfirm(req *models.ConfirmNewsLetterRequest, db *gorm.DB, extReq request.ExternalRequest) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if !utility.ValidNewsletterConfirmToken(email, req.Token, config.GetConfig().Server.Secret) {
		return ErrInvalidConfirmToken
	}

	var newsletter models.NewsLetter
	confirmed, err := newsletter.ConfirmNewsLetterByEmail(db, email)
	if err != nil {
		return err
	}

	if confirmed {
		queueNewsletterMail(names.SendNewsletterMail, models.SendNewsletterSubscriptionMail{Email: email}, extReq)
	}
	return nil
}
