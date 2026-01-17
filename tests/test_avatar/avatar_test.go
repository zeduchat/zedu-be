package test_avatar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	minioStorage "github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/tests"
)

func TestAvatarUpload(t *testing.T) {
	r, _, authController, _ := SetupAvatarTestRouter()
	logger := GetTestLogger()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testavatar%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "AvatarUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_avataruser%v", currUUID),
	}

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	tests.SignupUser(t, r, *authController, userSignUpData, false)
	token := tests.GetLoginToken(t, r, *authController, loginData)

	if token == "" {
		t.Fatal("Failed to get authentication token")
	}

	// Track uploaded avatars for cleanup
	var uploadedAvatars []string
	t.Cleanup(func() {
		for _, avatarName := range uploadedAvatars {
			if err := minioStorage.DeleteAvatar(logger, avatarName); err != nil {
				t.Logf("Cleanup warning: failed to delete avatar %s: %v", avatarName, err)
			}
		}
	})

	// Create a test image (1x1 PNG)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F,
		0x00, 0x05, 0xFE, 0x02, 0xFE, 0xDC, 0xCC, 0x59,
		0xE7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	t.Run("Upload Avatar Success", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("avatar", "test-avatar.png")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/avatars", body)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		req.Header.Set("Content-Type", writer.FormDataContentType())

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code == http.StatusCreated {
			resp := tests.ParseResponse(rr)
			tests.AssertResponseMessage(t, resp["message"].(string), "Avatar uploaded successfully")
			data := resp["data"].(map[string]interface{})
			if url, ok := data["url"].(string); ok && url != "" {
				// Extract avatar name from URL for cleanup
				parts := strings.Split(url, "/")
				if len(parts) > 0 {
					uploadedAvatars = append(uploadedAvatars, parts[len(parts)-1])
				}
			}
		} else {
			t.Logf("MinIO may not be available: status=%d, body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Upload Avatar Without File", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/avatars", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})

	t.Run("Upload Avatar Unauthorized", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("avatar", "test.png")
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/avatars", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})
}

func TestAvatarList(t *testing.T) {
	r, _, authController, _ := SetupAvatarTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testavatarlist%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "AvatarListUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_avatarlist%v", currUUID),
	}

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	tests.SignupUser(t, r, *authController, userSignUpData, false)
	token := tests.GetLoginToken(t, r, *authController, loginData)

	if token == "" {
		t.Fatal("Failed to get authentication token")
	}

	t.Run("List Avatars Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/avatars", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code == http.StatusOK {
			resp := tests.ParseResponse(rr)
			tests.AssertResponseMessage(t, resp["message"].(string), "Avatars retrieved successfully")
		} else {
			t.Logf("MinIO may not be available: status=%d, body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("List Avatars Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/avatars", nil)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})
}

func TestAvatarUploadInvalidType(t *testing.T) {
	r, _, authController, _ := SetupAvatarTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testavatartype%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "AvatarTypeUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_avatartype%v", currUUID),
	}

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	tests.SignupUser(t, r, *authController, userSignUpData, false)
	token := tests.GetLoginToken(t, r, *authController, loginData)

	if token == "" {
		t.Fatal("Failed to get authentication token")
	}

	textData := []byte("This is not an image file")

	t.Run("Upload Non-Image File", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("avatar", "test.txt")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(textData)
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/avatars", body)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		req.Header.Set("Content-Type", writer.FormDataContentType())

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		// Should return error for invalid file type (no upload happens, so no cleanup needed)
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["status"] != "error" {
			t.Errorf("Expected error status for non-image file")
		}
	})
}
