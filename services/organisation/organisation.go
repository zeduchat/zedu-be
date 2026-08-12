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
	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
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
		return nil, fmt.Errorf("failed to validate organisation logo, %v", err)
	}

	picUrl, err := UploadOrganisationLogo(logger, orgId, file, ext)

	if err != nil {
		return nil, fmt.Errorf("failed to upload organisation logo, %v", err)
	}

	plan := models.Plan{}
	planName := "Free"

	if req.Plan != "" {
		planName = req.Plan
	}

	err = db.Where("name = ?", planName).First(&plan).Error
	if err != nil {
		return nil, err
	}

	credits := plan.Credits

	if picUrl == "" {
		picUrl = req.LogoURL
	}

	org := models.Organisation{
		ID:            orgId,
		Name:          req.Name,
		Description:   strings.ToLower(req.Description),
		Location:      strings.ToLower(req.Location),
		Email:         strings.ToLower(req.Email),
		Type:          strings.ToLower(req.Type),
		OwnerID:       userId,
		CreditBalance: float64(credits),
		Country:       strings.ToLower(req.Country),
		LogoURL:       picUrl,
		OrgPlanID:     plan.ID,
	}

	err = org.CreateOrganisation(db)

	if err != nil {
		return nil, err
	}

	// create credit transaction
	credit_transaction := models.CreditTransaction{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgId,
		Amount:         float64(credits), // Initial Top up amout
		BalanceBefore:  0.00,
		BalanceAfter:   float64(org.CreditBalance),
		Type:           "Initial Top-up",
	}

	err = credit_transaction.CreateCreditTransaction(db)
	if err != nil {
		return nil, err
	}

	var user models.User

	user, err = user.GetUserByID(db, userId, org.ID)
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

	err = user.AddUserToOrganisation(db, &user, []any{&org})
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

	org.OrganisationSlug = slug.Make(req.Name)

	slugId := channel.ID
	webhookUrl := config.Config.App.WebhookApiUrl + fmt.Sprintf("/v1/webhooks/%s", slugId)
	webhook.WebhookSlug = slugId
	webhook.WebhookUrl = webhookUrl

	err = webhook.CreateWebhook(db)

	if err != nil {
		return nil, err
	}

	err = CreateOrgUserManagement(db, user.ID, org.ID)
	if err != nil {
		return nil, err
	}

	err = org.AddSystemAgentstoOrg(db, logger)
	if err != nil {
		logger.Error("Adding default agents to organisation failed, error: %v", err)
		return nil, err
	}

	orgResp, _ := org.GetOrgByID(db, orgId)
	orgResp.OrganisationSlug = slug.Make(orgResp.Name)

	return &orgResp, nil
}

func GetOrganisation(orgId string, userId string, db *gorm.DB) (*models.Organisation, error) {
	var org models.Organisation

	err := db.Preload("OrganisationPlan.PlanDetails").Where("id = ?", orgId).First(&org).Error
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

	if org.OrganisationPlan.ID == "" && org.OrgPlanID != "" {
		var plan models.Plan
		err = db.Where("id = ?", org.OrgPlanID).First(&plan).Error
		if err == nil {
			org.OrganisationPlan = models.OrganisationPlan{
				PlanDetails: plan,
			}
		}
	}

	if org.OrganisationPlan.PlanDetails.ID == "" {
		var freePlan models.Plan
		if err := db.Where("name = ?", "Free").First(&freePlan).Error; err == nil {
			org.OrganisationPlan = models.OrganisationPlan{
				OrganisationID: org.ID,
				PlanID:         freePlan.ID,
				Status:         "Active",
				StartedAt:      org.CreatedAt,
				PlanDetails:    freePlan,
			}
		}
	}

	var orgUserMgt models.OrgUserManagement
	userRoleInfo, err := orgUserMgt.GetUserRoleInOrganisation(db, userId, orgId)
	userRoleInfo.RoleName = strings.ToLower(userRoleInfo.RoleName)
	if err == nil && userRoleInfo.RoleName != "" {
		org.CurrentUserRoleInfo = userRoleInfo
	}

	org.OrganisationSlug = slug.Make(org.Name)

	// Check if the org has an active general invite link
	var generalInvite models.GeneralInvitation
	err = db.Where("organisation_id = ?", orgId).Order("created_at DESC").First(&generalInvite).Error
	if err == nil {
		if generalInvite.ActiveStatus {
			org.InviteLinkStatus = "enabled"
		} else {
			org.InviteLinkStatus = "disabled"
		}
	} else {
		org.InviteLinkStatus = "disabled"
	}

	return &org, nil
}

