package group

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/group"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AssignGroupChannel(c *gin.Context) {
	var (
		ids map[string]any = map[string]any{
			"organisation_id": c.Param("org_id"),
			"group_id":        c.Param("group_id"),
		}
		req models.AssignGroupChannelRequest
	)

	if _, err := uuid.Parse(ids["organisation_id"].(string)); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(ids["group_id"].(string)); err != nil {
		base.Logger.Error("Invalid group id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid group id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if len(req.Channels) == 0 {
		base.Logger.Error("No channel id provided")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "No channel id provided", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids["channel_ids"] = req.Channels

	err := group.AssignGroupChannel(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to assign group channels", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to assign group channels", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group channels assigned successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Group channels assigned successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) RemoveGroupChannel(c *gin.Context) {
	var (
		ids map[string]any = map[string]any{
			"organisation_id": c.Param("org_id"),
			"group_id":        c.Param("group_id"),
		}
		req models.AssignGroupChannelRequest
	)

	if _, err := uuid.Parse(ids["organisation_id"].(string)); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(ids["group_id"].(string)); err != nil {
		base.Logger.Error("Invalid group id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid group id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if len(req.Channels) == 0 {
		base.Logger.Error("No channel id provided")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "No channel id provided", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids["channel_ids"] = req.Channels

	err := group.RemoveGroupChannel(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to remove group channels", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to remove group channels", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group channels removed successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Group channels removed successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetGroupChannels(c *gin.Context) {
	var (
		ids map[string]string = map[string]string{
			"organisation_id": c.Param("org_id"),
			"group_id":        c.Param("group_id"),
		}
	)

	if _, err := uuid.Parse(ids["organisation_id"]); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(ids["group_id"]); err != nil {
		base.Logger.Error("Invalid group id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid group id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	channels, err := group.GetGroupChannels(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to get group channels", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to get group channels", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group channels fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Group channels fetched successfully", channels)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetChannelsNotInGroup(c *gin.Context) {
	var (
		org_id = c.Param("org_id")
	)

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	channels, err := group.GetChannelsNotInGroup(base.Db, org_id)
	if err != nil {
		base.Logger.Error("Failed to get channels not in group", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to get channels not in group", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Channels not in group fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channels not in group fetched successfully", channels)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) MoveGroupChannel(c *gin.Context) {
	var (
		ids map[string]string = map[string]string{
			"organisation_id": c.Param("org_id"),
			"new_group_id":    c.Param("group_id"),
			"channel_id":      c.Param("channel_id"),
		}
	)

	if _, err := uuid.Parse(ids["organisation_id"]); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(ids["new_group_id"]); err != nil {
		base.Logger.Error("Invalid group id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid group id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(ids["channel_id"]); err != nil {
		base.Logger.Error("Invalid channel id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid channel id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := group.MoveGroupChannel(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to move group channel", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to move group channel", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group channel moved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Group channel moved successfully", nil)
	c.JSON(http.StatusOK, rd)
}
