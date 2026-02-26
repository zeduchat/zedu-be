package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type TerminateSessionRequest struct {
	UserID            *string `json:"user_id"`
	GlobalTermination bool    `json:"global_termination"`
	AccessToken       *string `json:"access_token_id"`
}

type AccessToken struct {
	ID                        string    `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	OwnerID                   string    `gorm:"column:owner_id; type:uuid; not null" json:"owner_id"`
	IsLive                    bool      `gorm:"column:is_live; type:bool; default:false; not null" json:"is_live"`
	LoginAccessToken          string    `gorm:"column:login_access_token; type:text" json:"-"`
	LoginAccessTokenExpiresIn string    `gorm:"column:login_access_token_expires_in; type:varchar(250)" json:"-"`
	CreatedAt                 time.Time `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt                 time.Time `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
	SubAccessToken            string    `gorm:"column:sub_access_token; type:text" json:"-"`
	IsLiveNotificaionToken    bool      `gorm:"column:is_live_notification_token; type:bool; default:false; not null" json:"is_live_notification_token"`
}

func (a *AccessToken) GetAccessTokens(db *gorm.DB) error {
	err := postgresql.SelectFirstFromDb(db, &a)
	if err != nil {
		return fmt.Errorf("token selection failed: %v", err.Error())
	}
	return nil
}

func (a *AccessToken) GetMostRecentAccessToken(db *gorm.DB) (int, error) {
	err, nilErr := postgresql.SelectLatestFromDb(db, &a, "owner_id = ? and is_live = ?", a.OwnerID, true)
	if nilErr != nil {
		return http.StatusBadRequest, nilErr
	}

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to retrieve access token: %v", err)
	}
	return http.StatusOK, nil
}

func (a *AccessToken) GetByOwnerID(db *gorm.DB) (int, error) {
	err, nilErr := postgresql.SelectOneFromDb(db, &a, "owner_id = ? ", a.OwnerID)
	if nilErr != nil {
		return http.StatusBadRequest, nilErr
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (a *AccessToken) GetBySubToken(db *gorm.DB) (int, error) {
	err, nilErr := postgresql.SelectOneFromDb(db, &a, "sub_access_token = ? ", a.SubAccessToken)
	if nilErr != nil {
		return http.StatusBadRequest, nilErr
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (a *AccessToken) GetByID(db *gorm.DB) (int, error) {
	err, nilErr := postgresql.SelectOneFromDb(db, &a, "id = ? ", a.ID)
	if nilErr != nil {
		return http.StatusBadRequest, nilErr
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (a *AccessToken) GetLatestByOwnerIDAndIsLive(db *gorm.DB) (int, error) {
	err, nilErr := postgresql.SelectLatestFromDb(db, &a, "owner_id = ? and is_live = ? ", a.OwnerID, a.IsLive)
	if nilErr != nil {
		return http.StatusBadRequest, nilErr
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (a *AccessToken) CreateAccessToken(db *gorm.DB, tokenData any) error {
	if a.OwnerID == "" {
		return fmt.Errorf("owner id not provided to create access token")
	}

	if a.ID == "" {
		return fmt.Errorf("access id not provided to create access token")
	}

	var (
		access_token = tokenData.(map[string]string)["access_token"]
		exp          = tokenData.(map[string]string)["exp"]
	)

	a.IsLive = true
	a.LoginAccessToken = access_token
	a.LoginAccessTokenExpiresIn = exp
	a.SubAccessToken = randstr.String(32)
	a.IsLiveNotificaionToken = true
	err := postgresql.CreateOneRecord(db, &a)
	if err != nil {
		return fmt.Errorf("user creation failed: %v", err.Error())
	}
	return nil
}

func (a *AccessToken) RevokeAccessToken(db *gorm.DB) error {
	if a.ID == "" {
		return fmt.Errorf("access token id not provided to revoke access token")
	}
	a.IsLive = false

	_, err := postgresql.SaveAllFields(db, &a)
	return err
}

func (a *AccessToken) RevokeAccessTokenDelete(db *gorm.DB) error {
	if a.ID == "" {
		return fmt.Errorf("access token id not provided to revoke access token")
	}

	if err := postgresql.HardDeleteRecordFromDb(db, a); err != nil {
		return err
	}

	return nil
}

func (a *AccessToken) RevokeUserTokens(db *gorm.DB) (int, error) {
	statusCode, err := a.GetLatestByOwnerIDAndIsLive(db)
	if err != nil {
		return statusCode, fmt.Errorf("failed to retrieve user tokens: %v", err)
	}

	if err := a.RevokeAccessToken(db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to revoke user token: %v", err)
	}
	return http.StatusOK, nil
}

// RevokeTokensByUserIDs marks all live tokens owned by the given user IDs as revoked.
func (a *AccessToken) RevokeTokensByUserIDs(db *gorm.DB, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	err := db.Model(&AccessToken{}).
		Where("owner_id IN ? AND is_live = ?", userIDs, true).
		Update("is_live", false).Error
	if err != nil {
		return fmt.Errorf("failed to revoke tokens for users: %v", err)
	}
	return nil
}

func (a *AccessToken) RevokeAllTokens(db *gorm.DB) error {
	err := db.Model(&a).
		Where("is_live = ?", true).
		Update("is_live", false).Error
	if err != nil {
		return fmt.Errorf("failed to revoke all tokens: %v", err)
	}
	return nil
}

func (a *AccessToken) FetchAllUserSessions(db *gorm.DB, c *gin.Context, userID, requesterID string, isSuperAdmin bool) ([]AccessToken, postgresql.PaginationResponse, error) {
	var (
		sessions    []AccessToken
		ErrNotFound = errors.New("sessions not found")
	)

	var isOwner bool
	err := db.Model(&Organisation{}).
		Select("count(*) > 0").
		Where("owner_id = ? AND id IN (SELECT organisation_id FROM user_organisations WHERE user_id = ?)", requesterID, userID).
		Find(&isOwner).
		Error
	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	query := db.Model(&AccessToken{})
	if isOwner || isSuperAdmin {
		query = query.Where("owner_id = ?", userID)
	} else if requesterID == userID {
		query = query.Where("owner_id = ? AND owner_id = ?", userID, requesterID)
	} else {
		return nil, postgresql.PaginationResponse{}, ErrNotFound
	}

	pagination := postgresql.GetPagination(c)
	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&sessions,
		nil,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paginationResponse, ErrNotFound
		}
		return nil, paginationResponse, err
	}

	return sessions, paginationResponse, nil
}

func (a *AccessToken) GetAccessTokenByID(db *gorm.DB, tokenID string) (AccessToken, error) {

	query := db.Where("id = ?", tokenID)

	err := query.First(&a).Error

	if err != nil {
		return *a, err
	}

	return *a, nil
}

func (a *User) GetLatestAccessTokenByUserID(db *gorm.DB, userID string) (AccessToken, error) {
	var latestAccessToken AccessToken

	err := db.Where("owner_id = ?", userID).
		Order("created_at DESC").
		First(&latestAccessToken).
		Error

	if err != nil {
		return latestAccessToken, err
	}

	return latestAccessToken, nil
}
