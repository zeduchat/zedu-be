package user

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

func ReplaceUserRole(userID, orgID, roleID string, db *gorm.DB, c *gin.Context) (*models.User, error) {

	var (
		role   = models.OrgRole{}
		orgMgt = models.OrgUserManagement{}
	)

	userExists := postgresql.CheckExists(db, &orgMgt, "user_id = ? AND organisation_id = ?", userID, orgID)
	if !userExists {
		return nil, errors.New("user not in organization")
	}

	roleExists := postgresql.CheckExists(db, &role, "id = ?", roleID)
	if !roleExists {
		return nil, errors.New("invalid role")
	}

	userData, err := role.UpdateUserRole(db, userID, orgID, roleID, c)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	return userData, nil
}

func UpdateUserIdentity(userID string, roleID string, db *gorm.DB, c *gin.Context) (*models.User, int, error) {

	var (
		user = models.User{}
		role = models.UserRole{}
	)
	currentUser, err := middleware.GetUserClaims(c, db, "user_id")
	if err == nil {
		return nil, http.StatusNotFound, errors.New("user not found")
	}
	userid, ok := currentUser.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	user, code, err := GetUser(userid, db)
	if err != nil {
		return nil, code, err
	}

	isSuperAdmin := user.CheckUserIsAdmin(db)
	if !isSuperAdmin {
		return nil, http.StatusUnauthorized, errors.New("user is not a super admin")
	}

	userExists := postgresql.CheckExists(db, &user, "id = ?", userID)
	if !userExists {
		return nil, http.StatusUnauthorized, errors.New("invalid user")
	}

	roleExists := postgresql.CheckExists(db, &role, "id = ?", roleID)
	if !roleExists {
		return nil, http.StatusUnauthorized, errors.New("invalid role")
	}

	userData, err := role.UpdateUserIdentity(db, userID, roleID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf(err.Error())
	}

	return userData, http.StatusOK, nil
}
