package recentLogs

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/logs"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetTailLogs(c *gin.Context) {
	linesParam := c.DefaultQuery("lines", "100")

	lines, err := strconv.Atoi(linesParam)
	if err != nil || lines <= 0 {
		lines = 100
	}

	result, err := logs.TailLines("logs/app.log", lines)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to read logs", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(result))
}
