package models

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type Reaction struct {
	ID         string    `gorm:"type:uuid;primary_key" json:"id"`
	ThreadID   string    `gorm:"type:uuid" json:"thread_id"`
	MessageID  string    `gorm:"type:uuid" json:"message_id,omitempty"`
	UserID     string    `gorm:"type:uuid;index" json:"user_id"`
	ChannelsID string    `gorm:"type:uuid;index" json:"channels_id"`
	Reaction   string    `gorm:"type:text" json:"reaction"`
	Type       string    `gorm:"type:varchar(50);default:thread" json:"type"`
	ReactionID string    `gorm:"type:uuid" json:"reaction_id"`
	CreatedAt  time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type ReactionRequest struct {
	ID         string `json:"id,omitempty"`
	Reaction   string `json:"reaction" validate:"required"`
	ChannelsID string `json:"channels_id"`
	UserID     string `json:"user_id"`
	ThreadID   string `json:"thread_id" validate:"required"`
	MessageID  string `json:"message_id"`
	ReactionID string `json:"reaction_id"`
	Type       string `json:"type" validate:"required,oneof=thread reply"`
	Expression int    `json:"int"` // +1 for increment, -1 for decrement
}

type ReactionDetails struct {
	Reaction      string `json:"reaction"`
	ReactionId    string `json:"reaction_id"`
	ReactionCount string `json:"reaction_count"`
}

var ReactionMapping = map[string]any{
	"properties": map[string]any{
		"reaction":       map[string]string{"type": "text"},
		"reaction_id":    map[string]string{"type": "text"},
		"reaction_count": map[string]string{"type": "integer"},
	},
}

func (m *Reaction) CreateThreadReaction(db *gorm.DB) (int, bool, error) {
	var (
		checkThread   ThreadDocument
		reactionEntry Reaction
		new           bool
	)

	checkThread.ID = m.ThreadID
	checkThread.ChannelsID = m.ChannelsID

	exist, _, _ := checkThread.CheckExists()

	if !exist {
		return http.StatusBadRequest, new, errors.New("thread does not exist")
	}

	exists := postgresql.CheckExists(db, &reactionEntry, "reaction_id = ? AND thread_id = ? AND user_id = ?", m.ReactionID, m.ThreadID, m.UserID)
	if !exists {
		if err := postgresql.CreateOneRecord(db, &m); err != nil {
			new = true
			return http.StatusInternalServerError, new, err
		}
	}

	return http.StatusCreated, new, nil
}

func (m *Reaction) CreateReplyReaction(db *gorm.DB) (int, bool, error) {
	var (
		checkMessage  MessageDocument
		reactionEntry Reaction
		new           bool
	)

	checkMessage.ID = m.MessageID
	checkMessage.ChannelsID = m.ChannelsID

	exist, _, _ := checkMessage.CheckExists()

	if !exist {
		return http.StatusBadRequest, new, errors.New("reply message does not exist")
	}

	exists := postgresql.CheckExists(db, &reactionEntry, "reaction_id = ? AND message_id = ? AND user_id = ?", m.ReactionID, m.MessageID, m.UserID)
	if !exists {
		if err := postgresql.CreateOneRecord(db, &m); err != nil {
			new = true
			return http.StatusInternalServerError, new, err
		}
	}

	return http.StatusCreated, new, nil
}

func (m *Reaction) DeleteThreadReaction(db *gorm.DB) error {
	query := db.Where("reaction_id = ? AND thread_id = ? AND user_id = ?", m.ReactionID, m.ThreadID, m.UserID)

	if err := query.First(&m).Error; err != nil {
		return err
	}

	if err := query.Delete(&Reaction{}).Error; err != nil {
		return err
	}

	return nil
}

func (m *Reaction) DeleteReplyReaction(db *gorm.DB) error {
	query := db.Where("reaction_id = ? AND message_id = ? AND user_id = ?", m.ReactionID, m.MessageID, m.UserID)

	if err := query.First(&m).Error; err != nil {
		return err
	}

	if err := query.Delete(&Reaction{}).Error; err != nil {
		return err
	}

	return nil
}

func (m *Reaction) GetReactionUsernameByID(db *gorm.DB) (int, []string, error) {
	var (
		users []string
	)

	var (
		query   string
		queryId string
	)

	if m.Type == "reply" {
		query = "reactions.type = ? AND reactions.message_id = ? AND reactions.reaction_id = ?"
		queryId = m.MessageID

		var checkMessage MessageDocument
		checkMessage.ID = m.MessageID

		exist, _, err := checkMessage.CheckExists()
		if err != nil {
			return http.StatusInternalServerError, users, err
		}
		if !exist {
			return http.StatusBadRequest, users, errors.New("thread does not exist")
		}

	} else {
		query = "reactions.type = ? AND reactions.thread_id = ? AND reactions.reaction_id = ?"
		queryId = m.ThreadID

		var checkThread ThreadDocument
		checkThread.ID = m.ThreadID
		exist, _, err := checkThread.CheckExists()
		if err != nil {
			return http.StatusInternalServerError, users, err
		}
		if !exist {
			return http.StatusBadRequest, users, errors.New("reply message does not exist")
		}
	}

	if err := db.Table("reactions").
		Joins("JOIN profiles ON profiles.userid = reactions.user_id").
		Where(query, m.Type, queryId, m.ReactionID).
		Pluck("profiles.username", &users).Error; err != nil {
		return http.StatusInternalServerError, users, err
	}

	return http.StatusOK, users, nil
}
