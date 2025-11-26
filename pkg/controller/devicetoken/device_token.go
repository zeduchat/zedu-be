package devicetoken

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/device_token"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

// Register saves or updates a device token for the authenticated user.
func (base *Controller) Register(c *gin.Context) {
	var req models.RegisterDeviceTokenRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user claims not found", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userID, ok := userClaims["user_id"].(string)
	if !ok || userID == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user id missing in claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := service.RegisterDeviceToken(base.Db, base.Logger, req, userID)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	message := "device token registered successfully"
	if code == http.StatusOK {
		message = "device token updated successfully"
	}

	rd := utility.BuildSuccessResponse(code, message, resp)
	c.JSON(code, rd)
}
