package test_pagination

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

func paginationFor(t *testing.T, rawQuery string) postgresql.Pagination {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?"+rawQuery, nil)
	return postgresql.GetPagination(c)
}

func TestGetPaginationClampsLimitAndPage(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		wantPage  int
		wantLimit int
	}{
		{"defaults", "", 1, 20},
		{"valid values pass through", "page=3&limit=50", 3, 50},
		{"limit zero falls back to default", "limit=0", 1, 20},
		{"negative limit falls back to default", "limit=-5", 1, 20},
		{"limit above max is capped", "limit=999999", 1, 1000},
		{"largest client limit is kept", "limit=1000", 1, 1000},
		{"page zero becomes first page", "page=0&limit=10", 1, 10},
		{"non-numeric values use defaults", "page=x&limit=y", 1, 20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := paginationFor(t, tc.query)
			if got.Page != tc.wantPage || got.Limit != tc.wantLimit {
				t.Fatalf("GetPagination(%q) = page %d limit %d, want page %d limit %d",
					tc.query, got.Page, got.Limit, tc.wantPage, tc.wantLimit)
			}
		})
	}
}

func TestNormalizePaginationNeverReturnsZeroLimit(t *testing.T) {
	for _, limit := range []int{-1, 0, 1, 1000, 1001} {
		got := postgresql.NormalizePagination(postgresql.Pagination{Page: 1, Limit: limit})
		if got.Limit < 1 || got.Limit > 1000 {
			t.Fatalf("NormalizePagination limit %d gave %d, want 1..1000", limit, got.Limit)
		}
	}
}
