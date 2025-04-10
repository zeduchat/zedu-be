package dm

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	dm "github.com/hngprojects/telex_be/services/directMessage"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetGroupDMChannels(c *gin.Context) {

	var req models.GroupDMChannelsRequest

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

	resp, paginationResponse, statusCode, err := dm.GetGroupDMChannels(req, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	paginationData := map[string]interface{}{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(resp),
	}

	base.Logger.Info("Group DM channels retreived successfully")
	rd := utility.BuildSuccessResponse(statusCode, "Group DM channels retreived successfully", resp, paginationData)
	c.JSON(statusCode, rd)
}

func (base *Controller) GetUserGroupDMs(c *gin.Context) {

	var req models.GroupDMChannelsRequest

	req.UserId = c.Param("user_id")

	if _, err := uuid.Parse(req.UserId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", errors.New("failed to parse user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, statusCode, err := dm.GetUserGroupDMs(req, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("User retrived successfully")
	rd := utility.BuildSuccessResponse(statusCode, "User retrived successfully", resp)
	c.JSON(statusCode, rd)
}

func (base *Controller) CreateGroupDMChannel(c *gin.Context) {
	var req models.GroupDMChannelsRequest

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

	respData, statusCode, err := dm.CreateGroupDMChannel(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error creating dm channel", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("Group DM channel created successfully")
	rd := utility.BuildSuccessResponse(statusCode, "Group DM channel created successfully", respData)
	c.JSON(statusCode, rd)
}

func (base *Controller) DeleteGroupDMChannel(c *gin.Context) {
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

	statusCode, err := dm.DeleteGroupDMChannel(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group DM channel deleted successfully")
	rd := utility.BuildSuccessResponse(statusCode, "Group DM channel deleted successfully", nil)
	c.JSON(statusCode, rd)
}
