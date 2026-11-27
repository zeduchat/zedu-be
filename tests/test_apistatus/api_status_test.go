package test_apistatus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/router"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

const apiStatusTestSecret = "api-status-test-secret"

func statusReport(group string, executions int) []byte {
	type exec struct {
		Request struct {
			URL struct {
				Path []string `json:"path"`
			} `json:"url"`
		} `json:"request"`
		Response struct {
			ResponseTime int `json:"responseTime"`
		} `json:"response"`
	}
	execs := make([]exec, executions)
	for i := range execs {
		execs[i].Request.URL.Path = []string{"api", "v1", group}
		execs[i].Response.ResponseTime = 100
	}
	body, _ := json.Marshal(map[string]any{"run": map[string]any{"executions": execs}})
	return body
}

func postReport(t *testing.T, r *gin.Engine, secret, filename string, report []byte) int {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("result_file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	fw.Write(report)
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/api-status", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if secret != "" {
		req.Header.Set(router.ApiStatusSecretHeader, secret)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr.Code
}

func TestPostAPIStatusRequiresSecretAndUpserts(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	config.GetConfig().Server.ApiStatusSecret = apiStatusTestSecret
	r := gin.New()
	router.ApiStatus(r, "/api/v1", validator.New(), db, logger)

	group := "st" + strings.ReplaceAll(utility.GenerateUUID(), "-", "")[:12]
	apiGroup := fmt.Sprintf("%s API", group)
	countRows := func() int64 {
		var n int64
		db.Postgresql.Model(&models.APIStatus{}).Where("api_group = ?", apiGroup).Count(&n)
		return n
	}

	t.Run("no secret is rejected", func(t *testing.T) {
		tst.AssertStatusCode(t, postReport(t, r, "", "report.json", statusReport(group, 1)), http.StatusUnauthorized)
	})
	t.Run("wrong secret is rejected", func(t *testing.T) {
		tst.AssertStatusCode(t, postReport(t, r, "nope", "report.json", statusReport(group, 1)), http.StatusUnauthorized)
		if n := countRows(); n != 0 {
			t.Fatalf("rejected reports must not write rows, got %d", n)
		}
	})
	t.Run("too many executions is refused and writes nothing", func(t *testing.T) {
		tst.AssertStatusCode(t, postReport(t, r, apiStatusTestSecret, "report.json", statusReport(group, 5001)), http.StatusInternalServerError)
		if n := countRows(); n != 0 {
			t.Fatalf("expected no rows, got %d", n)
		}
	})
	t.Run("valid report twice keeps one row per group", func(t *testing.T) {
		tst.AssertStatusCode(t, postReport(t, r, apiStatusTestSecret, "report.json", statusReport(group, 2)), http.StatusOK)
		tst.AssertStatusCode(t, postReport(t, r, apiStatusTestSecret, "report.json", statusReport(group, 3)), http.StatusOK)
		if n := countRows(); n != 1 {
			t.Fatalf("expected exactly one row for %q, got %d", apiGroup, n)
		}
	})
	t.Run("client file name is never used as a path", func(t *testing.T) {
		name := fmt.Sprintf("../apistatus-%s.json", group)
		tst.AssertStatusCode(t, postReport(t, r, apiStatusTestSecret, name, statusReport(group, 1)), http.StatusOK)
		for _, p := range []string{filepath.Join("tmp", name), name} {
			if _, err := os.Stat(p); err == nil {
				os.Remove(p)
				t.Fatalf("upload was written to disk at %s", p)
			}
		}
	})
}
