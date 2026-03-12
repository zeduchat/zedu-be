package account

import (
	"errors"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

func CreateAccountDeletionRequest(db *gorm.DB, req models.CreateAccountDeletionRequest, orgID string) (*models.AccountDeletionRequest, error) {
	deletionRequest, err := req.ToAccountDeletionRequest(orgID)
	if err != nil {
		return nil, errors.New("failed to generate request ID")
	}

	err = postgresql.CreateOneRecord(db, deletionRequest)
	if err != nil {
		return nil, err
	}

	return deletionRequest, nil
}
