package agents

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func TriggerTick(db *storage.Database, logger *utility.Logger, req models.TriggerTickRequest) (string, int, error) {
	var (
		routing_key = "tick_test"
		// ch          models.Channels
		org models.Organisation
	)

	_, err := org.CheckOrgExists(req.OrganisationID, db.Postgresql)
	if err != nil {
		return "", http.StatusNotFound, err
	}

	// exists := postgresql.CheckExists(db.Postgresql, &ch, "id = ? AND organisation_id = ?", req.ChannelID, req.OrganisationID)
	// if !exists {
	// 	return "", http.StatusNotFound, errors.New("channel doesnt belong in organisation")
	// }

	payload := map[string]any{
		"message_content": map[string]any{
			"channel_id": req.ChannelID,
			"message":    "",
			// "thread_id":  feed.ThreadId,
			// "type":       feed.Type,
			// "user_id":    feed.UserId,
			"org_id": req.OrganisationID,
		},
		"channel_id": req.ChannelID,
		// "return_url": feed.ReturnUrl,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshaling payload for tick test: %v", err.Error()))
		return "", http.StatusBadRequest, fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	task := "telex_queue_processor.handle_new_message"

	err = rabbitmq.PushToRabbitQueue(logger, db.Postgresql, string(payloadBytes), routing_key, task)
	if err != nil {
		logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for ticktest: %v", err.Error()))
		return "", http.StatusBadRequest, fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	return "", http.StatusOK, nil
}

func GetActiveOutputIntegrations(db *gorm.DB, orgID string) ([]models.OutputIntegrationsResponse, error) {
	var (
		outputIntegrations []models.OutputIntegrationsResponse
		organisation       models.Organisation
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", orgID)
	if !exists {
		return outputIntegrations, fmt.Errorf("organisation does not exist")
	}

	baseURL := "https://system-integration.telex.im/"
	err := db.Table("integrations").
		Select(fmt.Sprintf("integrations.id, integrations.name, CONCAT('%s', Lower(integrations.name), '/channels') AS channels_url", baseURL)).
		Joins("LEFT JOIN organisation_integrations ON organisation_integrations.integration_id = integrations.id").
		Where("organisation_integrations.org_id = ? AND organisation_integrations.is_active = TRUE AND integrations.integration_type = 'o'", orgID).
		Scan(&outputIntegrations).Error

	if err != nil {
		return nil, err
	}

	return outputIntegrations, nil
}
