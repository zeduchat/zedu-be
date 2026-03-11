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

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/avatar"
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

// Default avatar URLs are generated deterministically based on user ID.

func ValidateCreateUserRequest(req models.CreateUserRequestModel, db *gorm.DB) (models.CreateUserRequestModel, error) {

	profile := models.Profile{}

	if req.Email != "" {
		req.Email = strings.ToLower(req.Email)
		formattedMail, checkBool := utility.EmailValid(req.Email)
		if !checkBool {
			return req, fmt.Errorf("email address is invalid")
		}
		req.Email = formattedMail
	}

	if req.PhoneNumber != "" {
		req.PhoneNumber = strings.ToLower(req.PhoneNumber)
		phone, _ := utility.PhoneValid(req.PhoneNumber)
		req.PhoneNumber = phone
		exists := postgresql.CheckExists(db, &profile, "phone = ?", req.PhoneNumber)
		if exists {
			return req, errors.New("user already exists with the given phone")
		}
	}
	return req, nil
}

func GetUser(userIDStr string, db *gorm.DB) (models.User, error) {
	var userResp models.User

	userResp, err := userResp.GetUserByID(db, userIDStr)
	if err != nil {
		return userResp, err
	}

	return userResp, nil
}

func CreateUser(c *gin.Context, extReq request.ExternalRequest, req models.CreateUserRequestModel, db *gorm.DB, logger *utility.Logger) (gin.H, int, error) {

	var (
		email              = strings.ToLower(req.Email)
		firstName          = strings.ToTitle(strings.ToLower(req.FirstName))
		lastName           = strings.ToTitle(strings.ToLower(req.LastName))
		phoneNumber        = req.PhoneNumber
		password           = req.Password
		responseData gin.H = gin.H{}
		userChk            = models.User{}
	)

	exists := postgresql.CheckExists(db, &userChk, "email = ?", req.Email)
	if exists {

		if !utility.CompareHash(req.Password, userChk.Password) {
			return responseData, 400, fmt.Errorf("Email address already exist, use another email or signin")
		}

		loginUser := models.LoginRequestModel{
			Email:    email,
			Password: req.Password,
		}

		return LoginUser(loginUser, db, c, extReq)
	}

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	name := req.UserName

	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	user := models.User{
		ID:             utility.GenerateUUID(),
		Name:           name,
		Email:          email,
		Password:       password,
		ProfileUpdated: true,
		IsOnboarded:    true,
		Profile: models.Profile{
			ID:        utility.GenerateUUID(),
			FirstName: name,
			LastName:  lastName,
			FullName:  firstName + " " + lastName,
			UserName:  name,
			Phone:     phoneNumber,
		},
	}

	createErr := user.CreateUser(db)

	// Audit the creation attempt, whether it succeeded or failed
	ipAddress := audit_utility.GetClientIP(c)
	success := createErr == nil
	description := fmt.Sprintf("User %s created account", user.Email)
	if !success {
		description = fmt.Sprintf("User %s failed to create account: %v", user.Email, createErr)
	}

	if auditErr := audit_utility.CreateAuditLog(
		db,
		user.ID,
		user.Email,
		"user",
		models.ActionUserCreate,
		models.ResourceUser,
		user.ID,
		"",
		"",
		description,
		ipAddress,
		c.GetHeader("User-Agent"),
		success,
	); auditErr != nil {
		logger.Error("failed to create audit log for user creation: " + auditErr.Error())
	}

	if createErr != nil {
		return nil, http.StatusInternalServerError, createErr
	}

	resetReq := models.SendWelcomeMail{
		Email: user.Email,
	}

	err = actions.AddNotificationToQueue(storage.DB.Redis, names.SendWelcomeMail, resetReq)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	exists = postgresql.CheckExists(db, &user, "email = ?", req.Email)
	if !exists {
		return responseData, 400, fmt.Errorf("invalid credentials")
	}

	user.IsActive = true
	if err := db.Save(&user).Error; err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("unable to update user status: %w", err)
	}

	userData, err := user.GetUserByID(db, user.ID)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("unable to fetch user: %w", err)
	}

	response, _ := extReq.SendExternalRequest("ipinfo_resolve_ip", ipAddress)

	info, _ := response.(external_models.IPInfoResponse)

	createPersonalOrgReq := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("%s", name),
		Description: fmt.Sprintf("%s's organization", name),
		Email:       email,
		Type:        "User Default Org",
		Country:     info.Country,
		Location:    info.Location,
	}

	org, err := organisation.CreateOrganisation(createPersonalOrgReq, db, user.ID, logger)

	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("failed to create user organization: %v", err)
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
		"user": map[string]any{
			"id":                        userData.ID,
			"user_id":                   userData.ID,
			"email":                     userData.Email,
			"username":                  userData.Name,
			"is_verified":               userData.IsVerified,
			"is_onboarded":              userData.IsOnboarded,
			"profile_updated":           userData.ProfileUpdated,
			"is_active":                 userData.IsActive,
			"current_org":               org.ID,
			"first_name":                userData.Profile.FirstName,
			"last_name":                 userData.Profile.LastName,
			"fullname":                  userData.Profile.FirstName + " " + userData.Profile.LastName,
			"phone":                     userData.Profile.Phone,
			"avatar_url":                userData.Profile.AvatarURL,
			"default_avatar_url":        avatar.GenerateDefaultAvatarURL(userData.ID),
			"expires_in":                strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at":                strconv.Itoa(int(userData.CreatedAt.Unix())),
			"updated_at":                strconv.Itoa(int(userData.UpdatedAt.Unix())),
			"current_organisation_slug": slug.Make(org.Name),
			"organisation":              org,
			"online":                    userData.Profile.Online,
		},
		"access_token":       tokenData.AccessToken,
		"notification_token": access_token.SubAccessToken,
	}
	return responseData, http.StatusCreated, nil
}

