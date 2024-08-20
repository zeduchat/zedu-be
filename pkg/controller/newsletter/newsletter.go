package newsletter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/newsletter"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) SubscribeNewsLetter(c *gin.Context) {
	var (
		req = models.NewsLetter{}
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	err = service.NewsLetterSubscribe(&req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		if err.Error() == "email already subscribed" {
			rd := utility.BuildErrorResponse(http.StatusConflict, "error", err.Error(), "failed to subscribe email", nil)
			c.JSON(http.StatusConflict, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("email subscribed successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "email subscribed successfully", nil)
	c.JSON(http.StatusCreated, rd)
}
