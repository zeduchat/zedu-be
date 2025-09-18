package organisation

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/organisation"
	service "github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) CreateOrganisation(c *gin.Context) {

	var (
		req = models.CreateOrgRequestModel{}
	)

	err := c.ShouldBind(&req)
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

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	respData, err := service.CreateOrganisation(req, base.Db.Postgresql, userId, base.Logger)
	if err != nil {
		base.Logger.Error("error creating organisation", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error creating organisation", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("organisation created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Organisation Created Successfully", respData)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to delete organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to delete organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	orgData, err := service.GetOrganisation(orgId, userId, base.Db.Postgresql)

	if err != nil {
		switch err.Error() {
		case "organisation not found":
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), "failed to retrieve organisation", nil)
			c.JSON(http.StatusNotFound, rd)
		case "user not authorised to retrieve this organisation":
			rd := utility.BuildErrorResponse(http.StatusForbidden, "error", err.Error(), "failed to retrieve organisation", nil)
			c.JSON(http.StatusForbidden, rd)
		default:
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to retrieve organisation", err.Error(), nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	base.Logger.Info("organisation retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "organisation retrieved successfully", orgData)

	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetAllChannelssInOrganisation(c *gin.Context) {
	orgID := c.Param("org_id")

	if _, err := uuid.Parse(orgID); err != nil {
		base.Logger.Info("error parsing org id")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid org id format", errors.New("failed to parse org id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to update organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	ids := models.IDS{
		UserID:         userId,
		OrganisationID: orgID,
	}

	respData, paginationResponse, code, err := organisation.GetAllChannelssInTeam(base.Db, c, ids)
	if err != nil {
		base.Logger.Info("error fetching channels")
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(respData),
	}

	base.Logger.Info("channels fetched successfully")
	rd := utility.BuildSuccessResponse(code, "channels fetched successfully", respData, paginationData)
	c.JSON(code, rd)
}

func (base *Controller) UpdateOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")
	var updateReq models.UpdateOrgRequestModel

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to update organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to update organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	if err := c.ShouldBind(&updateReq); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&updateReq)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	updatedOrg, code, err := service.UpdateOrganisation(orgId, userId, updateReq, base.Db.Postgresql, base.Logger)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "failed to update organisation", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("organisation updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Organisation updated successfully", updatedOrg)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to delete organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to delete organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	if err := service.DeleteOrganisation(orgId, userId, base.Db.Postgresql); err != nil {
		switch err.Error() {
		case "organisation not found":
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), "failed to delete organisation", nil)
			c.JSON(http.StatusNotFound, rd)
		case "user not authorised to delete this organisation":
			rd := utility.BuildErrorResponse(http.StatusForbidden, "error", err.Error(), "failed to delete organisation", nil)
			c.JSON(http.StatusForbidden, rd)
		default:
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to delete organisation", err.Error(), nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	base.Logger.Info("organisation deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusNoContent, "", nil)
	c.JSON(http.StatusNoContent, rd)
}

func (base *Controller) AddUserToOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")

	var req models.AddUserToOrgRequestModel
	if err := c.ShouldBind(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to update organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	err = service.AddUserToOrganisation(orgId, req, base.Db.Postgresql)

	if err != nil {
		switch err.Error() {
		case "organisation not found":
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), "failed to add user to organisation", nil)
			c.JSON(http.StatusNotFound, rd)
		case "user not found":
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), "failed to add user to organisation", nil)
			c.JSON(http.StatusNotFound, rd)
		case "user already added to organisation":
			rd := utility.BuildErrorResponse(http.StatusConflict, "error", err.Error(), "failed to add user to organisation", nil)
			c.JSON(http.StatusNotFound, rd)
		default:
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to add user to organisation", err.Error(), nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	base.Logger.Info("user added to organisation successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "user added to organisation successfully", nil)

	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetUsersBotsInOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")
	query := c.Query("query")
	includeBots := true
	if c.Query("include_bots") == "false" {
		includeBots = false
	}

	if _, err := uuid.Parse(orgId); err != nil {
		base.Logger.Error("error parsing org id")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to retrieve users and bots", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("unable to get user claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to retrieve users and bots", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	queryParams := map[string]any{
		"query":        query,
		"include_bots": includeBots,
	}

	users, paginationResponse, err := service.GetUsersBotsInOrganisation(orgId, userId, base.Db.Postgresql, c, queryParams)
	if err != nil {
		switch err.Error() {
		case "organisation not found":
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), "failed to retrieve users and bots", nil)
			c.JSON(http.StatusNotFound, rd)
		case "user does not have access to the organisation":
			rd := utility.BuildErrorResponse(http.StatusForbidden, "error", err.Error(), "failed to retrieve users and bots", nil)
			c.JSON(http.StatusNotFound, rd)
		default:
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to retrieve users and bots", err.Error(), nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(users),
	}

	base.Logger.Info("users and bots retrieved successfully")
	response := utility.BuildSuccessResponse(http.StatusOK, "users and bots retrieved successfully", users, paginationData)
	c.JSON(http.StatusOK, response)
}

func (base *Controller) GetLoadingMetrics(c *gin.Context) {
	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to retrieve loading metrics", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	loadingMetrics, err := organisation.LoadOrganisationMetrics(orgId, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to retrieve loading metrics", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("loading metrics retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "loading metrics retrieved successfully", loadingMetrics)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetStarted(c *gin.Context) {
	orgID := c.Param("org_id")

	if _, err := uuid.Parse(orgID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to retrieve loading metrics", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to retrieve users", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	ids := models.IDS{
		OrganisationID: orgID,
		UserID:         userId,
	}

	getstarted, err := organisation.FetchGetStarted(base.Db, ids)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to retrieve get-started info", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("loading metrics retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "get-started info retrieved successfully", getstarted)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAllOrganisations(c *gin.Context) {
	organizations, paginationResponse, err := service.GetAllOrganisations(base.Db.Postgresql, c)

	if err != nil {
		base.Logger.Error("Failed to fetch organisation", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to fetch organisation", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(organizations),
	}

	base.Logger.Info("organisation retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "organisation retrieved successfully", organizations, paginationData)

	c.JSON(http.StatusOK, rd)

}
