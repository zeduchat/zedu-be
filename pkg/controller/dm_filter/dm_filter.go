package dm_filter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	dmfilter "github.com/hngprojects/telex_be/services/dm_filter"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) DmFilter(c *gin.Context) {
	orgId := c.Param("org_id")
	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id", "organisation could not be found", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "could not perform search", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	dms, pg, statusCode, err := dmfilter.FilterData(base.Db, userId, orgId, c)

	if err != nil {

		rd := utility.BuildErrorResponse(statusCode, http.StatusText(statusCode), err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}
	resp := utility.BuildSuccessResponse(statusCode, "success", dms, pg)
	c.JSON(statusCode, resp)

}
