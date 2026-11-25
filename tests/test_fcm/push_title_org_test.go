package test_fcm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
)

func TestFormatTitleWithOrg(t *testing.T) {
	tests := []struct {
		name     string
		orgName  string
		title    string
		expected string
	}{
		{"dm title", "Acme", "john_doe", "Acme - message from john_doe"},
		{"channel title", "Acme", "#general", "Acme - message from #general"},
		{"no org name", "", "#general", "#general"},
		{"empty title", "Acme", "", "Acme"},
		{"already prefixed", "Acme", "Acme - message from john_doe", "Acme - message from john_doe"},
		{"title equals org", "Acme", "acme", "Acme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, push_notifications.FormatTitleWithOrg(tt.orgName, tt.title))
		})
	}
}
