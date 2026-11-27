package test_cronjobs

import (
	"errors"
	"testing"
	"time"

	"github.com/hngprojects/telex_be/cronjobs"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

// stopWithin calls StopCronJob and fails the test if it does not return in time.
func stopWithin(t *testing.T, jobName string, d time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cronjobs.StopCronJob(jobName) }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		t.Fatalf("StopCronJob(%q) still blocked after %v", jobName, d)
		return nil
	}
}

func TestStopCronJobNeverStarted(t *testing.T) {
	err := stopWithin(t, "test-stop-never-started", time.Second)
	if !errors.Is(err, cronjobs.ErrCronJobNotRunning) {
		t.Fatalf("err = %v, want ErrCronJobNotRunning", err)
	}
}

func TestStopCronJobAlreadyStopped(t *testing.T) {
	const jobName = "test-stop-already-stopped"
	cronjobs.RegisterCronJob(jobName, func(request.ExternalRequest, storage.Database) {}, 10*time.Millisecond)
	cronjobs.StartCronJob(request.ExternalRequest{}, storage.Database{}, jobName)

	if err := stopWithin(t, jobName, 2*time.Second); err != nil {
		t.Fatalf("first stop: unexpected error: %v", err)
	}
	err := stopWithin(t, jobName, time.Second)
	if !errors.Is(err, cronjobs.ErrCronJobNotRunning) {
		t.Fatalf("second stop: err = %v, want ErrCronJobNotRunning", err)
	}
}
