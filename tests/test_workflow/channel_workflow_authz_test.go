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

func TestChannelWorkflowAuthz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	owner := newWfUser(t, db, logger, true)
	other := newWfUser(t, db, logger, true)
	member := newWfUser(t, db, logger, false)
	addWfMember(t, db, logger, owner, member.id)

	channelA := models.Channels{ID: utility.GenerateUUID(), Name: "wf-authz-" + utility.GenerateUUID()[:8], Description: "workflow authz test", OrganisationID: owner.org, OwnerId: owner.id}
	if err := db.Postgresql.Create(&channelA).Error; err != nil {
		t.Fatalf("insert channel: %v", err)
	}
	wfA := insertWorkflow(t, db, owner.org, owner.id)
	wfB := insertWorkflow(t, db, other.org, other.id)
	defer func() {
		db.Postgresql.Where("channel_id = ?", channelA.ID).Delete(&models.ChannelWorkflow{})
		db.Postgresql.Where("id IN ?", []string{wfA, wfB}).Delete(&models.Workflow{})
		db.Postgresql.Exec("DELETE FROM channels WHERE id = ?", channelA.ID)
	}()

	r := newWfRouter(db, logger)
	mapped := func(wf string) bool {
		var n int64
		db.Postgresql.Model(&models.ChannelWorkflow{}).Where("channel_id = ? AND workflow_id = ?", channelA.ID, wf).Count(&n)
		return n > 0
	}
	post := func(token, ch, wf string) int {
		return doWf(r, http.MethodPost, "/api/v1/channel-workflows", token, map[string]string{"channel_id": ch, "workflow_id": wf}).Code
	}
	del := func(token, ch, wf string) int {
		return doWf(r, http.MethodDelete, fmt.Sprintf("/api/v1/channel-workflows/%s/channels/%s", wf, ch), token, nil).Code
	}
	expect := func(name string, got, want int) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %d want %d", name, got, want)
		}
	}

	// Attach is refused for anyone who cannot manage workflows in the channel's org.
	expect("outsider attaches victim workflow", post(other.token, channelA.ID, wfA), http.StatusNotFound)
	expect("outsider attaches own workflow to victim channel", post(other.token, channelA.ID, wfB), http.StatusNotFound)
	expect("member without can_manage_workflows attaches", post(member.token, channelA.ID, wfA), http.StatusForbidden)
	expect("owner attaches another org's workflow", post(owner.token, channelA.ID, wfB), http.StatusNotFound)
	expect("malformed ids", post(owner.token, "x", wfA), http.StatusBadRequest)
	if mapped(wfA) || mapped(wfB) {
		t.Fatal("an unauthorised attach created a mapping")
	}

	expect("owner attaches", post(owner.token, channelA.ID, wfA), http.StatusCreated)
	if !mapped(wfA) {
		t.Fatal("owner attach did not create the mapping")
	}

	// The toggle and the delete route cannot switch it off for outsiders either.
	expect("outsider toggles off", post(other.token, channelA.ID, wfA), http.StatusNotFound)
	expect("outsider detaches", del(other.token, channelA.ID, wfA), http.StatusNotFound)
	expect("member detaches", del(member.token, channelA.ID, wfA), http.StatusForbidden)
	if !mapped(wfA) {
		t.Fatal("an unauthorised request removed the mapping")
	}

	expect("owner detaches", del(owner.token, channelA.ID, wfA), http.StatusOK)
	if mapped(wfA) {
		t.Fatal("owner detach did not remove the mapping")
	}
	expect("owner toggles on", post(owner.token, channelA.ID, wfA), http.StatusCreated)
	expect("owner toggles off", post(owner.token, channelA.ID, wfA), http.StatusOK)
	if mapped(wfA) {
		t.Fatal("owner toggle-off did not remove the mapping")
	}
}
