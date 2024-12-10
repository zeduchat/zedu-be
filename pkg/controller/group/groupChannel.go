package group

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/group"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AssignGroupChannel(c *gin.Context) {
	var (
		ids map[string]string = map[string]string{
			"organisation_id": c.Param("org_id"),
			"group_id":        c.Param("group_id"),
			"channel_id":      c.Param("channel_id"),
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
	if _, err := uuid.Parse(ids["channel_id"]); err != nil {
		base.Logger.Error("Invalid channel id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid channel id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := group.AssignGroupChannel(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to assign group channel", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to assign group channel", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group channel assigned successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Group channel assigned successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) RemoveGroupChannel(c *gin.Context) {
	var (
		ids map[string]string = map[string]string{
			"organisation_id": c.Param("org_id"),
			"group_id":        c.Param("group_id"),
			"channel_id":      c.Param("channel_id"),
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
	if _, err := uuid.Parse(ids["channel_id"]); err != nil {
		base.Logger.Error("Invalid channel id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid channel id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := group.RemoveGroupChannel(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to remove group channel", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to remove group channel", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group channel removed successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Group channel removed successfully", nil)
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

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "user claims not found", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to get user claims", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	ids := map[string]string{
		"organisation_id": org_id,
		"user_id":         userId,
	}

	channels, err := group.GetChannelsNotInGroup(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to get user channels not in a group", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to get user channels not in a group", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("User channels not in a group fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User channels not in a group fetched successfully", channels)
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

func (base *Controller) GetDiscoverableChannels(c *gin.Context) {
	var (
		org_id = c.Param("org_id")
	)

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	ids := map[string]string{
		"organisation_id": org_id,
		"user_id":         userId,
	}

	channels, err := group.GetDiscoverableChannels(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to get discoverable channels for user", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to get discoverable channels for user", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Discoverable Channels for user fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Discoverable Channels for user fetched successfully", channels)
	c.JSON(http.StatusOK, rd)
}
