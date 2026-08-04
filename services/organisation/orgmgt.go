package organisation

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
)

func CreateOrgUserManagement(db *gorm.DB, userID, orgID string) error {
	var orgRole models.OrgRole

	orgRole, err := orgRole.GetAOrgRoleByName(db, models.OrgRoleNameAdministrator)
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
	var oum models.OrgUserManagement

	if !userCanOrOwner(db, ownerId, orgID, models.PermChangeUserOrgRole) {
		return oum, errors.New("you do not have permission to update members")
	}

	resp, err := oum.UpdateMember(db, orgID, userID, req)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func GetOrganisationInvites(c *gin.Context, db *gorm.DB, userID, orgID, invite_status string) ([]models.Invitation, postgresql.PaginationResponse, error) {
	var (
		o models.Organisation
	)

	ids := models.IDS{
		UserID:         userID,
		OrganisationID: orgID,
	}

	invitations, paginationResponse, err := o.GetOrganisationInvites(c, db, ids, invite_status)
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

func ChangeMemberActiveStatus(db *gorm.DB, rdb *redis.Client, logger *utility.Logger, c *gin.Context, req models.ChangeMemberActiveStatus, ids map[string]string) (int, error) {
	var (
		user      models.User
		adminUser models.User
		userID    = ids["user_id"]
		orgID     = ids["org_id"]
		adminID   = ids["admin_user_id"]
	)

	if userID == adminID {
		return http.StatusForbidden, errors.New("you cannot change your own active status")
	}

	if !user.CheckUserExists(db, userID) {
		return http.StatusNotFound, errors.New("user does not exist")
	}
	user.ID = userID

	if !adminUser.CheckUserExists(db, adminID) {
		return http.StatusUnauthorized, errors.New("admin user does not exist")
	}

	if !userCanOrOwner(db, adminID, orgID, models.PermChangeUserOrgRole) {
		return http.StatusForbidden, errors.New("you do not have permission to change member status")
	}

	tx := db.Begin()
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if req.Activate {
		if err := user.ActivateOrgMember(tx, orgID); err != nil {
			return http.StatusInternalServerError, err
		}
	} else {
		if err := user.DeactivateOrgMember(tx, orgID); err != nil {
			return http.StatusInternalServerError, err
		}

		// revoke the deactivated user's session
		var userToken models.AccessToken
		userToken.OwnerID = userID
		if _, err := userToken.GetMostRecentAccessToken(tx); err == nil {
			accessToken := models.AccessToken{ID: userToken.ID, OwnerID: userID}
			if err := accessToken.RevokeAccessToken(tx); err != nil {
				return http.StatusInternalServerError, fmt.Errorf("failed to revoke user session: %w", err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to commit transaction: %v", err)
	}
	committed = true

	if rdb != nil {
		isDeactivated := !req.Activate
		_ = rd.RedisSet(rdb, "user_deactivated:"+userID+":"+orgID, isDeactivated, 24*time.Hour)
		if logger != nil {
			logger.Info("updated redis entry for user deactivation status: user_deactivated:%s:%s", userID, orgID)
		}
	}

	return http.StatusOK, nil
}

func SearchUsersInOrganisation(db *gorm.DB, orgID, searchTerm string) ([]models.UserInOrgResponse, error) {
	var (
		o   models.Organisation
		oum models.OrgUserManagement
	)

	_, err := o.CheckOrgExists(orgID, db)
	if err != nil {
		return nil, err
	}

	users, err := oum.SearchUsersInOrganisation(db, orgID, searchTerm)

	if err != nil {
		return nil, err
	}

	return users, nil
}

func UpdateMemberRole(db *gorm.DB, rdb *redis.Client, logger *utility.Logger, ids models.IDS) (int, error) {
	var (
		oum models.OrgUserManagement
		r   models.OrgRole
	)

	targetRole, err := r.GetAOrgRoleById(db, ids.RoleID)
	if err != nil {
		return http.StatusNotFound, errors.New("provided role does not exist")
	}

	var currentMembership models.OrgUserManagement
	var targetCurrentRoleName string
	m, err := currentMembership.GetByIDs(db, ids.UserID, ids.OrganisationID)
	if err == nil && m.RoleID != "" {
		if cr, err := r.GetAOrgRoleById(db, m.RoleID); err == nil {
			targetCurrentRoleName = cr.Name
		}
	}

	isTargetAdminOrOwner := targetRole.Name == models.OrgRoleNameAdministrator ||
		targetRole.Name == models.OrgRoleNameOwner ||
		targetCurrentRoleName == models.OrgRoleNameAdministrator ||
		targetCurrentRoleName == models.OrgRoleNameOwner

	if isTargetAdminOrOwner {
		if !isUserOwnerOrOwnerRole(db, ids.OwnerID, ids.OrganisationID) {
			return http.StatusForbidden, errors.New("only organisation owners can manage administrator roles")
		}
	} else {
		if !userCanOrOwner(db, ids.OwnerID, ids.OrganisationID, models.PermChangeUserOrgRole) {
			return http.StatusForbidden, errors.New("you do not have permission to update member roles")
		}
	}

	err = oum.UpdateMemberRole(db, ids)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update member role: %w", err)
	}

	if rdb != nil {
		_, err = rd.RedisDelete(rdb, "user_org_role:"+ids.UserID+":"+ids.OrganisationID)
		if err != nil {
			logger.Error("failed to delete redis entry for user role: %s", "user_org_role:"+ids.UserID+":"+ids.OrganisationID)
		}
		logger.Info("updated redis entry for user role: %s", "user_org_role:"+ids.UserID+":"+ids.OrganisationID)

	}

	return http.StatusOK, nil
}
