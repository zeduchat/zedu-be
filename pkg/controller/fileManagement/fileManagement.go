package fileManagement

import (
	"fmt"
	"math"
	"net/http"

	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	services "github.com/hngprojects/telex_be/services/fileManagement"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

const maxFileSize = 200 * 1024 * 1024

// method for the validation heavy lifting.
func Validate(filename string) (string, error) {
	trimmed := strings.TrimSpace(filename)
	if strings.HasPrefix(trimmed, ".") {
		return "", fmt.Errorf("filename cannot start with a period")
	}
	if trimmed == "" {
		return "", fmt.Errorf("file name cannot be empty")
	}
	if len(trimmed) > 255 {
		return "", fmt.Errorf("file name too long")
	}
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9\s._-]+$`)
	if !validPattern.MatchString(trimmed) {
		return "", fmt.Errorf("filename contains invalid characters")
	}
	return trimmed, nil
}

// UploadFolderWithFiles Handles Folder Uploads
func (base *Controller) UploadFolderWithFiles(c *gin.Context) {
	var req models.FolderWithFilesRequest
	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	validationErr := base.Validator.Struct(&req)
	if validationErr != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(validationErr, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}
	for _, fileHeader := range req.Files {
		if fileHeader.Size > maxFileSize {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "File exceeds max size", fmt.Sprintf("File '%s' is too large", fileHeader.Filename), nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
	}
	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Unauthorized", "User not authenticated", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userID := userClaims["user_id"].(string)
	orgID := userClaims["org_id"].(string)
	folderWithFilesParams := models.UploadFolderWithFilesParams{
		FolderName: req.FolderName,
		ParentID:   req.ParentID,
		Files:      req.Files,
		OrgID:      orgID,
		UserID:     userID,
	}

	folder, files, err := services.UploadFolderWithFiles(base.Db.Postgresql, base.Logger, folderWithFilesParams)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Folder upload failed", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	response := models.FolderUploadResponse{
		Folder:    folder,
		Files:     files,
		FileCount: len(files),
	}
	base.Logger.Info("Folder uploaded successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Folder uploaded successfully", response)
	c.JSON(http.StatusCreated, rd)

}

func (base *Controller) UploadController(c *gin.Context) {
	var req models.UploadRequest

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	validationErr := base.Validator.Struct(&req)
	if validationErr != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(validationErr, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Unauthorized", "User not authenticated", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userID := userClaims["user_id"].(string)
	orgID := userClaims["org_id"].(string)

	var uploadedFiles []*models.File

	for _, fileHeader := range req.Files {
		if fileHeader.Size > maxFileSize {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "File exceeds max size", "File too large", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid file", err, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		defer file.Close()

		fileData, err := services.UploadFile(base.Db.Postgresql, base.Logger, models.UploadFileParams{
			File:     file,
			Header:   fileHeader,
			FolderID: req.FolderID,
			OrgID:    orgID,
			UserID:   userID,
		})
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Upload failed", err.Error(), nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}

		uploadedFiles = append(uploadedFiles, fileData)
	}

	base.Logger.Info("Files uploaded successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Files uploaded successfully", uploadedFiles)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetFileDetailsByID(c *gin.Context) {
	fileId := c.Param("id")
	if !utility.IsValidUUID(fileId) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid file ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	file, err := services.GetFileDetailsByID(base.Db.Postgresql, fileId)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "File not found", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Info("Files located successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "File located successfully", file)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteFileDetailsByID(c *gin.Context) {
	fileId := c.Param("id")
	if !utility.IsValidUUID(fileId) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid file ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	thread_id := c.Query("thread_id")

	if _, err := uuid.Parse(thread_id); thread_id != "" && err != nil {
		base.Logger.Error("invalid thread id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid thread id format", "failed to decode thread id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	file, err := services.GetFileDetailsByID(base.Db.Postgresql, fileId)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "File not found", err.Error(), nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	deleteErr := services.DeleteFileDetailsByID(base.Logger, base.Db, file, fileId, thread_id)
	if deleteErr != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "File not deleted", deleteErr.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Files deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "File deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateFileName(c *gin.Context) {
	fileID := c.Param("id")
	if !utility.IsValidUUID(fileID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid file ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userID := userClaims["user_id"].(string)
	orgID := userClaims["org_id"].(string)
	var req models.RenameFileRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	validationErr := base.Validator.Struct(&req)
	if validationErr != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(validationErr, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	trimmed, err := Validate(req.FileName)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Validation failed", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, err := services.UpdateFileName(base.Db.Postgresql, base.Logger, models.UpdateFileNameParams{
		FileID:      fileID,
		NewFileName: trimmed,
		OrgID:       orgID,
		UserID:      userID,
	})
	if err != nil {
		base.Logger.Error("error updating file name")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to update file name", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	base.Logger.Info("File name updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "File name updated successfully", resp)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CreateFolder(c *gin.Context) {
	var req struct {
		Name     string  `json:"name" binding:"required"`
		ParentID *string `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if req.ParentID != nil && !utility.IsValidUUID(*req.ParentID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid parent_id format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, _ := c.Get("userClaims")
	userClaims := claims.(jwt.MapClaims)
	userID := userClaims["user_id"].(string)
	orgID := userClaims["org_id"].(string)

	folder, err := services.CreateFolder(base.Db.Postgresql, models.CreateFolderParams{
		Name:           req.Name,
		OrganizationID: orgID,
		UserID:         userID,
	})
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to create folder", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusCreated, "Folder created successfully", folder)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetFolders(c *gin.Context) {
	claims, _ := c.Get("userClaims")
	userClaims := claims.(jwt.MapClaims)
	orgID := userClaims["org_id"].(string)

	parentID := c.Query("parent_id")
	var pID *string
	if parentID != "" {
		if !utility.IsValidUUID(parentID) {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid parent_id format", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		pID = &parentID
	}

	folders, err := services.GetFolders(base.Db.Postgresql, orgID, pID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch folders", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Folders fetched successfully", folders)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteFolder(c *gin.Context) {
	folderID := c.Param("id")
	if !utility.IsValidUUID(folderID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid folder ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	err := services.DeleteFolder(base.Db.Postgresql, folderID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to delete folder", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Folder deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) MoveFile(c *gin.Context) {
	fileID := c.Param("id")
	if !utility.IsValidUUID(fileID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid file ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	var req struct {
		FolderID string `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if !utility.IsValidUUID(req.FolderID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid folder ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	file, err := services.MoveFile(base.Db.Postgresql, fileID, req.FolderID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to move file", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "File moved successfully", file)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetFiles(c *gin.Context) {
	claims, _ := c.Get("userClaims")
	userClaims := claims.(jwt.MapClaims)
	userID := userClaims["user_id"].(string)
	orgID := userClaims["org_id"].(string)

	queryParams := make(map[string]string)
	queryParams["mode"] = c.Query("mode")

	folderID := c.Query("folder_id")
	if folderID != "" && !utility.IsValidUUID(folderID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid folder_id format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	queryParams["folder_id"] = folderID
	queryParams["search"] = c.Query("search")
	queryParams["type"] = c.Query("type")

	fileCategory := c.Query("file_category")
	if fileCategory != "" {
		validCategories := map[string]bool{
			"documents":    true,
			"spreadsheets": true,
			"images":       true,
			"videos":       true,
			"music":        true,
		}
		if !validCategories[strings.ToLower(fileCategory)] {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error",
				"Invalid file_category. Valid values: documents, spreadsheets, images, videos, music", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		queryParams["file_category"] = fileCategory
	}

	dateModified := c.Query("date_modified")
	if dateModified != "" {
		validDateFilters := map[string]bool{
			"today":        true,
			"last_7_days":  true,
			"last_30_days": true,
			"this_year":    true,
			"last_year":    true,
		}
		if !validDateFilters[strings.ToLower(dateModified)] {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error",
				"Invalid date_modified. Valid values: today, last_7_days, last_30_days, this_year, last_year", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		queryParams["date_modified"] = dateModified
	}

	pagination := postgresql.GetPagination(c)
	page, limit := pagination.Page, pagination.Limit

	files, count, err := services.GetFiles(base.Db.Postgresql, models.GetFilesParams{
		OrgID:       orgID,
		UserID:      userID,
		QueryParams: queryParams,
		Page:        pagination.Page,
		Limit:       pagination.Limit,
	})
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch files", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	totalPages := int(math.Ceil(float64(count) / float64(limit)))
	paginationResponse := postgresql.PaginationResponse{
		CurrentPage:     page,
		PageCount:       len(files),
		TotalPagesCount: totalPages,
		TotalItems:      count,
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Files fetched successfully", map[string]interface{}{
		"files":      files,
		"pagination": paginationResponse,
	})
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateFolderName(c *gin.Context) {

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Unauthorized", "User not authenticated", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	folderID := c.Param("id")
	if !utility.IsValidUUID(folderID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid folder ID format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userID := userClaims["user_id"].(string)
	orgID := userClaims["org_id"].(string)

	var req models.RenameFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	validationErr := base.Validator.Struct(&req)
	if validationErr != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(validationErr, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	trimmed, err := Validate(req.FolderName)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Validation failed", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	folder, err := services.UpdateFolder(base.Db.Postgresql, models.UpdateFolderParams{
		FolderID: folderID,
		Name:     trimmed,
		OrgID:    orgID,
		UserID:   userID,
	})

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to update folder name", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Folder name updated successfully", folder)
	c.JSON(http.StatusOK, rd)
}
