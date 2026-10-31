package admin

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCampaignReportRejectsInvalidRangeWithoutDBAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	controller := Controller{}
	r.GET("/report", controller.CampaignReport)
	for _, query := range []string{"", "?from=bad&to=2026-10-31", "?from=2026-01-01&to=2026-12-31"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/report"+query, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
