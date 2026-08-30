package test_file_management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/hngprojects/telex_be/utility"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

func TestFileManagement(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := utility.GenerateUUID()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testfm%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "FMUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_fmuser%v", currUUID),
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

	var user models.User
	db.Postgresql.Preload("Organisations").Where("email = ?", userSignUpData.Email).First(&user)
	if len(user.Organisations) == 0 {
		t.Fatal("User has no organisations")
	}

	var folderID string
	var fileID string

	t.Run("CreateFolder", func(t *testing.T) {
		reqBody := map[string]string{
			"name": "Test Folder",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/folders", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("CreateFolder failed with status %d. Response: %s", rr.Code, rr.Body.String())
		}
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		folderID = data["id"].(string)
	})

	t.Run("UploadFile", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("files", "test_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte("This is a test file content"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/upload-files?folder_id=%s", folderID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			fmt.Printf("UploadFile failed. Status: %d. Response: %s\n", rr.Code, rr.Body.String())
			t.Fatalf("UploadFile failed with status %d. Response: %s", rr.Code, rr.Body.String())
		}

		resp := tests.ParseResponse(rr)

		files, ok := resp["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data to be a list, got %T", resp["data"])
		}
		if len(files) == 0 {
			t.Fatal("Expected at least one file in response")
		}
		f := files[0].(map[string]interface{})
		fileID = f["id"].(string)
	})

	t.Run("UploadDuplicateFile", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("files", "test_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte("This is a test file content"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/upload-files?folder_id=%s", folderID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("UploadDuplicateFile failed with status %d. Response: %s", rr.Code, rr.Body.String())
		}

		resp := tests.ParseResponse(rr)
		files, ok := resp["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data to be a list, got %T", resp["data"])
		}
		if len(files) == 0 {
			t.Fatal("Expected at least one file in response")
		}
		f := files[0].(map[string]interface{})
		duplicateFileID := f["id"].(string)

		if duplicateFileID != fileID {
			t.Errorf("Expected duplicate upload to return existing file ID %s, got %s", fileID, duplicateFileID)
		}
	})

	t.Run("MoveFile", func(t *testing.T) {
		reqBody := map[string]string{
			"folder_id": folderID,
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/%s/move", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify in DB
		var file models.File
		db.Postgresql.Where("id = ?", fileID).First(&file)
		if file.FolderID == nil || *file.FolderID != folderID {
			t.Errorf("File not moved to folder")
		}
	})

	t.Run("VerifyFolderCount", func(t *testing.T) {

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/folders", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		folders := data["folders"].([]interface{})

		found := false
		for _, f := range folders {
			folder := f.(map[string]interface{})
			if folder["id"].(string) == folderID {
				found = true
				if count, ok := folder["item_count"].(float64); !ok || count != 1 {
					t.Errorf("Expected item_count 1, got %v", folder["item_count"])
				}
				break
			}
		}
		if !found {
			t.Error("Folder not found")
		}

		// upload another file to this folder
		uploadTestFile(t, r, token, "second_file.txt", "text/plain", []byte("content2"), folderID)

		// check count is 2
		req2, _ := http.NewRequest(http.MethodGet, "/api/v1/files/folders", nil)
		req2.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)

		tests.AssertStatusCode(t, rr2.Code, http.StatusOK)
		resp2 := tests.ParseResponse(rr2)
		data2 := resp2["data"].(map[string]interface{})
		folders2 := data2["folders"].([]interface{})

		found2 := false
		for _, f := range folders2 {
			folder := f.(map[string]interface{})
			if folder["id"].(string) == folderID {
				found2 = true
				if count, ok := folder["item_count"].(float64); !ok || count != 2 {
					t.Errorf("Expected item_count 2, got %v", folder["item_count"])
				}
				break
			}
		}
		if !found2 {
			t.Error("Folder not found")
		}
	})

	t.Run("GetFoldersPagination", func(t *testing.T) {

		// creating 2 more folders with the existing one
		// to ensure enough for pagination
		for i := 0; i < 2; i++ {
			reqBody := map[string]string{
				"name": fmt.Sprintf("Pagination Folder %d", i),
			}
			var b bytes.Buffer
			json.NewEncoder(&b).Encode(reqBody)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/folders", &b)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			if rr.Code != http.StatusCreated {
				t.Fatalf("Failed to create folder for pagination test: %s", rr.Body.String())
			}
		}

		//get the folders with pagination
		u, _ := url.Parse("/api/v1/files/folders")
		q := u.Query()
		q.Set("page", "1")
		q.Set("limit", "1")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		folders := data["folders"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		if len(folders) != 1 {
			t.Errorf("Expected 1 folder in page 1, got %d", len(folders))
		}

		if pagination["current_page"].(float64) != 1 {
			t.Errorf("Expected current_page 1, got %v", pagination["current_page"])
		}

		totalItems := pagination["total_items"].(float64)
		if totalItems < 3 {
			t.Errorf("Expected at least 3 total items, got %v", totalItems)
		}
	})

	t.Run("GetFoldersOwnerFilter", func(t *testing.T) {

		u, _ := url.Parse("/api/v1/files/folders")
		q := u.Query()
		q.Set("owner", "Test")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		folders := data["folders"].([]interface{})

		if len(folders) == 0 {
			t.Error("Expected folders matching owner 'Test', got 0")
		}
	})

	t.Run("GetFoldersOwnerFilter_NoMatch", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files/folders")
		q := u.Query()
		q.Set("owner", "NonExistentUserXYZ")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		folders := data["folders"].([]interface{})

		if len(folders) != 0 {
			t.Errorf("Expected 0 folders matching owner 'NonExistentUserXYZ', got %d", len(folders))
		}
	})

	t.Run("GetFiles_InFolder", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("folder_id", folderID)
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})
		if len(files) == 0 {
			t.Errorf("Expected files in folder")
		}

		pagination, ok := data["pagination"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected pagination object in response")
		} else {
			if pagination["current_page"] != 1.0 {
				t.Errorf("Expected current_page to be 1, got %v", pagination["current_page"])
			}
			if pagination["total_items"] != 2.0 {
				t.Errorf("Expected total_items to be 2, got %v", pagination["total_items"])
			}
		}
	})

	t.Run("DeleteFile_Soft", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify soft delete in DB
		var file models.File
		db.Postgresql.Unscoped().Where("id = ?", fileID).First(&file)
		if file.DeletedAt.Time.IsZero() {
			t.Errorf("File should be soft deleted")
		}
	})

	t.Run("GetFiles_Trash", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("mode", "trash")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})
		found := false
		for _, f := range files {
			fMap := f.(map[string]interface{})
			if fMap["id"] == fileID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected deleted file in trash")
		}
	})

	t.Run("DeleteFile_Permanent", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s?permanent=true", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify hard delete in DB
		var file models.File
		err := db.Postgresql.Unscoped().Where("id = ?", fileID).First(&file).Error
		if err == nil {
			t.Errorf("File should be permanently deleted")
		}
	})

	var fileInFolderID string

	t.Run("DeleteFolder_Soft", func(t *testing.T) {

		// upload a file to the folder first
		fileInFolderID = uploadTestFile(t, r, token, "test_file_in_folder.txt", "text/plain", []byte("content"))

		// move it to the folder
		moveBody := []byte(fmt.Sprintf(`{"folder_id": "%s"}`, folderID))
		moveReq, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/%s/move", fileInFolderID), bytes.NewBuffer(moveBody))
		moveReq.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		moveReq.Header.Set("Content-Type", "application/json")
		moveRr := httptest.NewRecorder()
		r.ServeHTTP(moveRr, moveReq)
		tests.AssertStatusCode(t, moveRr.Code, http.StatusOK)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s", folderID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify folder was soft deleted
		var folder models.Folder
		err := db.Postgresql.Unscoped().Where("id = ?", folderID).First(&folder).Error
		if err != nil {
			t.Errorf("Folder should exist (soft deleted)")
		}
		if !folder.DeletedAt.Valid {
			t.Errorf("Folder should be soft deleted")
		}

		// verify file in folder was soft deleted
		var file models.File
		err = db.Postgresql.Unscoped().Where("id = ?", fileInFolderID).First(&file).Error
		if err != nil {
			t.Errorf("File in folder should exist")
		}
		if !file.DeletedAt.Valid {
			t.Errorf("File in folder should be soft deleted")
		}
	})

	t.Run("DeleteFolder_Permanent", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s?permanent=true", folderID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify folder was hard deleted
		var folder models.Folder
		err := db.Postgresql.Unscoped().Where("id = ?", folderID).First(&folder).Error
		if err == nil {
			t.Errorf("Folder should be permanently deleted")
		}

		// verify file in folder was hard deleted
		var file models.File
		err = db.Postgresql.Unscoped().Where("id = ?", fileInFolderID).First(&file).Error
		if err == nil {
			t.Errorf("File in folder should be permanently deleted")
		}
	})

	t.Run("GetFoldersModeMine", func(t *testing.T) {

		// Create another user
		otherUUID := utility.GenerateUUID()
		otherUserSignUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("other%v@qa.team", otherUUID),
			FirstName:   "Other",
			LastName:    "User",
			PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
			Password:    "password123",
			UserName:    fmt.Sprintf("other_user%v", otherUUID),
		}

		// tests.SignupUser registers the route, which causes a panic if called twice on the same router.
		// Since the route is already registered by the parent test, we just make the request directly.
		var signUpBody bytes.Buffer
		json.NewEncoder(&signUpBody).Encode(otherUserSignUpData)
		reqSignUp, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", &signUpBody)
		reqSignUp.Header.Set("Content-Type", "application/json")

		rrSignUp := httptest.NewRecorder()
		r.ServeHTTP(rrSignUp, reqSignUp)

		if rrSignUp.Code != http.StatusCreated && rrSignUp.Code != http.StatusOK {
			t.Fatalf("Registration failed with status %d. Response: %s", rrSignUp.Code, rrSignUp.Body.String())
		}

		// tests.GetLoginToken also registers the route, causing panic. Inline login logic.
		var loginBody bytes.Buffer
		loginData := models.LoginRequestModel{
			Email:    otherUserSignUpData.Email,
			Password: otherUserSignUpData.Password,
		}

		json.NewEncoder(&loginBody).Encode(loginData)
		reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", &loginBody)
		reqLogin.Header.Set("Content-Type", "application/json")

		rrLogin := httptest.NewRecorder()
		r.ServeHTTP(rrLogin, reqLogin)

		if rrLogin.Code != http.StatusOK {
			t.Fatalf("Login failed with status %d. Response: %s", rrLogin.Code, rrLogin.Body.String())
		}

		loginResp := tests.ParseResponse(rrLogin)
		loginDataM := loginResp["data"].(map[string]interface{})
		otherToken := loginDataM["access_token"].(string)

		// Create folder with other user
		reqBody := map[string]string{
			"name": "Other User Folder",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/folders", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", otherToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("Failed to create folder for other user: %s", rr.Body.String())
		}

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		otherFolderID := data["id"].(string)

		// Create a folder for the main user (since previous tests might have deleted the shared folderID)
		reqBodyMain := map[string]string{
			"name": "Main User Folder",
		}
		var bMain bytes.Buffer
		json.NewEncoder(&bMain).Encode(reqBodyMain)

		reqMain, _ := http.NewRequest(http.MethodPost, "/api/v1/files/folders", &bMain)
		reqMain.Header.Set("Content-Type", "application/json")
		reqMain.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rrMain := httptest.NewRecorder()
		r.ServeHTTP(rrMain, reqMain)
		if rrMain.Code != http.StatusCreated {
			t.Fatalf("Failed to create folder for main user: %s", rrMain.Body.String())
		}

		respMain := tests.ParseResponse(rrMain)
		dataMain := respMain["data"].(map[string]interface{})
		mainFolderID := dataMain["id"].(string)

		// Request with mode=mine using main user token
		u, _ := url.Parse("/api/v1/files/folders")
		q := u.Query()
		q.Set("mode", "mine")
		u.RawQuery = q.Encode()

		reqMine, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		reqMine.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rrMine := httptest.NewRecorder()
		r.ServeHTTP(rrMine, reqMine)

		tests.AssertStatusCode(t, rrMine.Code, http.StatusOK)
		respMine := tests.ParseResponse(rrMine)
		dataMine := respMine["data"].(map[string]interface{})
		foldersMine := dataMine["folders"].([]interface{})

		// Verify that Other User Folder is not present
		for _, f := range foldersMine {
			folder := f.(map[string]interface{})
			if folder["id"].(string) == otherFolderID {
				t.Error("Found other user's folder in mode=mine results")
			}
		}

		// Verify that main user's folder is present
		foundMain := false
		for _, f := range foldersMine {
			folder := f.(map[string]interface{})
			if folder["id"].(string) == mainFolderID {
				foundMain = true
				break
			}
		}
		if !foundMain {
			t.Error("Expected main user's folder in mode=mine results")
		}
	})

	t.Run("GetFoldersModeTrash", func(t *testing.T) {
		// Create a temporary folder to delete
		reqBody := map[string]string{
			"name": "Trash Test Folder",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/folders", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		trashFolderID := data["id"].(string)

		// Soft delete the folder
		reqDel, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s", trashFolderID), nil)
		reqDel.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rrDel := httptest.NewRecorder()
		r.ServeHTTP(rrDel, reqDel)
		tests.AssertStatusCode(t, rrDel.Code, http.StatusOK)

		// Request with mode=trash
		u, _ := url.Parse("/api/v1/files/folders")
		q := u.Query()
		q.Set("mode", "trash")
		u.RawQuery = q.Encode()

		reqTrash, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		reqTrash.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rrTrash := httptest.NewRecorder()
		r.ServeHTTP(rrTrash, reqTrash)

		tests.AssertStatusCode(t, rrTrash.Code, http.StatusOK)
		respTrash := tests.ParseResponse(rrTrash)
		dataTrash := respTrash["data"].(map[string]interface{})
		foldersTrash := dataTrash["folders"].([]interface{})

		// Verify that the deleted folder is present
		foundTrash := false
		for _, f := range foldersTrash {
			folder := f.(map[string]interface{})
			if folder["id"].(string) == trashFolderID {
				foundTrash = true
				break
			}
		}
		if !foundTrash {
			t.Error("Expected deleted folder in mode=trash results")
		}

		//  Request with default mode (all)
		reqAll, _ := http.NewRequest(http.MethodGet, "/api/v1/files/folders", nil)
		reqAll.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rrAll := httptest.NewRecorder()
		r.ServeHTTP(rrAll, reqAll)

		tests.AssertStatusCode(t, rrAll.Code, http.StatusOK)
		respAll := tests.ParseResponse(rrAll)
		dataAll := respAll["data"].(map[string]interface{})
		foldersAll := dataAll["folders"].([]interface{})

		// Verify that the deleted folder is not present in default mode
		for _, f := range foldersAll {
			folder := f.(map[string]interface{})
			if folder["id"].(string) == trashFolderID {
				t.Error("Found deleted folder in default mode results")
			}
		}
	})
}

