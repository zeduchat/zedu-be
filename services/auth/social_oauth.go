package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Timothylock/go-signin-with-apple/apple"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/gosimple/slug"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/avatar"
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

		idTokenStr, ok := idTokenValue.(string)
		if !ok {
			return responseData, http.StatusBadRequest, fmt.Errorf("id_token is not a valid string")
		}

		resp, err := idtoken.Validate(context.Background(), idTokenStr, clientID)
		if err != nil {
			return responseData, http.StatusBadRequest, fmt.Errorf("failed to validate id_token: %v", err)
		}
		userClaims = resp.Claims
	}

	// Safely extract and validate claims
	email, ok := userClaims["email"].(string)
	if !ok || email == "" {
		return responseData, http.StatusBadRequest, fmt.Errorf("email claim missing or invalid in token")
	}
	email = strings.ToLower(email)

	name, ok := userClaims["name"].(string)
	if !ok || name == "" {
		return responseData, http.StatusBadRequest, fmt.Errorf("name claim missing or invalid in token")
	}
	username := strings.ToLower(name)

	// Safely extract picture URL (optional claim)
	pictureURL := ""
	if pic, ok := userClaims["picture"].(string); ok {
		pictureURL = pic
	}

	var user models.User

	reqUser = models.CreateUserRequestModel{
		Email: email,
	}
	formattedReq, err := ValidateCreateUserRequest(reqUser, db)
	if err != nil {
		return responseData, http.StatusBadRequest, fmt.Errorf("invalid user data: %v", err)
	}

	exists := postgresql.CheckExists(db, &user, "email = ?", formattedReq.Email)
	if exists {
		user, err = user.GetUserByEmail(db, formattedReq.Email)

		if err != nil {
			return responseData, http.StatusInternalServerError, fmt.Errorf("error fetching user %v", err.Error())
		}

		// Validate user status before allowing login
		if user.DeletedAt.Valid {
			return responseData, http.StatusForbidden, fmt.Errorf("account has been deleted")
		}

	} else {
		// Generate unique username to prevent collisions
		uniqueUsername := username
		for i := 0; i < 5; i++ {
			if !postgresql.CheckExists(db, &models.Profile{}, "user_name = ?", uniqueUsername) {
				break
			}
			// Append random suffix if username exists
			suffix := utility.GenerateUUID()[:8]
			uniqueUsername = fmt.Sprintf("%s_%s", username, suffix)
		}

		user = models.User{
			ID:             utility.GenerateUUID(),
			Name:           username,
			Email:          formattedReq.Email,
			IsVerified:     true,
			ProfileUpdated: true,
			IsOnboarded:    true,
			Profile: models.Profile{
				FullName:  username,
				UserName:  uniqueUsername,
				ID:        utility.GenerateUUID(),
				AvatarURL: pictureURL,
			},
		}

		// Wrap user and org creation in transaction to prevent orphaned users
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := user.CreateUser(tx); err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}

			sendWelcome = true

			ipAddress := audit_utility.GetClientIP(c)

			response, _ := extReq.SendExternalRequest("ipinfo_resolve_ip", ipAddress)

			info, _ := response.(external_models.IPInfoResponse)

			createPersonalOrgReq := models.CreateOrgRequestModel{
				Name:        username,
				Description: fmt.Sprintf("%s's organization", username),
				Email:       email,
				Type:        "User Default Org",
				Country:     info.Country,
				Location:    info.Location,
				LogoURL:     pictureURL,
			}

			org, err := organisation.CreateOrganisation(createPersonalOrgReq, tx, user.ID, logger)
			if err != nil {
				return fmt.Errorf("failed to create user organization: %w", err)
			}

			user.CurrentOrg = uuid.FromStringOrNil(org.ID)

			if err := tx.Save(&user).Error; err != nil {
				return fmt.Errorf("failed to update user org: %w", err)
			}

			return nil
		})

		if err != nil {
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
			"user_id":                   user.ID,
			"email":                     user.Email,
			"username":                  user.Name,
			"fullname":                  user.Name,
			"current_org":               user.CurrentOrg,
			"current_organisation_slug": slug.Make(org.Name),
			"is_verified":               user.IsVerified,
			"profile_updated":           user.ProfileUpdated,
			"is_onboarded":              user.IsOnboarded,
			"avatar_url":                user.Profile.AvatarURL,
			"default_avatar_url":        avatar.GenerateDefaultAvatarURL(user.ID),
			"expires_in":                strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at":                strconv.Itoa(int(user.CreatedAt.Unix())),
			"updated_at":                strconv.Itoa(int(user.UpdatedAt.Unix())),
			"organisation":              org,
			"online":                    user.Profile.Online,
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
// Google ID tokens are JWTs that:
// 1. Have exactly 3 parts separated by dots (header.payload.signature)
// 2. Start with "eyJ" (base64 encoded '{"')
// 3. Are longer than typical authorization codes (usually 100+ chars)
// Authorization codes are typically 40-70 characters and don't have dots.
func isGoogleIDToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	// JWT tokens are much longer than auth codes (typically 800+ chars)
	if len(token) < 100 {
		return false
	}
	// Check if it starts with base64 encoded JSON header
	return strings.HasPrefix(token, "eyJ")
}

func isAppleIDToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	if len(token) < 100 {
		return false
	}

	return strings.HasPrefix(token, "eyJ")
}

func CreateAppleUser(req models.AppleRequestModel, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest, logger *utility.Logger) (gin.H, int, error) {

	var (
		reqUser      models.CreateUserRequestModel
		sendWelcome  bool
		responseData gin.H
		org          models.Organisation
		idToken      string
	)

	if isAppleIDToken(req.Token) {
		idToken = req.Token
	} else {
		client := apple.New()

		secret, err := apple.GenerateClientSecret(config.Config.Apple.PRIVATE_KEY, config.Config.Apple.TEAM_ID, config.Config.Apple.CLIENT_ID, config.Config.Apple.KEY_ID)
		if err != nil {
			logger.Error("Failed to generate apple client secret: %v", err)
			return responseData, http.StatusInternalServerError, fmt.Errorf("failed to generate apple client secret: %v", err)
		}

		vReq := apple.AppValidationTokenRequest{
			ClientID:     config.Config.Apple.CLIENT_ID,
			ClientSecret: secret,
			Code:         req.Token,
		}

		var appleResp apple.ValidationResponse
		err = client.VerifyAppToken(c.Request.Context(), vReq, &appleResp)
		if err != nil {
			logger.Error("Failed to validate apple token: %v", err)
			return responseData, http.StatusBadRequest, fmt.Errorf("failed to validate apple token: %v", err)
		}

		idToken = appleResp.IDToken
	}

	claim, err := apple.GetClaims(idToken)
	if err != nil {
		logger.Error("Failed to get apple claims: %v", err)
		return responseData, http.StatusBadRequest, fmt.Errorf("failed to get apple claims: %v", err)
	}

	email, ok := (*claim)["email"].(string)
	if !ok || email == "" {
		return responseData, http.StatusBadRequest, fmt.Errorf("email missing in apple token")
	}

	logger.Info("Apple login successful for email: %s", email)

	name := strings.Split(email, "@")[0]
	username := strings.ToLower(name)

	var user models.User

	reqUser = models.CreateUserRequestModel{
		Email: email,
	}
	formattedReq, err := ValidateCreateUserRequest(reqUser, db)
	if err != nil {
		return responseData, http.StatusBadRequest, fmt.Errorf("invalid user data: %v", err)
	}

	exists := postgresql.CheckExists(db, &user, "email = ?", formattedReq.Email)
	if exists {
		logger.Info("Existing user found for email: %s", formattedReq.Email)
		user, err = user.GetUserByEmail(db, formattedReq.Email)

		if err != nil {
			logger.Error("Failed to fetch existing user: %v", err)
			return responseData, http.StatusInternalServerError, fmt.Errorf("error fetching user %v", err.Error())
		}

		if user.DeletedAt.Valid {
			return responseData, http.StatusForbidden, fmt.Errorf("account has been deleted")
		}

	} else {
		logger.Info("Creating new user for email: %s", formattedReq.Email)

		user = models.User{
			ID:             utility.GenerateUUID(),
			Name:           username,
			Email:          formattedReq.Email,
			IsVerified:     true,
			ProfileUpdated: true,
			IsOnboarded:    true,
			Profile: models.Profile{
				FullName: username,
				UserName: username,
				ID:       utility.GenerateUUID(),
			},
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			if err := user.CreateUser(tx); err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}

			sendWelcome = true

			ipAddress := audit_utility.GetClientIP(c)

			response, _ := extReq.SendExternalRequest("ipinfo_resolve_ip", ipAddress)

			info, _ := response.(external_models.IPInfoResponse)

			createPersonalOrgReq := models.CreateOrgRequestModel{
				Name:        username,
				Description: fmt.Sprintf("%s's organization", username),
				Email:       email,
				Type:        "User Default Org",
				Country:     info.Country,
				Location:    info.Location,
			}

			org, err := organisation.CreateOrganisation(createPersonalOrgReq, tx, user.ID, logger)
			if err != nil {
				return fmt.Errorf("failed to create user organization: %w", err)
			}

			user.CurrentOrg = uuid.FromStringOrNil(org.ID)

			if err := tx.Save(&user).Error; err != nil {
				return fmt.Errorf("failed to update user org: %w", err)
			}

			logger.Info("Successfully created user and personal organization for: %s", email)
			return nil
		})

		if err != nil {
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
			"user_id":                   user.ID,
			"email":                     user.Email,
			"username":                  user.Name,
			"fullname":                  user.Name,
			"current_org":               user.CurrentOrg,
			"current_organisation_slug": slug.Make(org.Name),
			"is_verified":               user.IsVerified,
			"profile_updated":           user.ProfileUpdated,
			"is_onboarded":              user.IsOnboarded,
			"avatar_url":                user.Profile.AvatarURL,
			"default_avatar_url":        avatar.GenerateDefaultAvatarURL(user.ID),
			"expires_in":                strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at":                strconv.Itoa(int(user.CreatedAt.Unix())),
			"updated_at":                strconv.Itoa(int(user.UpdatedAt.Unix())),
			"organisation":              org,
			"online":                    user.Profile.Online,
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