func GetAllChannelssInTeam(db *storage.Database, c *gin.Context, ids models.IDS) (models.GetUserChannelResp, postgresql.PaginationResponse, int, error) {
	var o models.Organisation

	channels, pag, code, err := o.GetAllChannelssInOrganisation(db, c, ids)
	if err != nil {
		return channels, pag, code, err
	}

	for i := range channels {
		parts := strings.Split(channels[i].ID, "-")
		lastPart := parts[len(parts)-1]
		channels[i].ChannelSlug = fmt.Sprintf("%s-%s", slug.Make(channels[i].Name), lastPart)
	}

	return channels, pag, code, nil
}

func UpdateOrganisation(orgId string, userId string, updateReq models.UpdateOrgRequestModel, db *gorm.DB, logger *utility.Logger) (*models.Organisation, int, error) {
	var org models.Organisation
	org, err := org.CheckOrgExists(orgId, db)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("organisation not found")
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
	)

	isOwner, err := org.IsOwnerOfOrganisation(db, userId, orgId)
	if err != nil {
		return err
	}
	if !isOwner {
		return errors.New("user not authorised to delete organisation")
	}

	org.ID = orgId
	err = org.ClearOrganisationResources(db)
	if err != nil {
		return err
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

	user, err = user.GetUserByID(db, req.UserId, orgId)
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

	err = user.AddUserToOrganisation(db, &user, []any{&org})

	if err != nil {
		return err
	}

	return nil
}

func GetUsersBotsInOrganisation(orgId, userId string, db *gorm.DB, c *gin.Context, queryParams map[string]any) ([]models.UserInOrgResponse, postgresql.PaginationResponse, error) {
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

	users, paginationResponse, err := fetchUsersBotsWithOrgManagement(orgId, userId, db, c, queryParams)
	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	return users, paginationResponse, nil
}

