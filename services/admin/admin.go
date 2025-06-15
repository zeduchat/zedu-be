package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func LoginAdmin(req models.LoginRequestModel, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest) (gin.H, int, error) {
	var (
		user         = models.User{}
		responseData gin.H
	)

	exists := postgresql.CheckExists(db, &user, "email = ?", req.Email)
	if !exists {
		return responseData, 400, fmt.Errorf("invalid credentials")
	}

	if !utility.CompareHash(req.Password, user.Password) {
		return responseData, 400, fmt.Errorf("invalid credentials")
	}

	tokenData, err := middleware.CreateToken(user, c)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	access_token := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: user.ID}

	err = access_token.CreateAccessToken(db, tokens)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}

	responseData = gin.H{
		"user": map[string]interface{}{
			"id":              user.ID,
			"email":           user.Email,
			"username":        user.Name,
			"is_verified":     user.IsVerified,
			"is_onboarded":    user.IsOnboarded,
			"profile_updated": user.ProfileUpdated,
			"is_active":       user.IsActive,
			"current_org":     user.CurrentOrg,
			"first_name":      user.Profile.FirstName,
			"last_name":       user.Profile.LastName,
			"fullname":        user.Profile.FirstName + " " + user.Profile.LastName,
			"phone":           user.Profile.Phone,
			"avatar_url":      user.Profile.AvatarURL,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusOK, nil
}
