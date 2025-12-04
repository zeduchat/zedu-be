package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	mediaPreferencesService "github.com/hngprojects/telex_be/services/mediaPreferences"
	"github.com/hngprojects/telex_be/utility"
)

// GetMediaPreferences retrieves media preferences for the authenticated user
func (base *Controller) GetMediaPreferences(c *gin.Context) {
	// Extract user_id from JWT claims
	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", errors.New("user claims not found"), nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userID, ok := userClaims["user_id"].(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid user_id in token", errors.New("user_id is not a string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Validate user_id format
	if _, err := uuid.Parse(userID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid user_id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Extract optional device_id from query params
	var deviceID *string
	if deviceIDParam := c.Query("device_id"); deviceIDParam != "" {
		// Validate device_id format
		if _, err := uuid.Parse(deviceIDParam); err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid device_id format", err, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		deviceID = &deviceIDParam
	}

	// Call service layer
	respData, code, err := mediaPreferencesService.GetMediaPreferences(userID, deviceID, base.Db.Postgresql, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Media preferences retrieved successfully", respData)
	c.JSON(http.StatusOK, rd)
}

// UpdateMediaPreferences updates media preferences for the authenticated user
func (base *Controller) UpdateMediaPreferences(c *gin.Context) {
	var req models.UpdateMediaPreferencesRequest

	// Extract user_id from JWT claims
	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", errors.New("user claims not found"), nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userID, ok := userClaims["user_id"].(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid user_id in token", errors.New("user_id is not a string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Validate user_id format
	if _, err := uuid.Parse(userID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid user_id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Bind request body
	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Validate request
	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	// Call service layer
	respData, code, err := mediaPreferencesService.UpdateMediaPreferences(req, userID, base.Db.Postgresql, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Media preferences updated successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "Media preferences updated successfully", respData)
	c.JSON(http.StatusOK, rd)
}
