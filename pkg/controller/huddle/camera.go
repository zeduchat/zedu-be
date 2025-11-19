package huddle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/huddle"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) UpdateCamera(c *gin.Context) {
	huddleID := c.Param("id")
	if huddleID == "" {
		base.Logger.Info("missing huddle ID")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "huddle ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.UpdateCameraRequest

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing camera update request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for camera update request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := huddle.UpdateCameraStatus(base.Db, base.Logger, huddleID, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to update camera status: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("camera status updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "camera status updated successfully", resp)
	c.JSON(http.StatusOK, rd)
}
