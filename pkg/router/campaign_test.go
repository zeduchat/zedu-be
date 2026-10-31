package router

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCampaignReportRequiresBackofficeAuthentication(t *testing.T) {
	previous := config.Config
	config.Config = &config.Configuration{Server: config.ServerConfiguration{Secret: "campaign-test-only"}}
	t.Cleanup(func() { config.Config = previous })
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Admin(r, "/api/v1", validator.New(), &storage.Database{}, nil)
	for _, header := range []string{"", "Bearer invalid"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/campaigns?from=2026-10-01&to=2026-10-31", nil)
		if header != "" {
			request.Header.Set("Authorization", header)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("unauthorized request got %d: %s", w.Code, w.Body.String())
		}
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"admin_id": "not-an-admin", "role": "member", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("campaign-test-only"))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/campaigns?from=2026-10-01&to=2026-10-31", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-admin got %d: %s", w.Code, w.Body.String())
	}
}

func TestCampaignReportAuthorizedResponse(t *testing.T) {
	previous := config.Config
	config.Config = &config.Configuration{Server: config.ServerConfiguration{Secret: "campaign-test-only"}}
	t.Cleanup(func() { config.Config = previous })
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"admin_id": "admin-test", "role": "admin", "access_uuid": "session-test", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("campaign-test-only"))
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT .*FROM "access_tokens"`).WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "is_live", "login_access_token"}).AddRow("session-test", "admin-test", true, token))
	mock.ExpectExec(`DELETE FROM "campaign_events"`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`WITH visits AS`).WillReturnRows(sqlmock.NewRows([]string{"source", "medium", "campaign", "visits", "leads", "registrations", "lead_rate", "registration_rate"}).AddRow("newsletter", "email", "launch", 10, 3, 2, 30.0, 20.0))
	r := gin.New()
	Admin(r, "/api/v1", validator.New(), &storage.Database{Postgresql: db}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/campaigns?from=2026-10-01&to=2026-10-31", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"registration_conversion_percent":20`) || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
