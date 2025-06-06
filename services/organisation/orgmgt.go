package organisation

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func CreateOrgUserManagement(db *gorm.DB, userID, orgID string) error {
	var orgRole models.OrgRole

	orgRole, err := orgRole.GetAOrgRoleByName(db, "Administrator")
	if err != nil {
		return err
	}

	orgUserManagement := models.OrgUserManagement{
		UserID:         userID,
		OrganisationID: orgID,
		RoleID:         orgRole.ID,
		Status:         "active",
	}

	err = orgUserManagement.CreateOrgUserManagement(db)
	if err != nil {
		return err
	}
	return nil
}

func CountMetrics(db *gorm.DB, userID, orgID string) (models.OrgUserMetricsResponse, error) {
	var (
		oum models.OrgUserManagement
	)

	countMetricsData, err := oum.CountMetrics(db, orgID)
	if err != nil {
		return countMetricsData, err
	}
	return countMetricsData, nil
}

func UpdateMember(db *gorm.DB, ownerId, orgID, userID string, req models.UpdateMemberRequest) (models.OrgUserManagement, error) {
	var (
		oum models.OrgUserManagement
		o   models.Organisation
	)

	isowner, err := o.IsOwnerOfOrganisation(db, ownerId, orgID)
	if err != nil {
		return oum, err
	}

	if !isowner {
		return oum, errors.New("user is not the owner of the organisation")
	}

	resp, err := oum.UpdateMember(db, orgID, userID, req)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func GetOrganisationInvites(c *gin.Context, db *gorm.DB, userID, orgID string) ([]models.Invitation, postgresql.PaginationResponse, error) {
	var (
		o models.Organisation
	)

	invitations, paginationResponse, err := o.GetOrganisationInvites(c, db, userID, orgID)
	if err != nil {
		return invitations, paginationResponse, err
	}

	return invitations, paginationResponse, nil
}
func UpdateDeviceNotification(db *gorm.DB, logger *utility.Logger, req models.DeviceNotificationSettings) (models.DeviceNotification, int, error) {

	respData, code, err := req.UpdateDeviceOrgNotification(db)

	if err != nil {
		logger.Error("failed to update notification settings: %v", err)
		return respData, code, fmt.Errorf("update failed")
	}

	logger.Info("updated user preference successfully")

	return respData, code, nil
}

func GetOrCreateDeviceNotification(db *gorm.DB, logger *utility.Logger, ids map[string]string) (models.DeviceNotification, error) {

	deviceNS := models.DeviceNotificationSettings{
		OrgID:      ids["org_id"],
		UserID:     ids["user_id"],
		DeviceType: ids["device_type"],
	}

	resp, err := deviceNS.GetOrCreateDeviceOrgNotification(db)

	if err != nil {

		logger.Error("failed to fetch device notification settings: %v", err)
		return resp, fmt.Errorf("failed to fetch device notification settings")

	}

	logger.Info("fetched user notification settings successfully")

	return resp, nil
}
