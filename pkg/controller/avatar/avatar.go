package avatar

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/avatar"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) UploadAvatar(ctx *gin.Context) {
	file, header, err := ctx.Request.FormFile("avatar")
	if err != nil {
		if err == http.ErrMissingFile {
			base.Logger.Error("No avatar file provided", "error", err)
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "No avatar file provided", err, nil)
			ctx.JSON(http.StatusBadRequest, rd)
			return
		}

		base.Logger.Error("Failed to parse avatar file", "error", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse avatar file", err, nil)
		ctx.JSON(http.StatusBadRequest, rd)
		return
	}
	defer file.Close()

	url, err := avatar.UploadAvatar(base.Logger, file, header)
	if err != nil {
		base.Logger.Error("Failed to upload avatar", "error", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to upload avatar", err, nil)
		ctx.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Avatar uploaded successfully", "url", url)
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Avatar uploaded successfully", gin.H{
		"url": url,
	})
	ctx.JSON(http.StatusCreated, rd)
}

func (base *Controller) ListAvatars(ctx *gin.Context) {
	avatars, err := avatar.ListAvatars(base.Logger)
	if err != nil {
		base.Logger.Error("Failed to list avatars", "error", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to list avatars", err, nil)
		ctx.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Avatars listed successfully", "count", len(avatars))
	rd := utility.BuildSuccessResponse(http.StatusOK, "Avatars retrieved successfully", avatars)
	ctx.JSON(http.StatusOK, rd)
}
