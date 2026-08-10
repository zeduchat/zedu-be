package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
)

type Profile struct {
	ID                string         `gorm:"type:uuid;primary_key" json:"profile_id"`
	FirstName         string         `gorm:"column:first_name; type:text; not null" json:"first_name"`
	LastName          string         `gorm:"column:last_name; type:text;not null" json:"last_name"`
	FullName          string         `gorm:"column:full_name; type:text;" json:"full_name"`
	UserName          string         `gorm:"column:user_name; type:text;" json:"username"`
	Phone             string         `gorm:"type:varchar(255)" json:"phone"`
	AvatarURL         string         `gorm:"type:varchar(255)" json:"avatar_url"`
	Userid            string         `gorm:"type:uuid;index:idx_profiles_userid;uniqueIndex:idx_user_org" json:"user_id"`
	OrganisationID    *string        `gorm:"type:uuid;null;index;uniqueIndex:idx_user_org" json:"organisation_id"`
	CreatedAt         time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	DisplayName       string         `gorm:"type:varchar(255)" json:"display_name"`
	Title             string         `gorm:"type:varchar(255)" json:"title"`
	NamePronunciation string         `gorm:"type:varchar(255)" json:"name_pronunciation"`
	Timezone          string         `gorm:"type:varchar(255)" json:"timezone"`
	Icon              string         `gorm:"type:varchar(255)" json:"icon"`
	Text              string         `gorm:"type:varchar(255)" json:"text"`
	PauseNotification bool           `gorm:"type:boolean;default:false" json:"pause_notification"`
	StatusTimeout     string         `gorm:"type:varchar(255)" json:"status_timeout"`
	StatusVisibility  string         `gorm:"type:varchar(255);default:'public'" json:"status_visibility"`
	RiverJobID        *int64         `gorm:"type:bigint;index" json:"river_job_id,omitempty"`
	WorkspaceID       string         `gorm:"type:varchar(255)" json:"workspace_id"`
	Track             string         `gorm:"type:varchar(255)" json:"track"`
	Links             pq.StringArray `gorm:"type:text[]" json:"links"`
	Online            bool           `gorm:"type:boolean;default:true" json:"online"`
	IsActive          bool           `gorm:"type:boolean;default:true" json:"is_active"`
	IsDeactivated     bool           `gorm:"column:is_deactivated;type:boolean;default:false" json:"is_deactivated"`
}

