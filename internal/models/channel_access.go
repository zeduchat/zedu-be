package models

import (
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/utility"
)

// UserCanAccessChannel reports whether userID is a member of channelID, which can be
// a regular channel (user_channels), a 1:1 DM (dm_channels, caller is owner or
// participant) or a group DM (channel_participants). Fails closed on bad input.
func UserCanAccessChannel(db *gorm.DB, channelID, userID string) (bool, error) {
	if !utility.IsValidUUID(channelID) || !utility.IsValidUUID(userID) {
		return false, nil
	}

	var count int64
	if err := db.Model(&UserChannels{}).
		Where("channels_id = ? AND user_id = ?", channelID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	// 1:1 DMs only (legacy rows may have NULL channel_type, matching the listing query at
	// direct_message.go:570). A group_dm row keeps its creator in user_id even after they
	// leave, so group membership must come from channel_participants.
	if err := db.Model(&DmChannels{}).
		Where("channel_id = ? AND (channel_type IS NULL OR channel_type = 'dm') AND (user_id = ? OR participant_id = ?)", channelID, userID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	if err := db.Model(&ChannelParticipant{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
