package models

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"google.golang.org/appengine/log"
	"gorm.io/gorm"
)

type Group struct {
	ID             string     `gorm:"type:uuid;primaryKey;" json:"id"`
	Name           string     `gorm:"column:name;type:varchar(255);not null;unique" json:"name"`
	Description    string     `gorm:"column:description;type:text" json:"description"`
	OrganisationID string     `gorm:"column:organisation_id;type:uuid;not null" json:"organisation_id"`
	Channels       []Channels `gorm:"foreignKey:GroupID;constraint:OnUpdate:CASCADE,onDelete:SET NULL" json:"channels"`
	Archived       bool       `gorm:"column:archived;default:false" json:"archived"`
	CreatedAt      time.Time  `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column: updated_at; not null; autoUpdateTime" json:"updated_at"`
	DeletedAt      time.Time  `gorm:"column: deleted_at; not null; autoDeleteTime" json:"deleted_at"`
}

type CreateGroupRequest struct {
	Name           string `json:"name" binding:"required"`
	OrganisationID string `json:"organisation_id"`
	Description    string `json:"description"`
}

type UpdateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Archived    bool   `json:"archived"`
}

type ToggleArchiveStatusRequest struct {
	Archived bool `json:"archived"`
}

type AssignGroupChannelRequest struct {
	Channels []string `json:"channels" binding:"required"`
}

func (g *Group) CreateGroup(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &g)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	return nil
}

func (g *Group) GetGroup(db *gorm.DB, ids map[string]string) (Group, error) {
	_, err := postgresql.SelectOneFromDb(db, &g, "organisation_id = ? AND id = ?", ids["organisation_id"], ids["group_id"])
	if err != nil {
		return Group{}, fmt.Errorf("failed to get group: %w", err)
	}
	return *g, nil
}

func (g *Group) GetGroups(db *gorm.DB, org_id string) ([]Group, error) {
	var (
		groups []Group
	)

	err := postgresql.SelectAllFromDb(db.Preload("Channels"), "", &groups, "organisation_id = ?", org_id)
	if err != nil {
		return groups, fmt.Errorf("failed to get groups: %w", err)
	}
	return groups, nil
}

func (g *Group) UpdateGroup(db *gorm.DB, req UpdateGroupRequest, ids map[string]string) error {
	g.ID = ids["group_id"]

	result, err := postgresql.UpdateFields(db, &g, req, "organisation_id = ? AND id = ?", ids["organisation_id"], ids["group_id"])
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("group not found")
	}

	_, err = postgresql.SelectOneFromDb(db, &g, "organisation_id = ? AND id = ?", ids["organisation_id"], ids["group_id"])
	if err != nil {
		return fmt.Errorf("failed to get updated group: %w", err)
	}

	return nil
}

func (g *Group) DeleteGroup(db *gorm.DB, ids map[string]string) error {

	exists := postgresql.CheckExists(db, &g, "organisation_id = ? AND id = ?", ids["organisation_id"], ids["group_id"])
	if !exists {
		return fmt.Errorf("group not found")
	}

	err := postgresql.DeleteRecordFromDb(db, &g)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	return nil
}

