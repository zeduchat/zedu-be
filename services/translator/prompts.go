package translator

import (
	"fmt"
	"net/http"

	"github.com/hngprojects/telex_be/internal/models"
	"gorm.io/gorm"
)

func CreatePrompt(db *gorm.DB, req models.Prompts) (int, error) {
	var p models.Prompts

	p.Name = req.Name
	p.Template = req.Template

	err := p.CreatePrompt(db)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("unable to create prompt: %v", err)
	}

	return http.StatusOK, nil
}

func GetPrompts(db *gorm.DB) ([]models.Prompts, int, error) {
	var p models.Prompts

	resp, code, err := p.GetAllPrompts(db)
	if err != nil {
		return resp, code, err
	}

	return resp, code, nil
}

func GetPrompt(db *gorm.DB, prompt_name string) (models.GetPromptResponse, int, error) {

	gpr, code, err := (&models.Prompts{}).GetPrompt(db, prompt_name)
	if err != nil {
		return gpr, code, err
	}

	return gpr, code, nil
}

func FetchUniqueSteps(db *gorm.DB) ([]models.Prompts, int, error) {
	var prompt models.Prompts 

	prompts, err := prompt.FetchUniquePrompts(db)
	if err != nil {
		return []models.Prompts{}, http.StatusInternalServerError, err
	}

	return prompts, http.StatusOK, nil
}
