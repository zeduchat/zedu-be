package organisation

import (
	"errors"
	"net/http"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateUserPinnedOrganisation(db *gorm.DB, ids models.IDS) (int, error) {
	var (
		pinnedOrg models.UserPinnedOrganisations
		org       models.Organisation
	)

	org, err := org.CheckOrgExists(ids.OrganisationID, db)
	if err != nil {
		return http.StatusNotFound, errors.New("organisation not found")
	}

	if pinnedOrg.CheckOrganisationPinned(db, ids) {
		return http.StatusBadRequest, errors.New("organisation already pinned")
	}

	pinnedOrg.UserID = ids.UserID
	pinnedOrg.OrgID = ids.OrganisationID
	pinnedOrg.ID = utility.GenerateUUID()

	currentNoOfPinnedOrgs, err := pinnedOrg.CountCurrentUserPinnedOrganisations(db, ids)
	if err != nil {
		return http.StatusInternalServerError, errors.New("failed to count current user pinned organisations: " + err.Error())
	}

	if currentNoOfPinnedOrgs >= 5 {
		err = pinnedOrg.RemoveOldestPinnedOrganisation(db, ids.UserID)
		if err != nil {
			return http.StatusInternalServerError, errors.New("failed to remove oldest pinned organisation: " + err.Error())
		}
	}

	err = pinnedOrg.CreateUserPinnedOrganisation(db)
	if err != nil {
		return http.StatusInternalServerError, errors.New("failed to create user pinned organisation")
	}

	return http.StatusCreated, nil
}

func GetUserPinnedOrganisation(db *gorm.DB, ids models.IDS) ([]models.GetUserPinnedOrganisationsResponse, int, error) {
	var (
		pinnedOrg models.UserPinnedOrganisations
		user      models.User
	)

	if !user.CheckUserExists(db, ids.UserID) {
		return []models.GetUserPinnedOrganisationsResponse{}, http.StatusNotFound, errors.New("user not found")
	}

	pinnedOrgs, err := pinnedOrg.GetUserPinnedOrganisations(db, ids)
	if err != nil {
		return []models.GetUserPinnedOrganisationsResponse{}, http.StatusInternalServerError, errors.New("failed to get user pinned organisations: " + err.Error())
	}

	return pinnedOrgs, http.StatusOK, nil
}

func UnpinOrganisation(db *gorm.DB, ids models.IDS) (int, error) {
	var (
		pinnedOrg models.UserPinnedOrganisations
		org       models.Organisation
	)

	org, err := org.CheckOrgExists(ids.OrganisationID, db)
	if err != nil {
		return http.StatusNotFound, errors.New("organisation not found")
	}

	if !pinnedOrg.CheckOrganisationPinned(db, ids) {
		return http.StatusBadRequest, errors.New("organisation not pinned")
	}

	err = pinnedOrg.UnpinOrganisation(db, ids)
	if err != nil {
		return http.StatusInternalServerError, errors.New("failed to unpin organisation: " + err.Error())
	}

	return http.StatusOK, nil
}
