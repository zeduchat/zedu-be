package contact

import (
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func AddToContactUs(contact models.ContactUs, db *gorm.DB, extReq request.ExternalRequest) error {

	contact.Message = utility.CleanStringInput(contact.Message)

	if err := contact.CreateContactUs(db); err != nil {
		return err
	}

	msgReq := models.SendContactUsMail{
		Email:       contact.Email,
		Name:        contact.Name,
		PhoneNumber: contact.PhoneNumber,
		Message:     contact.Message,
	}

	err := actions.AddNotificationToQueue(storage.DB.Redis, names.SendContactUsMail, msgReq)
	if err != nil {
		extReq.Logger.Error("Failed to send welcome email:", err)
	}

	return nil
}
