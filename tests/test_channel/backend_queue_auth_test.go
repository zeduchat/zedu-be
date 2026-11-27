package test_channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

const backendQueueTestSecret = "backend-queue-test-secret"

func postBackendQueue(t *testing.T, r *gin.Engine, secret string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/channels/backend-queue", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set(middleware.ServiceSecretHeader, secret)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestBackendQueueRequiresServiceSecretAndMembership(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	uid := utility.GenerateUUID()

	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	// The tests helpers register their own routes on every call, so each
	// fixture call gets a fresh engine and r only carries the route under test.
	fresh := func() *gin.Engine { return gin.New() }
	r := gin.New()
	r.Group("/api/v1/channels", middleware.RequireServiceSecret(middleware.ServiceSecretHeader, backendQueueTestSecret)).
		POST("/backend-queue", channelCtrl.SaveIncomingQueueMsg)

	newUser := func(prefix string) (string, string, string) {
		data := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("%s_%v@qa.team", prefix, uid),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   prefix,
			LastName:    "Queue",
			Password:    "password",
			UserName:    fmt.Sprintf("%s_%v", prefix, uid),
		}
		tst.SignupUser(t, fresh(), authCtrl, data, false)
		token := tst.GetLoginToken(t, fresh(), authCtrl, models.LoginRequestModel{Email: data.Email, Password: data.Password})
		return token, tst.GetUserIDFromToken(t, token, db), data.Email
	}
	newOrg := func(token, email, name string) string {
		orgID, _, _ := tst.CreateOrganisation(t, fresh(), db, orgCtrl, models.CreateOrgRequestModel{
			Name: name, Description: "d", Email: email, Type: "test", Location: "test", Country: "test",
		}, token)
		return orgID
	}

	ownerToken, ownerID, ownerEmail := newUser("bqowner")
	orgID := newOrg(ownerToken, ownerEmail, fmt.Sprintf("BQOrg%s", uid))
	chanA, _ := tst.CreateChannels(t, fresh(), channelCtrl, db, models.CreateChannelsRequest{
		Name: fmt.Sprintf("bqa_%s", uid), Username: fmt.Sprintf("bqa_%s", uid), OrganisationID: orgID, Description: "a",
	}, ownerToken)
	if chanA == "" {
		t.Fatalf("channel not created")
	}
	// Parent threads live in Elasticsearch, which the local harness cannot
	// write to, so these cases stop at or before the thread lookup.
	threadA := utility.GenerateUUID()

	outsiderToken, outsiderID, outsiderEmail := newUser("bqout")
	otherOrgID := newOrg(outsiderToken, outsiderEmail, fmt.Sprintf("BQOther%s", uid))

	installAgent := func(agentOrgID, ownerUserID string) string {
		agentID := utility.GenerateUUID()
		if err := db.Postgresql.Create(&models.OrganisationIntegrations{
			ID: utility.GenerateUUID(), OrgID: agentOrgID, IntegrationID: agentID, OwnerID: ownerUserID, IsActive: true,
		}).Error; err != nil {
			t.Fatalf("insert org integration: %v", err)
		}
		return agentID
	}
	foreignAgentID := installAgent(otherOrgID, outsiderID)
	localAgentID := installAgent(orgID, ownerID)

	msg := func(channelID, threadID, userID string) map[string]any {
		return map[string]any{"type": "message", "message": "<p>queued</p>", "channel_id": channelID, "thread_id": threadID, "user_id": userID, "org_id": orgID}
	}

	t.Run("no secret is rejected", func(t *testing.T) {
		tst.AssertStatusCode(t, postBackendQueue(t, r, "", msg(chanA, threadA, ownerID)).Code, http.StatusUnauthorized)
	})
	t.Run("wrong secret is rejected", func(t *testing.T) {
		tst.AssertStatusCode(t, postBackendQueue(t, r, "nope", msg(chanA, threadA, ownerID)).Code, http.StatusUnauthorized)
	})
	expectQueueError := func(t *testing.T, rr *httptest.ResponseRecorder, want string) {
		t.Helper()
		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
		if got := tst.ParseResponse(rr)["error"]; got != want {
			t.Fatalf("unexpected error: %v, want %q", got, want)
		}
	}
	t.Run("unknown channel is rejected", func(t *testing.T) {
		expectQueueError(t, postBackendQueue(t, r, backendQueueTestSecret, msg(utility.GenerateUUID(), threadA, ownerID)), "channel does not exist")
	})
	t.Run("non-member sender is rejected", func(t *testing.T) {
		body := msg(chanA, threadA, outsiderID)
		body["org_id"] = otherOrgID
		expectQueueError(t, postBackendQueue(t, r, backendQueueTestSecret, body), "sender is not a member of this channel")
	})
	t.Run("member sender passes membership and needs a thread of this channel", func(t *testing.T) {
		expectQueueError(t, postBackendQueue(t, r, backendQueueTestSecret, msg(chanA, threadA, ownerID)), "thread does not belong to channel")
	})
	t.Run("agent from another org is rejected", func(t *testing.T) {
		body := msg(chanA, threadA, "")
		body["agent_name"] = "Foreign Agent"
		body["agent_id"] = foreignAgentID
		body["org_id"] = otherOrgID
		expectQueueError(t, postBackendQueue(t, r, backendQueueTestSecret, body), "agent is not installed in this organisation")
	})
	t.Run("agent installed in the channel org passes the agent check", func(t *testing.T) {
		body := msg(chanA, threadA, "")
		body["agent_name"] = "Local Agent"
		body["agent_id"] = localAgentID
		expectQueueError(t, postBackendQueue(t, r, backendQueueTestSecret, body), "thread does not belong to channel")
	})
}
