package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/gosimple/slug"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/audit_utility"
)

func CreateGoogleUser(req models.GoogleRequestModel, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest, logger *utility.Logger) (gin.H, int, error) {

	var (
		userClaims   map[string]any
		reqUser      models.CreateUserRequestModel
		sendWelcome  bool
		responseData gin.H
		org          models.Organisation
	)

	clientID := config.Config.Google.CLIENT_ID

	// Detect if token is a JWT (ID token) or authorization code
	if isGoogleIDToken(req.Token) {
		// Direct ID token validation (mobile/web clients using Google Sign-In SDK)
		resp, err := idtoken.Validate(context.Background(), req.Token, clientID)
		if err != nil {
			return responseData, http.StatusBadRequest, fmt.Errorf("failed to validate id_token: %v", err)
		}
		userClaims = resp.Claims
	} else {
		// Authorization code exchange flow (traditional OAuth 2.0)
		googleOAuthConfig := &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: config.Config.Google.CLIENT_SECRET,
			RedirectURL:  config.Config.Google.REDIRECT_URI,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}

		token, err := googleOAuthConfig.Exchange(context.Background(), req.Token)
		if err != nil {
			return responseData, http.StatusBadRequest, fmt.Errorf("failed to exchange code: %v", err)
		}

		idTokenValue := token.Extra("id_token")
		if idTokenValue == nil {
			return responseData, http.StatusBadRequest, fmt.Errorf("id_token missing from token exchange")
		}

		resp, err := idtoken.Validate(context.Background(), idTokenValue.(string), clientID)
		if err != nil {
			return responseData, http.StatusBadRequest, fmt.Errorf("failed to validate id_token: %v", err)
		}
		userClaims = resp.Claims
	}

	var (
		email    = strings.ToLower(userClaims["email"].(string))
		username = strings.ToLower(userClaims["name"].(string))
		user     models.User
	)

	if email == "" || username == "" {
		return responseData, http.StatusNotFound, fmt.Errorf("token decode failed")
	}

	reqUser = models.CreateUserRequestModel{
		Email: email,
	}
	formattedReq, err := ValidateCreateUserRequest(reqUser, db)
	exists := postgresql.CheckExists(db, &user, "email = ?", formattedReq.Email)
	if exists {
		user, err = user.GetUserWithProfile(db, user.ID)

		if err != nil {
			return responseData, http.StatusInternalServerError, fmt.Errorf("error fetching user %v", err.Error())
		}

	} else {
		user = models.User{
			ID:             utility.GenerateUUID(),
			Name:           username,
			Email:          formattedReq.Email,
			IsVerified:     true,
			ProfileUpdated: true,
			IsOnboarded:    true,
			Profile: models.Profile{
				FullName:  username,
				UserName:  username,
				ID:        utility.GenerateUUID(),
				AvatarURL: userClaims["picture"].(string),
			},
		}
		err := user.CreateUser(db)
		sendWelcome = true
		if err != nil {
			return responseData, http.StatusInternalServerError, err
		}

		ipAddress := audit_utility.GetClientIP(c)

		response, _ := extReq.SendExternalRequest("ipinfo_resolve_ip", ipAddress)

		info, _ := response.(external_models.IPInfoResponse)

		createPersonalOrgReq := models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("%s", username),
			Description: fmt.Sprintf("%s's organization", username),
			Email:       email,
			Type:        "User Default Org",
			Country:     info.Country,
			Location:    info.Location,
			LogoURL:     userClaims["picture"].(string),
		}

		org, err := organisation.CreateOrganisation(createPersonalOrgReq, db, user.ID, logger)

		if err != nil {
			return responseData, http.StatusInternalServerError, fmt.Errorf("failed to create user organization: %v", err)
		}

		user.CurrentOrg = uuid.FromStringOrNil(org.ID)

		if err := db.Save(&user).Error; err != nil {
			return responseData, http.StatusInternalServerError, err
		}
	}

	tokenData, err := middleware.CreateToken(user, c)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("error saving token: %v", err.Error())
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	access_token := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: user.ID}

	err = access_token.CreateAccessToken(db, tokens)

	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("error saving token: %v", err.Error())
	}

	org, _ = org.GetOrgByID(db, user.CurrentOrg.String())

	responseData = gin.H{
		"user": map[string]any{
			"id":                        user.ID,
			"email":                     user.Email,
			"username":                  user.Name,
			"fullname":                  user.Name,
			"current_org":               user.CurrentOrg,
			"current_organisation_slug": slug.Make(org.Name),
			"is_verified":               user.IsVerified,
			"profile_updated":           user.ProfileUpdated,
			"is_onboarded":              user.IsOnboarded,
			"avatar_url":                user.Profile.AvatarURL,
			"expires_in":                strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at":                strconv.Itoa(int(user.CreatedAt.Unix())),
			"updated_at":                strconv.Itoa(int(user.UpdatedAt.Unix())),
		},
		"access_token":            tokenData.AccessToken,
		"access_token_expires_in": strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
		"notification_token":      access_token.SubAccessToken,
	}
	if sendWelcome {
		resetReq := models.SendWelcomeMail{
			Email: user.Email,
		}

		err = actions.AddNotificationToQueue(storage.DB.Redis, names.SendWelcomeMail, resetReq)
		if err != nil {
			return responseData, http.StatusInternalServerError, err
		}
	}

	audit_utility.LogUserLogin(c, db, extReq, user.ID, tokenData.AccessUuid, user.Organisations)

	return responseData, http.StatusCreated, nil
}

// isGoogleIDToken checks if the token is a JWT (ID token) or an authorization code.
// Google ID tokens are JWTs that start with "eyJ" (base64 encoded '{"')
// and contain exactly 3 parts separated by dots (header.payload.signature).
func isGoogleIDToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	return strings.HasPrefix(token, "eyJ")
}
