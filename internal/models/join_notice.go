package models

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// NoticeLabelForProfile resolves the display label for a channel join/leave
// system notice. The configured, non-empty profiles.display_name wins;
// otherwise the stored username is used. No error is returned: an empty
// label is valid and callers decide how to handle it.
func NoticeLabelForProfile(profile Profile) string {
	if label := strings.TrimSpace(profile.DisplayName); label != "" {
		return label
	}
	return strings.TrimSpace(profile.UserName)
}

// LoadJoinNoticeProfile fetches a user's channel-visible profile with a
// strictly read-only resolution: org-scoped profile first, then the
// unscoped base profile or the oldest profile, matching the legacy resolver.
// Unlike GetOrCreateProfileForOrg, it never binds,
// creates, or caches profiles — a join notice must not mutate membership data.
// A genuine database failure on the org-scoped query is propagated
// immediately; only a record-not-found falls through to the base profile.
func LoadJoinNoticeProfile(db *gorm.DB, userID, orgID string) (Profile, error) {
	var profile Profile
	if orgID != "" {
		if err := db.Where("userid = ? AND organisation_id = ?", userID, orgID).First(&profile).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return Profile{}, err
			}
		} else {
			return profile, nil
		}
	}
	if err := db.Where("userid = ? AND organisation_id IS NULL", userID).Order("created_at asc").First(&profile).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return Profile{}, err
		}
		// Signup may already have bound the original profile to a personal org.
		if err := db.Where("userid = ?", userID).Order("created_at asc").First(&profile).Error; err != nil {
			return Profile{}, err
		}
	}
	return profile, nil
}

// ErrJoinNoticeNoLabel is returned when a stored profile exists but carries
// no usable server-side label (both display_name and user_name are empty).
// Callers must skip the participant rather than publish a blank mention.
var ErrJoinNoticeNoLabel = errors.New("join notice: no server-side label available")

// NewChannelJoinMention builds the system-notice participant for a single
// channel joiner, resolving the label server-side from the stored profile.
// Precedence: stored display_name, then stored user_name; otherwise no
// notice is generated for that participant (ErrJoinNoticeNoLabel). The
// mention UserID is always preserved. No client-controlled string is ever
// used as a label.
func NewChannelJoinMention(db *gorm.DB, userID, orgID string) (UserMention, error) {
	profile, err := LoadJoinNoticeProfile(db, userID, orgID)
	if err != nil {
		return UserMention{UserID: userID}, err
	}
	if label := NoticeLabelForProfile(profile); label != "" {
		return UserMention{UserID: userID, Username: label}, nil
	}
	return UserMention{UserID: userID}, ErrJoinNoticeNoLabel
}

// BuildChannelJoinMentions resolves server-side labels for a batch of
// channel joiners, preserving their order and IDs. Entries that cannot be
// resolved to any label are skipped so no blank mention is ever stored.
// Profile lookup failures abort the batch to preserve stored participants.
func BuildChannelJoinMentions(db *gorm.DB, userIDs []string, orgID string) ([]UserMention, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	mentions := make([]UserMention, 0, len(userIDs))
	seen := make(map[string]bool, len(userIDs))
	for _, userID := range userIDs {
		if userID == "" || seen[userID] {
			continue
		}
		seen[userID] = true
		mention, err := NewChannelJoinMention(db, userID, orgID)
		if errors.Is(err, ErrJoinNoticeNoLabel) {
			continue
		}
		if err != nil {
			return nil, err
		}
		mentions = append(mentions, mention)
	}
	return mentions, nil
}
