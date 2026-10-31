package campaign

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAttribution(t *testing.T) {
	now := time.Now()
	base := Attribution{Source: "newsletter", VisitID: "c83e7b18-3be2-4ea1-a2b3-0b1c2d3e4f56", CapturedAt: now.Add(-time.Minute).UnixMilli(), Consent: true}
	if !base.Valid(now) {
		t.Fatal("valid campaign rejected")
	}
	for name, mutate := range map[string]func(*Attribution){
		"no consent":    func(a *Attribution) { a.Consent = false },
		"expired":       func(a *Attribution) { a.CapturedAt = now.Add(-Window - time.Hour).UnixMilli() },
		"future":        func(a *Attribution) { a.CapturedAt = now.Add(time.Hour).UnixMilli() },
		"personal data": func(a *Attribution) { a.Source = "person@example.com" },
		"bad visitor":   func(a *Attribution) { a.VisitID = "bad" },
	} {
		t.Run(name, func(t *testing.T) {
			a := base
			mutate(&a)
			if a.Valid(now) {
				t.Fatal("invalid campaign accepted")
			}
		})
	}
}
func TestMalformedOptionalMetadata(t *testing.T) {
	for _, raw := range []string{`"bad"`, `[]`, `{"utm_source":5}`, `null`} {
		var a Attribution
		if err := json.Unmarshal([]byte(raw), &a); err != nil {
			t.Fatal(err)
		}
		if a.Valid(time.Now()) {
			t.Fatal("accepted malformed metadata")
		}
	}
}
func TestDateRange(t *testing.T) {
	start, end, err := DateRange("2026-10-01", "2026-10-31")
	if err != nil || end.Sub(start) != 31*24*time.Hour {
		t.Fatal(start, end, err)
	}
	for _, pair := range [][2]string{{"oops", "2026-10-31"}, {"2026-11-01", "2026-10-01"}, {"2026-01-01", "2026-12-31"}} {
		if _, _, err := DateRange(pair[0], pair[1]); err == nil {
			t.Fatal("invalid range accepted")
		}
	}
}
