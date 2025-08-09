package agents

import (
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
		"github.com/hngprojects/telex_be/internal/models"

)

func CreateAgentSkill(orgId string, req models.CreateAgentSkillRequest, db *storage.Database, extReq request.ExternalRequest, userId string) (models.AgentSkillResponse, error) {
	return models.AgentSkillResponse{}, nil
}
