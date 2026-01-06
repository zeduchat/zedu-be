package test_onesignal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestRegisterOneSignalSubscriptionID(t *testing.T) {
	r, _ := SetupOnesignalTestRouter()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	testUser := CreateTestUser(t, db)

	token := generateTestToken(t, db, testUser.ID)

	tests := []struct {
		Name           string
		SubscriptionID string
		Platform       string
		ExpectedCode   int
		Message        string
		AuthToken      string
	}{
		{
			Name:           "Successful registration",
			SubscriptionID: "sub-test-123",
			Platform:       "ios",
			ExpectedCode:   http.StatusOK,
			Message:        "OneSignal subscription ID registered successfully",
			AuthToken:      token,
		},
		{
			Name:           "Missing subscription ID",
			SubscriptionID: "",
			Platform:       "android",
			ExpectedCode:   http.StatusBadRequest,
			Message:        "Invalid request body",
			AuthToken:      token,
		},
		{
			Name:           "Unauthorized - no token",
			SubscriptionID: "sub-test-456",
			Platform:       "web",
			ExpectedCode:   http.StatusUnauthorized,
			Message:        "Token could not be found!",
			AuthToken:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			reqBody := map[string]string{
				"subscription_id": tt.SubscriptionID,
				"platform":        tt.Platform,
			}

			jsonBody, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/v1/users/onesignal-subscription-id", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			if tt.AuthToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.AuthToken)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.ExpectedCode, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.ExpectedCode == http.StatusOK {
				assert.Contains(t, response["message"].(string), tt.Message)
				assert.Equal(t, testUser.ID, response["data"].(map[string]interface{})["user_id"])
				assert.Equal(t, tt.SubscriptionID, response["data"].(map[string]interface{})["subscription_id"])
			} else {
				assert.Contains(t, response["message"].(string), tt.Message)
			}
		})
	}

	testUser.OneSignalSubscriptionID = "sub-test-123"
	db.Postgresql.Save(&testUser)

	var user models.User
	err := db.Postgresql.Where("id = ?", testUser.ID).First(&user).Error
	require.NoError(t, err)
	assert.Equal(t, "sub-test-123", user.OneSignalSubscriptionID)
}

func TestUpdateOneSignalSubscriptionID(t *testing.T) {
	r, _ := SetupOnesignalTestRouter()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	testUser := CreateTestUser(t, db)

	token := generateTestToken(t, db, testUser.ID)
	testUser.OneSignalSubscriptionID = "sub-old-123"
	db.Postgresql.Save(&testUser)

	tests := []struct {
		Name           string
		SubscriptionID string
		ExpectedCode   int
		Message        string
	}{
		{
			Name:           "Successful update",
			SubscriptionID: "sub-new-456",
			ExpectedCode:   http.StatusOK,
			Message:        "OneSignal subscription ID updated successfully",
		},
		{
			Name:           "Missing subscription ID",
			SubscriptionID: "",
			ExpectedCode:   http.StatusBadRequest,
			Message:        "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			reqBody := map[string]string{
				"subscription_id": tt.SubscriptionID,
			}

			jsonBody, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest("PUT", "/api/v1/users/onesignal-subscription-id", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.ExpectedCode, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response["message"].(string), tt.Message)

			if tt.ExpectedCode == http.StatusOK {
				var user models.User
				err := db.Postgresql.Where("id = ?", testUser.ID).First(&user).Error
				require.NoError(t, err)
				assert.Equal(t, tt.SubscriptionID, user.OneSignalSubscriptionID)
			}
		})
	}
}

func TestGetOneSignalSubscriptionID(t *testing.T) {
	r, _ := SetupOnesignalTestRouter()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	testUser := CreateTestUser(t, db)

	token := generateTestToken(t, db, testUser.ID)
	testUser.OneSignalSubscriptionID = "sub-test-789"
	db.Postgresql.Save(&testUser)

	tests := []struct {
		Name           string
		SubscriptionID string
		ExpectedCode   int
		Message        string
	}{
		{
			Name:           "Successful retrieval",
			SubscriptionID: "sub-test-789",
			ExpectedCode:   http.StatusOK,
			Message:        "OneSignal subscription ID retrieved successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/users/onesignal-subscription-id", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.ExpectedCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response["message"].(string), tt.Message)
			assert.Equal(t, testUser.ID, response["data"].(map[string]interface{})["user_id"])
			assert.Equal(t, tt.SubscriptionID, response["data"].(map[string]interface{})["subscription_id"])
		})
	}
}

func generateTestToken(t *testing.T, db *storage.Database, userID string) string {
	cfg := config.GetConfig()
	accessUUID := utility.GenerateUUID()
	claims := jwt.MapClaims{
		"user_id":     userID,
		"org_id":      "00000000-0000-0000-0000-000000000000",
		"access_uuid": accessUUID,
		"exp":         time.Now().Add(time.Hour * 24).Unix(),
		"authorised":  true,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Server.Secret))
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	accessToken := models.AccessToken{
		ID:               accessUUID,
		LoginAccessToken: tokenString,
		OwnerID:          userID,
		IsLive:           true,
	}
	if err := db.Postgresql.Create(&accessToken).Error; err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}

	return tokenString
}
