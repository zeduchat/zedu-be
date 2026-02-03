package credentials

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/credentials"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateCredential(c *gin.Context) {
	var req models.CredentialRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.UserId = userClaims["user_id"].(string)

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := credentials.CreateCredentialService(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error creating credential", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Credential created successfully")
	rd := utility.BuildSuccessResponse(code, "Credential created successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) GetSkillCredentials(c *gin.Context) {
	var req models.CredentialRequest
	req.SkillId = c.Param("skill_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.SkillId); err != nil {
		base.Logger.Info("invalid skill id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid skill id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)
	req.UserId = userClaims["user_id"].(string)

	resp, code, err := credentials.GetSkillCredentialsService(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error fetching credentials", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Skill Credentials fetched successfully", resp)
	c.JSON(code, rd)
}

func (base *Controller) GetCredentialByID(c *gin.Context) {
	var credentialId = c.Param("credential_id")

	if _, err := uuid.Parse(credentialId); err != nil {
		base.Logger.Info("invalid skill id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid skill id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, code, err := credentials.GetCredentialByIDService(credentialId, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error fetching credential", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Credential fetched successfully", resp)
	c.JSON(code, rd)
}
