package models

import (
	"fmt"
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

type AgentWorkflowSkills struct {
	ID         string    `json:"id" gorm:"type:uuid;primaryKey"`
	AgentID    string    `json:"agent_id" gorm:"type:uuid;index"`
	SkillID    string    `json:"skill_id" gorm:"type:uuid;index"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
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

func (w *AgentWorkflowSkills) GetAgentWorkflowSkills(db *gorm.DB, agentID string) ([]GeneralAgentSkill, error) {
    var skills []GeneralAgentSkill
    
    err := db.Table("general_agent_skills").
        Select("general_agent_skills.*").
        Joins("JOIN agent_workflow_skills ON agent_workflow_skills.skill_id = general_agent_skills.id").
        Where("agent_workflow_skills.agent_id = ?", agentID).
        Find(&skills).Error
    
    if err != nil {
        return nil, fmt.Errorf("error fetching agent skills: %w", err)
    }
    
    return skills, nil
}