func LoginUser(req models.LoginRequestModel, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest) (gin.H, int, error) {
	var (
		user         = models.User{}
		responseData gin.H
		org          models.Organisation
	)

	exists := postgresql.CheckExists(db, &user, "email = ?", req.Email)
	if !exists {
		return responseData, 400, fmt.Errorf("invalid credentials")
	}

	if !utility.CompareHash(req.Password, user.Password) {
		return responseData, 400, fmt.Errorf("invalid email and password supplied")
	}

	// Update user active status and last login timestamp
	now := time.Now()
	user.IsActive = true
	user.LastLogInAt = &now

	if err := db.Save(&user).Error; err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("unable to update user status: %w", err)
	}

	userData, err := user.GetUserByID(db, user.ID)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("unable to fetch user: %w", err)
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

	org, _ = org.GetOrgByID(db, userData.CurrentOrg.String())

	responseData = gin.H{
		"user": map[string]any{
			"id":                        userData.ID,
			"email":                     userData.Email,
			"user_id":                   userData.ID,
			"username":                  userData.Name,
			"is_verified":               userData.IsVerified,
			"is_onboarded":              userData.IsOnboarded,
			"profile_updated":           userData.ProfileUpdated,
			"is_active":                 userData.IsActive,
			"current_org":               userData.CurrentOrg,
			"first_name":                userData.Profile.FirstName,
			"last_name":                 userData.Profile.LastName,
			"fullname":                  userData.Profile.FirstName + " " + userData.Profile.LastName,
			"phone":                     userData.Profile.Phone,
			"avatar_url":                userData.Profile.AvatarURL,
			"default_avatar_url":        avatar.GenerateDefaultAvatarURL(userData.ID),
			"expires_in":                strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at":                strconv.Itoa(int(userData.CreatedAt.Unix())),
			"updated_at":                strconv.Itoa(int(userData.UpdatedAt.Unix())),
			"current_organisation_slug": slug.Make(org.Name),
			"organisation":              org,
			"online":                    userData.Profile.Online,
		},
		"access_token":            tokenData.AccessToken,
		"notification_token":      access_token.SubAccessToken,
		"access_token_expires_in": strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	audit_utility.LogUserLogin(c, db, extReq, userData.ID, tokenData.AccessUuid, userData.Organisations)

	return responseData, http.StatusOK, nil
}

func LogoutUser(req models.LogoutReqModel, db *gorm.DB) (int, error) {
	var (
		user models.User
	)

	if err := db.Where("id = ?", req.UserId).First(&user).Error; err != nil {
		return http.StatusNotFound, fmt.Errorf("user not found: %w", err)
	}

	if err := db.Model(&user).Update("is_active", false).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error updating user: %w", err)
	}

	access_token := models.AccessToken{ID: req.AccessUuid, OwnerID: req.UserId}

	if req.Platform == "desktop" {
		access_token.IsLiveNotificaionToken = false
	}

	if err := access_token.RevokeAccessToken(db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error revoking user session: %w", err)
	}

	return http.StatusOK, nil
}

