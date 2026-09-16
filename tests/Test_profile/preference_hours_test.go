package test_profile

import (
	"testing"
	"time"

	"github.com/hngprojects/telex_be/services/notification_pref"
)

func TestIsTimeWithinRangeActiveHours(t *testing.T) {
	loc := time.UTC
	now9AM := time.Date(2026, 9, 14, 9, 0, 0, 0, time.UTC)
	now11PM := time.Date(2026, 9, 14, 23, 0, 0, 0, time.UTC)
	now3AM := time.Date(2026, 9, 14, 3, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		rangeStr   string
		testTime   time.Time
		wantResult bool
	}{
		{
			name:       "9 AM within 08:00 - 17:00",
			rangeStr:   "08:00 - 17:00",
			testTime:   now9AM,
			wantResult: true,
		},
		{
			name:       "11 PM outside 08:00 - 17:00",
			rangeStr:   "08:00 - 17:00",
			testTime:   now11PM,
			wantResult: false,
		},
		{
			name:       "11 PM within overnight range 22:00 - 06:00",
			rangeStr:   "22:00 - 06:00",
			testTime:   now11PM,
			wantResult: true,
		},
		{
			name:       "3 AM within overnight range 22:00 - 06:00",
			rangeStr:   "22:00 - 06:00",
			testTime:   now3AM,
			wantResult: true,
		},
		{
			name:       "9 AM outside overnight range 22:00 - 06:00",
			rangeStr:   "22:00 - 06:00",
			testTime:   now9AM,
			wantResult: false,
		},
		{
			name:       "12-hour AM/PM format support",
			rangeStr:   "9:00 AM - 5:00 PM",
			testTime:   now9AM,
			wantResult: true,
		},
		{
			name:       "Empty string returns true",
			rangeStr:   "",
			testTime:   now9AM,
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := notificationpref.IsTimeWithinRange(tt.rangeStr, loc, tt.testTime)
			if got != tt.wantResult {
				t.Errorf("IsTimeWithinRange(%q) = %v; want %v", tt.rangeStr, got, tt.wantResult)
			}
		})
	}
}
