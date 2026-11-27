package test_channel

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestChannelSearchScope covers #1171: channel search and lookup by name only return
// the caller's organisation's channels, hide private channels the caller hasn't
// joined and archived channels, don't embed member records, and treat LIKE
// wildcards literally.
func TestChannelSearchScope(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	pg := db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	newUser := func(tag string) models.User {
		u := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "Scope " + tag,
			Email:    fmt.Sprintf("chscope_%s_%v@qa.team", tag, currUUID),
			Password: password,
		}
		if err := pg.Create(&u).Error; err != nil {
			t.Fatalf("create user %s: %v", tag, err)
		}
		return u
	}
	userA, userA2, userB := newUser("a"), newUser("a2"), newUser("b")

	newOrg := func(tag string, owner models.User, members ...models.User) string {
		o := models.Organisation{
			ID:      utility.GenerateUUID(),
			Name:    fmt.Sprintf("ChScope%s%v", tag, currUUID),
			Email:   fmt.Sprintf("chscope_org_%s_%v@qa.team", tag, currUUID),
			OwnerID: owner.ID,
		}
		if err := pg.Create(&o).Error; err != nil {
			t.Fatalf("create org %s: %v", tag, err)
		}
		for _, m := range append([]models.User{owner}, members...) {
			if err := pg.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?) ON CONFLICT DO NOTHING", m.ID, o.ID).Error; err != nil {
				t.Fatalf("add org member: %v", err)
			}
		}
		return o.ID
	}
	orgA := newOrg("a", userA, userA2)
	orgB := newOrg("b", userB)

	term := "scope" + currUUID[len(currUUID)-12:]
	newChannel := func(name, org string, owner models.User, private, archived bool) string {
		ch := models.Channels{
			ID:             utility.GenerateUUID(),
			Name:           name,
			Description:    "scope test",
			OwnerId:        owner.ID,
			OrganisationID: org,
			IsPrivate:      private,
			Archived:       archived,
		}
		if err := pg.Create(&ch).Error; err != nil {
			t.Fatalf("create channel %s: %v", name, err)
		}
		if err := pg.Create(&models.UserChannels{ChannelsID: ch.ID, UserID: owner.ID, Username: owner.Name}).Error; err != nil {
			t.Fatalf("add channel member: %v", err)
		}
		return ch.ID
	}
	aPublic := newChannel(term, orgA, userA, false, false)
	aPrivate := newChannel(term+"-p", orgA, userA, true, false)
	aArchived := newChannel(term+"-old", orgA, userA, false, true)
	bPublic := newChannel(term, orgB, userB, false, false)
	bPrivate := newChannel(term+"-p", orgB, userB, true, false)
	_ = aArchived

	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()
	g := r.Group("/api/v1/channels", middleware.Authorize(pg))
	g.GET("/name/:channelName", channelCtrl.GetChannelsByName)
	g.GET("/search/:channelName", channelCtrl.SearchChannelsByNames)

	// Log in after the orgs exist; Authorize resolves the token's org to the user's org.
	token := func(u models.User) string {
		return tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: u.Email, Password: "password"})
	}
	tokA, tokA2, tokB := token(userA), token(userA2), token(userB)

	get := func(path, tok string) (int, []map[string]any) {
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		res := tst.ParseResponse(rr)
		var items []map[string]any
		if list, ok := res["data"].([]any); ok {
			for _, it := range list {
				if m, ok := it.(map[string]any); ok {
					items = append(items, m)
				}
			}
		}
		return rr.Code, items
	}
	ids := func(items []map[string]any) map[string]bool {
		out := map[string]bool{}
		for _, it := range items {
			if id, ok := it["channels_id"].(string); ok {
				out[id] = true
			}
			if _, ok := it["users"]; ok {
				t.Errorf("search item %v embeds users", it["channels_id"])
			}
		}
		return out
	}
	search := func(q string) string { return "/api/v1/channels/search/" + url.PathEscape(q) + "?limit=100" }

	cases := []struct {
		name    string
		tok     string
		path    string
		want    []string
		notWant []string
	}{
		{"org A creator sees own public and private", tokA, search(term), []string{aPublic, aPrivate}, []string{aArchived, bPublic, bPrivate}},
		{"org A non-member of private channel", tokA2, search(term), []string{aPublic}, []string{aPrivate, aArchived, bPublic, bPrivate}},
		{"org B user sees only org B", tokB, search(term), []string{bPublic, bPrivate}, []string{aPublic, aPrivate, aArchived}},
		{"percent is literal", tokB, search("%"), nil, []string{aPublic, aPrivate, aArchived}},
		{"underscore is literal", tokB, search("_"), nil, []string{aPublic, aPrivate, aArchived}},
		{"by name, org A", tokA2, "/api/v1/channels/name/" + term, []string{aPublic}, []string{bPublic}},
		{"by name, org B", tokB, "/api/v1/channels/name/" + term, []string{bPublic}, []string{aPublic}},
	}
	for _, tc := range cases {
		code, items := get(tc.path, tc.tok)
		if code != http.StatusOK {
			t.Errorf("%s: got status %d, want 200", tc.name, code)
			continue
		}
		got := ids(items)
		for _, id := range tc.want {
			if !got[id] {
				t.Errorf("%s: missing channel %s", tc.name, id)
			}
		}
		for _, id := range tc.notWant {
			if got[id] {
				t.Errorf("%s: unexpected channel %s", tc.name, id)
			}
		}
	}

	// A private channel name the caller can't see is a 404, not a leak.
	if code, _ := get("/api/v1/channels/name/"+term+"-p", tokA2); code != http.StatusNotFound {
		t.Errorf("by name, hidden private channel: got %d, want 404", code)
	}
}
