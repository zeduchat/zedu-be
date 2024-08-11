package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/profile"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetUserProfile(c *gin.Context) {

	userProfile, code, err := profile.GetUserProfile(base.Db.Postgresql, c)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "Failed to Fetch user profile", err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User profile retrieved successfully", userProfile)
	c.JSON(http.StatusOK, rd)
}
