package auth

import (
	"github.com/gin-gonic/gin"
	contract "github.com/hngprojects/telex_be/internal/campaign"
	"testing"
)

func TestReturningUsersDoNotRecordSignupConversions(t *testing.T) {
	// No database is configured: any accidental attempt to count a returning
	// account would fail this test. Both OAuth and email controllers share this gate.
	controller := Controller{}
	for _, result := range []gin.H{
		{"user": map[string]any{"id": "existing"}},
		{"is_new_user": false, "user": map[string]any{"id": "existing"}},
	} {
		controller.recordCampaign(&contract.Attribution{}, result)
	}
}
