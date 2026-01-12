package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	"github.com/hngprojects/telex_be/utility"
)

// OneSignalSubscriptionIDRequest represents the request body for registering/updating OneSignal subscription ID
type OneSignalSubscriptionIDRequest struct {
	SubscriptionID string `json:"subscription_id" binding:"required"`
	Platform       string `json:"platform"` // iOS, Android, Web (optional)
}

// RegisterOneSignalSubscriptionID registers a new OneSignal subscription ID for a user
func (base *Controller) RegisterOneSignalSubscriptionID(c *gin.Context) {
	var req OneSignalSubscriptionIDRequest

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Get user ID from claims
	userClaims := common.GetAllUserClaims(c)
	userID, ok := userClaims["user_id"].(string)

	if !ok || userID == "" {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", nil, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	// Validate subscription ID
	if req.SubscriptionID == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Subscription ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Fetch user
	user := &models.User{}
	fetchedUser, err := user.GetUserByID(base.Db.Postgresql, userID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "User not found", nil, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	// Update OneSignal subscription ID
	fetchedUser.OneSignalSubscriptionID = req.SubscriptionID
	err = fetchedUser.Update(base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("failed to update OneSignal subscription ID", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to register subscription ID", nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("OneSignal subscription ID registered successfully for user " + userID)
	rd := utility.BuildSuccessResponse(http.StatusOK, "OneSignal subscription ID registered successfully", gin.H{
		"user_id":         userID,
		"subscription_id": req.SubscriptionID,
	})
	c.JSON(http.StatusOK, rd)
}

// UpdateOneSignalSubscriptionID updates the OneSignal subscription ID for a user
func (base *Controller) UpdateOneSignalSubscriptionID(c *gin.Context) {
	var req OneSignalSubscriptionIDRequest

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Get user ID from claims
	userClaims := common.GetAllUserClaims(c)
	userID, ok := userClaims["user_id"].(string)

	if !ok || userID == "" {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", nil, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	// Validate subscription ID
	if req.SubscriptionID == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Subscription ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Fetch user
	user := &models.User{}
	fetchedUser, err := user.GetUserByID(base.Db.Postgresql, userID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "User not found", nil, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	// Check if user exists
	if fetchedUser.ID == "" {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "User not found", nil, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	// Update OneSignal subscription ID
	fetchedUser.OneSignalSubscriptionID = req.SubscriptionID
	err = fetchedUser.Update(base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("failed to update OneSignal subscription ID", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to update subscription ID", nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("OneSignal subscription ID updated successfully for user " + userID)
	rd := utility.BuildSuccessResponse(http.StatusOK, "OneSignal subscription ID updated successfully", gin.H{
		"user_id":         userID,
		"subscription_id": req.SubscriptionID,
	})
	c.JSON(http.StatusOK, rd)
}

// GetOneSignalSubscriptionID retrieves the OneSignal subscription ID for the current user
func (base *Controller) GetOneSignalSubscriptionID(c *gin.Context) {
	// Get user ID from claims
	userClaims := common.GetAllUserClaims(c)
	userID, ok := userClaims["user_id"].(string)

	if !ok || userID == "" {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", nil, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	// Fetch user's OneSignal subscription ID
	user := &models.User{}
	subscriptionID, err := user.GetOneSignalSubscriptionID(base.Db.Postgresql, userID)
	if err != nil {
		base.Logger.Error("failed to retrieve OneSignal subscription ID", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to retrieve subscription ID", nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "OneSignal subscription ID retrieved successfully", gin.H{
		"user_id":         userID,
		"subscription_id": subscriptionID,
	})
	c.JSON(http.StatusOK, rd)
}
