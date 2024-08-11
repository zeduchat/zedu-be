package teams

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateTeam(db *gorm.DB, req models.CreateTeamRequest, userId string) (models.Team, error) {

	team := models.Team{
		ID:          utility.GenerateUUID(),
		OwnerId:     userId,
		Name:        req.Name,
		Description: req.Description,
	}

	err := team.CreateTeam(db)
	if err != nil {
		return models.Team{}, err
	}
	return team, nil
}

func GetTeamByID(db *gorm.DB, teamID string) (models.Team, error) {
	var t models.Team

	team, err := t.GetTeamByID(db, teamID)

	if err != nil {
		return team, err
	}

	return team, nil
}

func GetAllRoomsInTeam(db *gorm.DB, teamID string) ([]models.Room, models.Team, error) {
	var t models.Team

	rooms, additionalInfo, err := t.GetAllRoomsInTeam(db, teamID)
	if err != nil {
		return rooms, models.Team{}, err
	}

	return rooms, additionalInfo, nil
}

func DeleteTeam(db *gorm.DB, userID, teamID string) error {
	var t models.Team

	t.ID = teamID

	err := t.DeleteTeam(db, userID)
	if err != nil {
		return err
	}
	return nil
}
