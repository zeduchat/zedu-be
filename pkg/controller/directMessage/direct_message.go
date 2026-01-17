package dm

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	dm "github.com/hngprojects/telex_be/services/directMessage"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateDmChannel(c *gin.Context) {
	var req models.DmChannelsRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	req.UserId = userId
	req.OrgId = c.Param("org_id")

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.OrgId); err != nil {
		base.Logger.Info("error parsing organisation id")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid organisation id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := dm.CreateDmChannel(req, base.Db.Postgresql, base.ExtReq, base.Db.Redis)
	if err != nil {
		base.Logger.Error("error creating dm channel", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Dm channel created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "DM channel created successfully", respData)
	c.JSON(http.StatusCreated, rd)

}

func (base *Controller) GetDmChannels(c *gin.Context) {

	var req models.DmChannelsRequest

	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	UserId := userClaims["user_id"].(string)

	req.UserId = UserId
	req.OrgId = c.Param("org_id")

	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, paginationResponse, code, err := dm.GetDmChannels(req, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("Dm channel retrived successfully")
	rd := utility.BuildSuccessResponse(code, "Dm channel retrived successfully", resp, paginationData)
	c.JSON(code, rd)
}

func (base *Controller) GetDmParticipants(c *gin.Context) {

	var req models.DmChannelsRequest

	req.ChannelId = c.Param("channel_id")
	req.OrgId = c.Param("org_id")

	if _, err := uuid.Parse(req.ChannelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	req.UserId = userClaims["user_id"].(string)

	includeMedia := c.Query("include_media") == "true"

	resp, code, err := dm.GetDmParticipants(req, base.Db, c, base.ExtReq, base.Db.Redis, includeMedia)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Participatint retrived successfully")
	rd := utility.BuildSuccessResponse(code, "Participants retrived successfully", resp)
	c.JSON(code, rd)
}

func (base *Controller) DeleteDmChannel(c *gin.Context) {
	var (
		req models.DmChannelsRequest
	)

	req.ChannelId = c.Param("channel_id")
	req.OrgId = c.Param("org_id")

	if _, err := uuid.Parse(req.ChannelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse organisation id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	req.UserId = userClaims["user_id"].(string)

	code, err := dm.DeleteDmChannel(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Dm channel deleted successfully")
	rd := utility.BuildSuccessResponse(code, "Dm channel deleted successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) GetDmChannelMedia(c *gin.Context) {
	var req models.DmChannelMediaRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	req.UserId = userClaims["user_id"].(string)
	req.OrgId = c.Param("org_id")
	req.ChannelId = c.Param("channel_id")
	req.MediaType = c.Query("type") // Optional: images, videos, documents, audio

	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.ChannelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, paginationResponse, code, err := dm.GetDmChannelMedia(req, base.Db, c)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("DM channel media retrieved successfully")
	rd := utility.BuildSuccessResponse(code, "DM channel media retrieved successfully", resp, paginationData)
	c.JSON(code, rd)
}

func (base *Controller) UpsertGroupDescription(c *gin.Context) {
	var req models.GroupDescriptionRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	req.UserId = userClaims["user_id"].(string)
	req.OrgId = c.Param("org_id")
	req.ChannelId = c.Param("channel_id")

	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.ChannelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

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

	code, err := dm.UpsertGroupDescription(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Group description updated successfully")
	rd := utility.BuildSuccessResponse(code, "Group description updated successfully", nil)
	c.JSON(code, rd)
}
