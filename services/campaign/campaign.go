package campaign

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	contract "github.com/hngprojects/telex_be/internal/campaign"
	"github.com/hngprojects/telex_be/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Record is called only after the corresponding business operation succeeds.
// Unique server-derived event keys make provider callbacks/retries idempotent.
func Record(db *gorm.DB, a *contract.Attribution, kind, subject string) error {
	now := time.Now().UTC()
	if a == nil || !a.Valid(now) {
		return nil
	}
	if kind != "visit" && kind != "lead_submitted" && kind != "sign_up" {
		return fmt.Errorf("unknown conversion")
	}
	if kind != "visit" && subject == "" {
		return fmt.Errorf("missing conversion identity")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		visit := models.CampaignEvent{ID: "visit:" + a.VisitID, VisitID: a.VisitID, Kind: "visit", Source: a.Source, Medium: a.Medium, CampaignName: a.Campaign, Content: a.Content, Term: a.Term, CreatedAt: time.UnixMilli(a.CapturedAt).UTC()}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&visit).Error; err != nil {
			return err
		}
		if kind == "visit" {
			return nil
		}
		if err := tx.First(&visit, "id = ?", "visit:"+a.VisitID).Error; err != nil {
			return err
		}
		if now.Sub(visit.CreatedAt) > contract.Window {
			return nil
		}
		event := visit
		event.ID = fmt.Sprintf("%x", sha256.Sum256([]byte(kind+":"+subject)))
		event.Kind = kind
		event.CreatedAt = now
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&event).Error
	})
}

type Metrics struct {
	Source           string  `json:"source"`
	Medium           string  `json:"medium"`
	Campaign         string  `json:"campaign"`
	Visits           int64   `json:"visits"`
	Leads            int64   `json:"leads"`
	Registrations    int64   `json:"registrations"`
	LeadRate         float64 `json:"lead_conversion_percent"`
	RegistrationRate float64 `json:"registration_conversion_percent"`
}

// Cohort rates use distinct attributed visits as the denominator, not page views.
func Report(db *gorm.DB, start, end time.Time) ([]Metrics, error) {
	rows := []Metrics{}
	err := db.Raw(`WITH visits AS (
 SELECT * FROM campaign_events WHERE kind = 'visit' AND created_at >= ? AND created_at < ?
 ), conversions AS (
 SELECT visit_id, COUNT(*) FILTER (WHERE kind = 'lead_submitted') AS leads,
 COUNT(*) FILTER (WHERE kind = 'sign_up') AS registrations
 FROM campaign_events WHERE kind <> 'visit' GROUP BY visit_id
 ) SELECT v.source, v.medium, v.campaign_name AS campaign, COUNT(*) AS visits,
 COALESCE(SUM(c.leads),0) AS leads, COALESCE(SUM(c.registrations),0) AS registrations,
 100.0 * COUNT(*) FILTER (WHERE c.leads > 0) / COUNT(*) AS lead_rate,
 100.0 * COUNT(*) FILTER (WHERE c.registrations > 0) / COUNT(*) AS registration_rate
 FROM visits v LEFT JOIN conversions c ON c.visit_id = v.visit_id
 GROUP BY v.source,v.medium,v.campaign_name ORDER BY visits DESC,v.source,v.medium,v.campaign_name`, start, end).Scan(&rows).Error
	return rows, err
}

// Run on the normal application scheduler, or on reporting reads. No signup
// waits for retention maintenance and no personal signup record is deleted.
func PurgeExpired(db *gorm.DB) error {
	return db.Where("created_at < ?", time.Now().UTC().AddDate(0, 0, -90)).Delete(&models.CampaignEvent{}).Error
}
