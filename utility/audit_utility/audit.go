package audit_utility

import (
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func LogUserLogin(c *gin.Context, db *gorm.DB, extReq request.ExternalRequest,
	userID, accessID string, Organisations []models.Organisation) error {
	ipAddress := c.ClientIP()
	var location, organisationID string
	response, err := extReq.SendExternalRequest("ipinfo_resolve_ip", ipAddress)
	if err != nil {
		location = "unknown"
	} else {
		if respMap, ok := response.(map[string]interface{}); ok {
			if city, ok := respMap["city"].(string); ok {
				location = city
			} else {
				location = "unknown"
			}
		} else {
			location = "error"
		}
	}

	if len(Organisations) > 0 {
		organisationID = Organisations[0].ID
	}

	browser := c.GetHeader("User-Agent")

	loginActivity := &models.LoginActivity{
		ID:             utility.GenerateUUID(),
		UserID:         userID,
		OrganisationID: &organisationID,
		AccessID:       accessID,
		LoginAt:        GetCurrentTime(),
		IPAddress:      ipAddress,
		Location:       location,
		Device:         browser,
	}

	return loginActivity.Create(db)
}
