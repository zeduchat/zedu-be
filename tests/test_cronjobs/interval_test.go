package test_cronjobs

import (
	"testing"
	"time"

	"github.com/hngprojects/telex_be/cronjobs"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

func TestUpdateCronJobIntervalBases(t *testing.T) {
	const jobName = "test-interval-bases"
	cronjobs.RegisterCronJob(jobName, func(request.ExternalRequest, storage.Database) {}, time.Minute)

	day := 24 * time.Hour
	cases := []struct {
		base   string
		number int
		want   time.Duration
	}{
		{"second", 2, 2 * time.Second},
		{"minute", 3, 3 * time.Minute},
		{"hour", 4, 4 * time.Hour},
		{"day", 2, 2 * day},
		{"week", 2, 14 * day},
		{"month", 1, 30 * day},
		{"month", 3, 90 * day},
		{"year", 1, 365 * day},
		{"year", 2, 730 * day},
	}
	for _, c := range cases {
		if err := cronjobs.UpdateCronJobInterval(request.ExternalRequest{}, storage.Database{}, jobName, c.number, c.base); err != nil {
			t.Fatalf("%d %s: unexpected error: %v", c.number, c.base, err)
		}
		got, ok := cronjobs.CronJobInterval(jobName)
		if !ok {
			t.Fatalf("job %s not registered", jobName)
		}
		if got != c.want {
			t.Errorf("%d %s: interval = %v (%.1f days), want %v (%.1f days)", c.number, c.base, got, got.Hours()/24, c.want, c.want.Hours()/24)
		}
	}
}
