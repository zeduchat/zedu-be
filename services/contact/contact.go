package contact

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func GetAllContactUs(c *gin.Context, db *gorm.DB) ([]models.ContactUs, *postgresql.PaginationResponse, int, error) {

	var contact models.ContactUs

	contacts, paginationResponse, err := contact.FetchAllContactUs(db, c)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return contacts, nil, http.StatusNoContent, nil
		}
		return contacts, nil, http.StatusBadRequest, err

	}

	return contacts, &paginationResponse, http.StatusOK, nil

}

func AddToContactUs(contact *models.ContactUs, db *gorm.DB) error {

	contact.Message = utility.CleanStringInput(contact.Message)

	if err := contact.CreateContactUs(db); err != nil {
		return err
	}

	msgReq := models.ContactUs{
		Email:   contact.Email,
		Name:    contact.Name,
		Message: contact.Message,
	}

	err := actions.AddNotificationToQueue(storage.DB.Redis, names.SendContactUsMail, msgReq)
	if err != nil {
		return err
	}

	return nil
}

func DeleteContactUs(ID string, db *gorm.DB) (int, error) {
	var (
		contact models.ContactUs
	)

	contact, err := contact.GetContactUsById(db, ID)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if err := contact.DeleteContactUs(db); err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func GetContactUsById(ID string, db *gorm.DB) (*models.ContactUs, error) {

	var contact models.ContactUs

	contactData, err := contact.GetContactUsById(db, ID)
	if err != nil {
		return nil, err
	}

	return &contactData, nil
}

func GetContactUsByEmail(email string, db *gorm.DB) (*[]models.ContactUs, error) {

	var contact models.ContactUs

	contactData, err := contact.GetContactUsByEmail(db, email)
	if err != nil {
		return nil, err
	}

	return &contactData, nil
}
