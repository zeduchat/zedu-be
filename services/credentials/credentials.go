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

func GetSkillCredentialsService(req models.CredentialRequest, db *gorm.DB) ([]models.SkillCredentialsResponse, int, error) {
	var cred models.Credential
	cred.OrgId = req.OrgId
	cred.UserId = req.UserId
	cred.SkillId = req.SkillId

	res, code, err := cred.GetSkillCredentials(db)
	return res, code, err
}

func GetCredentialByIDService(credentialId string, db *gorm.DB) (*models.CredentialsResponse, int, error) {
	var cred models.Credential
	cred.ID = credentialId

	res, code, err := cred.GetCredentialByID(db)
	return res, code, err
}
