package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type Organisation struct {
	ID                 string           `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name               string           `gorm:"type:varchar(255);not null" json:"name"`
	Description        string           `gorm:"type:text" json:"description"`
	Email              string           `gorm:"type:varchar(255);not null" json:"email"`
	Type               string           `gorm:"type:varchar(255)" json:"type"`
	Location           string           `gorm:"type:varchar(255)" json:"location"`
	Country            string           `gorm:"type:varchar(255)" json:"country"`
	OwnerID            string           `gorm:"type:uuid;" json:"owner_id"`
	LogoURL            string           `gorm:"type:varchar(255)" json:"logo_url"`
	CreditBalance      float64          `gorm:"type:decimal(10,2);default:0" json:"credit_balance"`
	ChannelssCount     int64            `gorm:"-" json:"channels_count"`
	TotalMessagesCount int64            `gorm:"-" json:"total_messages_count"`
	OrgRoles           []OrgRole        `gorm:"foreignKey:OrganisationID" json:"org_roles"`
	Users              []User           `gorm:"many2many:user_organisations;foreignKey:ID;joinForeignKey:organisation_id;References:ID;joinReferences:user_id"`
	CreatedAt          time.Time        `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time        `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt   `gorm:"index" json:"-"`
	Channels           []Channels       `gorm:"foreignKey:OrganisationID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"channels"`
	SubscriptionPlanId string           `gorm:"column:subscription_plan_id; type:varchar(255); default:'free'" json:"subscription_plan_id"`
	StripeCustomerID   string           `gorm:"column:stripe_customer_id; type:varchar(255)" json:"stripe_customer_id"`
	OrgPlanID          string           `gorm:"type:varchar(100);null;index" json:"org_plan_id"`
	OrganisationPlan   OrganisationPlan `gorm:"foreignKey:OrganisationID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"organisation_plan"`
}

type CreateOrgRequestModel struct {
	Name        string `json:"name" validate:"required,min=2,max=255"`
	Description string `json:"description" `
	Email       string `json:"email"`
	Type        string `json:"type" validate:"required"`
	Location    string `json:"location"`
	Country     string `json:"country" validate:"required"`
	LogoURL     string `json:"logo_url" `
	Plan        string `json:"plan"`
}

type UpdateOrgRequestModel struct {
	Name        string `json:"name"`
	Description string `json:"description" `
	Email       string `json:"email"`
	Type        string `json:"type"`
	Location    string `json:"location"`
	Country     string `json:"country"`
	LogoURL     string `json:"logo_url"`
}

type UserInOrgResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	AvatarURL   string    `json:"profile_url"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	EntityType  string    `json:"entity_type"` // "user" or "bot"
}

type OrgMetricsResponse struct {
	OrgUserInfo string   `json:"org_user_info"`
	OrgName     string   `json:"organisation_name"`
	UsersPhotos []string `json:"users_photos"`
}

type AddUserToOrgRequestModel struct {
	UserId string `json:"user_id" validate:"required"`
}

type ChannelResp []struct {
	Channels
	ThreadCount int64 `json:"thread_count"`
}

func (c *Organisation) CreateOrganisation(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &c)
	if err != nil {
		return err
	}
	return nil
}

func (c *Organisation) Delete(db *gorm.DB, orgId string) error {

	c.ID = orgId

	if err := postgresql.DeleteRecordWithNoModel(db,
		"DELETE FROM user_organisations WHERE organisation_id = ?",
		orgId); err != nil {
		return fmt.Errorf("failed to remove user_organisation mapping: %v", err)
	}

	err := postgresql.DeleteSpecificRecord(db, &OrgUserManagement{}, "organisation_id = ?", orgId)
	if err != nil {
		return errors.New("failed to remove organisation-user management mapping")
	}

	err = postgresql.DeleteRecordFromDb(db, c)
	if err != nil {
		return errors.New("failed to delete organisation")
	}
	return nil
}

func (c *Organisation) Update(db *gorm.DB) (*Organisation, error) {
	result, err := postgresql.SaveAllFields(db, &c)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("failed to update organisation")
	}

	return c, nil
}

func (c *Organisation) UpdateFeilds(db *gorm.DB, updateCon UpdateOrgRequestModel) (*Organisation, error) {

	update := make(map[string]interface{})

	data, _ := json.Marshal(updateCon)
	json.Unmarshal(data, &update)

	// Loop through update map and remove empty entries
	for key, value := range update {
		if value == "" {
			delete(update, key)
		}
	}

	result, err := postgresql.UpdateFields(db, &c, update, "id = ?", c.ID)
	if err != nil {
		return c, err
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("failed to update organisation")
	}

	return c, nil
}

func (o *Organisation) GetOrgByID(db *gorm.DB, orgID string) (Organisation, error) {
	var org Organisation

	exists := postgresql.CheckExists(db, &org, "id = ?", orgID)
	if !exists {
		return org, errors.New("organisation not found")
	}

	query := db.Where("id = ?", orgID)
	query = postgresql.PreloadEntities(query, &org, "OrgRoles", "OrgRoles.Permissions")

	if err := query.First(&org).Error; err != nil {
		return org, err
	}

	channelsCount, err := o.CountOrganisationChannelss(db, orgID)
	if err != nil {
		return org, err
	}

	org.ChannelssCount = channelsCount

	return org, nil
}

func (o *Organisation) GetOrgByEmail(db *gorm.DB, orgEmail string) (Organisation, error) {
	var org Organisation

	query := db.Where("email = ?", orgEmail)
	if err := query.First(&org).Error; err != nil {
		return org, err
	}

	return org, nil
}

func (o *Organisation) GetAllChannelssInOrganisation(db *gorm.DB, orgID string) (ChannelResp, error) {
	var (
		channels Channels
		thread   Threads
		chanResp ChannelResp
	)

	exists := postgresql.CheckExists(db, &o, "id = ?", orgID)
	if !exists {
		return chanResp, errors.New("organisation does not exist")
	}

	threadCountSubquery := db.Model(&thread).Select("count(*)").
		Where("threads.channels_id = channels.id").
		Where("threads.type = 'thread'")

	if err := db.Model(&channels).
		Select("channels.id, channels.name, channels.organisation_id, (?) AS thread_count",
			threadCountSubquery).
		Where("channels.organisation_id = ?", orgID).
		Scan(&chanResp).Error; err != nil {
		return nil, errors.New("error fetching channels")
	}

	return chanResp, nil
}

func (u *Organisation) GetOrganisationsByUserID(db *gorm.DB, userID string) ([]Organisation, error) {
	var (
		ErrNotFound   = errors.New("user not found")
		organisations = []Organisation{}
	)

	query := db.Model(&Organisation{}).
		Joins("INNER JOIN user_organisations uo ON organisations.id = uo.organisation_id").
		Where("uo.user_id = ?", userID)

	if err := query.Find(&organisations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return organisations, ErrNotFound
		}
		return organisations, err
	}
	if len(organisations) == 0 {
		return organisations, ErrNotFound
	}

	return organisations, nil
}

func (o *Organisation) GetUserOrganisations(db *gorm.DB, userID string) ([]Organisation, error) {
	var (
		orgs []Organisation
	)

	err := db.Table("organisations AS org").
		Select("org.*").
		Joins("JOIN org_user_managements AS oum ON org.id = oum.organisation_id").
		Where("oum.user_id = ?", userID).
		Find(&orgs).Error

	if err != nil {
		return orgs, err
	}
	return orgs, nil
}

func (u *Organisation) GetOrganisationsByUserIDs(db *gorm.DB, userID, requesterID string) ([]Organisation, error) {

	var (
		ErrNotFound   = errors.New("user not in your organisation")
		organisations = []Organisation{}
	)

	var isOwner bool
	err := db.Model(&Organisation{}).
		Select("count(*) > 0").
		Where("owner_id = ?", requesterID).
		Find(&isOwner).
		Error
	if err != nil {
		return nil, err
	}

	if isOwner {

		query := db.Model(&Organisation{}).
			Joins("INNER JOIN user_organisations uo ON organisations.id = uo.organisation_id").
			Where("uo.user_id = ?", userID).
			Where("organisations.owner_id = ?", requesterID)
		if err := query.Find(&organisations).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {

				return organisations, ErrNotFound
			}
			return organisations, err
		}
		if len(organisations) == 0 {
			return organisations, ErrNotFound
		}
		return organisations, nil
	}

	query := db.Model(&Organisation{}).
		Joins("INNER JOIN user_organisations uo ON organisations.id = uo.organisation_id").
		Where("uo.user_id = ?", requesterID)
	if err := query.Find(&organisations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return organisations, ErrNotFound
		}
		return organisations, err
	}

	return organisations, nil
}

func (o *Organisation) GetUsersAndBotsInOrganisation(c *gin.Context, db *gorm.DB, orgId string) ([]UserInOrgResponse, postgresql.PaginationResponse, error) {
	var users []UserInOrgResponse
	pagination := postgresql.GetPagination(c)

	offset := (pagination.Page - 1) * pagination.Limit

	if err := db.Table("users").
		Select("users.id, users.email, profiles.phone as phone, profiles.full_name as name, profiles.avatar_url as avatar_url, users.created_at ").
		Joins("JOIN user_organisations ON user_organisations.user_id = users.id").
		Joins("JOIN profiles ON profiles.userid = users.id").
		Where("user_organisations.organisation_id = ?", orgId).
		Offset(offset).
		Limit(pagination.Limit).
		Find(&users).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	var totalUsers int64
	if err := db.Table("users").
		Joins("JOIN user_organisations ON user_organisations.user_id = users.id").
		Joins("JOIN profiles ON profiles.userid = users.id").
		Where("user_organisations.organisation_id = ?", orgId).
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

func (o *Organisation) CheckOrgExists(orgId string, db *gorm.DB) (Organisation, error) {
	var (
		org Organisation
	)

	exists := postgresql.CheckExists(db, &o, "id = ?", orgId)
	if !exists {
		return org, errors.New("organisation not found")
	}

	err, _ := postgresql.SelectOneFromDb(db, &org, "id = ?", orgId)
	if err != nil {
		return org, err
	}

	return org, nil
}

func (o *Organisation) CheckUserIsMemberOfOrg(userId string, orgId string, db *gorm.DB) (bool, error) {

	exist := postgresql.CheckExistsInTable(db,
		"user_organisations",
		"user_id = ? AND organisation_id = ?",
		userId,
		orgId,
	)

	if !exist {
		return false, errors.New("user not a member of organisation")
	}

	return true, nil
}

func (o *Organisation) IsOwnerOfOrganisation(db *gorm.DB, requesterID, organisationID string) (bool, error) {
	org, err := o.GetOrgByID(db, organisationID)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve organisation: %v", err)
	}

	if org.OwnerID != requesterID {
		return false, errors.New("requester is not the owner of the organisation")
	}

	return true, nil
}

func (o *Organisation) CountOrganisationChannelss(db *gorm.DB, orgId string) (int64, error) {
	var rs []Channels

	err := postgresql.SelectAllFromDb(db, "", &rs, "organisation_id = ?", orgId)
	if err != nil {
		return 0, errors.New("error counting channels in an organisation")
	}
	return int64(len(rs)), nil
}

func (o *Organisation) GetOrganisationInvites(c *gin.Context, db *gorm.DB, userID, orgID string) ([]Invitation, postgresql.PaginationResponse, error) {
	var invitations []Invitation

	exists := postgresql.CheckExists(db, o, "id = ?", orgID)
	if !exists {
		return invitations, postgresql.PaginationResponse{}, errors.New("organisation not found")
	}

	exists = postgresql.CheckExists(db, &User{}, "id = ?", userID)
	if !exists {
		return invitations, postgresql.PaginationResponse{}, errors.New("user not found")
	}

	pagination := postgresql.GetPagination(c)
	query := db.Table("invitations AS i").
		Select("i.id, i.email, i.status, org_roles.name AS role, i.organisation_id, i.is_telex_user, i.created_at, i.expires_at").
		Joins("JOIN org_roles ON org_roles.id = i.role::uuid").
		Where("i.organisation_id = ?", orgID).
		Order("i.created_at DESC").
		Offset((pagination.Page - 1) * pagination.Limit).
		Limit(pagination.Limit)

	if err := query.Find(&invitations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return invitations, postgresql.PaginationResponse{}, errors.New("invitations not found")
		}
		return invitations, postgresql.PaginationResponse{}, err
	}

	var totalInvites int64
	if err := db.Table("invitations AS i").
		Where("i.organisation_id = ?", orgID).
		Count(&totalInvites).Error; err != nil {
		return invitations, postgresql.PaginationResponse{}, err
	}

	totalPages := int(math.Ceil(float64(totalInvites) / float64(pagination.Limit)))
	paginationResponse := postgresql.PaginationResponse{
		CurrentPage:     pagination.Page,
		PageCount:       pagination.Limit,
		TotalPagesCount: totalPages,
	}
	return invitations, paginationResponse, nil
}

func (o *Organisation) GetOrganisationDetails(db *gorm.DB, orgID string) (Organisation, error) {
	var org Organisation

	err := db.Where("id = ?", orgID).First(&org).Error
	if err != nil {
		return org, err
	}

	channelsCount, err := o.CountOrganisationChannelss(db, orgID)
	if err != nil {
		return org, err
	}

	org.ChannelssCount = channelsCount

	return org, nil
}

func (o *Organisation) FetchUsersInOrgProfile(db *gorm.DB, orgID string) ([]Profile, error) {
	var (
		org      Organisation
		profiles []Profile
	)
	exists := postgresql.CheckExists(db, &org, "id = ?", orgID)
	if !exists {
		return profiles, errors.New("organisation not found")
	}

	query := db.Table("profiles").
		Select("profiles.*").
		Joins("JOIN org_user_managements ON org_user_managements.user_id = profiles.userid").
		Where("org_user_managements.organisation_id = ?", orgID).
		Order("profiles.created_at DESC")

	if err := query.Find(&profiles).Error; err != nil {
		return profiles, fmt.Errorf("failed to fetch user profiles: %w", err)
	}

	o.Name = org.Name

	return profiles, nil
}

func (o *Organisation) AddSystemAgentstoOrg(db *gorm.DB) error {

	// this section creates default integration
	var orgIntResp []OrganisationIntegrations

	err := db.Model(&Integrations{}).
		Select("gen_random_uuid() AS id, id as integration_id,? as org_id, name as app_name, app_description, app_url, app_logo,json_url,false as is_active, true as is_system, NOW() as created_at, NOW() as updated_at", o.ID).
		Scan(&orgIntResp).Error

	if err != nil {
		return err
	}

	if len(orgIntResp) == 0 {
		return nil
	}

	for i := range orgIntResp {
		key, err := GenerateAgentKey()
		if err != nil {
			return err
		}
		orgIntResp[i].PreSharedKey = key
	}

	err = postgresql.CreateMultipleRecords(db, &orgIntResp, len(orgIntResp))

	if err != nil {
		return err
	}
	return nil
}

func (o *Organisation) FetchOrgUsers(db *gorm.DB, orgID string) ([]OrgUsersProfile, error) {
    var ouprof []OrgUsersProfile

    err := db.Table("org_user_managements AS oum").
        Select("profiles.full_name as name, profiles.avatar_url").
        Joins("JOIN users ON users.id = oum.user_id").
        Joins("JOIN profiles ON profiles.userid = users.id").
        Where("oum.organisation_id = ?", orgID).
        Limit(30).
        Find(&ouprof).Error

    if err != nil {
        return ouprof, err
    }

    for i := range ouprof {
        ouprof[i].IsOnline = true
    }

    return ouprof, nil
}

func (o *Organisation) FetchOrgChannelsPlusFirst3Members(db *storage.Database, orgID string) ([]OrgChannelsWithMemberAvatars, error) {
	var channels []struct {
		ID   string
		Name string
	}

	if err := db.Postgresql.Table("channels").
		Select("id, name").
		Where("organisation_id = ?", orgID).
		Find(&channels).Error; err != nil {
		return nil, err
	}

	var result []OrgChannelsWithMemberAvatars

	for _, ch := range channels {
		var avatars []string
		if err := db.Postgresql.Table("user_channels").
			Select("profiles.avatar_url").
			Joins("JOIN profiles ON profiles.userid = user_channels.user_id").
			Where("user_channels.channels_id = ? AND profiles.avatar_url != ''", ch.ID).
			Limit(3).
			Pluck("profiles.avatar_url", &avatars).Error; err != nil {
			return nil, err
		}

		for len(avatars) < 3 {
			avatars = append(avatars, "")
		}

		var totalMembers int64
		if err := db.Postgresql.Table("user_channels").
			Where("channels_id = ?", ch.ID).
			Count(&totalMembers).Error; err != nil {
			return nil, err
		}

		membersLeft := int(totalMembers) - 3
		if membersLeft < 0 {
			membersLeft = 0
		}

		var lastPostString string
		lastPostTime, err := FetchLastMessageTime(db, ch.ID)
		if err != nil {
			lastPostString = "Last post unavailable"
		} else if lastPostTime.IsZero() {
			lastPostString = "No posts yet"
		} else {
			duration := time.Since(lastPostTime)
			if duration < time.Minute {
				lastPostString = "Last post a few seconds ago"
			} else {
				lastPostString = fmt.Sprintf("Last post %s", lastPostTime.Format("Jan 2 15:04"))
			}
		}

		result = append(result, OrgChannelsWithMemberAvatars{
			Name:          ch.Name,
			MemberAvatars: avatars,
			MembersCount:  membersLeft,
			LastPostTime:  lastPostString,
		})
	}

	return result, nil
}

func FetchLastMessageTime(db *storage.Database, channelID string) (time.Time, error) {
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"channels_id.keyword": channelID, // channelID is your input
						},
					},
					{
						"term": map[string]interface{}{
							"type.keyword": "thread", // if you want only threads
						},
					},
				},
			},
		},
		"aggs": map[string]interface{}{
			"last_message": map[string]interface{}{
				"max": map[string]interface{}{
					"field": "created_at",
				},
			},
		},
	}

	var lastMsgResult any
	err := elastic.SelectAll(db.Elastic, ThreadIndexName, query, &lastMsgResult)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to fetch last message time: %w", err)
	}

	resultMap, ok := lastMsgResult.(map[string]any)
	if !ok {
		return time.Time{}, fmt.Errorf("failed to parse last message time result")
	}

	agg := resultMap["aggregations"].(map[string]interface{})["last_message"].(map[string]interface{})
	lastMsgEpoch, ok := agg["value"].(float64)
	if !ok || lastMsgEpoch == 0 {
		return time.Time{}, nil
	}

	lastMsgTime := time.UnixMilli(int64(lastMsgEpoch))

	return lastMsgTime, nil
}
