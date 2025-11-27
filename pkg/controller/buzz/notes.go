package buzz

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/buzz"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) CreateNote(c *gin.Context) {
	buzzID := c.Param("id")
	if buzzID == "" {
		base.Logger.Info("missing buzz ID")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "buzz ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.CreateBuzzNoteRequest

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing buzz note request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for buzz note request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := buzz.CreateBuzzNote(base.Db, base.Logger, buzzID, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to create buzz note: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("buzz note created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "buzz note created successfully", resp)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetNotes(c *gin.Context) {
	buzzID := c.Param("id")
	if buzzID == "" {
		base.Logger.Info("missing buzz ID")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "buzz ID is required", nil, nil)
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

	resp, code, err := buzz.GetBuzzNotes(base.Db, base.Logger, buzzID, userID.(string))
	if err != nil {
		base.Logger.Error("failed to fetch buzz notes: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("buzz notes fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "buzz notes retrieved successfully", resp)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateNote(c *gin.Context) {
	buzzID := c.Param("id")
	if buzzID == "" {
		base.Logger.Info("missing buzz ID")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "buzz ID is required", nil, nil)
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

	var req models.UpdateBuzzNoteRequest

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing buzz note update request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for buzz note update request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := buzz.UpdateBuzzNote(base.Db, base.Logger, buzzID, noteID, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to update buzz note: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("buzz note updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "buzz note updated successfully", resp)
	c.JSON(http.StatusOK, rd)
}