func fetchUsersBotsWithOrgManagement(orgId, userId string, db *gorm.DB, c *gin.Context, queryParams map[string]any) ([]models.UserInOrgResponse, postgresql.PaginationResponse, error) {
	var (
		queryBuilder strings.Builder
		args         = []any{}
		entities     []models.UserInOrgResponse
		pagination   = postgresql.GetPagination(c)
		offset       = (pagination.Page - 1) * pagination.Limit
		searchTerm   = strings.ToLower(queryParams["query"].(string))
		roleFilter   = strings.ToLower(queryParams["role"].(string))
		includeBots  = queryParams["include_bots"].(bool)
		likeTerm     = "%" + searchTerm + "%"
	)

	// --- User query ---
	queryBuilder.WriteString(`
		SELECT 
			u.id, 
			u.email, 
			p.phone AS phone_number,
			COALESCE(NULLIF(p.user_name, ''),
					 NULLIF(p.full_name, ''),
					 SUBSTRING(u.email FROM 1 FOR POSITION('@' IN u.email) - 1)) AS name,
			p.avatar_url AS avatar_url, 
			u.created_at, 
			CASE 
				WHEN o.is_deactivated THEN 'inactive'
				ELSE 'active'
			END AS status,
			org.name AS role,
			o.is_deactivated,
			'user' AS entity_type,
			p.online
		FROM org_user_managements o
		JOIN users u ON u.id = o.user_id
		LEFT JOIN profiles p ON p.userid = u.id AND (p.organisation_id IS NULL OR p.organisation_id = o.organisation_id)
		LEFT JOIN org_roles org ON org.id = o.role_id::uuid
		WHERE o.organisation_id = ?
		  AND (
			  u.name ILIKE ? OR 
			  u.email ILIKE ? OR 
			  p.user_name ILIKE ? OR 
			  p.phone ILIKE ? OR
			  p.first_name ILIKE ? OR
			  p.last_name ILIKE ?
		  )
	`)
	args = append(args, orgId, likeTerm, likeTerm, likeTerm, likeTerm, likeTerm, likeTerm)

	if roleFilter != "" {
		queryBuilder.WriteString(" AND org.name ILIKE ?")
		args = append(args, "%"+roleFilter+"%")
	}

	// --- Optionally add bot query ---
	if includeBots {
		queryBuilder.WriteString(`
			UNION ALL
			SELECT 
				oi.integration_id AS id,
				oi.app_name AS email,
				'' AS phone_number,
				oi.app_name AS name,
				oi.app_logo AS avatar_url,
				oi.created_at,
				CASE 
					WHEN oi.is_active THEN 'active'
					ELSE 'inactive'
				END AS status,
				'bot' AS role,
				false AS is_deactivated,
				'bot' AS entity_type,
				oi.is_active AS online
			FROM organisation_integrations oi
			WHERE oi.org_id = ? AND oi.is_archived = false
			  AND (oi.app_name ILIKE ?)
		`)
		args = append(args, orgId, likeTerm)
	}

	query := db.Table("(?) AS combined",
		db.Raw(queryBuilder.String(), args...)).
		Order("created_at DESC").
		Offset(offset).
		Limit(pagination.Limit)

	if err := query.Find(&entities).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, fmt.Errorf("failed to fetch org entities: %w", err)
	}

	for i := range entities {
		if entities[i].EntityType == "user" {
			entities[i].DefaultAvatarURL = avatar.GenerateDefaultAvatarURL(entities[i].ID)
		} else {
			entities[i].DefaultAvatarURL = entities[i].AvatarURL
		}
	}

	// Count total
	var totalCount int64
	countQuery := db.Table("(?) AS count_table", db.Raw(queryBuilder.String(), args...))
	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, fmt.Errorf("failed to count org entities: %w", err)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(pagination.Limit)))
	paginationResponse := postgresql.PaginationResponse{
		CurrentPage:     pagination.Page,
		PageCount:       pagination.Limit,
		TotalPagesCount: totalPages,
		TotalItems:      totalCount,
	}

	return entities, paginationResponse, nil
}

func RemoveMemberFromOrganisation(initiatingUserID, orgId, targetUserID string, db *storage.Database, logger *utility.Logger) error {
	var (
		org    models.Organisation
		orgmgt models.OrgUserManagement
	)

	isSelf := initiatingUserID == targetUserID
	isOwner, _ := org.IsOwnerOfOrganisation(db.Postgresql, initiatingUserID, orgId)
	canManage := isSelf || isOwner || userCanOrOwner(db.Postgresql, initiatingUserID, orgId, models.PermChangeUserOrgRole)

	if !canManage {
		return errors.New("user is not authorised to remove member from organisation")
	}

	err := orgmgt.RemoveMemberFromOrganisation(db, orgId, targetUserID, logger)
	if err != nil {
		return err
	}

	return nil
}

