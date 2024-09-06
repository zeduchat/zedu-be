package optin

import (
	"net/http"
	"strings"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateOptInRecord(req models.CreateOptIn, db *gorm.DB) (int, error) {
	req.Email = strings.ToLower(req.Email)

	OptIn := models.OptIn{
		ID:         utility.GenerateUUID(),
		Email:      req.Email,
		LastName:   req.LastName,
		FirstName:  req.FirstName,
	}

	code, err := OptIn.CreateOptInRecord(db, req.Email);
	if err != nil {
		return code, err
	}

	optInReq := models.SendSqueeze{
		Email:     req.Email,
		LastName:  req.LastName,
		FirstName: req.FirstName,
	}

	err = actions.AddNotificationToQueue(storage.DB.Redis, names.SendSqueeze, optInReq)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return code, nil
}