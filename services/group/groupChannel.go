package group

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

func AssignGroupChannel(db *storage.Database, ids map[string]string) error {
	var group models.Group

	err := group.AssignGroupChannel(db.Postgresql, ids)
	if err != nil {
		return err
	}
	return nil
}

func RemoveGroupChannel(db *storage.Database, ids map[string]string) error {
	var group models.Group

	err := group.RemoveGroupChannel(db.Postgresql, ids)
	if err != nil {
		return err
	}
	return nil
}

func GetGroupChannels(db *storage.Database, ids map[string]string) ([]models.Channels, error) {
	var group models.Group

	response, err := group.GetGroupChannels(db, ids)
	if err != nil {
		return []models.Channels{}, err
	}
	return response, nil
}

func MoveGroupChannel(db *storage.Database, ids map[string]string) error {
	var group models.Group

	err := group.MoveGroupChannel(db.Postgresql, ids)
	if err != nil {
		return err
	}
	return nil
}

func GetChannelsNotInGroup(db *storage.Database, ids map[string]string) (models.GetUserChannelResp, error) {
	var group models.Group

	response, err := group.GetChannelsNotInGroup(db, ids)
	if err != nil {
		return models.GetUserChannelResp{}, err
	}
	return response, nil
}

func GetDiscoverableChannels(db *storage.Database, ids map[string]string) ([]models.Channels, error) {
	var group models.Group

	response, err := group.GetDiscoverableChannels(db, ids)
	if err != nil {
		return []models.Channels{}, err
	}
	return response, nil
}