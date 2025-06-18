package organisation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) CreateUserPinnedOrganisation(c *gin.Context) {
	var req models.CreateUserPinnedOrganisationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("Failed to get user claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Validation failed", err.Error())
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	ids := models.IDS{
		UserID:         userId,
		OrganisationID: req.OrgID,
	}

	code, err := organisation.CreateUserPinnedOrganisation(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to create user pinned organisation", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Organisation pinned successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Organisation pinned successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetUserPinnedOrganisations(c *gin.Context) {

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("Failed to get user claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	ids := models.IDS{
		UserID: userId,
	}

	response, code, err := organisation.GetUserPinnedOrganisation(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to get user pinned organisation", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("User pinned organisations fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User pinned organisations fetched successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UnpinOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	ids := models.IDS{
		UserID: userId,
		OrganisationID: orgId,
	}

	code, err := organisation.UnpinOrganisation(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to unpin organisation", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Organisation unpinned successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Organisation unpinned successfully", nil)
	c.JSON(http.StatusOK, rd)
}
