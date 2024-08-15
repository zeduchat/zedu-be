package test_helpcenter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestHelpCenterFlow(t *testing.T) {
	router, authController := setup()
	currUUID := utility.GenerateUUID()
	token := initialise(currUUID, t, router, *authController, true)

	categoryPayload := models.CreateHelpCenterCategory{
		Name:        fmt.Sprintf("Category %s", currUUID),
		Description: "This is a test category",
	}

	categoryPayloadBytes, _ := json.Marshal(categoryPayload)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/help-center/categories", bytes.NewBuffer(categoryPayloadBytes))
	if err != nil {
		t.Fatalf("Failed to create new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
	
	var apiResponse ApiResponse

	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}

	createdCategoryID := apiResponse.Data.CategoryID

	fmt.Printf("Created Category ID: %s\n", createdCategoryID)

	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	articlePayload := models.CreateHelpCenterArticle{
		Title:      fmt.Sprintf("Article %s", currUUID),
		Content:    "This is a test article",
	}

	articlePayloadBytes, _ := json.Marshal(articlePayload)

	req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/help-center/articles/%s", createdCategoryID), bytes.NewBuffer(articlePayloadBytes))
	if err != nil {
		t.Fatalf("Failed to create new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	tests.AssertStatusCode(t, resp.Code, http.StatusCreated)

	var createdArticle models.HelpCenterArticle
	err = json.Unmarshal(resp.Body.Bytes(), &createdArticle)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
}

type ApiResponse struct {
	Status      string `json:"status"`
	StatusCode  int    `json:"status_code"`
	Message     string `json:"message"`
	Data        struct {
		CategoryID string `json:"category_id"`
	} `json:"data"`
}
	
	
	
func setup() (*gin.Engine, *auth.Controller) {
	router, hlpCntController := SetupHelpCenterTestRouter()
	authController := auth.Controller{
		Db:        hlpCntController.Db,
		Validator: hlpCntController.Validator,
		Logger:    hlpCntController.Logger,
		ExtReq:    hlpCntController.ExtReq,
	}
	return router, &authController
}