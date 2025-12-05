package test_file_management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

func TestRestoreFile(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testrestore%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "RestoreUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_restoreuser%v", currUUID),
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

	var fileID string

	// Helper to upload a file
	uploadFile := func() string {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Use valid JPEG magic numbers
		content := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
		filename := "test_restore.jpg"
		mimeType := "image/jpeg"

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "files", filename))
		h.Set("Content-Type", mimeType)

		part, err := writer.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		part.Write(content)
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-files", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Upload failed with status %d", rr.Code)
		}

		resp := tests.ParseResponse(rr)
		files := resp["data"].([]interface{})
		f := files[0].(map[string]interface{})
		return f["id"].(string)
	}

	t.Run("RestoreFile_Success", func(t *testing.T) {
		fileID = uploadFile()

		// delete the file first
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// restore the file
		req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/file/%s/restore", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify it's not deleted in DB
		var file models.File
		err := db.Postgresql.Where("id = ?", fileID).First(&file).Error
		if err != nil {
			t.Fatalf("Failed to find file in DB: %v", err)
		}

		if file.DeletedAt.Valid {
			t.Errorf("File should not be deleted after restore")
		}
	})

	t.Run("RestoreFile_NotDeleted", func(t *testing.T) {

		fileID := uploadTestFile(t, r, token, "test_restore_not_deleted.txt", "text/plain", []byte("test content"))

		/*
			try to restore it
			should fail as it's not deleted
		*/

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/file/%s/restore", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		/*
			file that was not deleted should
			not be found for restoration
		*/
		tests.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})

	t.Run("RestoreFile_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/file/%s/restore", randomID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})

	t.Run("RestoreFile_Unauthorized", func(t *testing.T) {
		// create another user
		otherUserUUID := uuid.New().String()
		otherUser := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("otherrestore%v@qa.team", otherUserUUID),
			FirstName:   "Other",
			LastName:    "User",
			PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
			Password:    "password123",
			UserName:    fmt.Sprintf("other_restore%v", otherUserUUID),
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(otherUser)
		reqReg, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", &b)
		reqReg.Header.Set("Content-Type", "application/json")
		rrReg := httptest.NewRecorder()
		r.ServeHTTP(rrReg, reqReg)

		if rrReg.Code != http.StatusCreated && rrReg.Code != http.StatusOK {
			t.Fatalf("Registration failed for second user: %v", rrReg.Body.String())
		}

		otherLoginData := models.LoginRequestModel{
			Email:    otherUser.Email,
			Password: otherUser.Password,
		}

		var bLogin bytes.Buffer
		json.NewEncoder(&bLogin).Encode(otherLoginData)
		reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", &bLogin)
		reqLogin.Header.Set("Content-Type", "application/json")
		rrLogin := httptest.NewRecorder()
		r.ServeHTTP(rrLogin, reqLogin)
		if rrLogin.Code != http.StatusOK {
			t.Fatalf("Login failed for second user: %v", rrLogin.Body.String())
		}
		data := tests.ParseResponse(rrLogin)
		dataM := data["data"].(map[string]any)
		otherToken := dataM["access_token"].(string)

		/*
			Try to restore the first user's file (which was restored in first test)
			Let's delete it again first to make it a valid candidate for restore if authorized
		*/

		reqDel, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s", fileID), nil)
		reqDel.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rrDel := httptest.NewRecorder()
		r.ServeHTTP(rrDel, reqDel)
		tests.AssertStatusCode(t, rrDel.Code, http.StatusOK)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/file/%s/restore", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", otherToken))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})
}
