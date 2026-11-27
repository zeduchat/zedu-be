package test_threads

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// Ziki #1358: org thread metrics must only be served to org members, and must not name
// private channels the caller isn't in.
func TestChannelMetricsAccess(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	threadCtrl := thread.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	newUser := func(tag string) (string, string) {
		id := utility.GenerateUUID()
		data := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("metrics%s%v@qa.team", tag, id),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "test",
			LastName:    tag,
			Password:    "password",
			UserName:    fmt.Sprintf("metrics_%s%v", tag, id),
		}
		tst.SignupUser(t, gin.Default(), authCtrl, data, false)
		token := tst.GetLoginToken(t, gin.Default(), authCtrl, models.LoginRequestModel{Email: data.Email, Password: data.Password})
		return token, tst.GetUserIDFromToken(t, token, db)
	}
	newOrg := func(token string) string {
		id := utility.GenerateUUID()
		orgID, _, _ := tst.CreateOrganisation(t, gin.Default(), db, orgCtrl, models.CreateOrgRequestModel{
			Name: fmt.Sprintf("MetricsOrg%s", id), Description: "metrics", Email: fmt.Sprintf("metricsorg%v@qa.team", id),
			Type: "type1", Location: "wakanda", Country: "wakanda",
		}, token)
		return orgID
	}

	ownerToken, ownerID := newUser("owner")
	memberToken, memberID := newUser("member")
	outsiderToken, _ := newUser("outsider")

	orgID := newOrg(ownerToken)
	newOrg(outsiderToken) // the outsider belongs to a different organisation

	if err := db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", memberID, orgID).Error; err != nil {
		t.Fatalf("add member to org: %v", err)
	}

	publicID, _ := tst.CreateChannels(t, gin.Default(), channelCtrl, db, models.CreateChannelsRequest{
		Name: fmt.Sprintf("metricspub%s", utility.GenerateUUID()), Username: fmt.Sprintf("metricspub%s", utility.GenerateUUID()),
		OrganisationID: orgID,
	}, ownerToken)
	privateID, _ := tst.CreateChannels(t, gin.Default(), channelCtrl, db, models.CreateChannelsRequest{
		Name: fmt.Sprintf("metricspriv%s", utility.GenerateUUID()), Username: fmt.Sprintf("metricspriv%s", utility.GenerateUUID()),
		OrganisationID: orgID, IsPrivate: true,
	}, ownerToken)
	if publicID == "" || privateID == "" {
		t.Fatalf("channel setup failed: public=%q private=%q", publicID, privateID)
	}

	// The real handler chain from pkg/router/thread.go.
	r := gin.New()
	r.GET("/api/v1/threads/organisations/:org_id/metrics", middleware.Authorize(db.Postgresql), middleware.RequireOrgMember(db.Postgresql), threadCtrl.GetChannelCountInfo)
	// The same guard in front of a stub, to check a member is let through without
	// depending on Elasticsearch for the aggregation.
	r.GET("/guard/:org_id", middleware.Authorize(db.Postgresql), middleware.RequireOrgMember(db.Postgresql), func(c *gin.Context) { c.Status(http.StatusOK) })

	get := func(path, token string) int {
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}

	t.Run("non-member of the org is denied", func(t *testing.T) {
		tst.AssertStatusCode(t, get("/api/v1/threads/organisations/"+orgID+"/metrics?days=365", outsiderToken), http.StatusForbidden)
	})

	t.Run("unknown org is denied the same way", func(t *testing.T) {
		tst.AssertStatusCode(t, get("/api/v1/threads/organisations/"+utility.GenerateUUID()+"/metrics", outsiderToken), http.StatusForbidden)
	})

	t.Run("invalid org id is rejected", func(t *testing.T) {
		tst.AssertStatusCode(t, get("/api/v1/threads/organisations/not-a-uuid/metrics", ownerToken), http.StatusBadRequest)
	})

	t.Run("org members pass the guard", func(t *testing.T) {
		tst.AssertStatusCode(t, get("/guard/"+orgID, memberToken), http.StatusOK)
		tst.AssertStatusCode(t, get("/guard/"+orgID, ownerToken), http.StatusOK)
	})

	t.Run("private channel is hidden from a member who is not in it", func(t *testing.T) {
		ids, err := models.VisibleOrgChannelIDs(db.Postgresql, orgID, memberID)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(ids, publicID) {
			t.Errorf("public channel missing from %v", ids)
		}
		if slices.Contains(ids, privateID) {
			t.Errorf("private channel %s leaked to a non-member", privateID)
		}
	})

	t.Run("private channel is visible to its member", func(t *testing.T) {
		ids, err := models.VisibleOrgChannelIDs(db.Postgresql, orgID, ownerID)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(ids, publicID) || !slices.Contains(ids, privateID) {
			t.Errorf("owner should see both channels, got %v", ids)
		}
	})
}
