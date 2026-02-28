package buzz

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/pkg/middleware"
	buzzsvc "github.com/hngprojects/telex_be/services/buzz"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) StartRecording(c *gin.Context) {
	buzzCode, ok := c.Params.Get("id")
	if !ok || !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code", errors.New("invalid buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id", errors.New("invalid user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rec, code, err := buzzsvc.StartBuzzRecording(base.Db, base.Logger, buzzID, userID)
	if err != nil {
		base.Logger.Error("failed to start recording: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("recording started for buzz %s by host %s", buzzID, userID)
	rd := utility.BuildSuccessResponse(http.StatusOK, "recording started successfully", rec)
	c.JSON(http.StatusOK, rd)
}


func (base *Controller) StopRecording(c *gin.Context) {
	buzzCode, ok := c.Params.Get("id")
	if !ok || !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code", errors.New("invalid buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id", errors.New("invalid user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rec, code, err := buzzsvc.StopBuzzRecording(base.Db, base.Logger, buzzID, userID)
	if err != nil {
		base.Logger.Error("failed to stop recording: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("recording stopped for buzz %s by host %s", buzzID, userID)
	rd := utility.BuildSuccessResponse(http.StatusOK, "recording stopped successfully", rec)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetRecordingStatus(c *gin.Context) {
	buzzCode, ok := c.Params.Get("id")
	if !ok || !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code", errors.New("invalid buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id", errors.New("invalid user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rec, code, err := buzzsvc.CheckRecordingStatus(base.Db, base.Logger, buzzID, userID)
	if err != nil {
		base.Logger.Error("failed to get recording status: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "recording status retrieved successfully", rec)
	c.JSON(http.StatusOK, rd)
}
