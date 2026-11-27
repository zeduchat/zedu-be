package test_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/agents"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func tokenOrgID(t *testing.T, token string) string {
	parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) {
		return []byte(config.GetConfig().Server.Secret), nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	claims, _ := parsed.Claims.(jwt.MapClaims)
	org, _ := claims["org_id"].(string)
	return org
}

func TestAgentTaskWritesAreOrgScoped(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	authCtrl := auth.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	tokenA := initialise(utility.GenerateUUID(), t, gin.Default(), authCtrl, false)
	tokenB := initialise(utility.GenerateUUID(), t, gin.Default(), authCtrl, false)
	orgA, orgB := tokenOrgID(t, tokenA), tokenOrgID(t, tokenB)
	if orgA == "" || orgB == "" || orgA == orgB {
		t.Fatalf("expected two distinct token orgs, got %q and %q", orgA, orgB)
	}

	ctrl := agents.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	g := r.Group("/api/v1/tasks", middleware.Authorize(db.Postgresql))
	g.POST("/:agent_id", ctrl.CreateAgentTasks)
	g.PUT("/:task_id/agents/:agent_id", ctrl.UpdateAgentTasks)
	g.DELETE("/:task_id/agents/:agent_id", ctrl.DeleteAgentTasks)
	do := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			json.NewEncoder(&buf).Encode(body)
		}
		req, _ := http.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	agentID := utility.GenerateUUID()
	t.Cleanup(func() { db.Postgresql.Where("agent_id = ?", agentID).Delete(&models.Task{}) })

	rr := do(http.MethodPost, "/api/v1/tasks/"+agentID, tokenA, map[string]any{"text": "org A task", "position": 1})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create task: %d %s", rr.Code, rr.Body.String())
	}
	data, _ := tst.ParseResponse(rr)["data"].(map[string]any)
	taskID, _ := data["id"].(string)
	if taskID == "" {
		t.Fatalf("no task id in %s", rr.Body.String())
	}
	path := fmt.Sprintf("/api/v1/tasks/%s/agents/%s", taskID, agentID)
	load := func() (models.Task, bool) {
		var task models.Task
		err := db.Postgresql.Where("id = ?", taskID).First(&task).Error
		return task, err == nil
	}

	if rr := do(http.MethodPut, path, tokenB, map[string]any{"text": "changed by B", "position": 9}); rr.Code != http.StatusNotFound {
		t.Fatalf("org B update: got %d want 404: %s", rr.Code, rr.Body.String())
	}
	if rr := do(http.MethodDelete, path, tokenB, nil); rr.Code != http.StatusNotFound {
		t.Fatalf("org B delete: got %d want 404: %s", rr.Code, rr.Body.String())
	}
	if task, ok := load(); !ok || task.Text != "org A task" || task.OrganisationID != orgA {
		t.Fatalf("org B changed org A's task: %+v (exists=%v)", task, ok)
	}

	if rr := do(http.MethodPut, path, tokenA, map[string]any{"text": "changed by A", "position": 2}); rr.Code != http.StatusOK {
		t.Fatalf("org A update: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	if task, _ := load(); task.Text != "changed by A" {
		t.Errorf("org A's update was not saved: %+v", task)
	}
	if rr := do(http.MethodDelete, path, tokenA, nil); rr.Code != http.StatusOK {
		t.Fatalf("org A delete: got %d want 200: %s", rr.Code, rr.Body.String())
	}
	if _, ok := load(); ok {
		t.Error("org A's delete did not remove the task")
	}
}
