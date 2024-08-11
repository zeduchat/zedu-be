package models

import (
	"errors"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Team struct {
	ID                 string         `gorm:"type:uuid;primary_key" json:"team_id"`
	Name               string         `gorm:"column:name;unique;type:text;not null" json:"name"`
	Description        string         `gorm:"column:description;type:text;not null" json:"description"`
	OwnerId            string         `gorm:"column:owner_id;type:uuid" json:"owner_id"`
	RoomsCount         int64           `gorm:"-" json:"rooms_count"`
	TotalMessagesCount int64            `gorm:"-" json:"total_messages_count"`
	CreatedAt          time.Time      `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	Rooms []Room `gorm:"foreignKey:TeamID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"rooms"`
}

type CreateTeamRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

func (t *Team) CreateTeam(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &t)
	if err != nil {
		return err
	}
	return nil
}

func (t *Team) GetTeamByID(db *gorm.DB, team_id string) (Team, error) {
	var (
		team Team
		r Room
	)

	exists := postgresql.CheckExists(db, &t, "id = ?", team_id)
	if !exists {
		return team, errors.New("team does not exist")
	}

	err, _ := postgresql.SelectOneFromDb(db.Preload("Rooms"), &team, "id = ?", team_id)
	if err != nil {
		return team, errors.New("error fetching team")
	}

	roomsCount, err := r.CountTeamRooms(db, team.ID)
	if err != nil {
		return team, err
	}

	team.RoomsCount = roomsCount

	return team, nil
}

func (t *Team) GetAllRoomsInTeam(db *gorm.DB, team_id string) ([]Room, Team ,error) {
	var (
		rooms []Room
		team Team

	)

	exists := postgresql.CheckExists(db, &t, "id = ?", team_id)
	if !exists {
		return rooms, team ,errors.New("team does not exist")
	}

	err := postgresql.SelectAllFromDb(db, "desc", &rooms, "team_id = ?", team_id)
	if err != nil {
		return rooms, team ,err
	}

	totalRoomCount := len(rooms)
	totalMessagesCount := int64(0)

	for _, room := range rooms {
		count, _ := room.CountRoomMessages(db, room.ID)
		totalMessagesCount += int64(count)
	}

	additionalInfo := Team{
		RoomsCount:         int64(totalRoomCount),
		TotalMessagesCount: totalMessagesCount,
	}

	
	return rooms, additionalInfo ,nil
}

func (t *Team) DeleteTeam(db *gorm.DB, userID string) error {

	exists := postgresql.CheckExists(db, &t, "id = ?", t.ID)
	if !exists {
		return errors.New("team does not exist")
	}

	if t.OwnerId != userID {
		return errors.New("user not authorized")
	}

	err := postgresql.DeleteRecordFromDb(db, &t)
	if err != nil {
		return errors.New("error deleting team")
	}
	return nil
}
