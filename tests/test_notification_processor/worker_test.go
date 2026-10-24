package test_notification_processor

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	np "github.com/hngprojects/telex_be/services/notification_processor"
	"github.com/hngprojects/telex_be/utility"
)

func newLogger() *utility.Logger { return utility.NewLogger() }
func newMetrics() *np.Metrics    { return &np.Metrics{} }
func newTracker() *np.CountTracker {
	return np.NewCountTracker(nil, newLogger())
}

func newWorker(t *testing.T) (np.Worker, chan np.WorkerSlot) {
	t.Helper()
	pool := make(chan np.WorkerSlot, 1)
	w := np.NewWorker(1, pool, newMetrics(), newLogger())
	return w, pool
}

func TestWorker_ProcessesJobSuccessfully(t *testing.T) {
	w, pool := newWorker(t)
	processed := make(chan struct{}, 1)

	np.ProcessFunc = func(_ np.Job, _ *utility.Logger) error {
		processed <- struct{}{}
		return nil
	}
	t.Cleanup(func() { np.ProcessFunc = np.ProcessNotification })

	w.Start(nil, nil, newTracker())
	slot := <-pool
	slot.JobChannel <- np.Job{Notification: models.PushNotificationRecord{ChannelType: "channel"}}

	select {
	case <-processed:
		assert.Equal(t, int64(1), atomic.LoadInt64(&w.Metrics.SuccessfulJobs))
		assert.Equal(t, int64(0), atomic.LoadInt64(&w.Metrics.FailedJobs))
	case <-time.After(2 * time.Second):
		t.Fatal("job was not processed within deadline")
	}
}

func TestWorker_FailedJobIncrementsFailedMetric(t *testing.T) {
	w, pool := newWorker(t)
	done := make(chan struct{}, 1)

	np.ProcessFunc = func(_ np.Job, _ *utility.Logger) error {
		defer func() { done <- struct{}{} }()
		return errors.New("downstream error")
	}
	t.Cleanup(func() { np.ProcessFunc = np.ProcessNotification })

	w.Start(nil, nil, newTracker())
	slot := <-pool
	slot.JobChannel <- np.Job{Notification: models.PushNotificationRecord{ChannelType: "channel"}}

	select {
	case <-done:
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, int64(1), atomic.LoadInt64(&w.Metrics.FailedJobs))
		assert.Equal(t, int64(0), atomic.LoadInt64(&w.Metrics.SuccessfulJobs))
	case <-time.After(2 * time.Second):
		t.Fatal("failed job did not complete within deadline")
	}
}

func TestWorker_PanicDoesNotCrashLoop(t *testing.T) {
	w, pool := newWorker(t)
	callCount := int32(0)

	np.ProcessFunc = func(_ np.Job, _ *utility.Logger) error {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			panic("simulated panic")
		}
		return nil
	}
	t.Cleanup(func() { np.ProcessFunc = np.ProcessNotification })

	w.Start(nil, nil, newTracker())

	slot := <-pool
	slot.JobChannel <- np.Job{Notification: models.PushNotificationRecord{}}

	slot = <-pool // worker re-registered means the loop survived the panic
	slot.JobChannel <- np.Job{Notification: models.PushNotificationRecord{}}

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount), "loop must continue after panic")
	assert.Equal(t, int64(1), atomic.LoadInt64(&w.Metrics.SuccessfulJobs))
	assert.Equal(t, int64(1), atomic.LoadInt64(&w.Metrics.FailedJobs))
}

func TestWorker_StopDoesNotBlockOrLeak(t *testing.T) {
	w, pool := newWorker(t)
	np.ProcessFunc = func(_ np.Job, _ *utility.Logger) error { return nil }
	t.Cleanup(func() { np.ProcessFunc = np.ProcessNotification })

	w.Start(nil, nil, newTracker())
	<-pool

	done := make(chan struct{})
	go func() {
		w.Stop()
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Worker.Stop() blocked")
	}
}

func TestWorker_RetryCountIncrementedOnFailure(t *testing.T) {
	w, pool := newWorker(t)
	requeuedWith := make(chan models.PushNotificationRecord, 1)

	np.ProcessFunc = func(_ np.Job, _ *utility.Logger) error {
		return errors.New("transient error")
	}
	np.RequeueFunc = func(job np.Job) {
		requeuedWith <- job.Notification
	}
	t.Cleanup(func() {
		np.ProcessFunc = np.ProcessNotification
		np.RequeueFunc = nil
	})

	w.Start(nil, nil, newTracker())
	slot := <-pool
	slot.JobChannel <- np.Job{Notification: models.PushNotificationRecord{
		ChannelType: "channel",
		RetryCount:  0,
		MaxRetries:  3,
	}}

	select {
	case rec := <-requeuedWith:
		assert.Equal(t, 1, rec.RetryCount)
		assert.NotEmpty(t, rec.LastError)
	case <-time.After(2 * time.Second):
		t.Fatal("notification was not requeued within deadline")
	}
}
