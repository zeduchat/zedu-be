package test_channel

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
)

// NoticeLabelForProfile prefers the configured display name and falls
// back to the stored username when no display name is configured.
func TestNoticeLabelForProfile(t *testing.T) {
	cases := []struct {
		name    string
		profile models.Profile
		want    string
	}{
		{
			name:    "display name wins over username and full name",
			profile: models.Profile{FullName: "Pauline Banye", UserName: "pauline.banye", DisplayName: "Lynn B"},
			want:    "Lynn B",
		},
		{
			name:    "display name is trimmed",
			profile: models.Profile{UserName: "pauline.banye", DisplayName: "  Lynn B  "},
			want:    "Lynn B",
		},
		{
			name:    "blank display name falls back to username",
			profile: models.Profile{FullName: "Pauline Banye", UserName: "pauline.banye", DisplayName: "   "},
			want:    "pauline.banye",
		},
		{
			name:    "no display name falls back to username",
			profile: models.Profile{FullName: "Pauline Banye", UserName: "pauline.banye"},
			want:    "pauline.banye",
		},
		{
			name:    "empty profile yields empty label",
			profile: models.Profile{},
			want:    "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := models.NoticeLabelForProfile(tc.profile); got != tc.want {
				t.Errorf("NoticeLabelForProfile() = %q, want %q", got, tc.want)
			}
		})
	}
}
