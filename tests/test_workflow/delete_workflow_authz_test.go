package test_workflow

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestDeleteWorkflowAuthz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	owner := newWfUser(t, db, logger, true)
	other := newWfUser(t, db, logger, true)
	member := newWfUser(t, db, logger, false)
	addWfMember(t, db, logger, owner, member.id)

	r := newWfRouter(db, logger)
	wfA := insertWorkflow(t, db, owner.org, owner.id)
	wfOwn := insertWorkflow(t, db, owner.org, owner.id)

	exists := func(id string) bool {
		var n int64
		db.Postgresql.Model(&models.Workflow{}).Where("id = ?", id).Count(&n)
		return n > 0
	}
	path := func(org, wf string) string { return fmt.Sprintf("/api/v1/organisations/%s/workflows/%s", org, wf) }

	cases := []struct {
		name  string
		path  string
		token string
		want  int
	}{
		{"outsider names the victim org", path(owner.org, wfA), other.token, http.StatusForbidden},
		{"outsider deletes through its own org", path(other.org, wfA), other.token, http.StatusNotFound},
		{"member without can_manage_workflows", path(owner.org, wfA), member.token, http.StatusForbidden},
		{"owner deletes unknown workflow", path(owner.org, utility.GenerateUUID()), owner.token, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := doWf(r, http.MethodDelete, tc.path, tc.token, nil)
			if rr.Code != tc.want {
				t.Fatalf("got %d want %d: %s", rr.Code, tc.want, rr.Body.String())
			}
			if !exists(wfA) {
				t.Fatal("workflow was deleted by an unauthorised request")
			}
		})
	}

	rr := doWf(r, http.MethodDelete, path(owner.org, wfOwn), owner.token, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("owner delete: got %d: %s", rr.Code, rr.Body.String())
	}
	if exists(wfOwn) {
		t.Fatal("owner delete returned 200 but the row remains")
	}

	db.Postgresql.Where("id = ?", wfA).Delete(&models.Workflow{})
}
