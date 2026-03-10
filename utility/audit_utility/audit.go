package audit_utility

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func LogUserLogin(c *gin.Context, db *gorm.DB, extReq request.ExternalRequest,
	userID, accessID string, Organisations []models.Organisation) error {
	ipAddress := GetClientIP(c)

	var location, organisationID string
	response, err := extReq.SendExternalRequest("ipinfo_resolve_ip", ipAddress)
	if err != nil {
		location = "error"
	}

	info, ok := response.(external_models.IPInfoResponse)
	if !ok {
		location = "error"
	}
	location = info.City

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
		Device:         getBrowserName(browser),
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

func getBrowserName(userAgent string) string {
	userAgent = strings.ToLower(userAgent)

	switch {
	case strings.Contains(userAgent, "brave"):
		return "Brave"
	case strings.Contains(userAgent, "opr") || strings.Contains(userAgent, "opera"):
		return "Opera"
	case strings.Contains(userAgent, "vivaldi"):
		return "Vivaldi"
	case strings.Contains(userAgent, "chrome"):
		return "Chrome"
	case strings.Contains(userAgent, "firefox"):
		return "Firefox"
	case strings.Contains(userAgent, "safari") && !strings.Contains(userAgent, "chrome"):
		return "Safari"
	case strings.Contains(userAgent, "edg"):
		return "Edge"
	case strings.Contains(userAgent, "msie") || strings.Contains(userAgent, "trident"):
		return "Internet Explorer"
	default:
		return "Unknown"
	}
}

func CreateAuditLog(db *gorm.DB, actorID, actorEmail, actorRole string, action models.AuditAction, resourceType models.ResourceType, resourceID string, oldValues, newValues, description, ipAddress, userAgent string, success bool) error {
	audit := models.AuditLog{
		ID:           utility.GenerateUUID(),
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		ActorRole:    actorRole,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OldValues:    oldValues,
		NewValues:    newValues,
		Description:  description,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Success:      success,
	}

	return audit.CreateAuditLog(db)
}
