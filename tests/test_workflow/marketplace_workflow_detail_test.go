package test_workflow

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestPublicMarketplaceWorkflowDetail(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	gw := models.GeneralWorkflow{ID: utility.GenerateUUID(), Name: "marketplace detail test", Description: "public detail"}
	if err := db.Postgresql.Create(&gw).Error; err != nil {
		t.Fatalf("insert general workflow: %v", err)
	}
	defer db.Postgresql.Where("id = ?", gw.ID).Delete(&models.GeneralWorkflow{})

	r := newWfRouter(db, logger)

	rr := doWf(r, http.MethodGet, "/api/v1/workflows/"+gw.ID, "", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("anonymous GET: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	data, _ := tst.ParseResponse(rr)["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected one workflow, got %v", rr.Body.String())
	}
	if got, _ := data[0].(map[string]any)["id"].(string); got != gw.ID {
		t.Fatalf("got workflow %q want %q", got, gw.ID)
	}

	if rr := doWf(r, http.MethodGet, "/api/v1/workflows/not-a-uuid", "", nil); rr.Code != http.StatusBadRequest {
		t.Fatalf("malformed id: got %d want 400", rr.Code)
	}
}
