package models

import (
	"time"
)

type Task struct {
	ID         string    `json:"id" gorm:"type:uuid;primaryKey"`
	WorkflowID string    `json:"workflow_id" gorm:"type:uuid;index"`
	Text       string    `json:"text"`
	Position   int       `json:"position"` //order
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type TaskSkill struct {
	ID         string `gorm:"primaryKey"`
	TaskID     string
	SkillID    string
	Confidence float64
}

type UpdateWorkflowTasksRequest struct {
	WorkflowID string `json:"-"`
	Tasks      []struct {
		ID       *string `json:"id,omitempty"`
		Text     string  `json:"text"`
		Position int     `json:"position"`
	} `json:"tasks"`
}
