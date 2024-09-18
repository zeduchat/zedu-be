package blog

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	service "github.com/hngprojects/telex_be/services/blog"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) SubmitFeedback(c *gin.Context) {
	var req models.BlogFeedbackReq

	if err := c.ShouldBind(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.BlogID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid blog id format", "failed to create blog", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	err := service.SubmitFeedback(req, base.Db.Postgresql)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("blog feedback submitted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "blog feedback submitted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetFeedbackCount(c *gin.Context) {
	blogID := c.Param("id")

	if _, err := uuid.Parse(blogID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid blog id format", "failed to retrieve blog", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	posCount, negCount, err := service.GetFeedbackCount(blogID, base.Db.Postgresql)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to count feedback", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("blog feedback counts retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "blog feedback counts retrieved successfully", gin.H{
		"positive_feedback": posCount,
		"negative_feedback": negCount,
	})
	c.JSON(http.StatusOK, rd)
}