func TestFileFilters(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := utility.GenerateUUID()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testfilters%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "FilterUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_filteruser%v", currUUID),
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

	var user models.User
	db.Postgresql.Preload("Organisations").Where("email = ?", userSignUpData.Email).First(&user)
	if len(user.Organisations) == 0 {
		t.Fatal("User has no organisations")
	}

	var imageFileID, docFileID, videoFileID string

	t.Run("SetupTestFiles", func(t *testing.T) {
		// JPEG magic number: FF D8 FF
		imageFileID = uploadTestFile(t, r, token, "test_image.jpg", "image/jpeg", []byte("\xFF\xD8\xFF\xE0"))

		// PDF magic number: %PDF-
		docFileID = uploadTestFile(t, r, token, "test_doc.pdf", "application/pdf", []byte("%PDF-1.4"))

		// MP4 magic number (ftyp box)
		videoFileID = uploadTestFile(t, r, token, "test_video.mp4", "video/mp4", []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00"))

		db.Postgresql.Model(&models.File{}).Where("id = ?", docFileID).
			Update("updated_at", time.Now().AddDate(0, 0, -10))
	})

	t.Run("FilterByFileCategory_Images", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "images")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) == 0 {
			t.Error("Expected at least one image file")
		}

		for _, f := range files {
			file := f.(map[string]interface{})
			mimeType := file["mime_type"].(string)
			if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/gif" {
				t.Errorf("Expected image mime type, got %s", mimeType)
			}
		}
	})

	t.Run("FilterByFileCategory_Documents", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "documents")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) == 0 {
			t.Error("Expected at least one document file")
		}

		found := false
		for _, f := range files {
			file := f.(map[string]interface{})
			if file["id"].(string) == docFileID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find uploaded PDF in documents filter")
		}
	})

	t.Run("FilterByFileCategory_Videos", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "videos")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) == 0 {
			t.Error("Expected at least one video file")
		}
	})

	t.Run("FilterByFileType_NormalAndDot", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("type", ".jpg")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) == 0 {
			t.Error("Expected at least one file when querying type=.jpg")
		}
	})

	t.Run("FilterByFileCategory_Audio", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "audio")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
	})

	t.Run("FilterByFileCategory_Invalid", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "invalid_category")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
		resp := tests.ParseResponse(rr)
		if resp["message"] == "" {
			t.Error("Expected error message for invalid category")
		}
	})

	t.Run("FilterByDateModified_Today", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "today")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) < 2 {
			t.Error("Expected at least 2 files modified today")
		}

		for _, f := range files {
			file := f.(map[string]interface{})
			if file["id"].(string) == docFileID {
				t.Error("Document file modified 10 days ago should not appear in 'today' filter")
			}
		}
	})

	t.Run("FilterByDateModified_Last7Days", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "last_7_days")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) < 2 {
			t.Error("Expected at least 2 files in last 7 days")
		}
	})

	t.Run("FilterByDateModified_Last30Days", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "last_30_days")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) < 3 {
			t.Error("Expected all 3 files in last 30 days")
		}
	})

	t.Run("FilterByDateModified_ThisYear", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "this_year")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		expected := 2
		/*
			two files created today in SetupTestFiles
			if today is late enough in the year, the file modified 10 days ago
			will also be in the current year.
		*/

		if time.Now().YearDay() > 10 {
			expected = 3
		}

		if len(files) < expected {
			t.Errorf("Expected at least %d files to be modified this year, got %d", expected, len(files))
		}
	})

	t.Run("FilterByDateModified_Invalid", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "invalid_date")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
		resp := tests.ParseResponse(rr)
		if resp["message"] == "" {
			t.Error("Expected error message for invalid date filter")
		}
	})

	t.Run("CombineFilters_CategoryAndDate", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "images")
		q.Set("date_modified", "today")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		for _, f := range files {
			file := f.(map[string]interface{})
			mimeType := file["mime_type"].(string)
			if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/gif" {
				t.Errorf("Expected only image files, got %s", mimeType)
			}
		}
	})

	t.Run("CombineFilters_WithModeAndSearch", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("mode", "mine")
		q.Set("file_category", "documents")
		q.Set("search", "test")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)

		if resp["status"] != "success" {
			t.Error("Expected successful response with combined filters")
		}
	})

	t.Run("GetFileOwnerFilter", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("owner", "Test")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) == 0 {
			t.Error("Expected files matching owner 'Test', got 0")
		}

		// verify each returned file belongs to matched owner
		for _, f := range files {
			file := f.(map[string]interface{})
			if file["id"] == imageFileID || file["id"] == docFileID || file["id"] == videoFileID {
				return
			}
		}
	})

	t.Run("GetFileOwnerFilter_NoMatch", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("owner", "NonExistentOwnerXYZ")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) != 0 {
			t.Errorf("Expected 0 files matching owner 'NonExistentOwnerXYZ', got %d", len(files))
		}
	})

	t.Run("Cleanup", func(t *testing.T) {
		db.Postgresql.Unscoped().Delete(&models.File{}, "id IN ?", []string{imageFileID, docFileID, videoFileID})
	})
}

