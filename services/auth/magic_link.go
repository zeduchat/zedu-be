package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/audit_utility"
)

func MagicLinkRequest(userEmail, url string, db *gorm.DB) (string, int, error) {

	var (
		user      = models.User{}
		magicLink = models.MagicLink{}
		config    = config.GetConfig()
	)

	magicExist, err := magicLink.GetMagicLinkByEmail(db, userEmail)
	if err != nil {
		return "error", http.StatusUnauthorized, err
	}

	if magicExist != nil {
		if err := magicExist.DeleteMagicLink(db); err != nil {
			return "error", http.StatusInternalServerError, err
		}
	}

	exists := postgresql.CheckExists(db, &user, "email = ?", userEmail)
	if !exists {
		return "error", http.StatusNotFound, fmt.Errorf("user not found")
	}

	resetToken, err := utility.GenerateOTP(6)

	if err != nil {
		return "error", http.StatusInternalServerError, err
	}

	magic := models.MagicLink{
		ID:        utility.GenerateUUID(),
		Email:     strings.ToLower(userEmail),
		Token:     resetToken,
		ExpiresAt: time.Now().Add(time.Duration(config.App.ResetPasswordDuration) * time.Minute),
	}

	err = magic.CreateMagicLink(db)
	if err != nil {
		return "error", http.StatusInternalServerError, err
	}

	magic_link := fmt.Sprintf("%vauth/login/magic-link?token=%v", url, resetToken)

	resetReq := models.SendMagicLink{
		Email:     userEmail,
		MagicLink: magic_link,
	}

	err = actions.AddNotificationToQueue(storage.DB.Redis, names.SendMagicLink, resetReq)
	if err != nil {
		return "error", http.StatusInternalServerError, err
	}

	return "success", http.StatusOK, nil
}

func VerifyMagicLinkToken(req models.VerifyMagicLinkRequest, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest) (gin.H, int, error) {

	var (
		user         = models.User{}
		responseData gin.H
		magicLink    = models.MagicLink{}
		org          models.Organisation
	)

	magicExist, err := magicLink.GetMagicLinkByToken(db, req.Token)
	if err != nil {
		return responseData, http.StatusUnauthorized, errors.New("invalid or expired token")
	}

	exists := postgresql.CheckExists(db, &user, "email = ?", magicExist.Email)
	if !exists {
		return responseData, http.StatusBadRequest, errors.New("invalid credentials")
	}

	userData, err := user.GetUserByEmail(db, magicExist.Email)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("unable to fetch user")
	}

	tokenData, err := middleware.CreateToken(user, c)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("error saving token")
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	access_token := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: user.ID}

	err = access_token.CreateAccessToken(db, tokens)

	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("error saving token")
	}

	if err := magicExist.DeleteMagicLink(db); err != nil {
		return responseData, http.StatusInternalServerError, err
	}

	org, _ = org.GetOrgByID(db, userData.CurrentOrg.String())

	responseData = gin.H{
		"user": map[string]any{
			"id":                        userData.ID,
			"email":                     userData.Email,
			"username":                  userData.Name,
			"is_verified":               userData.IsVerified,
			"is_onboarded":              userData.IsOnboarded,
			"profile_updated":           userData.ProfileUpdated,
			"is_active":                 userData.IsActive,
			"current_org":               userData.CurrentOrg,
			"current_organisation_slug": slug.Make(org.Name),
			"first_name":                userData.Profile.FirstName,
			"last_name":                 userData.Profile.LastName,
			"fullname":                  userData.Profile.FirstName + " " + userData.Profile.LastName,
			"phone":                     userData.Profile.Phone,
			"avatar_url":                userData.Profile.AvatarURL,
			"expires_in":                strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at":                strconv.Itoa(int(userData.CreatedAt.Unix())),
			"updated_at":                strconv.Itoa(int(userData.UpdatedAt.Unix())),
		},
		"access_token":            tokenData.AccessToken,
		"notification_token":      access_token.SubAccessToken,
		"access_token_expires_in": strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	audit_utility.LogUserLogin(c, db, extReq, user.ID, tokenData.AccessUuid, user.Organisations)

	return responseData, http.StatusOK, nil
}
