package activity

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/activity"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetUserMentionsActivity(c *gin.Context) {
	orgID := c.Param("org_id")

	if !utility.IsValidUUID(orgID) {
		base.Logger.Error("invalid org id format: %s", orgID)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid org id format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("unable to get user claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userID := userClaims["user_id"].(string)

	mentions, paginationResponse, code, err := service.GetUserMentionsActivity(userID, orgID, base.Db, c, base.Logger)
	if err != nil {
		base.Logger.Error(fmt.Sprintf("failed to get user mentions activity: %v", err))
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "mentions retrieved successfully", mentions, paginationResponse)
	c.JSON(http.StatusOK, rd)
}