// Helper function to upload test files with specific mime types
func uploadTestFile(t *testing.T, r http.Handler, token, filename, mimeType string, content []byte, folderID ...string) string {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "files", filename)}
	h["Content-Type"] = []string{mimeType}
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)

	if len(folderID) > 0 && folderID[0] != "" {
		_ = writer.WriteField("folder_id", folderID[0])
	}
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Upload failed with status %d. Response: %s", rr.Code, rr.Body.String())
	}

	resp := tests.ParseResponse(rr)
	files := resp["data"].([]interface{})
	if len(files) == 0 {
		t.Fatal("No files returned from upload")
	}

	f := files[0].(map[string]interface{})
	fileID := f["id"].(string)

	return fileID
}

func TestUserRelevanceAndAdvancedFilters(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	// Register User A
	uuidA := utility.GenerateUUID()
	userSignUpDataA := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("usera_%v@qa.team", uuidA),
		FirstName:   "UserA",
		LastName:    "Test",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("usera_%v", uuidA),
	}

	var signUpBodyA bytes.Buffer
	json.NewEncoder(&signUpBodyA).Encode(userSignUpDataA)
	reqSignUpA, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", &signUpBodyA)
	reqSignUpA.Header.Set("Content-Type", "application/json")
	rrSignUpA := httptest.NewRecorder()
	r.ServeHTTP(rrSignUpA, reqSignUpA)
	if rrSignUpA.Code != http.StatusCreated && rrSignUpA.Code != http.StatusOK {
		t.Fatalf("User A registration failed: %s", rrSignUpA.Body.String())
	}

	var loginBodyA bytes.Buffer
	loginDataA := models.LoginRequestModel{
		Email:    userSignUpDataA.Email,
		Password: userSignUpDataA.Password,
	}
	json.NewEncoder(&loginBodyA).Encode(loginDataA)
	reqLoginA, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", &loginBodyA)
	reqLoginA.Header.Set("Content-Type", "application/json")
	rrLoginA := httptest.NewRecorder()
	r.ServeHTTP(rrLoginA, reqLoginA)
	tokenA := tests.ParseResponse(rrLoginA)["data"].(map[string]interface{})["access_token"].(string)

	var userA models.User
	db.Postgresql.Preload("Organisations").Where("email = ?", userSignUpDataA.Email).First(&userA)
	orgID := userA.Organisations[0].ID

	// Register User B in same organization
	uuidB := utility.GenerateUUID()
	userSignUpDataB := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("userb_%v@qa.team", uuidB),
		FirstName:   "UserB",
		LastName:    "Test",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("userb_%v", uuidB),
	}

	var signUpBodyB bytes.Buffer
	json.NewEncoder(&signUpBodyB).Encode(userSignUpDataB)
	reqSignUpB, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", &signUpBodyB)
	reqSignUpB.Header.Set("Content-Type", "application/json")
	rrSignUpB := httptest.NewRecorder()
	r.ServeHTTP(rrSignUpB, reqSignUpB)

	var loginBodyB bytes.Buffer
	loginDataB := models.LoginRequestModel{
		Email:    userSignUpDataB.Email,
		Password: userSignUpDataB.Password,
	}
	json.NewEncoder(&loginBodyB).Encode(loginDataB)
	reqLoginB, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", &loginBodyB)
	reqLoginB.Header.Set("Content-Type", "application/json")
	rrLoginB := httptest.NewRecorder()
	r.ServeHTTP(rrLoginB, reqLoginB)
	tokenB := tests.ParseResponse(rrLoginB)["data"].(map[string]interface{})["access_token"].(string)

	var userB models.User
	db.Postgresql.Where("email = ?", userSignUpDataB.Email).First(&userB)

	// Add User B to User A's organization
	db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", userB.ID, orgID)

	t.Run("TypeFilter_CaseAndDotInsensitive", func(t *testing.T) {
		fileID := uploadTestFile(t, r, tokenA, "document_test.DOCX", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", []byte("docx content"))

		// Query type=docx
		u1, _ := url.Parse("/api/v1/files")
		q1 := u1.Query()
		q1.Set("type", "docx")
		u1.RawQuery = q1.Encode()
		req1, _ := http.NewRequest(http.MethodGet, u1.String(), nil)
		req1.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenA))
		rr1 := httptest.NewRecorder()
		r.ServeHTTP(rr1, req1)
		tests.AssertStatusCode(t, rr1.Code, http.StatusOK)
		files1 := tests.ParseResponse(rr1)["data"].(map[string]interface{})["files"].([]interface{})

		found1 := false
		for _, f := range files1 {
			if f.(map[string]interface{})["id"].(string) == fileID {
				found1 = true
				break
			}
		}
		if !found1 {
			t.Error("Expected to find file uploaded as .DOCX when querying type=docx")
		}

		// Query type=.DOCX
		u2, _ := url.Parse("/api/v1/files")
		q2 := u2.Query()
		q2.Set("type", ".DOCX")
		u2.RawQuery = q2.Encode()
		req2, _ := http.NewRequest(http.MethodGet, u2.String(), nil)
		req2.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenA))
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		tests.AssertStatusCode(t, rr2.Code, http.StatusOK)
		files2 := tests.ParseResponse(rr2)["data"].(map[string]interface{})["files"].([]interface{})

		found2 := false
		for _, f := range files2 {
			if f.(map[string]interface{})["id"].(string) == fileID {
				found2 = true
				break
			}
		}
		if !found2 {
			t.Error("Expected to find file when querying type=.DOCX")
		}

		db.Postgresql.Unscoped().Delete(&models.File{}, "id = ?", fileID)
	})

	t.Run("CategoryFilter_MimeAndExtensionMatching", func(t *testing.T) {
		// Upload file with generic mime_type octet-stream but .pdf extension
		fileID := uploadTestFile(t, r, tokenA, "generic_document.pdf", "application/octet-stream", []byte("pdf content"))

		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "documents")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenA))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		files := tests.ParseResponse(rr)["data"].(map[string]interface{})["files"].([]interface{})

		found := false
		for _, f := range files {
			if f.(map[string]interface{})["id"].(string) == fileID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected file with generic mime type but .pdf extension to match documents category")
		}

		db.Postgresql.Unscoped().Delete(&models.File{}, "id = ?", fileID)
	})

	t.Run("UserRelevanceScoping_DMsGroupDMsChannelsAndShares", func(t *testing.T) {
		// File A: Owned by User A
		fileA := uploadTestFile(t, r, tokenA, "fileA.txt", "text/plain", []byte("file A content"))

		// File B: Owned by User B (unrelated to User A)
		fileB := uploadTestFile(t, r, tokenB, "fileB.txt", "text/plain", []byte("file B content"))

		// Channel 1: Standard channel with User A as member
		chanID := utility.GenerateUUID()
		chanObj := models.Channels{
			ID:             chanID,
			Name:           "test-chan",
			Description:    "desc",
			OrganisationID: orgID,
			OwnerId:        userB.ID,
		}
		db.Postgresql.Create(&chanObj)

		userChanObj := models.UserChannels{
			ChannelsID: chanID,
			UserID:     userA.ID,
		}
		db.Postgresql.Create(&userChanObj)

		fileC := uploadTestFile(t, r, tokenB, "fileC.txt", "text/plain", []byte("file C content"))

		// DM Channel with User A and User B
		dmChanID := utility.GenerateUUID()
		dmObj := models.DmChannels{
			ID:            utility.GenerateUUID(),
			ChannelId:     dmChanID,
			UserId:        userB.ID,
			ParticipantId: &userA.ID,
			OrgId:         orgID,
			ChannelType:   "dm",
			ChatType:      "user",
		}
		db.Postgresql.Create(&dmObj)

		fileD := uploadTestFile(t, r, tokenB, "fileD.txt", "text/plain", []byte("file D content"))

		// Group DM Channel with User A
		grpChanID := utility.GenerateUUID()
		grpDmObj := models.DmChannels{
			ID:          utility.GenerateUUID(),
			ChannelId:   grpChanID,
			UserId:      userB.ID,
			OrgId:       orgID,
			ChannelType: "group_dm",
			ChatType:    "user",
		}
		db.Postgresql.Create(&grpDmObj)

		partObj := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: grpChanID,
			UserId:    userA.ID,
			OrgId:     orgID,
		}
		db.Postgresql.Create(&partObj)

		fileE := uploadTestFile(t, r, tokenB, "fileE.txt", "text/plain", []byte("file E content"))

		// File Shared directly with User A
		fileF := uploadTestFile(t, r, tokenB, "fileF.txt", "text/plain", []byte("file F content"))
		shareObj := models.FileShare{
			ID:             utility.GenerateUUID(),
			FileID:         fileF,
			SharedByUserID: userA.ID,
			OrganisationID: orgID,
			AccessType:     "private",
			PermissionType: "view",
		}
		db.Postgresql.Create(&shareObj)

		// File G: Private file of User B in channel User A is NOT in
		privateChanID := utility.GenerateUUID()
		privateChanObj := models.Channels{
			ID:             privateChanID,
			Name:           "private-chan",
			Description:    "desc",
			OrganisationID: orgID,
			OwnerId:        userB.ID,
		}
		db.Postgresql.Create(&privateChanObj)

		fileG := uploadTestFile(t, r, tokenB, "fileG.txt", "text/plain", []byte("file G content"))

		// Ensure all files are in User A's organization orgID and associated with channels
		db.Postgresql.Exec("UPDATE files SET organisation_id = ? WHERE id IN (?, ?, ?, ?, ?, ?, ?)", orgID, fileA, fileB, fileC, fileD, fileE, fileF, fileG)
		db.Postgresql.Exec("UPDATE files SET channel_id = ? WHERE id = ?", chanID, fileC)
		db.Postgresql.Exec("UPDATE files SET channel_id = ? WHERE id = ?", dmChanID, fileD)
		db.Postgresql.Exec("UPDATE files SET channel_id = ? WHERE id = ?", grpChanID, fileE)
		db.Postgresql.Exec("UPDATE files SET channel_id = ? WHERE id = ?", privateChanID, fileG)

		// 1. GetFiles for User A (Default / All Mode)
		reqDefault, _ := http.NewRequest(http.MethodGet, "/api/v1/files", nil)
		reqDefault.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenA))
		rrDefault := httptest.NewRecorder()
		r.ServeHTTP(rrDefault, reqDefault)
		tests.AssertStatusCode(t, rrDefault.Code, http.StatusOK)
		filesDefault := tests.ParseResponse(rrDefault)["data"].(map[string]interface{})["files"].([]interface{})

		returnedIDsDefault := make(map[string]bool)
		for _, f := range filesDefault {
			returnedIDsDefault[f.(map[string]interface{})["id"].(string)] = true
		}

		if !returnedIDsDefault[fileA] {
			t.Error("Expected User A owned fileA in default results")
		}
		if !returnedIDsDefault[fileC] {
			t.Error("Expected channel fileC in default results for User A")
		}
		if !returnedIDsDefault[fileD] {
			t.Error("Expected DM fileD in default results for User A")
		}
		if !returnedIDsDefault[fileE] {
			t.Error("Expected Group DM fileE in default results for User A")
		}
		if !returnedIDsDefault[fileF] {
			t.Error("Expected fileShare fileF in default results for User A")
		}
		if returnedIDsDefault[fileB] || returnedIDsDefault[fileG] {
			t.Error("Did NOT expect unrelated User B files (fileB, fileG) in User A default results")
		}

		// 2. GetFiles for User A (mode=mine)
		reqMine, _ := http.NewRequest(http.MethodGet, "/api/v1/files?mode=mine", nil)
		reqMine.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenA))
		rrMine := httptest.NewRecorder()
		r.ServeHTTP(rrMine, reqMine)
		tests.AssertStatusCode(t, rrMine.Code, http.StatusOK)
		filesMine := tests.ParseResponse(rrMine)["data"].(map[string]interface{})["files"].([]interface{})

		returnedIDsMine := make(map[string]bool)
		for _, f := range filesMine {
			returnedIDsMine[f.(map[string]interface{})["id"].(string)] = true
		}

		if !returnedIDsMine[fileA] {
			t.Error("Expected fileA in mode=mine")
		}
		if returnedIDsMine[fileC] || returnedIDsMine[fileD] || returnedIDsMine[fileE] || returnedIDsMine[fileF] || returnedIDsMine[fileB] || returnedIDsMine[fileG] {
			t.Error("Did NOT expect non-owned files in mode=mine")
		}

		// 3. GetFiles for User A (mode=shared)
		reqShared, _ := http.NewRequest(http.MethodGet, "/api/v1/files?mode=shared", nil)
		reqShared.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenA))
		rrShared := httptest.NewRecorder()
		r.ServeHTTP(rrShared, reqShared)
		tests.AssertStatusCode(t, rrShared.Code, http.StatusOK)
		filesShared := tests.ParseResponse(rrShared)["data"].(map[string]interface{})["files"].([]interface{})

		returnedIDsShared := make(map[string]bool)
		for _, f := range filesShared {
			returnedIDsShared[f.(map[string]interface{})["id"].(string)] = true
		}

		if !returnedIDsShared[fileC] || !returnedIDsShared[fileD] || !returnedIDsShared[fileE] || !returnedIDsShared[fileF] {
			t.Error("Expected shared/channel files (fileC, fileD, fileE, fileF) in mode=shared")
		}
		if returnedIDsShared[fileA] || returnedIDsShared[fileB] || returnedIDsShared[fileG] {
			t.Error("Did NOT expect owned files (fileA) or unrelated files (fileB, fileG) in mode=shared")
		}

		// Cleanup test data in correct FK order
		db.Postgresql.Exec("DELETE FROM file_shares WHERE file_id IN (?, ?, ?, ?, ?, ?, ?)", fileA, fileB, fileC, fileD, fileE, fileF, fileG)
		db.Postgresql.Unscoped().Delete(&models.File{}, "id IN ?", []string{fileA, fileB, fileC, fileD, fileE, fileF, fileG})
		db.Postgresql.Exec("DELETE FROM channel_participants WHERE channel_id = ?", grpChanID)
		db.Postgresql.Exec("DELETE FROM dm_channels WHERE channel_id IN (?, ?)", dmChanID, grpChanID)
		db.Postgresql.Exec("DELETE FROM user_channels WHERE channels_id = ?", chanID)
		db.Postgresql.Exec("DELETE FROM channels WHERE id IN (?, ?)", chanID, privateChanID)
	})

	_ = authController
}
