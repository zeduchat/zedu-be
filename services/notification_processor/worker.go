package notification_processor

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

// ProcessFunc and RequeueFunc are exported so tests can inject stubs without real dependencies.
var (
	ProcessFunc  = ProcessNotification
	RequeueFunc  func(job Job)
)

type Metrics struct {
	ActiveWorkers  int64
	QueuedJobs     int64
	FailedJobs     int64
	SuccessfulJobs int64
}

type Job struct {
	Notification models.PushNotificationRecord
}

type Worker struct {
	ID         int
	JobChannel chan Job
	WorkerPool chan WorkerSlot
	QuitChan   chan bool
	Metrics    *Metrics
	Busy       int32
	JobCount   int32
	Logger     *utility.Logger
}

type WorkerSlot struct {
	WorkerID   int
	JobChannel chan Job
}

func NewWorker(id int, workerPool chan WorkerSlot, metrics *Metrics, logger *utility.Logger) Worker {
	return Worker{
		ID:         id,
		JobChannel: make(chan Job),
		WorkerPool: workerPool,
		QuitChan:   make(chan bool, 1),
		Metrics:    metrics,
		Logger:     logger,
	}
}

func (w Worker) Start(rdb *redis.Client, db *gorm.DB, tracker *CountTracker) {
	go func() {
		for {
			select {
			case w.WorkerPool <- WorkerSlot{WorkerID: w.ID, JobChannel: w.JobChannel}:
			case <-w.QuitChan:
				return
			}

			select {
			case job, ok := <-w.JobChannel:
				if !ok {
					return
				}
				atomic.StoreInt32(&w.Busy, 1)
				atomic.AddInt32(&w.JobCount, 1)

				err := processWithRecovery(job, w.Logger)
				if err != nil {
					w.Logger.Error("Worker %d: job error: %v", w.ID, err)
					atomic.AddInt64(&w.Metrics.FailedJobs, 1)
					tracker.Track(string(job.Notification.ChannelType), "failed")
					w.requeueOrPersist(job, err, rdb, db, tracker)
				} else {
					atomic.AddInt64(&w.Metrics.SuccessfulJobs, 1)
					tracker.Track(string(job.Notification.ChannelType), "success")
				}

				atomic.AddInt32(&w.JobCount, -1)
				atomic.StoreInt32(&w.Busy, 0)

			case <-w.QuitChan:
				return
			}
		}
	}()
}

func (w Worker) Stop() {
	select {
	case w.QuitChan <- true:
	default:
	}
}

func (w Worker) requeueOrPersist(job Job, lastErr error, rdb *redis.Client, db *gorm.DB, tracker *CountTracker) {
	const defaultMaxRetries = 3
	maxRetries := job.Notification.MaxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}

	job.Notification.LastError = lastErr.Error()

	if job.Notification.RetryCount >= maxRetries {
		w.Logger.Error("Worker %d: max retries reached, persisting dead letter (channel=%s)", w.ID, job.Notification.ChannelId)
		w.persistDeadLetter(job, db, tracker)
		return
	}

	job.Notification.RetryCount++

	if RequeueFunc != nil {
		RequeueFunc(job)
		return
	}

	if err := job.Notification.PushToRetryQueue(rdb); err != nil {
		w.Logger.Error("Worker %d: requeue failed, falling back to dead letter: %v", w.ID, err)
		w.persistDeadLetter(job, db, tracker)
	} else {
		w.Logger.Info("Worker %d: requeued notification (attempt %d/%d)", w.ID, job.Notification.RetryCount, maxRetries)
	}
}

func (w Worker) persistDeadLetter(job Job, db *gorm.DB, tracker *CountTracker) {
	payload, err := json.Marshal(job.Notification)
	if err != nil {
		w.Logger.Error("Worker %d: failed to marshal dead letter payload: %v", w.ID, err)
		return
	}

	rec := models.DeadLetterNotification{
		ID:          utility.GenerateUUID(),
		ChannelId:   job.Notification.ChannelId,
		OrgId:       job.Notification.OrgId,
		ChannelType: job.Notification.ChannelType,
		Payload:     string(payload),
		RetryCount:  job.Notification.RetryCount,
		LastError:   job.Notification.LastError,
	}

	if saveErr := rec.Save(db); saveErr != nil {
		w.Logger.Error("Worker %d: failed to persist dead letter: %v", w.ID, saveErr)
	} else {
		w.Logger.Info("Worker %d: dead letter saved (channel=%s)", w.ID, job.Notification.ChannelId)
	}

	tracker.Track(string(job.Notification.ChannelType), "dead_letter")
}

func processWithRecovery(job Job, logger *utility.Logger) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic processing notification: %v", r)
			logger.Error("recovered panic in notification worker: %v", r)
		}
	}()
	return ProcessFunc(job, logger)
}

func PrintMetrics(m *Metrics) string {
	return fmt.Sprintf("📊 Metrics | ActiveWorkers: %d | QueuedJobs: %d | Success: %d | Failed: %d\n",
		atomic.LoadInt64(&m.ActiveWorkers),
		atomic.LoadInt64(&m.QueuedJobs),
		atomic.LoadInt64(&m.SuccessfulJobs),
		atomic.LoadInt64(&m.FailedJobs),
	)
}

func workerIdleFor(_ Worker) time.Duration { return 0 }
