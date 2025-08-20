package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
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

func CreateGoogleUser(req models.GoogleRequestModel, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest) (gin.H, int, error) {

	var (
		userClaims   map[string]any
		reqUser      models.CreateUserRequestModel
		sendWelcome  bool
		responseData gin.H
	)

	var googleOAuthConfig = &oauth2.Config{
		ClientID:     config.Config.Google.CLIENT_ID,
		ClientSecret: config.Config.Google.CLIENT_SECRET,
		RedirectURL:  config.Config.Google.REDIRECT_URI,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	token, err := googleOAuthConfig.Exchange(context.Background(), req.Token)
	if err != nil {
		return responseData, http.StatusBadRequest, fmt.Errorf("failed to exchange code: %v", err)
	}

	idToken := token.Extra("id_token")
	if idToken == nil {
		return responseData, http.StatusBadRequest, fmt.Errorf("id_token missing from token exchange")
	}

	resp, err := idtoken.Validate(context.Background(), idToken.(string), googleOAuthConfig.ClientID)
	userClaims = resp.Claims
	if err != nil {
		return responseData, http.StatusBadRequest, fmt.Errorf("an error occured: %v", err.Error())
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

	responseData = gin.H{
		"user": map[string]any{
			"id":              user.ID,
			"email":           user.Email,
			"username":        user.Name,
			"fullname":        user.Name,
			"current_org":     user.CurrentOrg,
			"is_verified":     user.IsVerified,
			"profile_updated": user.ProfileUpdated,
			"is_onboarded":    user.IsOnboarded,
			"avatar_url":      user.Profile.AvatarURL,
			"expires_in":      strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at":      strconv.Itoa(int(user.CreatedAt.Unix())),
			"updated_at":      strconv.Itoa(int(user.UpdatedAt.Unix())),
		},
		"access_token": tokenData.AccessToken,
		"access_token_expires_in":      strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
		"notification_token": access_token.SubAccessToken,
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
