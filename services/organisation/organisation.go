package organisation

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func ValidateCreateOrgRequest(req models.CreateOrgRequestModel, db *gorm.DB) (models.CreateOrgRequestModel, int, error) {

	org := models.Organisation{}

	if req.Email != "" {
		req.Email = strings.ToLower(req.Email)
		formattedMail, checkBool := utility.EmailValid(req.Email)
		if !checkBool {
			return req, http.StatusUnprocessableEntity, fmt.Errorf("email address is invalid")
		}
		req.Email = formattedMail

	}

	exists := postgresql.CheckExists(db, &org, "name = ?", strings.ToLower(req.Name))
	if exists {
		return req, http.StatusConflict, fmt.Errorf("organisation already exists with the given name")
	}

	return req, http.StatusOK, nil
}

func CreateOrganisation(req models.CreateOrgRequestModel, db *gorm.DB, userId string, logger *utility.Logger) (*models.Organisation, error) {

	orgId := utility.GenerateUUID()
	file, ext, err := utility.ValidatePicture(req.LogoURL)

	if err != nil {
		return nil, errors.New("failed to validate organisation logo")
	}

	picUrl, err := UploadOrganisationLogo(logger, orgId, file, ext)

	if err != nil {
		return nil, errors.New("failed to upload organisation logo")
	}

	org := models.Organisation{
		ID:          orgId,
		Name:        strings.ToLower(req.Name),
		Description: strings.ToLower(req.Description),
		Location:    strings.ToLower(req.Location),
		Email:       strings.ToLower(req.Email),
		Type:        strings.ToLower(req.Type),
		OwnerID:     userId,
		Country:     strings.ToLower(req.Country),
		LogoURL:     picUrl,
	}

	err = org.CreateOrganisation(db)

	if err != nil {
		return nil, err
	}

	var user models.User

	user, err = user.GetUserByID(db, userId)
	if err != nil {
		return nil, err
	}

	user.CurrentOrg, err = uuid.FromString(org.ID)
	if err != nil {
		return nil, err
	}

	err = user.Update(db)
	if err != nil {
		return nil, err
	}

	err = user.AddUserToOrganisation(db, &user, []interface{}{&org})
	if err != nil {
		return nil, err
	}

	channel := models.Channels{
		ID:             utility.GenerateUUID(),
		Name:           "Default",
		Description:    fmt.Sprintf("%s's default channel", org.Name),
		OwnerId:        user.ID,
		OrganisationID: org.ID,
	}

	var joinChannelsReq models.JoinChannelsRequest

	joinChannelsReq.ChannelsID = channel.ID
	joinChannelsReq.UserID = userId
	joinChannelsReq.Username = user.Profile.UserName

	err = channel.CreateChannels(db, storage.DB.TypeSense)
	if err != nil {
		return nil, err
	}

	_, err = channel.AddUserToChannels(db, joinChannelsReq)
	if err != nil {
		return nil, err
	}

	webhook := models.Webhook{
		ID:          utility.GenerateUUID(),
		ChannelId:   channel.ID,
		OwnerId:     userId,
		Status:      "active",
		WebhookName: fmt.Sprintf("%s's webhook", channel.Name),
	}

	slug := strings.Split(webhook.ID, "-")[4]
	webhookUrl := config.Config.App.WebhookApiUrl + fmt.Sprintf("/v1/webhooks/%s", slug)
	webhook.WebhookSlug = slug
	webhook.WebhookUrl = webhookUrl

	err = webhook.CreateWebhook(db)

	if err != nil {
		return nil, err
	}

	err = CreateOrgUserManagement(db, user.ID, org.ID)
	if err != nil {
		return nil, err
	}

	return &org, nil
}

func GetOrganisation(orgId string, userId string, db *gorm.DB) (*models.Organisation, error) {
	var org models.Organisation
	org, err := org.CheckOrgExists(orgId, db)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organisation not found")
		}
		return nil, err
	}

	isMember, err := org.CheckUserIsMemberOfOrg(userId, orgId, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user not authorised to retrieve this organisation")
	}

	return &org, nil
}

func GetAllChannelssInTeam(db *gorm.DB, orgID string) (models.ChannelResp, error) {
	var o models.Organisation

	channels, err := o.GetAllChannelssInOrganisation(db, orgID)
	if err != nil {
		return channels, err
	}

	return channels, nil
}

func UpdateOrganisation(orgId string, userId string, updateReq models.UpdateOrgRequestModel, db *gorm.DB, logger *utility.Logger) (*models.Organisation, error) {
	var org models.Organisation
	org, err := org.CheckOrgExists(orgId, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organisation not found")
		}
		return nil, err
	}

	isMember, err := org.CheckUserIsMemberOfOrg(userId, orgId, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user not authorised to update this organisation")
	}

	if updateReq.Email != "" && updateReq.Email != org.Email {
		updateReq.Email = strings.ToLower(updateReq.Email)
		formattedMail, checkBool := utility.EmailValid(updateReq.Email)
		if !checkBool {
			return nil, errors.New("email address is invalid")
		}
		updateReq.Email = formattedMail
		exists := postgresql.CheckExists(db, &org, "email = ?", updateReq.Email)
		if exists {
			return nil, errors.New("organisation already exists with the given email")
		}
	}
	file, ext, err := utility.ValidatePicture(updateReq.LogoURL)

	if err != nil {
		return nil, errors.New("failed to validate organisation logo")
	}

	picUrl, err := UploadOrganisationLogo(logger, orgId, file, ext)

	if err != nil {
		return nil, errors.New("failed to upload organisation logo")
	}

	updateReq.LogoURL = picUrl

	copier.Copy(&org, &updateReq)
	return org.Update(db)
}

