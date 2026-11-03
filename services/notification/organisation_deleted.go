package notifications

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/send"
)

func BuildOrganisationDeletedEmail(extReq request.ExternalRequest, data models.OrganisationDeletionMail) (*send.EmailRequest, error) {
	if !data.IncludeProfileInDeletionMail {
		return nil, errors.New("deletion-mail profile consent is required")
	}
	if len(data.Recipients) == 0 {
		return nil, errors.New("organisation deletion email has no recipients")
	}

	email, err := send.NewEmailRequest(extReq, data.Recipients, "Subject: Organisation deleted", "organisation-deleted.html", "", map[string]any{
		"organisation_name": data.OrganisationName,
		"deleted_by_name":   data.DeletedByName,
		"deleted_by_avatar": data.DeletedByAvatarURL,
	})
	if err != nil {
		return nil, err
	}
	email.BlindCarbonCopy = true
	return email, nil
}

func (n NotificationObject) SendOrganisationDeletedMail() error {
	var data models.OrganisationDeletionMail
	if err := json.Unmarshal([]byte(n.Notification.Data), &data); err != nil {
		return fmt.Errorf("error decoding organisation deletion email data: %w", err)
	}
	email, err := BuildOrganisationDeletedEmail(n.ExtReq, data)
	if err != nil {
		return err
	}
	return email.Send()
}
