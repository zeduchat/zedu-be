package huddle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/huddle"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) CreateNote(c *gin.Context) {
	huddleID := c.Param("id")
	if huddleID == "" {
		base.Logger.Info("missing huddle ID")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "huddle ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.CreateHuddleNoteRequest

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing huddle note request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for huddle note request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := huddle.CreateHuddleNote(base.Db, base.Logger, huddleID, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to create huddle note: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("huddle note created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "huddle note created successfully", resp)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetNotes(c *gin.Context) {
	huddleID := c.Param("id")
	if huddleID == "" {
		base.Logger.Info("missing huddle ID")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "huddle ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	resp, code, err := huddle.GetHuddleNotes(base.Db, base.Logger, huddleID, userID.(string))
	if err != nil {
		base.Logger.Error("failed to fetch huddle notes: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("huddle notes fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "huddle notes retrieved successfully", resp)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateNote(c *gin.Context) {
	huddleID := c.Param("id")
	if huddleID == "" {
		base.Logger.Info("missing huddle ID")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "huddle ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	noteID := c.Param("note_id")
	if noteID == "" {
		base.Logger.Info("missing note ID")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "note ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.UpdateHuddleNoteRequest

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing huddle note update request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for huddle note update request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := huddle.UpdateHuddleNote(base.Db, base.Logger, huddleID, noteID, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to update huddle note: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("huddle note updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "huddle note updated successfully", resp)
	c.JSON(http.StatusOK, rd)
}