func DeleteOrganisation(orgId string, userId string, db *gorm.DB) error {
	var org models.Organisation

	isOwner, err := org.IsOwnerOfOrganisation(db, userId, orgId)
	if err != nil {
		return err
	}
	if !isOwner {
		return errors.New("user not authorised to delete this organisation")
	}

	return org.Delete(db, orgId)
}

func AddUserToOrganisation(orgId string, req models.AddUserToOrgRequestModel, db *gorm.DB) error {
	var (
		org  models.Organisation
		user models.User
	)

	org, err := org.CheckOrgExists(orgId, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("organisation not found")
		}
		return err
	}

	user, err = user.GetUserByID(db, req.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	isMember, err := org.CheckUserIsMemberOfOrg(req.UserId, orgId, db)
	if err != nil {
		return err
	}
	if isMember {
		return errors.New("user already added to organisation")
	}

	err = user.AddUserToOrganisation(db, &user, []interface{}{&org})

	if err != nil {
		return err
	}

	return nil
}

func GetUsersInOrganisation(orgId, userId string, db *gorm.DB, c *gin.Context) ([]models.UserInOrgResponse, postgresql.PaginationResponse, error) {
	var org models.Organisation

	_, err := org.CheckOrgExists(orgId, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, postgresql.PaginationResponse{}, errors.New("organisation not found")
		}
		return nil, postgresql.PaginationResponse{}, err
	}

	isMember, err := org.CheckUserIsMemberOfOrg(userId, orgId, db)
	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}
	if !isMember {
		return nil, postgresql.PaginationResponse{}, errors.New("user does not have access to the organisation")
	}

	users, paginationResponse, err := fetchUsersWithOrgManagement(orgId, db, c)
	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	return users, paginationResponse, nil
}

func fetchUsersWithOrgManagement(orgId string, db *gorm.DB, c *gin.Context) ([]models.UserInOrgResponse, postgresql.PaginationResponse, error) {
	var users []models.UserInOrgResponse
	pagination := postgresql.GetPagination(c)
	offset := (pagination.Page - 1) * pagination.Limit

	if err := db.Table("users AS u").
		Select(`u.id, u.email, p.phone AS phone_number, p.full_name AS name, 
			p.avatar_url AS avatar_url, u.created_at, o.status, org.name AS role`).
		Joins("JOIN user_organisations AS uo ON uo.user_id = u.id").
		Joins("JOIN profiles AS p ON p.userid = u.id").
		Joins("JOIN org_user_managements AS o ON o.user_id = u.id AND o.organisation_id = ?", orgId).
		Joins("JOIN org_roles AS org ON org.id = o.role_id::uuid").
		Where("uo.organisation_id = ?", orgId).
		Offset(offset).
		Limit(pagination.Limit).
		Find(&users).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	var totalUsers int64
	if err := db.Table("users AS u").
		Joins("JOIN user_organisations AS uo ON uo.user_id = u.id").
		Where("uo.organisation_id = ?", orgId).
		Count(&totalUsers).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	totalPages := int(math.Ceil(float64(totalUsers) / float64(pagination.Limit)))
	paginationResponse := postgresql.PaginationResponse{
		CurrentPage:     pagination.Page,
		PageCount:       pagination.Limit,
		TotalPagesCount: totalPages,
	}

	return users, paginationResponse, nil
}

func RemoveMemberFromOrganisation(ownerId, orgId, userId string, db *gorm.DB) error {
	var (
		org    models.Organisation
		orgmgt models.OrgUserManagement
	)

	isowner, err := org.IsOwnerOfOrganisation(db, ownerId, orgId)
	if err != nil {
		return err
	}

	if !isowner {
		return errors.New("user is not the owner of the organisation")
	}

	err = orgmgt.RemoveMemberFromOrganisation(db, orgId, userId)

	if err != nil {
		return err
	}

	return nil
}

func AddMemberToOrganisation(ownerId, orgId string, req models.OrgUserCreateRequest, db *gorm.DB) error {
	var (
		org    models.Organisation
		orgmgt models.OrgUserManagement
	)

	isowner, err := org.IsOwnerOfOrganisation(db, ownerId, orgId)
	if err != nil {
		return err
	}

	if !isowner {
		return errors.New("user is not the owner of the organisation")
	}

	orgmgt.RoleID = req.RoleID
	orgmgt.UserID = req.UserID
	orgmgt.OrganisationID = orgId
	orgmgt.Status = "active"

	err = orgmgt.AddUserToOrganisation(db, orgId, req.UserID)

	if err != nil {
		return err
	}

	return nil
}

func LoadOrganisationMetrics(orgId string, db *gorm.DB) (models.OrgMetricsResponse, error) {
	var (
		o   models.Organisation
		ogm models.OrgMetricsResponse
	)

	metrics, err := o.LoadOrganisationMetrics(db, orgId)
	if err != nil {
		return ogm, err
	}
	return metrics, nil
}

func UploadOrganisationLogo(logger *utility.Logger, uniqueId string, file []byte, ext string) (string, error) {
	if file != nil {
		logoId := strings.Split(uniqueId, "-")[4]
		filename := fmt.Sprintf("org_logo_%s%s", logoId, ext)

		picUrl, err := minio.UploadProfilePic(logger, filename, bytes.NewReader(file), int64(len(file)))
		if err != nil {
			return "", err
		}
		return picUrl, nil
	}
	return "", nil
}
