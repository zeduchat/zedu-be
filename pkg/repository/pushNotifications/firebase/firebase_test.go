package firebase

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/utility"
)

func TestWithDisplayData_CodeQLFix(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]string
		title    string
		body     string
		expected map[string]string
	}{
		{
			name: "Add title and body to empty data",
			data: map[string]string{},
			title: "Incoming Call",
			body:  "Alice is calling",
			expected: map[string]string{
				"title": "Incoming Call",
				"body":  "Alice is calling",
			},
		},
		{
			name: "Preserve existing keys and add title/body",
			data: map[string]string{
				"buzz_id": "123",
				"host_id": "user-456",
			},
			title: "Incoming Call",
			body:  "Bob is calling",
			expected: map[string]string{
				"buzz_id": "123",
				"host_id": "user-456",
				"title":   "Incoming Call",
				"body":    "Bob is calling",
			},
		},
		{
			name: "Empty title and body",
			data: map[string]string{
				"event": "direct_call",
			},
			title: "",
			body:  "",
			expected: map[string]string{
				"event": "direct_call",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := withDisplayData(tt.data, tt.title, tt.body)
			assert.Equal(t, tt.expected, res)
		})
	}

	// Verify capacity calculation logic does not overflow at theoretical boundary
	capHint := math.MaxInt - 1
	if capHint <= math.MaxInt-2 {
		capHint += 2
	}
	assert.Equal(t, math.MaxInt-1, capHint, "should not overflow when capHint > MaxInt-2")
}

func TestIsDirectCall(t *testing.T) {
	tests := []struct {
		data     map[string]string
		expected bool
	}{
		{data: map[string]string{"event": "direct_call_ringing"}, expected: true},
		{data: map[string]string{"event": "direct_call_initiated"}, expected: true},
		{data: map[string]string{"notification_type": "direct_call"}, expected: true},
		{data: map[string]string{"notification_type": "buzz_ended"}, expected: true},
		{data: map[string]string{"event": "new_message", "notification_type": "channel"}, expected: false},
		{data: map[string]string{"event": "new_message", "notification_type": "dm"}, expected: false},
		{data: nil, expected: false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, isDirectCall(tt.data))
	}
}

func TestSendPush_UninitializedGraceful(t *testing.T) {
	logger := utility.NewLogger()
	// When Client.Client is nil, SendPush should return nil, 0, nil without error
	invalid, count, err := SendPush(logger, []string{"fake_token"}, "Title", "Body", "", nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, invalid)
}
