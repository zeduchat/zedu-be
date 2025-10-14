package credentials

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

func CreateCredentialService(req models.CredentialRequest, db *gorm.DB) (int, error) {

	if _, err := req.CreateCredential(db); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusCreated, nil
}