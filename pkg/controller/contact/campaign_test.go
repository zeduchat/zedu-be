package contact

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	contract "github.com/hngprojects/telex_be/internal/campaign"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCampaignVisitInvalidMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	controller := Controller{Db: &storage.Database{}}
	r.POST("/visit", controller.CampaignVisit)
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{`, http.StatusBadRequest},
		{`{"utm_source":"` + strings.Repeat("a", 3000) + `"}`, http.StatusBadRequest},
		{`{"consent":false}`, http.StatusNoContent},
		{`{"utm_source":12}`, http.StatusNoContent},
	} {
		request := httptest.NewRequest(http.MethodPost, "/visit", strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)
		if w.Code != tc.status {
			t.Fatalf("got %d, expected %d: %s", w.Code, tc.status, w.Body.String())
		}
	}
}

func TestCampaignVisitDatabaseFailureIsReportedForRetry(t *testing.T) {
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin().WillReturnError(errors.New("offline"))
	r := gin.New()
	controller := Controller{Db: &storage.Database{Postgresql: db}}
	r.POST("/visit", controller.CampaignVisit)
	body, _ := json.Marshal(contract.Attribution{Source: "email", Consent: true, VisitID: "c83e7b18-3be2-4ea1-a2b3-0b1c2d3e4f56", CapturedAt: time.Now().Add(-time.Minute).UnixMilli()})
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/visit", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, request)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatal(w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
