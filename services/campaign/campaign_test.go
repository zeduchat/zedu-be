package campaign

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	contract "github.com/hngprojects/telex_be/internal/campaign"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func mockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		conn.Close()
	})
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db, mock
}
func sample() *contract.Attribution {
	return &contract.Attribution{Source: "email", Campaign: "launch", VisitID: "c83e7b18-3be2-4ea1-a2b3-0b1c2d3e4f56", CapturedAt: time.Now().Add(-time.Minute).UnixMilli(), Consent: true}
}

func TestRecordDeduplicatesServerIdentity(t *testing.T) {
	db, mock := mockDB(t)
	a := sample()
	eventID := fmt.Sprintf("%x", sha256.Sum256([]byte("sign_up:user-123")))
	for i := 0; i < 2; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "campaign_events".*ON CONFLICT DO NOTHING`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`SELECT .*FROM "campaign_events"`).WillReturnRows(sqlmock.NewRows([]string{"id", "visit_id", "kind", "source", "campaign_name", "created_at"}).AddRow("visit:"+a.VisitID, a.VisitID, "visit", "email", "launch", time.UnixMilli(a.CapturedAt)))
		mock.ExpectExec(`INSERT INTO "campaign_events".*ON CONFLICT DO NOTHING`).WithArgs(eventID, a.VisitID, "sign_up", "email", "", "launch", "", "", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, int64(1-i)))
		mock.ExpectCommit()
		if err := Record(db, a, "sign_up", "user-123"); err != nil {
			t.Fatal(err)
		}
	}
}
func TestInvalidOptionalAttributionDoesNotAccessDatabase(t *testing.T) {
	if err := Record(nil, nil, "sign_up", "user"); err != nil {
		t.Fatal(err)
	}
	a := sample()
	a.Consent = false
	if err := Record(nil, a, "sign_up", "user"); err != nil {
		t.Fatal(err)
	}
}
func TestWriteFailureRollsBack(t *testing.T) {
	db, mock := mockDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "campaign_events"`).WillReturnError(errors.New("offline"))
	mock.ExpectRollback()
	if err := Record(db, sample(), "visit", ""); err == nil {
		t.Fatal("expected failure for logging/retry")
	}
}
func TestReportAggregatesAttributedCohorts(t *testing.T) {
	db, mock := mockDB(t)
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	mock.ExpectQuery(`WITH visits AS`).WithArgs(start, end).WillReturnRows(sqlmock.NewRows([]string{"source", "medium", "campaign", "visits", "leads", "registrations", "lead_rate", "registration_rate"}).AddRow("email", "newsletter", "launch", 10, 3, 2, 30.0, 20.0))
	rows, err := Report(db, start, end)
	if err != nil || len(rows) != 1 || rows[0].RegistrationRate != 20 || rows[0].Leads != 3 {
		t.Fatal(rows, err)
	}
}
