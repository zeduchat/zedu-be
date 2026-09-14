package test_buzz

import (
	"encoding/json"
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
)

func TestInviteUsersToBuzzResponseFailureReasons(t *testing.T) {
	resp := models.InviteUsersToBuzzResponse{
		BuzzID:          "buzz-123",
		InvitedUserIDs:  []string{},
		FailedUserIDs:   []string{"user-456"},
		FailureReasons:  []string{"user-456 is already a participant in this buzz"},
		InvitationsSent: 0,
		Message:         "no invitations were sent",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal InviteUsersToBuzzResponse: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	reasons, exists := jsonMap["failure_reasons"]
	if !exists {
		t.Fatalf("expected JSON to contain 'failure_reasons' key")
	}

	reasonsList, ok := reasons.([]interface{})
	if !ok || len(reasonsList) != 1 {
		t.Fatalf("expected failure_reasons to be array with 1 item, got %v", reasons)
	}

	if reasonsList[0] != "user-456 is already a participant in this buzz" {
		t.Errorf("expected failure reason 'user-456 is already a participant in this buzz', got %v", reasonsList[0])
	}
}
