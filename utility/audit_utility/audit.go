package audit_utility

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func LogUserLogin(c *gin.Context, db *gorm.DB, extReq request.ExternalRequest,
	userID, accessID string, Organisations []models.Organisation) error {
	ipAddress := GetClientIP(c)

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
	accessId, _ := uuid.FromString(accessID)
	loginActivity := &models.LoginActivity{
		ID:             utility.GenerateUUID(),
		UserID:         userID,
		OrganisationID: &organisationID,
		AccessID:       &accessId,
		LoginAt:        GetCurrentTime(),
		IPAddress:      ipAddress,
		Location:       location,
		Device:         browser,
		IsLive:         true,
	}

	return loginActivity.Create(db)
}

func GetClientIP(c *gin.Context) string {

	ip := c.GetHeader("X-Forwarded-For")
	if ip != "" {
		ip = strings.Split(ip, ",")[0]
		if !isPrivateIP(ip) {
			return ip
		}
	}

	ip = c.GetHeader("X-Real-IP")
	if ip != "" && !isPrivateIP(ip) {
		return ip
	}

	ip, _, _ = net.SplitHostPort(c.Request.RemoteAddr)
	return ip
}

func isPrivateIP(ip string) bool {
	privateIPBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}

	for _, block := range privateIPBlocks {
		_, cidr, _ := net.ParseCIDR(block)
		parsedIP := net.ParseIP(ip)
		if cidr.Contains(parsedIP) {
			return true
		}
	}
	return false
}
