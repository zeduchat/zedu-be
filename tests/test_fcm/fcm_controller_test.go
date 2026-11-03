package test_fcm

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestCreateFcmToken_Endpoints(t *testing.T) {
	r, _ := SetupFCMTestRouter()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	user := CreateTestUser(t, db)
	authToken := GenerateTestToken(t, db, user.ID)

	t.Run("Success with platform android", func(t *testing.T) {
		body := map[string]string{
			"fcm_token": "token_android_" + utility.GenerateUUID(),
			"platform":  "android",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fcmtoken", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+authToken)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "success", resp["status"])
		data, ok := resp["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "firebase", data["provider"])
	})

	t.Run("Success with mixed case platform Android normalized", func(t *testing.T) {
		body := map[string]string{
			"fcm_token": "token_mixed_" + utility.GenerateUUID(),
			"platform":  "Android",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fcmtoken", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+authToken)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Default platform to web when omitted", func(t *testing.T) {
		body := map[string]string{
			"fcm_token": "token_web_" + utility.GenerateUUID(),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fcmtoken", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+authToken)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Reject invalid platform", func(t *testing.T) {
		body := map[string]string{
			"fcm_token": "token_invalid_" + utility.GenerateUUID(),
			"platform":  "windows_phone",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fcmtoken", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+authToken)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Reject missing fcm_token", func(t *testing.T) {
		body := map[string]string{
			"platform": "android",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fcmtoken", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+authToken)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Reject unauthorized request", func(t *testing.T) {
		body := map[string]string{
			"fcm_token": "some_token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fcmtoken", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestDeleteFcmToken_Endpoints(t *testing.T) {
	r, _ := SetupFCMTestRouter()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	user := CreateTestUser(t, db)
	authToken := GenerateTestToken(t, db, user.ID)

	t.Run("Success deleting fcm token", func(t *testing.T) {
		body := map[string]string{
			"fcm_token": "token_to_delete_" + utility.GenerateUUID(),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/fcmtoken", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+authToken)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "success", resp["status"])
	})

	t.Run("Reject delete with missing token", func(t *testing.T) {
		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/fcmtoken", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+authToken)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Reject delete without authorization", func(t *testing.T) {
		body := map[string]string{
			"fcm_token": "some_token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/fcmtoken", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
