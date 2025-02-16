package integrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func GetAllIntegrationApp(c *gin.Context, org_id string, db *gorm.DB) (models.IntegrationResp, error) {
	var integrations models.Integrations

	resp, err := integrations.GetAllIntegrationApp(db, org_id, c)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func GetCustomIntegrationApp(c *gin.Context, org_id string, db *gorm.DB, extReq request.ExternalRequest) (models.IntegrationResp, postgresql.PaginationResponse, error, int) {
	var org_integrations models.OrganisationIntegrations

	var int_resp = models.IntegrationResp{}

	resp, paginationResult, err, code := org_integrations.GetCustomIntegrationApp(db, org_id, c)

	if err != nil {
		return nil, postgresql.PaginationResponse{}, err, code
	}

	for _, org_integrations := range resp {

		json_url := org_integrations.JSONUrl
		data := map[string]string{"url": json_url}

		response, err := extReq.SendExternalRequest(request.IntegrationJsonContent, data)

		if err != nil {
			integration := models.Integrations{
				ID:             org_integrations.IntegrationID,
				Name:           "Unavailable",
				JSONUrl:        org_integrations.JSONUrl,
				AppDescription: "This integration is currently unavailable.",
				Category:       "Unavailable",
				IsActive:       false,
				Status:         "failed",
				CreatedAt:      org_integrations.CreatedAt,
				UpdatedAt:      org_integrations.UpdatedAt,
			}

			int_resp = append(int_resp, struct {
				models.Integrations
				Linked bool "json:\"linked\""
			}{
				Integrations: integration,
				Linked:       true,
			})
			continue
		}

		response_data := response.(map[string]interface{})

		data_r := response_data["data"].(map[string]interface{})

		description := data_r["descriptions"].(map[string]interface{})
		category, ok := data_r["integration_category"].(string)

		if !ok || category == "" {

			category = "Undefined"
		}

		integration := models.Integrations{
			ID:             org_integrations.IntegrationID,
			Name:           description["app_name"].(string),
			JSONUrl:        org_integrations.JSONUrl,
			AppUrl:         description["app_url"].(string),
			AppLogo:        description["app_logo"].(string),
			AppDescription: description["app_description"].(string),
			Category:       category,
			Status:         "success",
			IsActive:       org_integrations.IsActive,
			CreatedAt:      org_integrations.CreatedAt,
			UpdatedAt:      org_integrations.UpdatedAt,
		}

		int_resp = append(int_resp, struct {
			models.Integrations
			Linked bool "json:\"linked\""
		}{
			Integrations: integration,
			Linked:       true,
		})
	}

	return int_resp, paginationResult, nil, code
}

func GetSystemIntegrationApps(c *gin.Context, db *gorm.DB, extReq request.ExternalRequest) (models.IntegrationResp, postgresql.PaginationResponse, error, int) {
	var integrations models.Integrations

	var int_resp = models.IntegrationResp{}

	resp, paginationResult, err, code := integrations.GetSystemIntegrationApps(db, c)

	if err != nil {
		return nil, postgresql.PaginationResponse{}, err, code
	}

	for _, org_integrations := range resp {

		json_url := org_integrations.JSONUrl
		data := map[string]string{"url": json_url}

		response, err := extReq.SendExternalRequest(request.IntegrationJsonContent, data)

		if err != nil {
			integration := models.Integrations{
				ID:             org_integrations.ID,
				Name:           "Unavailable",
				JSONUrl:        org_integrations.JSONUrl,
				AppDescription: "This integration is currently unavailable.",
				Category:       "Unavailable",
				IsActive:       false,
				Status:         "failed",
				CreatedAt:      org_integrations.CreatedAt,
				UpdatedAt:      org_integrations.UpdatedAt,
			}

			int_resp = append(int_resp, struct {
				models.Integrations
				Linked bool "json:\"linked\""
			}{
				Integrations: integration,
				Linked:       true,
			})
			continue
		}

		response_data := response.(map[string]interface{})

		data_r := response_data["data"].(map[string]interface{})

		description := data_r["descriptions"].(map[string]interface{})

		category, ok := data_r["integration_category"].(string)

		if !ok || category == "" {

			category = "Undefined"
		}

		integration := models.Integrations{
			ID:             org_integrations.ID,
			Name:           description["app_name"].(string),
			JSONUrl:        org_integrations.JSONUrl,
			AppUrl:         description["app_url"].(string),
			AppLogo:        description["app_logo"].(string),
			AppDescription: description["app_description"].(string),
			Category:       category,
			Status:         "success",
			IsActive:       org_integrations.IsActive,
			CreatedAt:      org_integrations.CreatedAt,
			UpdatedAt:      org_integrations.UpdatedAt,
		}

		int_resp = append(int_resp, struct {
			models.Integrations
			Linked bool "json:\"linked\""
		}{
			Integrations: integration,
			Linked:       true,
		})
	}

	return int_resp, paginationResult, nil, code
}

func GetSystemIntegrationApp(c *gin.Context, db *gorm.DB, int_id string, extReq request.ExternalRequest) (models.Integrations, error, int) {
	var integrations models.Integrations

	resp, err, code := integrations.GetSystemIntegrationApp(db, int_id, c)

	if err != nil {
		return models.Integrations{}, err, code
	}

	json_url := resp.JSONUrl
	data := map[string]string{"url": json_url}

	response, err := extReq.SendExternalRequest(request.IntegrationJsonContent, data)

	if err != nil {
		extReq.Logger.Error("An error occurred while fetching integration json, err: %s ", err)
		integration := models.Integrations{
			ID:             resp.ID,
			Name:           "Unavailable",
			JSONUrl:        resp.JSONUrl,
			AppDescription: "This integration is currently unavailable.",
			Category:       "Unavailable",
			IsActive:       false,
			Status:         "failed",
			CreatedAt:      resp.CreatedAt,
			UpdatedAt:      resp.UpdatedAt,
		}

		return integration, nil, code
	}

	response_data := response.(map[string]interface{})

	data_r := response_data["data"].(map[string]interface{})

	description := data_r["descriptions"].(map[string]interface{})

	integration := models.Integrations{
		ID:             resp.ID,
		Name:           description["app_name"].(string),
		JSONUrl:        resp.JSONUrl,
		Status:         "success",
		AppUrl:         description["app_url"].(string),
		AppLogo:        description["app_logo"].(string),
		AppDescription: description["app_description"].(string),
		IsActive:       resp.IsActive,
		CreatedAt:      resp.CreatedAt,
		UpdatedAt:      resp.UpdatedAt,
	}

	return integration, nil, code
}

func UpdateIntegrationApp(req models.UpdateIntegration, ids map[string]string, db *gorm.DB) (models.Integrations, error) {
	var integration models.Integrations

	updatedIntegration, err := integration.UpdateIntegration(db, ids, req)
	if err != nil {
		return models.Integrations{}, err
	}

	return updatedIntegration, nil
}

func DeleteIntegrationApp(ids map[string]string, db *gorm.DB) error {
	var integration models.Integrations

	err := integration.DeleteIntegration(db, ids)
	if err != nil {
		return err
	}

	return nil
}

// Delete Org Custom Integration
func DeleteCustomIntegrationApp(ids map[string]string, db *gorm.DB) (error, int) {
	var org_integration models.OrganisationIntegrations

	err, code := org_integration.DeleteCustomIntegration(db, ids)
	if err != nil {
		return err, code
	}

	return nil, code
}

func ChangeIntegrationStatus(ids map[string]string, req models.ChangeIntegrationStatus, db *gorm.DB, extReq request.ExternalRequest) error {
	var integration models.OrganisationIntegrations

	err := integration.ChangeStatus(db, req, ids, extReq)
	if err != nil {
		return err
	}

	return nil
}

func ChangeIntegrationSendBackStatus(ids map[string]string, req models.ChangeIntegrationStatus, db *gorm.DB) error {
	var integration models.OrganisationChannelsIntegrations

	err := integration.ChangeSendBackStatus(db, req, ids)
	if err != nil {
		return err
	}

	return nil
}

func UpdateJSONSchema(ids map[string]string, req models.UpdateJSONSchemaRequest, db *gorm.DB) error {
	var orgIntegration models.OrganisationIntegrations

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if !exists {
		return errors.New("organisation does not have that integration")
	}

	err := orgIntegration.UpdateJSONSchema(db, req, ids)
	if err != nil {
		return err
	}

	return nil
}

func CreateCustomIntegration(org_id string, req models.CustomIntegrationRequest, db *gorm.DB, extReq request.ExternalRequest) error {

	var (
		orgIntegration      models.OrganisationIntegrations
		integrationSettings models.CustomIntegrationsSetting
	)

	data := map[string]string{"url": req.JSONUrl}

	response, err := extReq.SendExternalRequest(request.IntegrationJsonContent, data)

	if err != nil {
		return errors.New("Failed to create custom integration, invalid JSON supplied")
	}

	response_data := response.(map[string]interface{})
	data_r, ok := response_data["data"].(map[string]interface{})

	if !ok {
		return errors.New("Failed to Create Custom Integration, data field does not exist")
	}

	// validate description entry

	err = models.ValidateIntegrationData(data_r)

	if err != nil {
		return err
	}

	settings, ok := data_r["settings"]
	if !ok {
		return errors.New("Failed to create custom integration, settings field does not exist")
	}

	settings_data := map[string]interface{}{"settings": settings}

	// serialize the settings json

	settingJsonData, err := json.Marshal(settings_data)
	if err != nil {
		return fmt.Errorf("error serializing to JSON: %v", err)
	}

	serialized_settings := string(settingJsonData)

	// create integration in db

	orgIntegration.OrgID = org_id
	orgIntegration.JSONUrl = req.JSONUrl
	orgIntegration.IntegrationID = utility.GenerateUUID()
	orgIntegration.IsActive = false
	orgIntegration.IsSystem = false
	orgIntegration.ID = utility.GenerateUUID()

	err = orgIntegration.CreateOrganisationIntegration(db)

	if err != nil {
		return err
	}

	integrationSettings.ID = utility.GenerateUUID()
	integrationSettings.SettingEntry = serialized_settings
	integrationSettings.OrgID = org_id
	integrationSettings.IsSystem = false
	integrationSettings.IntegrationID = orgIntegration.IntegrationID

	err = integrationSettings.CreateIntegrationSettings(db)

	if err != nil {
		return errors.New("Failed to create integration settings")
	}

	return nil
}

// Update CustomIntegration
func UpdateCustomIntegration(ids map[string]string, req models.CustomIntegrationRequest, db *gorm.DB, extReq request.ExternalRequest) error {

	var orgIntegration models.OrganisationIntegrations

	data := map[string]string{"url": req.JSONUrl}

	_, err := extReq.SendExternalRequest(request.IntegrationJsonContent, data)

	if err != nil {
		return errors.New("Failed to Update Custom Integration, invalid JSON supplied")
	}

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if !exists {
		return errors.New("organisation does not have that integration")
	}

	err = orgIntegration.UpdateCustomIntegration(db, req, ids)

	if err != nil {
		return err
	}

	return nil
}

func GetOrganisationChannelIntegrations(db *gorm.DB, channel_id, org_id string, c *gin.Context, extReq request.ExternalRequest) (models.IntegrationResp, postgresql.PaginationResponse, int, error) {
	var (
		ocIntegrations models.OrganisationChannelsIntegrations
	)

	var int_resp = models.IntegrationResp{}

	integrations, paginationResponse, code, err := ocIntegrations.GetOrganisationChannelIntegrations(db, channel_id, org_id, c)

	if err != nil {
		return nil, paginationResponse, code, err
	}

	for _, org_integrations := range integrations {

		json_url := org_integrations.JSONUrl
		data := map[string]string{"url": json_url}

		response, err := extReq.SendExternalRequest(request.IntegrationJsonContent, data)

		if err != nil {

			integration := models.Integrations{
				ID:             org_integrations.IntegrationID,
				Name:           "Unavailable",
				JSONUrl:        org_integrations.JSONUrl,
				AppDescription: "This integration is currently unavailable.",
				Category:       "Unavailable",
				IsActive:       false,
				Status:         "failed",
				CreatedAt:      org_integrations.CreatedAt,
				UpdatedAt:      org_integrations.UpdatedAt,
			}

			int_resp = append(int_resp, struct {
				models.Integrations
				Linked bool "json:\"linked\""
			}{
				Integrations: integration,
				Linked:       true,
			})

			continue
		}

		response_data := response.(map[string]interface{})

		data_r := response_data["data"].(map[string]interface{})

		description := data_r["descriptions"].(map[string]interface{})

		category, ok := data_r["integration_category"].(string)

		if !ok || category == "" {

			category = "Undefined"
		}

		integration := models.Integrations{
			ID:             org_integrations.IntegrationID,
			Name:           description["app_name"].(string),
			JSONUrl:        org_integrations.JSONUrl,
			AppUrl:         description["app_url"].(string),
			AppLogo:        description["app_logo"].(string),
			AppDescription: description["app_description"].(string),
			Category:       category,
			Status:         "success",
			IsActive:       org_integrations.IsActive,
			CreatedAt:      org_integrations.CreatedAt,
			UpdatedAt:      org_integrations.UpdatedAt,
		}

		int_resp = append(int_resp, struct {
			models.Integrations
			Linked bool "json:\"linked\""
		}{
			Integrations: integration,
			Linked:       true,
		})
	}

	return int_resp, paginationResponse, code, nil
}

func ActivateChannelIntegration(ids map[string]string, req models.ActivateChannelIntegration, db *gorm.DB) error {
	var (
		ocIntegrations  models.OrganisationChannelsIntegrations
		orgIntegrations models.OrganisationIntegrations
		channels        models.Channels
	)

	exists := postgresql.CheckExists(db, &orgIntegrations, "org_id = ? AND integration_id = ?", ids["organisation_id"], ids["integration_id"])
	if !exists {
		return errors.New("organisation does not have that integration")
	}

	exists = postgresql.CheckExists(db, &channels, "id = ? AND organisation_id = ?", ids["channel_id"], ids["organisation_id"])
	if !exists {
		return errors.New("organisation does not have that channel")
	}

	err := ocIntegrations.ActivateChannelIntegration(db, req, ids)
	if err != nil {
		return err
	}

	return nil
}

func IntegrationChannels(ids map[string]string, db *gorm.DB) (gin.H, error) {
	var (
		ocIntegrations  models.OrganisationChannelsIntegrations
		orgIntegrations models.OrganisationIntegrations
		res             gin.H
	)

	exists := postgresql.CheckExists(db, &orgIntegrations, "org_id = ? AND integration_id = ?", ids["organisation_id"], ids["integration_id"])
	if !exists {
		return nil, errors.New("organisation does not have that integration")
	}

	response, is_allChannels, err := ocIntegrations.FetchIntegrationChannels(db, ids)
	if err != nil {
		return res, err
	}

	res = gin.H{
		"is_allchannels": !is_allChannels,
		"channels":       response,
	}

	return res, nil
}

func CheckIntegrationIsActive(ids map[string]string, db *gorm.DB) (gin.H, error) {
	var (
		ocIntegrations  models.OrganisationChannelsIntegrations
		orgIntegrations models.OrganisationIntegrations
		res             gin.H
	)

	exists := postgresql.CheckExists(db, &orgIntegrations, "org_id = ? AND integration_id = ?", ids["organisation_id"], ids["integration_id"])
	if !exists {
		return nil, errors.New("organisation does not exist or have that integration")
	}

	status, err := ocIntegrations.CheckIntegrationIsActive(db, ids)
	if err != nil {
		return res, err
	}

	res = gin.H{
		"status": status,
	}

	return res, nil
}

func UpdateCustomIntegrationSettings(ids map[string]string, req models.CustomIntegrationSettingRequest, db *gorm.DB, extReq request.ExternalRequest) error {

	var (
		orgIntegration models.OrganisationIntegrations
		ucis           models.CustomIntegrationsSetting
	)

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if !exists {
		return errors.New("organisation does not exist or have that integration")
	}

	settings := req.SettingEntry
	settingJsonData, err := json.Marshal(settings)

	if err != nil {
		return fmt.Errorf("error serializing to JSON: %v", err)
	}

	serialized_settings := string(settingJsonData)
	req.SerializedEntry = serialized_settings

	err = ucis.UpdateCustomIntegrationSettings(db, req, ids)

	if err != nil {
		return err
	}

	return nil
}

func GetCustomIntegrationSettings(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) (map[string]interface{}, int, error) {

	var (
		orgIntegration models.OrganisationIntegrations
		ucis           models.CustomIntegrationsSetting

		deserialize_settings map[string]interface{}
	)

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if !exists {
		return deserialize_settings, http.StatusNotFound, errors.New("Integration not connnected yet")
	}

	exists = postgresql.CheckExists(db, &ucis, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if !exists {
		return deserialize_settings, http.StatusNotFound, errors.New("Integration not connnected yet")
	}

	settings := ucis.SettingEntry

	// unserialize the settings text

	err := json.Unmarshal([]byte(settings), &deserialize_settings)

	deserialize_settings["is_system"] = ucis.IsSystem
	deserialize_settings["is_active"] = orgIntegration.IsActive

	if err != nil {
		return deserialize_settings, http.StatusInternalServerError, fmt.Errorf("Error deserializing JSON: %v", err)
	}

	return deserialize_settings, http.StatusOK, nil
}

func GetCustomIntegrationStatus(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) (map[string]bool, int, error) {

	var (
		orgIntegration models.OrganisationIntegrations
		status         map[string]bool
	)

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if !exists {
		return status, http.StatusNotFound, errors.New("Integration not connnected yet")
	}

	status = map[string]bool{}

	status["is_system"] = orgIntegration.IsSystem
	status["is_active"] = orgIntegration.IsActive

	return status, http.StatusOK, nil
}
