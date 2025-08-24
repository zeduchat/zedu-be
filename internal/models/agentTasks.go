package models

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	AgentID   string    `json:"agent_id" gorm:"type:uuid;index"`
	Text      string    `json:"text"`
	Position  int       `json:"position"` //order
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type TaskSkill struct {
	ID      string `json:"id" gorm:"type:uuid;primaryKey"`
	TaskID  string `json:"task_id"`
	SkillID string `json:"skill_id"`
}

type UpdateAgentTasksRequest struct {
	AgentID string `json:"-"`
	Tasks   []struct {
		ID       *string `json:"id,omitempty"`
		Text     string  `json:"text"`
		Position int     `json:"position"`
	} `json:"tasks"`
}

func (t *Task) GetAgentTasks(db *gorm.DB, agentID string) ([]Task, error) {
	var tasks []Task
	err := db.Where("agent_id = ?", agentID).Order("position").Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}
