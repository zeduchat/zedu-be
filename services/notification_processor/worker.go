package notification_processor

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
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
	StopChan   chan bool
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
		QuitChan:   make(chan bool),
		Metrics:    metrics,
		StopChan:   make(chan bool, 1),
		Logger:     logger,
	}
}

func (w Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- WorkerSlot{
				WorkerID:   w.ID,
				JobChannel: w.JobChannel,
			} // Register worker

			select {
			case job := <-w.JobChannel:
				w.Logger.Info("Worker %d: Handling job: %+v\n", w.ID, job.Notification.ChannelType)
				atomic.AddInt32(&w.JobCount, 1)
				atomic.StoreInt32(&w.Busy, 1)

				err := ProcessNotification(job, w.Logger)
				if err != nil {
					w.Logger.Info("Worker %d: error sending notification: %v\n", w.ID, err)
					atomic.AddInt64(&w.Metrics.FailedJobs, 1)
				} else {
					atomic.AddInt64(&w.Metrics.SuccessfulJobs, 1)
				}

				atomic.AddInt32(&w.JobCount, -1)
				if atomic.LoadInt32(&w.JobCount) == 0 {
					atomic.StoreInt32(&w.Busy, 0)
				}

			case <-w.QuitChan:
				w.Logger.Info("Worker %d stopping\n", w.ID)
				return

			case <-w.StopChan:
				w.Logger.Info("Worker %d received graceful stop signal\n", w.ID)

				// Wait for jobs to finish
				for {
					if atomic.LoadInt32(&w.JobCount) == 0 {
						w.Logger.Info("Worker %d has no more jobs. Exiting gracefully.\n", w.ID)
						return
					}
					time.Sleep(100 * time.Millisecond) // 💤 small wait
				}
			}
		}
	}()
}

func (w Worker) Stop() {
	go func() {
		w.QuitChan <- true
	}()
}

func PrintMetrics(m *Metrics) string {
	return fmt.Sprintf("📊 Metrics | ActiveWorkers: %d | QueuedJobs: %d | Success: %d | Failed: %d\n",
		atomic.LoadInt64(&m.ActiveWorkers),
		atomic.LoadInt64(&m.QueuedJobs),
		atomic.LoadInt64(&m.SuccessfulJobs),
		atomic.LoadInt64(&m.FailedJobs),
	)
}
