package test_admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/agents"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/utility"
)

type agentListResponse struct {
	Data []struct {
		ID            string  `json:"id"`
		IntegrationID *string `json:"integration_id"`
		AppName       string  `json:"app_name"`
		CreditUsed    float64 `json:"credit_used"`
	} `json:"data"`
}

// TestBackofficeAgentsSortBy covers #1659: sort_by must never reach ORDER BY
// as raw SQL, and sort_by=credit_used must actually sort.
func TestBackofficeAgentsSortBy(t *testing.T) {
	r, _, logger, db := SetupAdminTestRouter()
	ctrl := agents.Controller{Db: db, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r.GET("/api/v1/backoffice/agents/all", middleware.AdminAuthorize(db.Postgresql), ctrl.GetAllCustomAgent)

	token := CreateSuperAdminAndGetToken(t, r, db)
	orgID := CreateOrganizationWithCredit(t, db.Postgresql, 0)

	prefix := "sorttest" + utility.RandomString(8)
	base := time.Now().Add(-time.Hour)
	// older has the higher credit use, so created_at desc and credit_used desc disagree.
	type row struct {
		name    string
		created time.Time
		credit  float64
	}
	rows := []row{
		{prefix + "-older", base, 50},
		{prefix + "-newer", base.Add(time.Minute), 5},
	}
	var oiIDs, intIDs []string
	for _, rw := range rows {
		oiID, intID := utility.GenerateUUID(), utility.GenerateUUID()
		oiIDs = append(oiIDs, oiID)
		intIDs = append(intIDs, intID)
		if err := db.Postgresql.Exec(
			`INSERT INTO organisation_integrations (id, org_id, integration_id, app_name, json_url, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, 'https://example.com/agent.json', true, ?, ?)`,
			oiID, orgID, intID, rw.name, rw.created, rw.created).Error; err != nil {
			t.Fatalf("insert organisation_integration: %v", err)
		}
		if err := db.Postgresql.Create(&models.CreditUsage{ID: utility.GenerateUUID(), OrganisationID: orgID, AgentID: intID, Amount: rw.credit}).Error; err != nil {
			t.Fatalf("insert credit usage: %v", err)
		}
	}
	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM credit_usages WHERE agent_id IN ?", intIDs)
		db.Postgresql.Exec("DELETE FROM organisation_integrations WHERE id IN ?", oiIDs)
	})

	get := func(t *testing.T, sortBy, sortOrder string) (int, agentListResponse) {
		t.Helper()
		q := url.Values{"is_system": {"false"}, "is_active": {"true"}, "search": {prefix}}
		if sortBy != "" {
			q.Set("sort_by", sortBy)
		}
		if sortOrder != "" {
			q.Set("sort_order", sortOrder)
		}
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/agents/all?"+q.Encode(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var body agentListResponse
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		return rr.Code, body
	}
	names := func(b agentListResponse) []string {
		out := make([]string, len(b.Data))
		for i, d := range b.Data {
			out[i] = d.AppName
		}
		return out
	}

	cases := []struct {
		name      string
		sortBy    string
		sortOrder string
		want      []string
	}{
		{"default sort is created_at desc", "", "", []string{rows[1].name, rows[0].name}},
		{"allowed column ascending", "updated_at", "asc", []string{rows[0].name, rows[1].name}},
		{"name maps to app_name", "name", "asc", []string{rows[1].name, rows[0].name}},
		{"unknown column falls back", "not_a_column", "", []string{rows[1].name, rows[0].name}},
		{"SQL expression is not executed", "created_at desc, (SELECT 1/0)", "", []string{rows[1].name, rows[0].name}},
		// Falls back to created_at asc; on unpatched code this sorted by app_name.
		{"subquery is not executed", "(SELECT CASE WHEN true THEN app_name END)", "asc", []string{rows[0].name, rows[1].name}},
		{"credit_used sorts by credit", "credit_used", "desc", []string{rows[0].name, rows[1].name}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, body := get(t, tc.sortBy, tc.sortOrder)
			if code != http.StatusOK {
				t.Fatalf("status = %d, want 200", code)
			}
			if got := names(body); fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Fatalf("order = %v, want %v", got, tc.want)
			}
		})
	}
}
