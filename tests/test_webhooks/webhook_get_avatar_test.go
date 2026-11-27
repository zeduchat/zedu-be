package test_webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/router"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestGetWebhookKeepsAvatarURL checks that GET webhook deliveries read
// avatar_url from the query string, so the thread they store carries it
// (#1652). Elasticsearch is replaced by a stub that records indexed documents.
func TestGetWebhookKeepsAvatarURL(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	var (
		mu   sync.Mutex
		docs []map[string]any
	)
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/_doc/") {
			body, _ := io.ReadAll(r.Body)
			var doc map[string]any
			if json.Unmarshal(body, &doc) == nil {
				mu.Lock()
				docs = append(docs, doc)
				mu.Unlock()
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"result":"created"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer stub.Close()
	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{stub.URL}})
	require.NoError(t, err)

	db := *storage.Connection()
	db.Elastic = es

	r := gin.Default()
	router.Webhook(r, "/api/v1", validator.New(), &db, logger)

	orgID := utility.GenerateUUID()
	require.NoError(t, db.Postgresql.Exec("INSERT INTO organisations (id, name, email, created_at) VALUES (?, ?, ?, now())",
		orgID, "avatar-"+orgID[:8], "avatar_"+orgID[:8]+"@qa.team").Error)
	channelID := utility.GenerateUUID()
	require.NoError(t, db.Postgresql.Exec(`INSERT INTO channels (id, name, description, organisation_id, created_at, " deleted_at") VALUES (?, ?, 'webhook avatar test', ?, now(), now())`,
		channelID, "avatar-"+channelID[:8], orgID).Error)
	slug := utility.GenerateUUID()[:12]
	hook := models.Webhook{
		ID:          utility.GenerateUUID(),
		WebhookName: "avatar",
		Status:      "active",
		OwnerId:     utility.GenerateUUID(),
		WebhookUrl:  "https://example.com/" + slug,
		WebhookSlug: slug,
		ChannelId:   channelID,
	}
	require.NoError(t, db.Postgresql.Create(&hook).Error)
	t.Cleanup(func() {
		db.Postgresql.Unscoped().Where("id = ?", hook.ID).Delete(&models.Webhook{})
		db.Postgresql.Exec("DELETE FROM channels WHERE id = ?", channelID)
		db.Postgresql.Exec("DELETE FROM organisations WHERE id = ?", orgID)
	})

	avatar := "https://example.com/avatar.png"
	q := url.Values{}
	q.Set("event_name", "deploy")
	q.Set("username", "ci")
	q.Set("status", "success")
	q.Set("message", "done")
	q.Set("avatar_url", avatar)

	for _, path := range []string{"/api/v1/webhooks/feed/" + slug, "/v1/webhooks/" + slug + "/return"} {
		t.Run(path, func(t *testing.T) {
			mu.Lock()
			docs = nil
			mu.Unlock()

			req, err := http.NewRequest(http.MethodGet, path+"?"+q.Encode(), nil)
			require.NoError(t, err)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			t.Logf("GET -> %d %s", rr.Code, rr.Body.String())

			mu.Lock()
			defer mu.Unlock()
			require.Len(t, docs, 1, "expected the webhook thread to be indexed")
			assert.Equal(t, channelID, docs[0]["channels_id"])
			assert.Equal(t, avatar, docs[0]["avatar_url"])
		})
	}
}
