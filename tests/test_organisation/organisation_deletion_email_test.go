package test_organisation

import (
	"strings"
	"testing"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	notifications "github.com/hngprojects/telex_be/services/notification"
	"github.com/hngprojects/telex_be/utility"
)

func TestOrganisationDeletionEmailUsesConsentedProfileAndBccRecipients(t *testing.T) {
	config.Setup(utility.NewLogger(), "../../app")
	data := models.OrganisationDeletionMail{
		Recipients:                   []string{"owner@example.test", "member@example.test"},
		OrganisationName:             "Test Organisation",
		DeletedByName:                "Test Owner",
		DeletedByAvatarURL:           "https://cdn.example.test/owner.png",
		IncludeProfileInDeletionMail: true,
	}

	email, err := notifications.BuildOrganisationDeletedEmail(request.ExternalRequest{Test: true}, data)
	if err != nil {
		t.Fatal(err)
	}
	if !email.BlindCarbonCopy {
		t.Fatal("organisation members must not see each other's email addresses")
	}
	if len(email.To) != len(data.Recipients) {
		t.Fatalf("expected %d recipients, got %d", len(data.Recipients), len(email.To))
	}
	for _, expected := range []string{data.OrganisationName, data.DeletedByName, data.DeletedByAvatarURL} {
		if !strings.Contains(email.Body, expected) {
			t.Fatalf("rendered deletion email does not include consented value %q", expected)
		}
	}
	if err := email.Send(); err != nil {
		t.Fatalf("test email should validate without sending SMTP traffic: %v", err)
	}
}

func TestOrganisationDeletionEmailRequiresProfileConsent(t *testing.T) {
	_, err := notifications.BuildOrganisationDeletedEmail(request.ExternalRequest{Test: true}, models.OrganisationDeletionMail{
		Recipients: []string{"member@example.test"},
	})
	if err == nil {
		t.Fatal("email containing profile details should require explicit consent")
	}
}
