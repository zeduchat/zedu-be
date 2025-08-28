package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/mongodb"
	"github.com/hngprojects/telex_be/utility"
)

func Authorize(db *gorm.DB) gin.HandlerFunc {
	// if no role is passed it would assume default user role
	return func(c *gin.Context) {

		var (
			tokenStr     string
			access_token models.AccessToken
			oum          models.OrgUserManagement
		)

		bearerToken := c.GetHeader("Authorization")
		strArr := strings.Split(bearerToken, " ")
		if len(strArr) == 2 {
			tokenStr = strArr[1]
		}

		if tokenStr == "" {
			r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token could not be found!", "Unauthorized", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, r)
			return
		}

		token, err := TokenValid(tokenStr)
		if err != nil {
			r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, r)
			return
		}

		// access user claims
		claims := token.Claims.(jwt.MapClaims)

		// check if user id exists and fetch it
		userID, ok := claims["user_id"].(string) //convert the interface to string
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil))
			return
		}

		org_id, ok := claims["org_id"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil))
			return
		}

		if org_id == "00000000-0000-0000-0000-000000000000" {
			org := models.Organisation{}
			orgs, _ := org.GetOrganisationsByUserID(db, userID)
			if len(orgs) > 0 {
				claims["org_id"] = orgs[0].ID
			}

			// update user current org entry
			user := models.User{}
			user, _ = user.GetUserByID(db, userID)
			user.CurrentOrg, err = uuid.FromString(orgs[0].ID)
			err = user.Update(db)
		}

		ids := models.IDS{
			OrganisationID: org_id,
			UserID:         userID,
		}

		if oum.CheckIsUserDeactivated(db, ids) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "User is deactivated from organisation", "Unauthorized", nil))
			return
		}

		// check if access id exists and fetch it
		accessID, ok := claims["access_uuid"].(string) //convert the interface to string
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil))
			return
		}

		// check user session and also if token is valid in stored session
		access_token = models.AccessToken{ID: accessID}
		if code, err := access_token.GetByID(db); err != nil {
			c.AbortWithStatusJSON(code, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil))
			return
		}

		// check if session is valid
		if access_token.LoginAccessToken != tokenStr || userID != access_token.OwnerID || !access_token.IsLive {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Session is invalid!", "Unauthorized", nil))
			return
		}

		c.Set("userClaims", claims)

		// call the next handler
		c.Next()

	}
}

func GetIdFromToken(c *gin.Context) (string, any) {
	var tokenStr string
	bearerToken := c.GetHeader("Authorization")
	strArr := strings.Split(bearerToken, " ")
	if len(strArr) == 2 {
		tokenStr = strArr[1]
	}

	if tokenStr == "" {
		r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token could not be found!", "Unauthorized", nil)
		return "", r
	}

	token, err := TokenValid(tokenStr)
	if err != nil {
		r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil)
		return "", r
	}

	// access user claim
	claims := token.Claims.(jwt.MapClaims)
	id, ok := claims["user_id"].(string)
	if !ok {
		return "", utility.BuildErrorResponse(http.StatusForbidden, "error", "Forbidden", "Unauthorized", nil)
	}
	return id, ""
}

func APIKeyAuthMiddleware(db *gorm.DB, logger *utility.Logger, isDBAuth bool) gin.HandlerFunc {
	return func(c *gin.Context) {

		if isDBAuth {
			store := mongodb.MongoStore{}

			if !store.IsClientAvailable() {
				logger.Error("MongoDB Client is still connecting. Please try again...")
				rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "MongoDB client is still connecting. Please try again...", "Bad Request", nil)
				c.JSON(http.StatusBadRequest, rd)
				c.Abort()
				return
			}
		}

		apiKey := c.GetHeader("X-AGENT-API-KEY")
		if apiKey == "" {
			rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "API key not found", "Unauthorized", nil)
			c.JSON(http.StatusUnauthorized, rd)
			c.Abort()
			return
		}

		org_id_slug, agent_id_slug, err := models.ValidateAgentApiKey(db, apiKey)
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "invalid API key", err.Error(), nil)
			c.JSON(http.StatusUnauthorized, rd)
			c.Abort()
			return
		}

		ids := models.IDS{
			OrganisationID: org_id_slug,
			AgentID:        agent_id_slug,
		}

		organisation_id, agent_id, err := VerifyCredentials(db, ids)
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "invalid API key", err.Error(), nil)
			c.JSON(http.StatusUnauthorized, rd)
			c.Abort()
			return
		}

		c.Set("org_id", organisation_id)
		c.Set("agent_id", agent_id)

		c.Next()
	}
}

func VerifyCredentials(db *gorm.DB, ids models.IDS) (string, string, error) {
	var orgint models.OrganisationIntegrations

	err := db.Where("org_id = ? AND integration_id = ?", ids.OrganisationID, ids.AgentID).First(&orgint).Error
	if err != nil {
		return "", "", fmt.Errorf("agent does not exist in organisation")
	}

	return orgint.OrgID, orgint.IntegrationID, nil
}

func AdminAuthorize(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		var (
			tokenStr     string
			access_token models.AccessToken
		)

		bearerToken := c.GetHeader("Authorization")
		strArr := strings.Split(bearerToken, " ")
		if len(strArr) == 2 {
			tokenStr = strArr[1]
		}

		if tokenStr == "" {
			r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token could not be found!", "Unauthorized", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, r)
			return
		}

		token, err := TokenValid(tokenStr)
		if err != nil {
			r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, r)
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		userID, ok := claims["admin_id"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil))
			return
		}

		// check if access id exists and fetch it
		accessID, ok := claims["access_uuid"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil))
			return
		}

		access_token = models.AccessToken{ID: accessID}
		if code, err := access_token.GetByID(db); err != nil {
			c.AbortWithStatusJSON(code, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Token is invalid!", "Unauthorized", nil))
			return
		}

		if access_token.LoginAccessToken != tokenStr || userID != access_token.OwnerID || !access_token.IsLive {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Session is invalid!", "Unauthorized", nil))
			return
		}

		c.Set("adminClaims", claims)

		// call the next handler
		c.Next()

	}
}
