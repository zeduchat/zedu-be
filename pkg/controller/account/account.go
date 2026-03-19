package account

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/account"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

func (base *Controller) CreateAccountDeletionRequest(c *gin.Context) {
	var req models.CreateAccountDeletionRequest

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

	orgID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "org_id")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Failed to retrieve organization", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	orgIDStr, ok := orgID.(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Invalid organization ID format", nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	deletionRequest, err := account.CreateAccountDeletionRequest(base.Db.Postgresql, req, orgIDStr)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to create deletion request", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusCreated, "Account deletion request submitted successfully", deletionRequest)
	c.JSON(http.StatusCreated, rd)
}