type ProfileSummary struct {
	ID                string   `json:"id"`
	Email             string   `json:"email"`
	Phone             string   `json:"phone"`
	FirstName         string   `json:"first_name"`
	LastName          string   `json:"last_name"`
	FullName          string   `json:"full_name"`
	UserName          string   `json:"username"`
	AvatarURL         string   `json:"avatar_url"`
	DefaultAvatarURL  string   `json:"default_avatar_url"`
	UserId            string   `json:"user_id"`
	OrganisationID    string   `json:"organisation_id"`
	Deactivated       bool     `json:"deactivated"`
	IsDeactivated     bool     `json:"is_deactivated"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	DeletedAt         string   `json:"deleted_at"`
	ProfileUpdated    bool     `json:"profile_updated"`
	IsOnboarded       bool     `json:"is_onboarded"`
	DisplayName       string   `json:"display_name"`
	Title             string   `json:"title"`
	NamePronunciation string   `json:"name_pronounciation"`
	Timezone          string   `json:"timezone"`
	Icon              string   `json:"icon"`
	Text              string   `json:"text"`
	PauseNotification bool     `json:"pause_notification"`
	StatusTimeout     string   `json:"status_timeout"`
	WorkspaceID       string   `json:"workspace_id"`
	Track             string   `json:"track"`
	Links             []string `json:"links"`
	Online            bool     `json:"online"`
}

type UpdateUserProfileRequest struct {
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	FullName          string `json:"full_name"`
	UserName          string `json:"username"`
	AvatarURL         string `json:"avatar_url"`
	AvatarFile        string `json:"avatar_file"`
	DisplayName       string `json:"display_name"`
	Title             string `json:"title"`
	NamePronunciation string `json:"name_pronounciation"`
	Timezone          string `json:"timezone"`
	AvatarUpdate      bool
	WorkspaceID       string   `json:"workspace_id"`
	Track             string   `json:"track"`
	Links             []string `json:"links"`
}

type UpdateProfileStatus struct {
	Icon              string `json:"icon"`
	Text              string `json:"text"`
	PauseNotification bool   `json:"pause_notification"`
	StatusTimeout     string `json:"status_timeout"`
	StatusExpiry      string `json:"status_expiry"` // Natural language expiry like "30 minutes"
	ClearStatus       bool   `json:"clear_status"`
	Online            bool   `json:"online"`
	StatusVisibility  string `json:"status_visibility"` // Stored in DB but not used in new API
	UserId            string
	OrgId             string
}

// PartialStatusUpdate holds optional fields for status patch requests.
// Pointer fields let us detect whether a field was supplied so we can
// avoid overwriting existing values.
type PartialStatusUpdate struct {
	Text        *string `json:"text"`
	Emoji       *string `json:"emoji"`
	Expiry      *string `json:"expiry"`
	ClearStatus *bool   `json:"clear_status"`
	UserID      string  `json:"-"`
}

// SetStatusRequest holds fields for setting a new status via POST.
// Text is required; emoji and expiry are optional.
type SetStatusRequest struct {
	Text   string  `json:"text" validate:"required,min=1,max=255"`
	Emoji  *string `json:"emoji,omitempty" validate:"omitempty,max=64,no_whitespace,emoji"`
	Expiry *string `json:"expiry,omitempty" validate:"omitempty,status_expiry"`
	UserID string  `json:"-"`
}

type UpdateUserPresenceRequest struct {
	IsActive bool   `json:"online" validate:"boolean"`
	UserID   string `json:"-"`
	OrgID    string `json:"-"`
}

// UserStatus represents the persisted status for responses.
type UserStatus struct {
	Text       string `json:"text"`
	Emoji      string `json:"emoji"`
	Expiry     int64  `json:"expiry"`
	Visibility string `json:"visibility"`
	Online     bool   `json:"online"`
}

func (j *Profile) UpdateProfileFields(db *gorm.DB, req UpdateUserProfileRequest, userId string, logger *utility.Logger, orgId ...string) (*Profile, error) {
	var targetOrg string
	if len(orgId) > 0 {
		targetOrg = orgId[0]
	}

	targetProfile, err := j.GetOrCreateProfileForOrg(db, userId, targetOrg)
	if err != nil {
		return nil, errors.New("Profile does not exist")
	}

	if req.DisplayName != "" && req.UserName == "" {
		req.UserName = req.DisplayName
	}

	profileUpdates := Profile{
		FullName:          req.FullName,
		UserName:          req.UserName,
		Phone:             req.Phone,
		AvatarURL:         req.AvatarURL,
		DisplayName:       req.DisplayName,
		Title:             req.Title,
		NamePronunciation: req.NamePronunciation,
		Timezone:          req.Timezone,
		WorkspaceID:       req.WorkspaceID,
		Track:             req.Track,
		Links:             pq.StringArray(req.Links),
	}

	query := "id = ?"
	result, err := postgresql.UpdateFieldsAndReturn(db, &j, profileUpdates, query, targetProfile.ID)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("failed to update user profile")
	}

	if storage.DB != nil && storage.DB.Redis != nil {
		c, err := rd.RedisDelete(storage.DB.Redis, getProfileCacheKey(userId, targetOrg))

		if err != nil {
			utility.LogError(logger, "Failed to delete profile from redis: %v, user_id: %s, org_id: %s", err, userId, targetOrg)
		}

		utility.LogInfo(logger, "Deleted %d profile key(s) from redis, user_id: %s, org_id: %s", c, userId, targetOrg)

	}

	return j, nil
}

func (j *Profile) UpdateProfileStatus(db *gorm.DB, req UpdateProfileStatus, logger *utility.Logger) (UserStatus, int, error) {
	targetProf, err := j.GetOrCreateProfileForOrg(db, req.UserId, req.OrgId, logger)
	if err != nil {
		return UserStatus{}, http.StatusNotFound, errors.New("profile does not exist")
	}

	*j = targetProf

	// Parse status expiry if provided
	var expiryTimestamp int64
	if req.StatusExpiry != "" {
		expiryTimestamp, err = j.ParseStatusExpiry(req.StatusExpiry)
		if err != nil {
			return UserStatus{}, http.StatusBadRequest, err
		}
	}

	// Build updates map
	updates := map[string]any{
		"pause_notification": req.PauseNotification,
		"online":             req.Online,
	}

	// Handle text/icon - only update if provided
	if req.Text != "" || req.Icon != "" {
		updates["text"] = req.Text
		updates["icon"] = req.Icon
	}

	// Handle clear status
	if req.ClearStatus {
		updates["text"] = ""
		updates["icon"] = ""
		updates["status_timeout"] = ""
		ctx := context.Background()
		if j.RiverJobID != nil && storage.DB.River != nil {
			_, cancelErr := storage.DB.River.JobCancel(ctx, *j.RiverJobID)
			if cancelErr != nil {
				logger.Error("failed to cancel clear status job %d: %v", *j.RiverJobID, cancelErr)
			} else {
				logger.Info("Cancelled clear status job %d", *j.RiverJobID)
			}
		}
		updates["river_job_id"] = nil
	} else {
		// Handle status timeout/expiry
		if req.StatusExpiry != "" {
			if expiryTimestamp > 0 {
				updates["status_timeout"] = strconv.FormatInt(expiryTimestamp, 10)
			} else {
				updates["status_timeout"] = ""
			}
		} else if req.StatusTimeout != "" {
			// Use pre-formatted timeout if provided
			updates["status_timeout"] = req.StatusTimeout
		}

	}

	// Apply updates to target profile by primary key ID
	if err := db.Model(&Profile{}).Where("id = ?", targetProf.ID).Updates(updates).Error; err != nil {
		return UserStatus{}, http.StatusInternalServerError, errors.New("failed to update user profile")
	}

	updatedProfile := Profile{}
	// Reload profile to get updated values
	if err := db.Where("id = ?", targetProf.ID).First(&updatedProfile).Error; err != nil {
		return UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to reload profile: %w", err)
	}

	*j = updatedProfile

	if storage.DB != nil && storage.DB.Redis != nil {
		key1 := getProfileCacheKey(req.UserId, req.OrgId)
		c1, err1 := rd.RedisDelete(storage.DB.Redis, key1)

		if err1 != nil {
			utility.LogError(logger, "Failed to delete profile from redis: %v, user_id: %s, org_id: %s", err1, req.UserId, req.OrgId)
		}
		utility.LogInfo(logger, "Deleted %d profile key(s) from redis for user_id: %s, org_id: %s", c1, req.UserId, req.OrgId)

		key2 := getProfileCacheKey(req.UserId, "")
		_, _ = rd.RedisDelete(storage.DB.Redis, key2)
	}

	// Build response status
	expiry := int64(0)
	if !req.ClearStatus && j.StatusTimeout != "" {
		if parsed, err := strconv.ParseInt(j.StatusTimeout, 10, 64); err == nil {
			expiry = parsed
		}
	}

	visibility := "public"
	if j.StatusVisibility != "" {
		visibility = j.StatusVisibility
	}

	status := UserStatus{
		Text:       j.Text,
		Emoji:      j.Icon,
		Expiry:     expiry,
		Visibility: visibility,
		Online:     j.Online,
	}

	if req.ClearStatus {
		status.Text = ""
		status.Emoji = ""
		status.Expiry = 0
	}

	// Publish notification
	if logger != nil {
		logger.Info("status updated/cleared", "user_id", req.UserId)
		notification := Notification[ProfileStatusUpdated]
		notification.SectionType = ChannelsSection
		notification.NotificationId = utility.GenerateUUID()
		notification.ModificationDetails = &ModificationDetails{
			UserId: req.UserId,
		}
		notification.Content = struct {
			UserID string     `json:"user_id"`
			Status UserStatus `json:"status"`
		}{
			UserID: req.UserId,
			Status: status,
		}

		channelID := req.OrgId
		if err := centrifuge.PublishChannel(logger, channelID, notification); err != nil {
			logger.Error("failed to publish status update event", "error", err, "channel_id", channelID)
		}
	}

	return status, http.StatusOK, nil
}

// ParseStatusExpiry converts a natural language expiry string to a Unix timestamp.
func (p *Profile) ParseStatusExpiry(expiryStr string) (int64, error) {
	if expiryStr == "" {
		return 0, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(expiryStr))
	now := time.Now()

	var loc *time.Location
	if p.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(p.Timezone)
		if err != nil {
			loc = time.UTC
		}
	} else {
		loc = time.UTC
	}

	nowInTz := now.In(loc)

	switch normalized {
	case "30 seconds", "30 second":
		return nowInTz.Add(30 * time.Second).Unix(), nil

	case "30 minutes", "30 minute":
		return nowInTz.Add(30 * time.Minute).Unix(), nil

	case "1 hour", "1 hr":
		return nowInTz.Add(1 * time.Hour).Unix(), nil

	case "today":
		endOfDay := time.Date(
			nowInTz.Year(),
			nowInTz.Month(),
			nowInTz.Day(),
			23, 59, 59, 0,
			loc,
		)
		return endOfDay.Unix(), nil

	case "this week":
		daysUntilSunday := (7 - int(nowInTz.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		endOfWeek := time.Date(
			nowInTz.Year(),
			nowInTz.Month(),
			nowInTz.Day(),
			23, 59, 59, 0,
			loc,
		).AddDate(0, 0, daysUntilSunday)
		return endOfWeek.Unix(), nil

	case "don't remove", "dont remove", "do not remove", "Don't clear":
		return 0, nil

	default:
		return 0, fmt.Errorf("invalid expiry value: %s", expiryStr)
	}
}

func (p *Profile) GetUserByUsername(db *gorm.DB, userName string) (Profile, error) {
	var user Profile

	query := db.Where("user_name = ?", userName)

	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}

func (p *Profile) SetProfileImageToEmpty(db *gorm.DB, userId string, logger *utility.Logger, orgId ...string) error {
	var targetOrg string
	if len(orgId) > 0 {
		targetOrg = orgId[0]
	}

	targetProf, err := p.GetOrCreateProfileForOrg(db, userId, targetOrg)
	if err != nil {
		return err
	}

	result := db.Model(&Profile{}).Where("id = ?", targetProf.ID).Update("avatar_url", "")
	if result.Error != nil {
		return result.Error
	}

	if storage.DB != nil && storage.DB.Redis != nil {
		c, err := rd.RedisDelete(storage.DB.Redis, getProfileCacheKey(userId, targetOrg))

		if err != nil {
			utility.LogError(logger, "Failed to delete profile from redis: %v, user_id: %s, org_id: %s", err, userId, targetOrg)
		}
		utility.LogInfo(logger, "Deleted %d profile key(s) from redis, user_id: %s, org_id: %s", c, userId, targetOrg)

	}

	return nil
}

func (p *Profile) GetProfileByUserId(db *gorm.DB, userId string, orgId ...string) error {
	var targetOrg string
	if len(orgId) > 0 {
		targetOrg = orgId[0]
	}
	prof, err := p.GetOrCreateProfileForOrg(db, userId, targetOrg)
	if err != nil {
		return err
	}
	*p = prof
	return nil
}

func (p *Profile) GetProfileByUserIdAndOrgId(db *gorm.DB, userId string, orgId string) (Profile, error) {
	return p.GetOrCreateProfileForOrg(db, userId, orgId)
}

func (p *Profile) GetOrgID() string {
	if p != nil && p.OrganisationID != nil {
		return *p.OrganisationID
	}
	return ""
}

func getProfileCacheKey(userID, orgID string) string {
	if orgID == "" {
		return fmt.Sprintf("user:profile:%s:default", userID)
	}
	return fmt.Sprintf("user:profile:%s:%s", userID, orgID)
}

func (p *Profile) GetOrCreateProfileForOrg(db *gorm.DB, userID string, orgID string, logger ...*utility.Logger) (Profile, error) {
	var appLogger *utility.Logger
	if len(logger) > 0 {
		appLogger = logger[0]
	}

	if userID == "" {
		utility.LogError(appLogger, "[GetOrCreateProfileForOrg] Error: userID is empty")
		return Profile{}, errors.New("user_id is required")
	}

	cacheKey := getProfileCacheKey(userID, orgID)
	if storage.DB != nil && storage.DB.Redis != nil {
		if cachedBytes, err := rd.RedisGet(storage.DB.Redis, cacheKey); err == nil && len(cachedBytes) > 0 {
			var cachedProf Profile
			if jsonErr := json.Unmarshal(cachedBytes, &cachedProf); jsonErr == nil && cachedProf.ID != "" {
				utility.LogInfo(appLogger, "[GetOrCreateProfileForOrg] Redis cache hit: found profile ID=%s for userID=%s, orgID=%s", cachedProf.ID, userID, orgID)
				return cachedProf, nil
			}
		}
	}

	resolvedProf, err := p.resolveProfileForOrg(db, userID, orgID, appLogger)
	if err == nil && resolvedProf.ID != "" && storage.DB != nil && storage.DB.Redis != nil {
		if setErr := rd.RedisSet(storage.DB.Redis, cacheKey, resolvedProf, 12*time.Hour); setErr != nil {
			utility.LogError(appLogger, "[GetOrCreateProfileForOrg] Failed to set profile cache in redis for key %s: %v", cacheKey, setErr)
		}
		utility.LogInfo(appLogger, "[GetOrCreateProfileForOrg] Set profile cache in redis for key %s (TTL: 12h)", cacheKey)

	}
	return resolvedProf, err
}

func (p *Profile) GetOrCreateMultipleProfilesForOrg(db *gorm.DB, userIDs []string, orgID string, logger ...*utility.Logger) (map[string]Profile, error) {

	var appLogger *utility.Logger
	if len(logger) > 0 {
		appLogger = logger[0]
	}

	utility.LogInfo(appLogger, "[GetOrCreateMultipleProfilesForOrg] Starting resolution for orgID=%s", orgID)

	resultMap := make(map[string]Profile)
	if len(userIDs) == 0 {
		return resultMap, nil
	}

	uniqueUserIDs := make([]string, 0, len(userIDs))
	seen := make(map[string]bool)
	for _, uID := range userIDs {
		trimmed := strings.TrimSpace(uID)
		if trimmed != "" && trimmed != "WEBHOOK" && !seen[trimmed] {
			seen[trimmed] = true
			uniqueUserIDs = append(uniqueUserIDs, trimmed)
		}
	}

	if len(uniqueUserIDs) == 0 {
		return resultMap, nil
	}

	missedIDs := make([]string, 0, len(uniqueUserIDs))

	if storage.DB != nil && storage.DB.Redis != nil {
		keys := make([]string, len(uniqueUserIDs))
		for i, uID := range uniqueUserIDs {
			keys[i] = getProfileCacheKey(uID, orgID)
		}

		vals, err := rd.RedisMGet(storage.DB.Redis, keys...)
		if err == nil && len(vals) == len(uniqueUserIDs) {
			for i, val := range vals {
				uID := uniqueUserIDs[i]
				if val != nil {
					var cachedProf Profile
					var unmarshalErr error

					switch v := val.(type) {
					case string:
						unmarshalErr = json.Unmarshal([]byte(v), &cachedProf)
					case []byte:
						unmarshalErr = json.Unmarshal(v, &cachedProf)
					}

					if unmarshalErr == nil && cachedProf.ID != "" {
						resultMap[uID] = cachedProf
						utility.LogInfo(appLogger, "[GetOrCreateMultipleProfilesForOrg] Redis MGET cache hit for userID=%s, profileID=%s", uID, cachedProf.ID)
						continue
					}
				}
				missedIDs = append(missedIDs, uID)
			}
		} else {
			missedIDs = append(missedIDs, uniqueUserIDs...)
		}
	} else {
		missedIDs = append(missedIDs, uniqueUserIDs...)
	}

	utility.LogInfo(appLogger, "[GetOrCreateMultipleProfilesForOrg] Resolving uncached Missed IDs: %v", missedIDs)

	for _, uID := range missedIDs {
		resolvedProf, err := p.resolveProfileForOrg(db, uID, orgID, appLogger)
		if err == nil {
			resultMap[uID] = resolvedProf
			if storage.DB != nil && storage.DB.Redis != nil && resolvedProf.ID != "" {
				cacheKey := getProfileCacheKey(uID, orgID)
				if setErr := rd.RedisSet(storage.DB.Redis, cacheKey, resolvedProf, 12*time.Hour); setErr != nil {

					utility.LogError(appLogger, "[GetOrCreateMultipleProfilesForOrg] Failed to set profile cache in redis for key %s: %v", cacheKey, setErr)

				}
				utility.LogInfo(appLogger, "[GetOrCreateMultipleProfilesForOrg] Set profile cache in redis for key %s (TTL: 12h)", cacheKey)
			}
		} else {
			utility.LogError(appLogger, "[GetOrCreateMultipleProfilesForOrg] Failed to resolve profile for userID=%s: %v", uID, err)
		}
	}

	utility.LogInfo(appLogger, "[GetOrCreateMultipleProfilesForOrg] Resolved profiles with length of  %d", len(resultMap))

	return resultMap, nil
}

func (p *Profile) resolveProfileForOrg(db *gorm.DB, userID string, orgID string, appLogger *utility.Logger) (Profile, error) {
	utility.LogInfo(appLogger, "[GetOrCreateProfileForOrg] Starting resolution for userID=%s, orgID=%s", userID, orgID)
	var profile Profile

	if orgID != "" {
		if err := db.Where("userid = ? AND organisation_id = ?", userID, orgID).First(&profile).Error; err == nil {
			utility.LogInfo(appLogger, "[GetOrCreateProfileForOrg] Fast-path hit: found profile ID=%s for userID=%s, orgID=%s", profile.ID, userID, orgID)
			return profile, nil
		}
	}

	var existingProfiles []Profile
	err := db.Where("userid = ?", userID).Order("created_at asc").Find(&existingProfiles).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		utility.LogError(appLogger, "[GetOrCreateProfileForOrg] Error fetching existing profiles for userID=%s: %v", userID, err)
		return profile, fmt.Errorf("failed to fetch existing profile: %w", err)
	}

	var userOrgIDs []string
	db.Table("user_organisations").
		Where("user_id = ?", userID).
		Distinct("organisation_id").
		Pluck("organisation_id", &userOrgIDs)

	if len(existingProfiles) > 0 {
		bindUnassignedBaseProfiles(db, userID, orgID, existingProfiles, userOrgIDs, appLogger)
		orgMap, baseProfile := populateMissingOrgProfiles(db, userID, orgID, existingProfiles, userOrgIDs, appLogger)

		if orgID != "" {
			if targetProf, ok := orgMap[orgID]; ok {
				utility.LogInfo(appLogger, "[GetOrCreateProfileForOrg] Completed resolution for userID=%s, orgID=%s, returning profile ID=%s", userID, orgID, targetProf.ID)
				return targetProf, nil
			}
			var fetchedProf Profile
			if err := db.Where("userid = ? AND organisation_id = ?", userID, orgID).First(&fetchedProf).Error; err == nil {
				utility.LogInfo(appLogger, "[GetOrCreateProfileForOrg] Completed resolution for userID=%s, orgID=%s, returning fetched profile ID=%s", userID, orgID, fetchedProf.ID)
				return fetchedProf, nil
			}
		}
		utility.LogInfo(appLogger, "[GetOrCreateProfileForOrg] Completed resolution for userID=%s, returning base profile ID=%s", userID, baseProfile.ID)
		return baseProfile, nil
	}

	return createInitialDefaultProfile(db, userID, orgID, appLogger)
}

func bindUnassignedBaseProfiles(db *gorm.DB, userID string, orgID string, existingProfiles []Profile, userOrgIDs []string, appLogger *utility.Logger) {
	utility.LogInfo(appLogger, "[bindUnassignedBaseProfiles] Starting check for userID=%s, orgID=%s across %d profile(s)", userID, orgID, len(existingProfiles))

	assignedOrgs := make(map[string]bool)
	for _, prof := range existingProfiles {
		if prof.OrganisationID != nil && *prof.OrganisationID != "" {
			assignedOrgs[*prof.OrganisationID] = true
		}
	}

	availableOrgs := make([]string, 0)
	if orgID != "" && !assignedOrgs[orgID] {
		availableOrgs = append(availableOrgs, orgID)
	}
	for _, oid := range userOrgIDs {
		if oid != "" && !assignedOrgs[oid] && oid != orgID {
			availableOrgs = append(availableOrgs, oid)
		}
	}

	boundCount := 0
	orgIdx := 0
	for i := range existingProfiles {
		if existingProfiles[i].OrganisationID == nil || *existingProfiles[i].OrganisationID == "" {
			if orgIdx < len(availableOrgs) {
				targetOrg := availableOrgs[orgIdx]
				existingProfiles[i].OrganisationID = &targetOrg
				_ = db.Model(&Profile{}).Where("id = ?", existingProfiles[i].ID).Update("organisation_id", targetOrg).Error
				assignedOrgs[targetOrg] = true
				orgIdx++
				boundCount++
			}

			if existingProfiles[i].AvatarURL == "" {
				existingProfiles[i].AvatarURL = avatar.GenerateDefaultAvatarURL(userID)
				_ = db.Model(&Profile{}).Where("id = ?", existingProfiles[i].ID).Update("avatar_url", existingProfiles[i].AvatarURL).Error
			}
		}
	}
	utility.LogInfo(appLogger, "[bindUnassignedBaseProfiles] Completed for userID=%s: bound %d unassigned profile(s)", userID, boundCount)
}

func populateMissingOrgProfiles(db *gorm.DB, userID string, orgID string, existingProfiles []Profile, userOrgIDs []string, appLogger *utility.Logger) (map[string]Profile, Profile) {
	utility.LogInfo(appLogger, "[populateMissingOrgProfiles] Starting population for userID=%s, target orgID=%s", userID, orgID)

	allOrgIDs := make([]string, len(userOrgIDs))
	copy(allOrgIDs, userOrgIDs)
	if orgID != "" {
		found := false
		for _, o := range allOrgIDs {
			if o == orgID {
				found = true
				break
			}
		}
		if !found {
			allOrgIDs = append(allOrgIDs, orgID)
		}
	}

	existingOrgMap := make(map[string]Profile)
	for _, prof := range existingProfiles {
		if prof.OrganisationID != nil && *prof.OrganisationID != "" {
			existingOrgMap[*prof.OrganisationID] = prof
		}
	}

	var baseProfile Profile
	for _, prof := range existingProfiles {
		if prof.OrganisationID == nil || *prof.OrganisationID == "" {
			baseProfile = prof
			break
		}
	}
	if baseProfile.ID == "" && len(existingProfiles) > 0 {
		baseProfile = existingProfiles[0]
	}
	createdCount := 0

	for _, mappedOrgID := range allOrgIDs {
		if mappedOrgID == "" {
			continue
		}
		if _, exists := existingOrgMap[mappedOrgID]; !exists {
			orgCopy := mappedOrgID
			avatarURL := baseProfile.AvatarURL
			if avatarURL == "" {
				avatarURL = avatar.GenerateDefaultAvatarURL(userID)
			}
			newProf := Profile{
				ID:                utility.GenerateUUID(),
				Userid:            userID,
				OrganisationID:    &orgCopy,
				FirstName:         baseProfile.FirstName,
				LastName:          baseProfile.LastName,
				FullName:          baseProfile.FullName,
				UserName:          baseProfile.UserName,
				DisplayName:       baseProfile.DisplayName,
				Title:             baseProfile.Title,
				NamePronunciation: baseProfile.NamePronunciation,
				Timezone:          baseProfile.Timezone,
				Phone:             baseProfile.Phone,
				WorkspaceID:       baseProfile.WorkspaceID,
				Track:             baseProfile.Track,
				Links:             baseProfile.Links,
				AvatarURL:         avatarURL,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			}
			if createErr := db.Create(&newProf).Error; createErr == nil {
				existingOrgMap[mappedOrgID] = newProf
				createdCount++
			} else {
				utility.LogError(appLogger, "[populateMissingOrgProfiles] Error creating profile for userID=%s, orgID=%s: %v", userID, mappedOrgID, createErr)
				var raceFetched Profile
				if fetchErr := db.Where("userid = ? AND organisation_id = ?", userID, orgCopy).First(&raceFetched).Error; fetchErr == nil {
					existingOrgMap[mappedOrgID] = raceFetched
				}
			}

		}
	}

	utility.LogInfo(appLogger, "[populateMissingOrgProfiles] Completed: created %d profile(s) for userID=%s across %d mapped org(s)", createdCount, userID, len(allOrgIDs))
	return existingOrgMap, baseProfile
}

func createInitialDefaultProfile(db *gorm.DB, userID string, orgID string, appLogger *utility.Logger) (Profile, error) {
	utility.LogInfo(appLogger, "[createInitialDefaultProfile] Creating initial default profile for userID=%s, orgID=%s", userID, orgID)
	var userObj User
	userName := "User"
	if fetchUserErr := db.Where("id = ?", userID).First(&userObj).Error; fetchUserErr == nil && userObj.Name != "" {
		userName = userObj.Name
	}

	var orgPtr *string
	if orgID != "" {
		orgPtr = &orgID
	}

	newProfile := Profile{
		ID:             utility.GenerateUUID(),
		Userid:         userID,
		OrganisationID: orgPtr,
		FirstName:      userName,
		FullName:       userName,
		UserName:       userName,
		AvatarURL:      avatar.GenerateDefaultAvatarURL(userID),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Online:         true,
		IsActive:       true,
	}

	var profile Profile
	if createErr := db.Create(&newProfile).Error; createErr != nil {
		if fetchErr := db.Where("userid = ? AND organisation_id = ?", userID, orgID).First(&profile).Error; fetchErr == nil {
			utility.LogInfo(appLogger, "[createInitialDefaultProfile] Initial profile race condition resolved, found ID=%s for userID=%s", profile.ID, userID)
			return profile, nil
		}
		utility.LogError(appLogger, "[createInitialDefaultProfile] Error creating initial profile for userID=%s: %v", userID, createErr)
		return profile, fmt.Errorf("failed to create initial profile: %w", createErr)
	}
	utility.LogInfo(appLogger, "[createInitialDefaultProfile] Successfully created initial profile ID=%s for userID=%s, orgID=%s", newProfile.ID, userID, orgID)
	return newProfile, nil
}

// updateProfileLinks updates the `links` text[] column for a user's profile.
// helper removed: Links are updated as part of the main update using pq.StringArray
