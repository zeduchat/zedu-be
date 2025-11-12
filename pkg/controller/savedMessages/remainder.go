package savedMessages

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/savedMessages"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) SetRemainder(c *gin.Context) {
	var req models.SetRemainderRequest
	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserId = userId
	req.OrgId = orgId

	statusCode, err := savedMessages.SetRemainder(req, base.Db, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), nil, nil)
		c.JSON(statusCode, rd)
		return
	}

	rd := utility.BuildSuccessResponse(statusCode, "Remainder set successfully", nil)
	c.JSON(http.StatusOK, rd)
}
