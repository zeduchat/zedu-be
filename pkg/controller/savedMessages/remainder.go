package savedMessages

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/savedMessages"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) SetRemainder(c *gin.Context) {
	var req models.SetRemainderRequest
	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
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

	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserId = userId
	req.OrgId = orgId

	statusCode, err := savedMessages.SetRemainder(req, base.Db, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), nil, nil)
		c.JSON(statusCode, rd)
		return
	}

	rd := utility.BuildSuccessResponse(statusCode, "Remainder set successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) MarkCompleteSavedMessage(c *gin.Context) {
	var req models.MarkCompleteSavedMessageRequest
	orgId := c.Param("org_id")
	smId := c.Param("smId")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
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

	if _, err := uuid.Parse(smId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid saved message id format", errors.New("failed to parse saved message id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserId = userId
	req.OrgId = orgId
	req.SavedMessageID = smId

	statusCode, err := savedMessages.MarkCompleteSavedMessage(req, base.Db, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), nil, nil)
		c.JSON(statusCode, rd)
		return
	}

	rd := utility.BuildSuccessResponse(statusCode, "Saved message marked as complete successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ArchiveSavedMessage(c *gin.Context) {
	var req models.ArchiveSavedMessageRequest
	orgId := c.Param("org_id")
	smId := c.Param("smId")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(smId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid saved message id format", errors.New("failed to parse saved message id"), nil)
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

	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserId = userId
	req.OrgId = orgId
	req.SavedMessageID = smId

	statusCode, err := savedMessages.ArchiveSavedMessage(req, base.Db, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), nil, nil)
		c.JSON(statusCode, rd)
		return
	}

	rd := utility.BuildSuccessResponse(statusCode, "Saved message archived successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetCompletedSavedMessages(c *gin.Context) {
	org_id := c.Param("org_id")
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	if _, err := uuid.Parse(org_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(userId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user_id id format", errors.New("failed to parse user_id id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		OrganisationID: org_id,
		UserID:         userId,
	}

	pagination := postgresql.GetPagination(c)

	resp, paginationResponse, err := savedMessages.GetCompletedSavedMessages(base.Db, base.Logger, ids, pagination)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to get completed saved messages", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Completed saved messages retrieved successfully", resp, paginationData)
	c.JSON(http.StatusOK, rd)

}
func (base *Controller) GetArchivedSavedMessages(c *gin.Context) {
	org_id := c.Param("org_id")
	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	if _, err := uuid.Parse(org_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(userId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user_id id format", errors.New("failed to parse user_id id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		OrganisationID: org_id,
		UserID:         userId,
	}

	pagination := postgresql.GetPagination(c)

	resp, paginationResponse, err := savedMessages.GetArchivedSavedMessages(base.Db, base.Logger, ids, pagination)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to get archived saved messages", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Archived saved messages retrieved successfully", resp, paginationData)
	c.JSON(http.StatusOK, rd)

}