func CreateAdmin(req models.CreateUserRequestModel, db *gorm.DB, c *gin.Context) (gin.H, int, error) {

	var (
		email        = strings.ToLower(req.Email)
		firstName    = strings.Title(strings.ToLower(req.FirstName))
		lastName     = strings.Title(strings.ToLower(req.LastName))
		username     = strings.ToLower(req.UserName)
		phoneNumber  = req.PhoneNumber
		password     = req.Password
		responseData gin.H
		userChk      = models.User{}
	)

	exists := postgresql.CheckExists(db, &userChk, "email = ?", req.Email)
	if exists {
		return responseData, 400, fmt.Errorf("Email address already exist, use another email or signin")
	}

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     username,
		Email:    email,
		Password: password,
		Profile: models.Profile{
			ID:        utility.GenerateUUID(),
			FirstName: firstName,
			LastName:  lastName,
			FullName:  firstName + " " + lastName,
			UserName:  username,
			Phone:     phoneNumber,
		},
	}

	err = user.CreateUser(db)
	if err != nil {
		return nil, http.StatusInternalServerError, err
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
		"user": map[string]string{
			"id":         user.ID,
			"user_id":    user.ID,
			"email":      user.Email,
			"username":   user.Name,
			"first_name": user.Profile.FirstName,
			"last_name":  user.Profile.LastName,
			"fullname":   user.Profile.FirstName + " " + user.Profile.LastName,
			"phone":      user.Profile.Phone,
			"expires_in": strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at": strconv.Itoa(int(user.CreatedAt.Unix())),
			"updated_at": strconv.Itoa(int(user.UpdatedAt.Unix())),
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusCreated, nil
}

func GetOnboardStatus(owner_id string, db *gorm.DB) (gin.H, int, error) {

	var (
		responseData gin.H
		user         models.User
	)

	user, err := user.GetUserByID(db, owner_id)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching user: %w", err)
	}

	status := user.IsOnboarded

	responseData = gin.H{
		"status": status,
		"online": user.Profile.Online,
	}

	return responseData, http.StatusOK, nil
}

func UpdateOnboardStatus(owner_id string, db *gorm.DB) (gin.H, int, error) {

	var (
		responseData gin.H
		user         models.User
	)

	user, err := user.GetUserByID(db, owner_id)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching user: %w", err)
	}

	user.IsOnboarded = true
	err = user.Update(db)

	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error updating user status: %w", err)
	}

	responseData = gin.H{}

	return responseData, http.StatusOK, nil
}

func FetchUser(userId string, db *gorm.DB) (gin.H, int, error) {
	var (
		user         = models.User{}
		responseData gin.H
		org          models.Organisation
	)

	userData, err := user.GetUserByID(db, userId)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("unable to fetch user: %w", err)
	}

	org, _ = org.GetOrgByID(db, userData.CurrentOrg.String())

	responseData = gin.H{
		"user": map[string]any{
			"id":                        userData.ID,
			"user_id":                   userData.ID,
			"email":                     userData.Email,
			"username":                  userData.Name,
			"is_verified":               userData.IsVerified,
			"is_onboarded":              userData.IsOnboarded,
			"profile_updated":           userData.ProfileUpdated,
			"is_active":                 userData.IsActive,
			"current_org":               userData.CurrentOrg,
			"first_name":                userData.Profile.FirstName,
			"last_name":                 userData.Profile.LastName,
			"fullname":                  userData.Profile.FirstName + " " + userData.Profile.LastName,
			"phone":                     userData.Profile.Phone,
			"avatar_url":                userData.Profile.AvatarURL,
			"default_avatar_url":        avatar.GenerateDefaultAvatarURL(userData.ID),
			"created_at":                strconv.Itoa(int(userData.CreatedAt.Unix())),
			"updated_at":                strconv.Itoa(int(userData.UpdatedAt.Unix())),
			"current_organisation_slug": slug.Make(org.Name),
			"organisation":              org,
			"online":                    userData.Profile.Online,
		},
	}

	return responseData, http.StatusOK, nil
}
