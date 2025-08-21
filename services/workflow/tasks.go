package workflow

import (
	"errors"
	"net/http"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func UpdateWorkflowTasks(db *gorm.DB, req models.UpdateWorkflowTasksRequest) (int, []models.Task, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return http.StatusInternalServerError, nil, tx.Error
	}

	// fetch tasks belonging to this workflow
	var existingTasks []models.Task
	err := postgresql.SelectAllFromDb(tx, "", &existingTasks, "workflow_id = ?", req.WorkflowID)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, nil, err
	}

	existingMap := make(map[string]models.Task)
	for _, t := range existingTasks {
		existingMap[t.ID] = t
	}

	incomingIDs := make(map[string]bool)
	var updatedTasks []models.Task

	for _, taskReq := range req.Tasks {
		if taskReq.ID != nil { // try update
			existingTask, ok := existingMap[*taskReq.ID]
			if !ok {
				tx.Rollback()
				return http.StatusNotFound, nil, errors.New("task not found: " + *taskReq.ID)
			}

			// update only if something changed
			changed := false
			if existingTask.Text != taskReq.Text {
				existingTask.Text = taskReq.Text
				changed = true
			}
			if existingTask.Position != taskReq.Position {
				existingTask.Position = taskReq.Position
				changed = true
			}

			if changed {
				if err := tx.Save(&existingTask).Error; err != nil {
					tx.Rollback()
					return http.StatusInternalServerError, nil, err
				}
			}

			incomingIDs[existingTask.ID] = true
			updatedTasks = append(updatedTasks, existingTask)

		} else { // create new task
			newTask := models.Task{
				ID:         utility.GenerateUUID(),
				WorkflowID: req.WorkflowID,
				Text:       taskReq.Text,
				Position:   taskReq.Position,
			}
			if err := tx.Create(&newTask).Error; err != nil {
				tx.Rollback()
				return http.StatusInternalServerError, nil, err
			}
			incomingIDs[newTask.ID] = true
			updatedTasks = append(updatedTasks, newTask)
		}
	}

	// delete tasks not in incoming list
	for id := range existingMap {
		if !incomingIDs[id] {
			err := postgresql.HardDeleteSpecificRecord(tx, &models.Task{}, "id = ?", id)
			if err != nil {
				tx.Rollback()
				return http.StatusInternalServerError, nil, err
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, updatedTasks, nil
}
