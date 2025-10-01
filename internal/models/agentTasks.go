package models

import (
	"errors"
	"net/http"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Tasks []Task

type Task struct {
	ID             string    `json:"id" gorm:"type:uuid;primaryKey"`
	AgentID        string    `json:"agent_id" gorm:"type:uuid;index"`
	OrganisationID string    `json:"org_id" gorm:"type:uuid;index"`
	Text           string    `gorm:"type:text" json:"text"`
	Position       int       `gorm:"type:int" json:"position"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type TaskSkill struct {
	ID      string `json:"id" gorm:"type:uuid;primaryKey"`
	TaskID  string `json:"task_id"`
	SkillID string `json:"skill_id"`
}

type UpdateAgentTasksRequest struct {
	Text     string `json:"text"`
	Position int    `json:"position"`
}

type CreateAgentTasksRequest struct {
	AgentID        string `json:"agentId"`
	OrganisationID string `json:"org_id"`
	Text           string `json:"text"`
	Position       int    `json:"position"`
}

func (t *Task) GetAgentTasks(db *gorm.DB, agentID, orgID string) ([]Task, error) {
	var tasks []Task
	err := db.Where("agent_id = ? AND organisation_id = ?", agentID, orgID).Order("position").Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (t *Task) CreateTasks(db *gorm.DB) (int, error) {
	err := postgresql.CreateOneRecord(db, &t)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusCreated, nil
}

func (t *Task) UpdateAgentTasks(db *gorm.DB, req UpdateAgentTasksRequest, ids IDS) (int, error) {
	var (
		task Task
	)

	exists := postgresql.CheckExists(db, &task, "id = ? AND agent_id = ?", ids.TaskID, ids.AgentID)
	if !exists {
		return http.StatusNotFound, errors.New("task not found for the agent")
	}

	res, err := postgresql.UpdateFields(db, &task, req, "id = ?", ids.TaskID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if res.RowsAffected == 0 {
		return http.StatusNotFound, gorm.ErrRecordNotFound
	}

	return http.StatusOK, nil
}

func (t *Task) DeleteAgentTasks(db *gorm.DB, ids IDS) (int, error) {
	var (
		task Task
	)

	exists := postgresql.CheckExists(db, &task, "id = ? AND agent_id = ?", ids.TaskID, ids.AgentID)
	if !exists {
		return http.StatusNotFound, errors.New("task not found for the agent")
	}

	err := postgresql.HardDeleteSpecificRecord(db, &task, "id = ? AND agent_id = ?", ids.TaskID, ids.AgentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, err
		}
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