func AddMemberToOrganisation(ownerId, orgId string, req models.OrgUserCreateRequest, db *gorm.DB) (int, error) {
	var (
		org    models.Organisation
		orgmgt models.OrgUserManagement
	)

	isowner, err := org.IsOwnerOfOrganisation(db, ownerId, orgId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if !isowner {
		return http.StatusForbidden, errors.New("user is not the owner of the organisation")
	}

	orgmgt.RoleID = req.RoleID
	orgmgt.UserID = req.UserID
	orgmgt.OrganisationID = orgId
	orgmgt.Status = "active"

	code, err := orgmgt.AddUserToOrganisation(db)
	if err != nil {
		return code, err
	}

	return http.StatusOK, nil
}

func LoadOrganisationMetrics(orgId string, invitationToken string, db *gorm.DB) (models.OrgMetricsResponse, error) {
	var (
		o         models.Organisation
		ogm       models.OrgMetricsResponse
		userNames []string
		userInfo  string
		isNewUser bool
		userEmail string
	)

	if invitationToken != "" {
		var invite models.Invitation
		if err := db.Where("token = ?", invitationToken).First(&invite).Error; err == nil {
			userEmail = invite.Email
			var user models.User
			exists := postgresql.CheckExists(db, &user, "email = ?", invite.Email)
			isNewUser = !exists
		}
	}

	userPhotos := make([]string, 0, 5)

	profiles, err := o.FetchUsersInOrgProfile(db, orgId)
	if err != nil {
		return ogm, err
	}

	if len(profiles) == 0 {
		return models.OrgMetricsResponse{
			OrgUserInfo: "No members yet",
			OrgName:     o.Name,
			UsersPhotos: []string{},
			IsNewUser:   isNewUser,
			UserEmail:   userEmail,
		}, nil
	}

	for _, profile := range profiles {
		if profile.AvatarURL != "" {
			userPhotos = append(userPhotos, profile.AvatarURL)
		}
		if profile.FirstName != "" {
			userNames = append(userNames, profile.FirstName)
		}
	}

	if len(userPhotos) > 5 {
		userPhotos = userPhotos[:5]
	}

	switch len(userNames) {
	case 0:
		userInfo = "No named members yet"
	case 1:
		userInfo = fmt.Sprintf("%s is in this organisation", userNames[0])
	case 2:
		userInfo = fmt.Sprintf("%s and %s are in this organisation",
			userNames[0], userNames[1])
	default:
		userInfo = fmt.Sprintf("%s, %s and %d others are in this organisation",
			userNames[0], userNames[1], len(userNames)-2)
	}

	response := models.OrgMetricsResponse{
		OrgUserInfo: userInfo,
		OrgName:     o.Name,
		UsersPhotos: userPhotos,
		IsNewUser:   isNewUser,
		UserEmail:   userEmail,
	}

	return response, nil
}


func FetchGetStarted(db *storage.Database, ids models.IDS) (models.OrgGetStartedResponse, error) {
	var (
		profile models.Profile
		org     models.Organisation
	)

	err := profile.GetProfileByUserId(db.Postgresql, ids.UserID, ids.OrganisationID)
	if err != nil {
		return models.OrgGetStartedResponse{}, err
	}

	userProfiles, err := org.FetchOrgUsers(db.Postgresql, ids)
	if err != nil {
		return models.OrgGetStartedResponse{}, fmt.Errorf("failed to fetch user profiles: %v", err.Error())
	}

	orgChans, err := org.FetchOrgChannelsPlusFirst3Members(db, ids.OrganisationID)
	if err != nil {
		return models.OrgGetStartedResponse{}, fmt.Errorf("failed to fetch channels in organisation with member images: %s", err.Error())
	}

	ogsr := models.OrgGetStartedResponse{
		UserName:       profile.UserName,
		OrgUserProfile: userProfiles,
		OrgChannel:     orgChans,
	}

	return ogsr, nil
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

func GetAllOrganisations(db *gorm.DB, c *gin.Context) ([]models.Organisation, postgresql.PaginationResponse, error) {
	var org = models.Organisation{}

	organisations, paginationResult, err := org.GetAllOrganisations(db, c)

	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	return organisations, paginationResult, nil
}
