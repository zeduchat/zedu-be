// Package campaign defines the optional, consented acquisition contract.
package campaign

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

const Window = 30 * 24 * time.Hour

var label = regexp.MustCompile(`^[A-Za-z0-9 _./:-]{1,100}$`)
var visitor = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)

type Attribution struct {
	Source     string `json:"utm_source"`
	Medium     string `json:"utm_medium"`
	Campaign   string `json:"utm_campaign"`
	Content    string `json:"utm_content"`
	Term       string `json:"utm_term"`
	VisitID    string `json:"visit_id"`
	CapturedAt int64  `json:"captured_at"`
	Consent    bool   `json:"consent"`
}

// Bad optional metadata must never reject an otherwise valid signup/enquiry.
func (a *Attribution) UnmarshalJSON(data []byte) error {
	type plain Attribution
	var value plain
	if json.Unmarshal(data, &value) != nil {
		*a = Attribution{}
		return nil
	}
	*a = Attribution(value)
	for _, field := range []*string{&a.Source, &a.Medium, &a.Campaign, &a.Content, &a.Term} {
		*field = strings.TrimSpace(*field)
	}
	return nil
}

func (a Attribution) Valid(now time.Time) bool {
	captured := time.UnixMilli(a.CapturedAt)
	if !a.Consent || !visitor.MatchString(a.VisitID) || a.Source == "" || captured.After(now) || now.Sub(captured) > Window {
		return false
	}
	for _, value := range []string{a.Source, a.Medium, a.Campaign, a.Content, a.Term} {
		if value != "" && (!label.MatchString(value) || value != strings.TrimSpace(value)) {
			return false
		}
	}
	return true
}

func VisitID(a *Attribution) string {
	if a != nil && a.Valid(time.Now()) {
		return a.VisitID
	}
	return ""
}

func DateRange(from, to string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return start, start, err
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return start, end, err
	}
	end = end.AddDate(0, 0, 1)
	if !end.After(start) || end.Sub(start) > 90*24*time.Hour {
		return start, end, &rangeError{}
	}
	return start, end, nil
}

type rangeError struct{}

func (*rangeError) Error() string { return "date range must be between 1 and 90 days" }
