package group

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func CreateGroup(db *storage.Database, req models.CreateGroupRequest) (*models.Group, error) {

	group := models.Group{
		ID:             utility.GenerateUUID(),
		Name:           req.Name,
		Description:    req.Description,
		OrganisationID: req.OrganisationID,
	}

	err := group.CreateGroup(db.Postgresql)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func GetGroup(db *storage.Database, ids map[string]string) (models.Group, error) {
	var (
		g   models.Group
		org models.Organisation
	)

	_, err := org.CheckOrgExists(ids["organisation_id"], db.Postgresql)
	if err != nil {
		return models.Group{}, err
	}

	response, err := g.GetGroup(db.Postgresql, ids)
	if err != nil {
		return models.Group{}, err
	}
	return response, nil
}

func GetGroups(db *storage.Database, org_id string) ([]models.Group, error) {
	var (
		g   models.Group
		org models.Organisation
	)

	_, err := org.CheckOrgExists(org_id, db.Postgresql)
	if err != nil {
		return []models.Group{}, err
	}

	response, err := g.GetGroups(db.Postgresql, org_id)
	if err != nil {
		return []models.Group{}, err
	}
	return response, nil
}

func UpdateGroup(db *storage.Database, req models.UpdateGroupRequest, ids map[string]string) (*models.Group, error) {
	var (
		org models.Organisation
		g   models.Group
	)

	_, err := org.CheckOrgExists(ids["organisation_id"], db.Postgresql)
	if err != nil {
		return nil, err
	}

	err = g.UpdateGroup(db.Postgresql, req, ids)
	if err != nil {
		return nil, err
	}

	return &g, nil
}

func DeleteGroup(db *storage.Database, ids map[string]string) error {
	var (
		org models.Organisation
		g   models.Group
	)

	_, err := org.CheckOrgExists(ids["organisation_id"], db.Postgresql)
	if err != nil {
		return err
	}

	err = g.DeleteGroup(db.Postgresql, ids)
	if err != nil {
		return err
	}

	return nil
}
