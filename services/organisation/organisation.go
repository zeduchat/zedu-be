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
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func ValidateCreateOrgRequest(req models.CreateOrgRequestModel, db *gorm.DB) (models.CreateOrgRequestModel, int, error) {

	if req.Email != "" {
		req.Email = strings.ToLower(req.Email)
		formattedMail, checkBool := utility.EmailValid(req.Email)
		if !checkBool {
			return req, http.StatusUnprocessableEntity, fmt.Errorf("email address is invalid")
		}
		req.Email = formattedMail

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

	// Check if the organisation name already exists
	exists := postgresql.CheckExists(db, &models.Organisation{}, "name = ? AND owner_id = ?", org.Name, userId)
	if exists {
		return nil, errors.New("organisation already exists with the given name")
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
		Name:           "general",
		Description:    fmt.Sprintf("%s's general channel", org.Name),
		OwnerId:        user.ID,
		OrganisationID: org.ID,
	}

	var joinChannelsReq models.JoinChannelsRequest

	joinChannelsReq.ChannelsID = channel.ID
	joinChannelsReq.UserID = userId
	joinChannelsReq.Username = user.Profile.UserName

	err = channel.CreateChannel(db)
	if err != nil {
		return nil, err
	}

	_, err = channel.AddUserToChannel(db, joinChannelsReq)
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

	slug := channel.ID
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

	// this section creates default integration
	var orgIntResp []models.OrganisationIntegrations

	err = db.Model(&models.Integrations{}).
		Select("gen_random_uuid() AS id, id as integration_id,? as org_id, json_url,false as is_active, true as is_system, NOW() as created_at, NOW() as updated_at", org.ID).
		Scan(&orgIntResp).Error

	if err != nil {
		return nil, err
	}

	err = postgresql.CreateMultipleRecords(db, &orgIntResp, len(orgIntResp))

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

func UpdateOrganisation(orgId string, userId string, updateReq models.UpdateOrgRequestModel, db *gorm.DB, logger *utility.Logger) (*models.Organisation, int, error) {
	var org models.Organisation
	org, err := org.CheckOrgExists(orgId, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, errors.New("organisation not found")
		}
		return nil, http.StatusBadRequest, err
	}

	isMember, err := org.CheckUserIsMemberOfOrg(userId, orgId, db)
	if err != nil {
		return nil, http.StatusForbidden, err
	}
	if !isMember {
		return nil, http.StatusForbidden, errors.New("user not authorised to update this organisation")
	}

	if updateReq.Email != "" && updateReq.Email != org.Email {
		updateReq.Email = strings.ToLower(updateReq.Email)
		formattedMail, checkBool := utility.EmailValid(updateReq.Email)
		if !checkBool {
			return nil, http.StatusUnprocessableEntity, errors.New("email address is invalid")
		}
		updateReq.Email = formattedMail
		exists := postgresql.CheckExists(db, &org, "email = ?", updateReq.Email)
		if exists {
			return nil, http.StatusBadRequest, errors.New("organisation already exists with the given email")
		}
	}
	file, ext, err := utility.ValidatePicture(updateReq.LogoURL)

	if err != nil {
		return nil, http.StatusUnprocessableEntity, errors.New("failed to validate organisation logo")
	}

	picUrl, err := UploadOrganisationLogo(logger, orgId, file, ext)

	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to upload organisation logo")
	}

	if picUrl != "" {
		updateReq.LogoURL = picUrl
	}

	res, err := org.UpdateFeilds(db, updateReq)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to update organisation details")
	}

	return res, http.StatusOK, nil
}

func DeleteOrganisation(orgId string, userId string, db *gorm.DB) error {
	var (
		org models.Organisation
		ch  models.Channels
		oi  models.OrganisationIntegrations
	)

	isOwner, err := org.IsOwnerOfOrganisation(db, userId, orgId) //already checks if org exists
	if err != nil {
		return err
	}
	if !isOwner {
		return errors.New("user not authorised to delete this organisation")
	}

	//remove all channels in that organisation
	err = postgresql.DeleteSpecificRecord(db, &ch, "organisation_id = ?", orgId)
	if err != nil {
		return errors.New("unable to delete organisation channels")
	}

	//remove all organisation-integrations mapping
	err = postgresql.DeleteSpecificRecord(db, &oi, "org_id = ?", orgId)
	if err != nil {
		return errors.New("unable to remove organisation integrations")
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

func GetUsersAndBotsInOrganisation(orgId, userId string, db *gorm.DB, c *gin.Context) ([]models.UserInOrgResponse, postgresql.PaginationResponse, error) {
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

	users, paginationResponse, err := fetchUsersWithOrgManagement(orgId, userId, db, c)
	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	return users, paginationResponse, nil
}

func fetchUsersWithOrgManagement(orgId, userId string, db *gorm.DB, c *gin.Context) ([]models.UserInOrgResponse, postgresql.PaginationResponse, error) {
    var (
        users []models.UserInOrgResponse
        pagination = postgresql.GetPagination(c)
        offset = (pagination.Page - 1) * pagination.Limit
    )

    // Query for both users and agents
    query := db.Table("(?) AS combined", 
        db.Raw(`
            (SELECT 
                u.id, 
                u.email, 
                p.phone AS phone_number,
                COALESCE(NULLIF(p.user_name, ''),
                         NULLIF(p.full_name, ''),
                         SUBSTRING(u.email FROM 1 FOR POSITION('@' IN u.email) - 1)) AS name,
                p.avatar_url AS avatar_url, 
                u.created_at, 
                o.status,
                org.name AS role,
                'user' AS entity_type
            FROM org_user_managements o
            JOIN users u ON u.id = o.user_id
            LEFT JOIN profiles p ON p.userid = u.id
            LEFT JOIN org_roles org ON org.id = o.role_id::uuid
            WHERE o.organisation_id = ?)
            
            UNION ALL
            
            (SELECT 
                oi.integration_id AS id,
                'agent' AS email,
                '' AS phone_number,
                oi.app_name AS name,
                oi.app_logo AS avatar_url,
                oi.created_at,
                CASE 
                    WHEN oi.is_active THEN 'active'
                    ELSE 'inactive'
                END AS status,
                'bot' AS role,
                'bot' AS entity_type
            FROM organisation_integrations oi
            WHERE oi.org_id = ? AND oi.is_archived = false)
        `, orgId, orgId)).
        Order("created_at DESC").
        Offset(offset).
        Limit(pagination.Limit)

    // Execute the combined query
    if err := query.Find(&users).Error; err != nil {
        return nil, postgresql.PaginationResponse{}, fmt.Errorf("failed to fetch users and bots: %w", err)
    }

    // Get total count of users and bots
    var totalCount int64
    if err := db.Table("(?) as count_table", 
        db.Raw(`
            (SELECT user_id FROM org_user_managements WHERE organisation_id = ?)
            UNION ALL
            (SELECT integration_id FROM organisation_integrations 
             WHERE org_id = ? AND is_archived = false)
        `, orgId, orgId)).
        Count(&totalCount).Error; err != nil {
        return nil, postgresql.PaginationResponse{}, fmt.Errorf("failed to count users and bots: %w", err)
    }

    totalPages := int(math.Ceil(float64(totalCount) / float64(pagination.Limit)))
    paginationResponse := postgresql.PaginationResponse{
        CurrentPage:     pagination.Page,
        PageCount:       pagination.Limit,
        TotalPagesCount: totalPages,
        // TotalItems:      int(totalCount),
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

	err = orgmgt.AddUserToOrganisation(db)

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
