package test_channel

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestChannelSearchAccessAndNameLookup(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()
	validatorRef := validator.New()
	currUUID := utility.GenerateUUID()
	userData := models.CreateUserRequestModel{
		Email: fmt.Sprintf("channelsearch_%s@example.test", currUUID), Password: "password",
		FirstName: "Search", LastName: "Tester", UserName: "search_" + currUUID,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
	}
	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	var userID, orgID, otherOrgID string
	var fixtureChannelIDs []string
	t.Cleanup(func() {
		cleanup := func(err error) {
			if err != nil {
				t.Errorf("clean up channel search test data: %v", err)
			}
		}
		if userID == "" {
			var user models.User
			if err := db.Postgresql.Where("email = ?", userData.Email).First(&user).Error; err == nil {
				userID = user.ID
			}
		}
		if userID != "" {
			cleanup(db.Postgresql.Where("user_id = ?", userID).Delete(&models.UserChannels{}).Error)
			cleanup(db.Postgresql.Where("owner_id = ?", userID).Delete(&models.AccessToken{}).Error)
			cleanup(db.Postgresql.Unscoped().Where("user_id = ?", userID).Delete(&models.LoginActivity{}).Error)
			cleanup(db.Postgresql.Where("actor_id = ?", userID).Delete(&models.AuditLog{}).Error)
			cleanup(db.Postgresql.Unscoped().Where("userid = ?", userID).Delete(&models.Profile{}).Error)
			if orgID != "" || otherOrgID != "" {
				orgIDs := make([]string, 0, 2)
				if orgID != "" {
					orgIDs = append(orgIDs, orgID)
				}
				if otherOrgID != "" {
					orgIDs = append(orgIDs, otherOrgID)
				}
				cleanup(db.Postgresql.Where("user_id = ? AND organisation_id IN ?", userID, orgIDs).Delete(&models.OrgUserManagement{}).Error)
				cleanup(db.Postgresql.Exec("DELETE FROM user_organisations WHERE user_id = ? AND organisation_id IN ?", userID, orgIDs).Error)
			}
		}
		if len(fixtureChannelIDs) > 0 {
			cleanup(db.Postgresql.Where("channels_id IN ?", fixtureChannelIDs).Delete(&models.UserChannels{}).Error)
			cleanup(db.Postgresql.Where("id IN ?", fixtureChannelIDs).Delete(&models.Channels{}).Error)
		}
		if userID != "" {
			cleanup(db.Postgresql.Model(&models.User{}).Where("id = ?", userID).Update("current_org", nil).Error)
		}
		if orgID != "" {
			cleanup(db.Postgresql.Unscoped().Where("id = ?", orgID).Delete(&models.Organisation{}).Error)
		}
		if otherOrgID != "" {
			cleanup(db.Postgresql.Unscoped().Where("id = ?", otherOrgID).Delete(&models.Organisation{}).Error)
		}
		if userID != "" {
			cleanup(db.Postgresql.Unscoped().Where("id = ?", userID).Delete(&models.User{}).Error)
		}
	})
	r := gin.Default()
	authRoutes := r.Group("/api/v1/auth")
	authRoutes.POST("/register", authCtrl.RegisterUser)
	authRoutes.POST("/login", authCtrl.LoginUser)
	tst.SignupUser(t, r, authCtrl, userData, false)
	login := models.LoginRequestModel{Email: userData.Email, Password: userData.Password}
	token := tst.GetLoginToken(t, r, authCtrl, login)
	var user models.User
	if err := db.Postgresql.Where("email = ?", userData.Email).First(&user).Error; err != nil {
		t.Fatalf("load test user: %v", err)
	}
	userID = user.ID
	orgID = utility.GenerateUUID()
	otherOrgID = utility.GenerateUUID()
	var plan models.Plan
	if err := db.Postgresql.Where("name = ?", "Free").First(&plan).Error; err != nil {
		t.Fatalf("load free plan for test organisations: %v", err)
	}
	orgs := []models.Organisation{
		{ID: orgID, Name: "ChannelSearch" + currUUID, Description: "search test", Email: userData.Email, Type: "test", Location: "test", Country: "test", OwnerID: userID, OrgPlanID: plan.ID},
		{ID: otherOrgID, Name: "Other" + currUUID, Email: "other_" + currUUID + "@example.test", Type: "test", Location: "test", Country: "test", OwnerID: userID, OrgPlanID: plan.ID},
	}
	for _, org := range orgs {
		if err := db.Postgresql.Create(&org).Error; err != nil {
			t.Fatalf("create test organisation %q: %v", org.ID, err)
		}
	}
	user.CurrentOrg = uuid.FromStringOrNil(orgID)
	if err := db.Postgresql.Model(&user).Update("current_org", orgID).Error; err != nil {
		t.Fatalf("set test user's current organisation: %v", err)
	}
	if err := user.AddUserToOrganisation(db.Postgresql, &user, []any{&orgs[0]}); err != nil {
		t.Fatalf("add test user to test organisation: %v", err)
	}
	var ownerRole models.OrgRole
	if err := db.Postgresql.Where("LOWER(name) = LOWER(?)", models.OrgRoleNameOwner).First(&ownerRole).Error; err != nil {
		t.Fatalf("load owner role for test organisation membership: %v", err)
	}
	if err := db.Postgresql.Create(&models.OrgUserManagement{UserID: userID, OrganisationID: orgID, Status: "active", RoleID: ownerRole.ID}).Error; err != nil {
		t.Fatalf("create test organisation membership: %v", err)
	}
	token = tst.GetLoginToken(t, r, authCtrl, login)
	prefix := "scope" + currUUID[:8]
	type fixture struct {
		name     string
		orgID    string
		private  bool
		archived bool
		member   bool
	}
	fixtures := []fixture{
		{name: prefix + "-public-member", orgID: orgID, member: true},
		{name: prefix + "-private-member", orgID: orgID, private: true, member: true},
		{name: prefix + "-public-outsider", orgID: orgID},
		{name: prefix + "-private-outsider", orgID: orgID, private: true},
		{name: prefix + "-archived-member", orgID: orgID, member: true, archived: true},
		{name: prefix + "-foreign-org-member", orgID: otherOrgID, member: true},
		{name: prefix + "-percent%literal", orgID: orgID, member: true},
		{name: prefix + "-under_score", orgID: orgID, member: true},
		{name: prefix + `-back\slash`, orgID: orgID, member: true},
		{name: prefix + "-percentXliteral", orgID: orgID, member: true},
		{name: prefix + "-underXscore", orgID: orgID, member: true},
		{name: prefix + `-backXslash`, orgID: orgID, member: true},
	}
	for _, f := range fixtures {
		id := utility.GenerateUUID()
		ch := models.Channels{ID: id, Name: f.name, Description: "search fixture", OrganisationID: f.orgID, OwnerId: userID, IsPrivate: f.private, Archived: f.archived}
		if err := db.Postgresql.Create(&ch).Error; err != nil {
			t.Fatalf("create channel %q: %v", f.name, err)
		}
		fixtureChannelIDs = append(fixtureChannelIDs, id)
		if f.member {
			if err := db.Postgresql.Create(&models.UserChannels{ChannelsID: id, UserID: userID, Username: userData.UserName}).Error; err != nil {
				t.Fatalf("add user to channel %q: %v", f.name, err)
			}
		}
	}

	channelRoutes := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	channelRoutes.GET("/search/:channelName", channelCtrl.SearchChannelsByNames)
	channelRoutes.GET("/name/:channelName", channelCtrl.GetChannelsByName)
	requestChannels := func(path string) (*httptest.ResponseRecorder, string) {
		req, err := http.NewRequest(http.MethodGet, path, nil)
		if err != nil {
			t.Fatalf("make request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr, rr.Body.String()
	}
	assertFound := func(t *testing.T, body string, names ...string) {
		t.Helper()
		for _, name := range names {
			if !strings.Contains(body, name) {
				t.Errorf("expected %q in response: %s", name, body)
			}
		}
	}
	assertAbsent := func(t *testing.T, body string, names ...string) {
		t.Helper()
		for _, name := range names {
			if strings.Contains(body, name) {
				t.Errorf("did not expect %q in response: %s", name, body)
			}
		}
	}

	t.Run("search applies organisation membership and archived filters from JWT", func(t *testing.T) {
		rr, body := requestChannels("/api/v1/channels/search/" + url.PathEscape(prefix))
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		assertFound(t, body, fixtures[0].name, fixtures[1].name)
		assertAbsent(t, body, fixtures[2].name, fixtures[3].name, fixtures[4].name, fixtures[5].name)
		assertAbsent(t, body, userData.Email, `"users"`, `"email"`)
	})

	t.Run("partial search finds a member channel", func(t *testing.T) {
		rr, body := requestChannels("/api/v1/channels/search/" + url.PathEscape(prefix+"-private"))
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		assertFound(t, body, fixtures[1].name)
		assertAbsent(t, body, fixtures[3].name)
	})

	for _, special := range []struct{ literal, expected, similar string }{
		{"%", "percent%literal", "percentXliteral"}, {"_", "under_score", "underXscore"}, {"\\", `back\slash`, "backXslash"},
	} {
		special := special
		t.Run("LIKE metacharacter "+special.literal+" is literal", func(t *testing.T) {
			rr, body := requestChannels("/api/v1/channels/search/" + url.PathEscape(prefix+"-"+special.literal))
			tst.AssertStatusCode(t, rr.Code, http.StatusOK)
			assertFound(t, body, prefix+"-"+special.expected)
			assertAbsent(t, body, prefix+"-"+special.similar)
		})
	}

	t.Run("name lookup requires caller organisation membership and active channel", func(t *testing.T) {
		for _, f := range fixtures[:6] {
			path := "/api/v1/channels/name/" + url.PathEscape(f.name)
			rr, body := requestChannels(path)
			if f.member && f.orgID == orgID && !f.archived {
				tst.AssertStatusCode(t, rr.Code, http.StatusOK)
				assertFound(t, body, f.name)
			} else {
				tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
				assertAbsent(t, body, f.name)
			}
		}
	})
}
