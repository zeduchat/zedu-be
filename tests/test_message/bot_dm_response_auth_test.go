package test_message

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	dmCtrl "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

const botDMResponseTestSecret = "bot-dm-response-test-secret"

const zeroThreadID = "00000000-0000-0000-0000-000000000000"

func TestBotDMResponseRequiresServiceSecretAndChannelAgent(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	ctrl := dmCtrl.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.New()
	r.POST("/api/v1/dms/bot-dm-response",
		middleware.RequireServiceSecret(middleware.ServiceSecretHeader, botDMResponseTestSecret),
		ctrl.BotDMResponse)

	post := func(secret string, body map[string]any) int {
		t.Helper()
		buf, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/dms/bot-dm-response", bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
		if secret != "" {
			req.Header.Set(middleware.ServiceSecretHeader, secret)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}

	// Fixtures: an agent installed in orgA, a second agent installed only in
	// orgB, a bot DM in orgA with the first agent, a bot DM in orgA whose
	// participant is the orgB-only agent, and a DM with no participant.
	orgA, orgB := utility.GenerateUUID(), utility.GenerateUUID()
	agentA, agentB := utility.GenerateUUID(), utility.GenerateUUID()
	install := func(orgID, agentID string) {
		if err := db.Postgresql.Create(&models.OrganisationIntegrations{
			ID: utility.GenerateUUID(), OrgID: orgID, IntegrationID: agentID, OwnerID: utility.GenerateUUID(), IsActive: true,
		}).Error; err != nil {
			t.Fatalf("insert org integration: %v", err)
		}
	}
	install(orgA, agentA)
	install(orgB, agentB)

	newDM := func(participant *string) string {
		channelID := utility.GenerateUUID()
		if err := db.Postgresql.Create(&models.DmChannels{
			ID: utility.GenerateUUID(), UserId: utility.GenerateUUID(), ChannelId: channelID, OrgId: orgA,
			ParticipantId: participant, ChatType: "bot", ChannelType: "dm",
		}).Error; err != nil {
			t.Fatalf("insert dm channel: %v", err)
		}
		return channelID
	}
	dmWithAgentA := newDM(&agentA)
	dmWithForeignAgent := newDM(&agentB)
	dmWithoutParticipant := newDM(nil)

	marketplaceChannel := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.UnauthenticatedUser{
		UserIdentifier: utility.GenerateUUID(), ChannelID: marketplaceChannel, Subtoken: utility.GenerateUUID(),
		LastAgentChatID: zeroThreadID,
	}).Error; err != nil {
		t.Fatalf("insert marketplace session: %v", err)
	}

	body := func(channelID, threadID, agentID string) map[string]any {
		return map[string]any{"channel_id": channelID, "thread_id": threadID, "agent_id": agentID, "message": "hello", "type": "message"}
	}

	cases := []struct {
		name   string
		secret string
		body   map[string]any
		want   int
	}{
		{"no secret is rejected", "", body(dmWithAgentA, utility.GenerateUUID(), agentA), http.StatusUnauthorized},
		{"wrong secret is rejected", "nope", body(dmWithAgentA, utility.GenerateUUID(), agentA), http.StatusUnauthorized},
		{"unknown channel is not found", botDMResponseTestSecret, body(utility.GenerateUUID(), utility.GenerateUUID(), agentA), http.StatusNotFound},
		{"agent_id of another agent is forbidden", botDMResponseTestSecret, body(dmWithAgentA, utility.GenerateUUID(), agentB), http.StatusForbidden},
		{"participant installed only in another org is forbidden", botDMResponseTestSecret, body(dmWithForeignAgent, utility.GenerateUUID(), ""), http.StatusForbidden},
		{"dm without participant is forbidden", botDMResponseTestSecret, body(dmWithoutParticipant, utility.GenerateUUID(), ""), http.StatusForbidden},
		{"zero thread to a non-marketplace channel is not found", botDMResponseTestSecret, body(dmWithAgentA, zeroThreadID, agentA), http.StatusNotFound},
		{"zero thread to a marketplace session channel is published", botDMResponseTestSecret, body(marketplaceChannel, zeroThreadID, agentA), http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tst.AssertStatusCode(t, post(tc.secret, tc.body), tc.want)
		})
	}
}
