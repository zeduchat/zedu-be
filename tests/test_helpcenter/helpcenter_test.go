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
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestHelpCenterFlow(t *testing.T) {
	router, authController := setup()
	currUUID := utility.GenerateUUID()
	userToken := initialise(currUUID, t, router, *authController, true)
	token := createAdminAndGetToken(t, router, authController.Db, models.RoleAdmin)

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
		Title:   fmt.Sprintf("Article %s", currUUID),
		Content: "This is a test article",
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

	// Ordinary user tokens are refused on both write routes.
	tests.AssertStatusCode(t, doRequest(router, http.MethodPost, "/api/v1/help-center/categories", userToken, categoryPayloadBytes).Code, http.StatusUnauthorized)
	tests.AssertStatusCode(t, doRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/help-center/articles/%s", createdCategoryID), userToken, articlePayloadBytes).Code, http.StatusUnauthorized)

	// Reads stay public.
	tests.AssertStatusCode(t, doRequest(router, http.MethodGet, "/api/v1/help-center/categories", "", nil).Code, http.StatusOK)
	tests.AssertStatusCode(t, doRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/help-center/articles/categories/%s", createdCategoryID), "", nil).Code, http.StatusOK)
}

func TestHelpCenterWritesRejectNonAdmins(t *testing.T) {
	router, authController := setup()
	currUUID := utility.GenerateUUID()
	userToken := initialise(currUUID, t, router, *authController, true)
	adminToken := createAdminAndGetToken(t, router, authController.Db, models.RoleAdmin)

	categoryBody, _ := json.Marshal(models.CreateHelpCenterCategory{Name: fmt.Sprintf("Injected %s", currUUID), Description: "injected category"})
	articleBody, _ := json.Marshal(models.CreateHelpCenterArticle{Title: fmt.Sprintf("Injected %s", currUUID), Content: "injected article"})

	cases := []struct {
		name  string
		token string
		want  int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"ordinary user token", userToken, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tests.AssertStatusCode(t, doRequest(router, http.MethodPost, "/api/v1/help-center/categories", tc.token, categoryBody).Code, tc.want)
		})
	}

	// An admin can create a category, but a user still cannot add articles to it.
	resp := doRequest(router, http.MethodPost, "/api/v1/help-center/categories", adminToken, categoryBody)
	tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
	var apiResponse ApiResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &apiResponse); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	articlePath := fmt.Sprintf("/api/v1/help-center/articles/%s", apiResponse.Data.CategoryID)
	tests.AssertStatusCode(t, doRequest(router, http.MethodPost, articlePath, userToken, articleBody).Code, http.StatusUnauthorized)
	tests.AssertStatusCode(t, doRequest(router, http.MethodPost, articlePath, "", articleBody).Code, http.StatusUnauthorized)
	tests.AssertStatusCode(t, doRequest(router, http.MethodPost, articlePath, adminToken, articleBody).Code, http.StatusCreated)
}

func doRequest(router *gin.Engine, method, path, token string, body []byte) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func createAdminAndGetToken(t *testing.T, router *gin.Engine, db *storage.Database, role string) string {
	t.Helper()
	const rawPassword = "password"
	hashedPassword, _ := utility.HashPassword(rawPassword)
	adminUser := models.Admin{
		ID:       utility.GenerateUUID(),
		Name:     "Help Center Admin Test",
		Email:    fmt.Sprintf("hcadmin%s@qa.team", utility.RandomString(5)),
		Role:     role,
		IsActive: true,
		Password: hashedPassword,
	}
	if err := db.Postgresql.Create(&adminUser).Error; err != nil {
		t.Fatalf("Failed to create admin: %v", err)
	}

	loginBody, _ := json.Marshal(map[string]string{"email": adminUser.Email, "password": rawPassword})
	resp := doRequest(router, http.MethodPost, "/api/v1/backoffice/login", "", loginBody)
	if resp.Code != http.StatusOK {
		t.Fatalf("Admin login failed, status: %d, body: %s", resp.Code, resp.Body.String())
	}
	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &loginResp); err != nil || loginResp.Data.AccessToken == "" {
		t.Fatalf("Admin login response missing access_token: %s", resp.Body.String())
	}
	return loginResp.Data.AccessToken
}

type ApiResponse struct {
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Data       struct {
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
