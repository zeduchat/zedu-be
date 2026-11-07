package links

import (
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

func GetChannelLinks(db *gorm.DB, channelID string, pagination postgresql.Pagination) ([]models.ChannelLinkResponse, postgresql.PaginationResponse, error) {
	records, paginationResp, err := models.SelectChannelLinks(db, channelID, pagination.Page, pagination.Limit)
	if err != nil {
		return nil, paginationResp, err
	}
	if len(records) == 0 {
		return []models.ChannelLinkResponse{}, paginationResp, nil
	}

	userIDs := make([]string, 0, len(records))
	for _, record := range records {
		userIDs = append(userIDs, record.UserID)
	}

	var profModel models.Profile
	profilesByUser, err := profModel.GetOrCreateMultipleProfilesForOrg(db, userIDs, records[0].OrganisationID)
	if err != nil {
		return nil, paginationResp, err
	}

	response := make([]models.ChannelLinkResponse, 0, len(records))
	for _, record := range records {
		profile := profilesByUser[record.UserID]
		response = append(response, models.ChannelLinkResponse{
			URL:       record.URL,
			MessageID: record.MessageID,
			ThreadID:  record.ThreadID,
			ChannelID: record.ChannelID,
			UserID:    record.UserID,
			Username:  profile.UserName,
			AvatarURL: profile.AvatarURL,
			CreatedAt: record.CreatedAt,
		})
	}

	return response, paginationResp, nil
}