// ====================================CHANNELS====================================
func (group *Group) AssignGroupChannel(db *gorm.DB, ids map[string]string) error {
	var (
		ch  Channels
		org Organisation
	)

	_, err := org.CheckOrgExists(ids["organisation_id"], db)
	if err != nil {
		return err
	}

	exists := postgresql.CheckExists(db, &group, "organisation_id = ? AND id = ?", ids["organisation_id"], ids["group_id"])
	if !exists {
		return fmt.Errorf("group not found")
	}

	exists = postgresql.CheckExists(db, &ch, "id = ?", ids["channel_id"])
	if !exists {
		return fmt.Errorf("channel not found")
	}

	exists = postgresql.CheckExists(db, &ch, "organisation_id = ? AND id = ? ", ids["organisation_id"], ids["channel_id"])
	if !exists {
		return fmt.Errorf("channel %s is not part of the %s organisation", ch.Name, org.Name)
	}

	// check if channel is already assigned to a group
	exists = postgresql.CheckExists(db, &ch, "group_id = ?", ids["group_id"])
	if exists {
		return fmt.Errorf("channel %s is already assigned to %s group", ch.Name, group.Name)
	}

	update := map[string]any{
		"group_id": ids["group_id"],
	}

	result, err := postgresql.UpdateFields(db, &ch, update, "id = ?", ids["channel_id"])
	if err != nil {
		return fmt.Errorf("failed to assign group channel: %w", err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("channel not found")
	}

	return nil
}

func (group *Group) RemoveGroupChannel(db *gorm.DB, ids map[string]string) error {
	var (
		ch  Channels
		org Organisation
	)

	_, err := org.CheckOrgExists(ids["organisation_id"], db)
	if err != nil {
		return err
	}

	exists := postgresql.CheckExists(db, &group, "organisation_id = ? AND id = ?", ids["organisation_id"], ids["group_id"])
	if !exists {
		return fmt.Errorf("group not found")
	}

	exists = postgresql.CheckExists(db, &ch, "id = ?", ids["channel_id"])
	if !exists {
		return fmt.Errorf("channel not found")
	}

	exists = postgresql.CheckExists(db, &ch, "organisation_id = ? AND id = ? ", ids["organisation_id"], ids["channel_id"])
	if !exists {
		return fmt.Errorf("channel %s is not part of the %s organisation", ch.Name, org.Name)
	}

	// check if channel is already assigned to a group
	exists = postgresql.CheckExists(db, &ch, "group_id = ?", ids["group_id"])
	if !exists {
		return fmt.Errorf("channel %s is not assigned to %s group", ch.Name, group.Name)
	}

	update := map[string]any{
		"group_id": nil,
	}

	result, err := postgresql.UpdateFields(db, &ch, update, "id = ?", ids["channel_id"])
	if err != nil {
		return fmt.Errorf("failed to unassign group channel: %w", err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("channel not found")
	}

	return nil
}

func (group *Group) GetGroupChannels(db *storage.Database, ids map[string]string) ([]Channels, error) {
	var (
		channels []Channels
		org      Organisation
		user     User
	)

	_, err := org.CheckOrgExists(ids["organisation_id"], db.Postgresql)
	if err != nil {
		return channels, err
	}

	exists := postgresql.CheckExists(db.Postgresql, &user, "id = ?", ids["user_id"])
	if !exists {
		return channels, fmt.Errorf("user not found")
	}


	exists = postgresql.CheckExists(db.Postgresql, &group, "organisation_id = ? AND id = ?", ids["organisation_id"], ids["group_id"])
	if !exists {
		return channels, fmt.Errorf("group not found")
	}

	err = db.Postgresql.Model(&Channels{}).
		Select("channels.*").
		Joins("join user_channels on channels.id = user_channels.channels_id").
		Where("channels.organisation_id = ? AND channels.group_id = ? AND channels.archived = false AND user_channels.user_id = ?", ids["organisation_id"], ids["group_id"], ids["user_id"]).
		Order("channels.created_at").
		Scan(&channels).Error
	if err != nil {
		return channels, fmt.Errorf("failed to get group channels: %w", err)
	}

	//get the thread count for each channel
	getThreadCountFromElastic := func(es *elasticsearch.Client, channelID string) int {
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []map[string]interface{}{
						{
							"term": map[string]interface{}{
								"channels_id.keyword": channelID,
							},
						},
						{
							"term": map[string]interface{}{
								"type.keyword": "thread",
							},
						},
					},
				},
			},
			"size": 0,
			"aggs": map[string]interface{}{
				"thread_count": map[string]interface{}{
					"value_count": map[string]interface{}{
						"field": "thread_id.keyword",
					},
				},
			},
		}

		var countInfo any
		err := elastic.SelectAll(es, "threads", query, &countInfo)
		if err != nil {
			log.Errorf(context.Background(), "error fetching thread count from elastic: %v", err)
			return 0
		}

		count := countInfo.(map[string]interface{})["aggregations"].(map[string]interface{})["thread_count"].(map[string]interface{})["value"].(float64)

		return int(count)
	}

	for i, channel := range channels {
		count := getThreadCountFromElastic(db.Elastic, channel.ID)
		fmt.Println("count for grouped channels", count)
		channels[i].MessageCount = int64(count)
	}

	return channels, nil
}

