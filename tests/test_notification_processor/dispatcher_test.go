package test_notification_processor

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	np "github.com/hngprojects/telex_be/services/notification_processor"
	"github.com/hngprojects/telex_be/utility"
)

func TestDispatcher_JobsReachWorkers(t *testing.T) {
	processed := int64(0)

	np.ProcessFunc = func(_ np.Job, _ *utility.Logger) error {
		atomic.AddInt64(&processed, 1)
		return nil
	}
	t.Cleanup(func() { np.ProcessFunc = np.ProcessNotification })

	d := np.NewDispatcher(3, 10, nil, newLogger(), newTracker())
	d.Run(nil, nil)

	for i := 0; i < 5; i++ {
		d.JobQueue <- np.Job{Notification: models.PushNotificationRecord{ChannelType: "channel"}}
		atomic.AddInt64(&d.JobQueueCount, 1)
	}

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, int64(5), atomic.LoadInt64(&processed))
}

func TestDispatcher_NoGoroutineLeakOnBusyWorkers(t *testing.T) {
	block := make(chan struct{})
	callCount := int32(0)

	np.ProcessFunc = func(_ np.Job, _ *utility.Logger) error {
		atomic.AddInt32(&callCount, 1)
		<-block
		return nil
	}
	t.Cleanup(func() {
		np.ProcessFunc = np.ProcessNotification
		close(block)
	})

	d := np.NewDispatcher(2, 10, nil, newLogger(), newTracker())
	d.Run(nil, nil)

	d.JobQueue <- np.Job{Notification: models.PushNotificationRecord{}}
	d.JobQueue <- np.Job{Notification: models.PushNotificationRecord{}}
	time.Sleep(100 * time.Millisecond)

	before := runtime.NumGoroutine()

	go func() { d.JobQueue <- np.Job{Notification: models.PushNotificationRecord{}} }()
	time.Sleep(100 * time.Millisecond)

	after := runtime.NumGoroutine()
	assert.LessOrEqual(t, after-before, 2, "dispatcher must not leak goroutines when workers are busy")
}

func TestDispatcher_Shutdown(t *testing.T) {
	np.ProcessFunc = func(_ np.Job, _ *utility.Logger) error { return nil }
	t.Cleanup(func() { np.ProcessFunc = np.ProcessNotification })

	d := np.NewDispatcher(2, 10, nil, newLogger(), newTracker())
	d.Run(nil, nil)

	done := make(chan struct{})
	go func() {
		d.Shutdown()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Dispatcher.Shutdown() blocked")
	}
}

func TestCountTracker_NeverBlocksNotificationPath(t *testing.T) {
	tracker := np.NewCountTracker(nil, newLogger())

	done := make(chan struct{})
	go func() {
		for i := 0; i < 2000; i++ {
			tracker.Track("channel", "success")
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("CountTracker.Track() blocked")
	}
}

func TestCountTracker_NilMongoIsNoOp(t *testing.T) {
	tracker := np.NewCountTracker(nil, newLogger())

	assert.NotPanics(t, func() {
		for i := 0; i < 100; i++ {
			tracker.Track("channel", "success")
			tracker.Track("dm", "failed")
			tracker.Track("thread", "dead_letter")
		}
	})
}

func TestDispatcher_MetricsReflectProcessedJobs(t *testing.T) {
	np.ProcessFunc = func(_ np.Job, _ *utility.Logger) error { return nil }
	t.Cleanup(func() { np.ProcessFunc = np.ProcessNotification })

	d := np.NewDispatcher(2, 10, nil, newLogger(), newTracker())
	d.Run(nil, nil)

	const jobCount = 4
	for i := 0; i < jobCount; i++ {
		d.JobQueue <- np.Job{Notification: models.PushNotificationRecord{}}
		atomic.AddInt64(&d.JobQueueCount, 1)
	}

	time.Sleep(400 * time.Millisecond)
	assert.Equal(t, int64(jobCount), atomic.LoadInt64(&d.Metrics.SuccessfulJobs))
	assert.Equal(t, int64(0), atomic.LoadInt64(&d.Metrics.FailedJobs))
}
