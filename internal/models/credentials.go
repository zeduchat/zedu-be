package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/config"
	"gorm.io/gorm"
)


type Credential struct {
	ID               string                `gorm:"type:uuid;primaryKey" json:"id"`
	Name             string                `gorm:"type:text" json:"name"`
	OrgId		     string                `gorm:"type:uuid;index" json:"org_id"`
	AgentId		     string                `gorm:"type:uuid;index" json:"agent_id"`
	UserId		     string                `gorm:"type:uuid;index" json:"user_id"`
	SkillId 		 string                `gorm:"type:uuid;index" json:"skill_id"`
	Credentials      []byte                `gorm:"type:bytea" json:"credentials"`
	CreatedAt        time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
	UpdatedAt        time.Time             `gorm:"type:timestamp;default:current_timestamp" json:"-"`
}

type CredentialRequest struct {
	OrgId       string                 `json:"org_id" validate:"required"`
	AgentId     string                 `json:"agent_id" validate:"required"`
	UserId      string                 `json:"user_id" validate:"required"`
	SkillId     string                 `json:"skill_id" validate:"required"`
	Name        string                 `json:"name" validate:"required"`
	Credentials JSONBMap               `json:"credentials" validate:"required"`
}


type CredentialsResponse struct {
	ID          string                 `json:"id"`
	OrgId       string                 `json:"org_id" `
	AgentId     string                 `json:"agent_id"`
	UserId      string                 `json:"user_id"`
	SkillId     string                 `json:"skill_id"`
	Name        string                 `json:"name"`
	Credentials JSONBMap               `json:"credentials"`
}
type SkillCredentialsResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
}


func EncryptJSON(data interface{}, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, jsonBytes, nil)
	return ciphertext, nil
}


func DecryptJSON(ciphertext []byte, result interface{}) error {
	var config = config.GetConfig()
	var encryptionKey = []byte(config.Server.EncryptionKey)
	if len(encryptionKey) != 32 {
		return errors.New("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	jsonBytes, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}



func (cred *CredentialRequest) CreateCredential(db *gorm.DB) (int, error) {
	if err := ValidateAgentIDs(db, cred.OrgId, []string{cred.AgentId}); err != nil {
		return http.StatusBadRequest, err
	}

	if err := ValidateSkillIDs(db, cred.OrgId, []string{cred.SkillId}); err != nil {
		return http.StatusBadRequest, err
	}

	var config = config.GetConfig()
	var encryptionKey = []byte(config.Server.EncryptionKey)


	var existing Credential
	err := db.Where("org_id = ? AND agent_id = ? AND skill_id = ? AND name = ?", cred.OrgId, cred.AgentId, cred.SkillId, cred.Name).First(&existing).Error
	
	if err == nil {
		encrypted, err := EncryptJSON(cred.Credentials, encryptionKey)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to encrypt credential: %w", err)
		}

		existing.Credentials = encrypted
		existing.UpdatedAt = time.Now()

		if err := db.Save(&existing).Error; err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to update credential: %w", err)
		}

		return http.StatusOK, nil 
	} 


	encrypted, err := EncryptJSON(cred.Credentials, encryptionKey)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to encrypt credential: %w", err)
	}

	dbCredential := Credential{
		ID:          uuid.New().String(),
        OrgId:       cred.OrgId,
        AgentId:     cred.AgentId,
        UserId:      cred.UserId,
        SkillId:     cred.SkillId,
        Name:        cred.Name,
        Credentials: encrypted, 
    }


	if err := db.Create(&dbCredential).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create credential: %w", err)
	}


	return http.StatusCreated, nil
}


func (cred *Credential) GetSkillCredentials(db *gorm.DB) ([]SkillCredentialsResponse, int, error) {

	if err := ValidateSkillIDs(db, cred.OrgId, []string{cred.SkillId}); err != nil {
		return nil, http.StatusBadRequest, err
	}

    var dbCredentials []SkillCredentialsResponse
    err := db.Model(&Credential{}).
		Select("id, name").
        Where("user_id = ? AND org_id = ? AND skill_id = ?", cred.UserId, cred.OrgId, cred.SkillId).
        Find(&dbCredentials).Error
    
    if err != nil {
        return nil, http.StatusInternalServerError, err
    }

	if dbCredentials == nil {
        return []SkillCredentialsResponse{}, http.StatusOK, nil
    }

    return dbCredentials, http.StatusOK, nil
}


func (cred *Credential) GetCredentialByID(db *gorm.DB) (*CredentialsResponse, int, error) {
	res := CredentialsResponse{}

    var dbCredential Credential
    err := db.Model(&Credential{}).
        Where("id = ?", cred.ID).
        First(&dbCredential).Error
    
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return &res, http.StatusNotFound, fmt.Errorf("credential not found")
        }
        return &res, http.StatusInternalServerError, err
    }

    var decryptedCreds JSONBMap 
    if err := DecryptJSON(dbCredential.Credentials, &decryptedCreds); err != nil {
        return &res, http.StatusInternalServerError, fmt.Errorf("failed to decrypt credentials: %w", err)
    }

    res.ID = dbCredential.ID
    res.Name = dbCredential.Name
    res.OrgId = dbCredential.OrgId
    res.AgentId = dbCredential.AgentId
    res.UserId = dbCredential.UserId
    res.SkillId = dbCredential.SkillId
    res.Credentials = decryptedCreds 

    return &res, http.StatusOK, nil
}