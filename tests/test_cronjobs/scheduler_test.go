package test_cronjobs

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/hngprojects/telex_be/cronjobs"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

const longInterval = time.Hour

// countingJob registers a job with a long interval that counts its runs.
func countingJob(jobName string) *atomic.Int32 {
	runs := &atomic.Int32{}
	cronjobs.RegisterCronJob(jobName, func(request.ExternalRequest, storage.Database) { runs.Add(1) }, longInterval)
	return runs
}

func waitForRuns(t *testing.T, jobName string, runs *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for runs.Load() < want {
		if time.Now().After(deadline) {
			t.Fatalf("%s ran %d times, want %d", jobName, runs.Load(), want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// returnsWithin fails the test if fn does not return in time.
func returnsWithin(t *testing.T, what string, d time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { fn(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s still blocked after %v (interval is %v)", what, d, longInterval)
	}
}

func TestStopCronJobDuringInterval(t *testing.T) {
	const jobName = "test-stop-during-interval"
	runs := countingJob(jobName)
	cronjobs.StartCronJob(request.ExternalRequest{}, storage.Database{}, jobName)
	waitForRuns(t, jobName, runs, 1)

	returnsWithin(t, "StopCronJob", time.Second, func() { cronjobs.StopCronJob(jobName) })
}

func TestRestartCronJobDuringInterval(t *testing.T) {
	const jobName = "test-restart-during-interval"
	runs := countingJob(jobName)
	cronjobs.StartCronJob(request.ExternalRequest{}, storage.Database{}, jobName)
	waitForRuns(t, jobName, runs, 1)

	returnsWithin(t, "RestartCronJob", time.Second, func() {
		cronjobs.RestartCronJob(request.ExternalRequest{}, storage.Database{}, jobName)
	})
	waitForRuns(t, jobName, runs, 2)
	returnsWithin(t, "StopCronJob", time.Second, func() { cronjobs.StopCronJob(jobName) })
}

func TestSetupCronJobsStartsEachJobOnce(t *testing.T) {
	jobs := []string{"test-setup-first", "test-setup-second"}
	runs := map[string]*atomic.Int32{}
	for _, jobName := range jobs {
		runs[jobName] = countingJob(jobName)
	}

	go cronjobs.SetupCronJobs(request.ExternalRequest{}, storage.Database{}, jobs)

	for _, jobName := range jobs {
		waitForRuns(t, jobName, runs[jobName], 1)
	}
	time.Sleep(200 * time.Millisecond)
	for _, jobName := range jobs {
		if got := runs[jobName].Load(); got != 1 {
			t.Errorf("%s ran %d times after setup, want 1", jobName, got)
		}
		returnsWithin(t, "StopCronJob("+jobName+")", time.Second, func() { cronjobs.StopCronJob(jobName) })
	}
}

func TestCronJobNeverRunsConcurrentlyWithItself(t *testing.T) {
	const jobName = "test-no-concurrent-runs"
	var active, maxActive atomic.Int32
	cronjobs.RegisterCronJob(jobName, func(request.ExternalRequest, storage.Database) {
		n := active.Add(1)
		for {
			m := maxActive.Load()
			if n <= m || maxActive.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
		active.Add(-1)
	}, longInterval)

	// Two schedulers for the same job, as happened when SetupCronJobs started
	// a job and then restarted it.
	cronjobs.StartCronJob(request.ExternalRequest{}, storage.Database{}, jobName)
	cronjobs.StartCronJob(request.ExternalRequest{}, storage.Database{}, jobName)
	time.Sleep(300 * time.Millisecond)

	if got := maxActive.Load(); got != 1 {
		t.Errorf("job ran %d times concurrently, want 1", got)
	}
}