func (group *Group) GetChannelsNotInGroup(db *storage.Database, ids map[string]string) (GetUserChannelResp, error) {
	var (
		org     Organisation
		c       = context.Background()
		org_id  = ids["organisation_id"]
		user_id = ids["user_id"]
	)
	chanResp := GetUserChannelResp{}
	channels := []Channels{}

	_, err := org.CheckOrgExists(org_id, db.Postgresql)
	if err != nil {
		return chanResp, err
	}

	err = db.Postgresql.Model(&Channels{}).
		Select("channels.id, channels.name, channels.description, channels.organisation_id, channels.owner_id, channels.archived, channels.group_id, channels.created_at").
		Joins("join user_channels on channels.id = user_channels.channels_id").
		Where("channels.organisation_id = ? AND channels.group_id IS NULL AND channels.archived = false AND user_channels.user_id = ?", org_id, user_id).
		Order("channels.created_at").
		Scan(&channels).Error
	if err != nil {
		return chanResp, fmt.Errorf("failed to get channels not in group: %w", err)
	}

	getThreadCountFromElastic := func(es *elasticsearch.Client, channelID string) int {
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []map[string]interface{}{
						{
							"term": map[string]interface{}{
								"channels_id.keyword": channelID,
							},
						},
						{
							"term": map[string]interface{}{
								"type.keyword": "thread",
							},
						},
					},
				},
			},
			"size": 0,
			"aggs": map[string]interface{}{
				"thread_count": map[string]interface{}{
					"value_count": map[string]interface{}{
						"field": "thread_id.keyword",
					},
				},
			},
		}

		var countInfo any
		err := elastic.SelectAll(es, "threads", query, &countInfo)
		if err != nil {
			log.Errorf(c, "error fetching thread count from elastic: %v", err)
			return 0
		}

		count := countInfo.(map[string]interface{})["aggregations"].(map[string]interface{})["thread_count"].(map[string]interface{})["value"].(float64)

		return int(count)

	}

	for _, channel := range channels {
		count := getThreadCountFromElastic(db.Elastic, channel.ID)
		fmt.Println("count for channels not in group", count)
		channel.MessageCount = int64(count)
		chanResp = append(chanResp, struct {
			Channels
			WebhookUrl  string `json:"webhook_url"`
			ThreadCount int64  `json:"thread_count"`
			Access      bool   `json:"access"`
		}{
			Channels:    channel,
			WebhookUrl:  "",
			ThreadCount: int64(count),
			Access:      true,
		})
	}

	return chanResp, nil
}

func (group *Group) MoveGroupChannel(db *gorm.DB, ids map[string]string) error {
	var (
		ch  Channels
		org Organisation
	)
	_, err := org.CheckOrgExists(ids["organisation_id"], db)
	if err != nil {
		return err
	}

	exists := postgresql.CheckExists(db, &group, "organisation_id = ? AND id = ?", ids["organisation_id"], ids["new_group_id"])
	if !exists {
		return fmt.Errorf("group not found")
	}

	exists = postgresql.CheckExists(db, &ch, "id = ?", ids["channel_id"])
	if !exists {
		return fmt.Errorf("channel not found")
	}

	exists = postgresql.CheckExists(db, &ch, "organisation_id = ? AND id = ? ", ids["organisation_id"], ids["channel_id"])
	if !exists {
		return fmt.Errorf("channel %s is not part of the %s organisation", ch.Name, org.Name)
	}

	exists = postgresql.CheckExists(db, &ch, "group_id = ?", ids["new_group_id"])
	if exists {
		return fmt.Errorf("channel %s is already assigned to %s group", ch.Name, group.Name)
	}

	update := map[string]any{
		"group_id": ids["new_group_id"],
	}

	result, err := postgresql.UpdateFields(db, &ch, update, "id = ?", ids["channel_id"])
	if err != nil {
		return fmt.Errorf("failed to move group channel: %w", err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("channel not found")
	}

	return nil
}

func (group *Group) GetDiscoverableChannels(db *storage.Database, ids map[string]string) ([]Channels, error) {
	var (
		channels = []Channels{}
		org      Organisation
	)

	_, err := org.CheckOrgExists(ids["organisation_id"], db.Postgresql)
	if err != nil {
		return channels, err
	}

	if err := db.Postgresql.Model(&Channels{}).
		Select("channels.id, channels.name, channels.description, channels.organisation_id, channels.owner_id, channels.archived, channels.group_id, channels.created_at").
		Where("channels.organisation_id = ? AND channels.archived = false AND channels.id NOT IN (SELECT user_channels.channels_id FROM user_channels WHERE user_channels.user_id = ?)", ids["organisation_id"], ids["user_id"]).
		Order("channels.created_at").
		Scan(&channels).Error; err != nil {
		return channels, fmt.Errorf("error fetching discoverable channels: %w", err)
	}

	return channels, nil
}
