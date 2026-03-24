package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/auth"
	telexaudit "github.com/hngprojects/telex_be/services/telexAudit"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GoogleLogin(c *gin.Context) {

	var (
		req = models.GoogleRequestModel{}
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

	respData, code, err := auth.CreateGoogleUser(req, base.Db.Postgresql, c, base.ExtReq, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user sign in successfully")

	err = telexaudit.LoginAudit(base.Db, base.Logger, respData)
	if err != nil {
		base.Logger.Error("error publishing login audit: ", err.Error())
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "user sign in successfully", respData)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) AppleLogin(c *gin.Context) {

	var (
		req = models.AppleRequestModel{}
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

	respData, code, err := auth.CreateAppleUser(req, base.Db.Postgresql, c, base.ExtReq, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user sign in successfully")

	err = telexaudit.LoginAudit(base.Db, base.Logger, respData)
	if err != nil {
		base.Logger.Error("error publishing login audit: ", err.Error())
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "user sign in successfully", respData)
	c.JSON(http.StatusOK, rd)

}
