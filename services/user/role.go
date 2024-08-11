package user

import (
	"errors"
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

func ReplaceUserRole(userID string, roleID string, db *gorm.DB) (*models.User, error) {

	var (
		user = models.User{}
		role = models.OrgRole{}
	)

	userExists := postgresql.CheckExists(db, &user, "id = ?", userID)
	if !userExists {
		return nil, errors.New("invalid user")
	}

	roleExists := postgresql.CheckExists(db, &role, "id = ?", roleID)
	if !roleExists {
		return nil, errors.New("invalid role")
	}

	userData, err := role.UpdateUserRole(db, userID, roleID)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	return userData, nil
}
