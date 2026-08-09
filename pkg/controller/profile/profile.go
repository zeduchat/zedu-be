package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/profile"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetUserProfile(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(400, "error", "Failed to Fetch user profile", "No user found", nil)
		c.JSON(400, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	orgId, _ := userClaims["org_id"].(string)

	targetUserID := userId
	memberID := c.Param("user_id")
	if memberID != "" {
		code, err := profile.IsSameOrganization(base.Db.Postgresql, orgId, userId, memberID)
		if err != nil {
			rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
			c.JSON(code, rd)
			return
		}
		targetUserID = memberID
	}

	userProfile, code, err := profile.GetUserProfile(base.Db.Postgresql, targetUserID, orgId)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "Failed to Fetch user profile", err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User profile retrieved successfully", userProfile)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ChangeProfileStatus(c *gin.Context) {

	var req models.UpdateProfileStatus

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	req.UserId = userId
	req.OrgId = userClaims["org_id"].(string)

	status, code, err := profile.UpdateProfileStatusWithJobScheduling(req, base.Db, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "Failed to update user profile", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User profile status updated successfully", status)
	c.JSON(code, rd)
}

func (base *Controller) UpdateProfile(c *gin.Context) {
	var req models.UpdateUserProfileRequest
	var fileBase64Img string
	var file []byte
	var ext string

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.Email = c.Request.FormValue("email")
	req.UserName = c.Request.FormValue("username")
	req.FullName = c.Request.FormValue("full_name")
	req.Phone = c.Request.FormValue("phone")
	req.DisplayName = c.Request.FormValue("display_name")
	req.Timezone = c.Request.FormValue("timezone")
	req.Title = c.Request.FormValue("title")
	req.NamePronunciation = c.Request.FormValue("name_pronounciation")
	req.WorkspaceID = c.Request.FormValue("workspace_id")
	req.Track = c.Request.FormValue("track")
	links := c.Request.Form["links"]
	if len(links) > 0 {
		req.Links = links
	}

	if req.FullName != "" && (len(req.FullName) < 2 || len(req.FullName) > 100) {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Name must be between 2 and 100 characters", nil, nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if req.WorkspaceID != "" {
		if len(req.WorkspaceID) > 100 {
			rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Workspace ID too long", nil, nil)
			c.JSON(http.StatusUnprocessableEntity, rd)
			return
		}

		// allow alphanum, spaces, dashes, max 100 chars for workspaceId
		for _, r := range req.WorkspaceID {
			if !(r == '-' || r == ' ' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Workspace ID contains invalid characters", nil, nil)
				c.JSON(http.StatusUnprocessableEntity, rd)
				return
			}
		}
	}

	if len(req.Links) > 0 {
		for _, link := range req.Links {
			if !utility.IsValidURL(link) {
				rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Invalid link URL: "+link, nil, nil)
				c.JSON(http.StatusUnprocessableEntity, rd)
				return
			}
		}
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	err = c.Request.ParseMultipartForm(5 << 20)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Parse form error: %v", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base64Image := c.Request.FormValue("avatar_url")
	if base64Image != "" {
		fileBase64Img = base64Image
	}

	_, imgFileHeader, err := c.Request.FormFile("avatar_file")
	if err != nil && err != http.ErrMissingFile {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to parse avatar file", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if imgFileHeader != nil && imgFileHeader.Filename != "" {
		base64Image, err := utility.ConvertToBase64(imgFileHeader)
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		fileBase64Img = base64Image
	}

	if fileBase64Img != "" {
		validFile, validExt, err := utility.ValidatePicture(fileBase64Img)
		if err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		file = validFile
		ext = validExt
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	orgId, _ := userClaims["org_id"].(string)

	code, data, err := profile.UpdateUserProfile(req, base.Db.Postgresql, base.Logger, userId, ext, file, orgId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Profile not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(code, "error", "Failed to update Profile", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	base.Logger.Info("Profile updated successfully")
	rd := utility.BuildSuccessResponse(code, "Profile updated successfully", data)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteUserProfileImage(c *gin.Context) {

	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	orgId, _ := userClaims["org_id"].(string)

	code, err := profile.DeleteUserProfileImage(base.Db.Postgresql, base.Logger, userId, orgId)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Profile not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(code, "error", "Failed to update Profile", err, nil)
			c.JSON(code, rd)
		}
		return
	}

	base.Logger.Info("Profile image deleted successfully")
	rd := utility.BuildSuccessResponse(code, "Profile image deleted successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) ChangeProfilePresence(c *gin.Context) {
	var req models.UpdateUserPresenceRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("unable to get user claims", nil)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	orgId := userClaims["org_id"].(string)

	req.UserID = userId
	req.OrgID = orgId

	code, err := profile.UpdateUserPresence(req, base.Db.Postgresql, base.Logger)
	if err != nil {
		base.Logger.Error("Failed to update user presence", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to update user presence", err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("User presence updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User presence updated successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) GetProfilePresence(c *gin.Context) {
	userID := c.Param("user_id")
	orgID := ""
	if userID == "" {
		claims, exists := c.Get("userClaims")
		if !exists {
			base.Logger.Error("unable to get user claims", nil)
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		userClaims := claims.(jwt.MapClaims)
		userID = userClaims["user_id"].(string)
		orgID, _ = userClaims["org_id"].(string)
	}

	presence, code, err := profile.GetUserPresence(userID, orgID, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to fetch user presence", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch user presence", err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("User presence retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User presence retrieved successfully", gin.H{"online": presence})
	c.JSON(code, rd)
}
