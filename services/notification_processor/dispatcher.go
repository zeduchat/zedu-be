package notification_processor

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type Dispatcher struct {
	WorkerPool    chan WorkerSlot
	Workers       map[int]*Worker
	MaxWorkers    int
	MaxJobs       int
	JobQueue      chan Job
	JobQueueCount int64
	scaleLock     sync.Mutex
	Metrics       *Metrics
	DB            *storage.Database
	Logger        *utility.Logger
	Tracker       *CountTracker
	done          chan struct{}
}

func NewDispatcher(startWorkers int, maxJobs int, db *storage.Database, logger *utility.Logger, tracker *CountTracker) *Dispatcher {
	return &Dispatcher{
		WorkerPool: make(chan WorkerSlot, startWorkers),
		MaxWorkers: startWorkers,
		MaxJobs:    maxJobs,
		JobQueue:   make(chan Job, startWorkers),
		Workers:    make(map[int]*Worker),
		Metrics:    &Metrics{},
		DB:         db,
		Logger:     logger,
		Tracker:    tracker,
		done:       make(chan struct{}),
	}
}

func (d *Dispatcher) Run(rdb *redis.Client, db *gorm.DB) {
	for i := 0; i < d.MaxWorkers; i++ {
		d.startWorker(i, rdb, db)
	}
	go d.dispatch()
	d.Logger.Info("Started Dispatchers...")
}

func (d *Dispatcher) Shutdown() {
	close(d.done)
}

func (d *Dispatcher) startWorker(id int, rdb *redis.Client, db *gorm.DB) {
	d.scaleLock.Lock()
	defer d.scaleLock.Unlock()

	atomic.AddInt64(&d.Metrics.ActiveWorkers, 1)
	worker := NewWorker(id, d.WorkerPool, d.Metrics, d.Logger)
	worker.Start(rdb, db, d.Tracker)
	d.Workers[id] = &worker
	d.Logger.Info(PrintMetrics(d.Metrics))
}

func (d *Dispatcher) stopWorker(id int) {
	d.scaleLock.Lock()
	defer d.scaleLock.Unlock()

	worker, ok := d.Workers[id]
	if !ok {
		return
	}

	worker.Stop()
	delete(d.Workers, id)
	atomic.AddInt64(&d.Metrics.ActiveWorkers, -1)
	d.Logger.Info("Worker %d stopped\n", id)
	d.Logger.Info(PrintMetrics(d.Metrics))
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case <-d.done:
			return
		case job := <-d.JobQueue:
			select {
			case slot := <-d.WorkerPool:
				slot.JobChannel <- job
				atomic.AddInt64(&d.JobQueueCount, -1)
			case <-d.done:
				return
			}
		}
	}
}

func (d *Dispatcher) ScaleWorkers(queueLength int, rdb *redis.Client, db *gorm.DB) {
	desiredWorkers := queueLength / d.MaxJobs

	if desiredWorkers > d.MaxWorkers {
		desiredWorkers = d.MaxWorkers
	}
	if desiredWorkers < 10 {
		desiredWorkers = 10
	}

	d.scaleLock.Lock()
	currentWorkers := len(d.Workers)
	d.scaleLock.Unlock()

	if desiredWorkers > currentWorkers {
		for i := currentWorkers; i < desiredWorkers; i++ {
			d.startWorker(i, rdb, db)
		}
		d.Logger.Info("Scaling up: %d workers\n", desiredWorkers)
	} else if desiredWorkers < currentWorkers {
		toStop := currentWorkers - desiredWorkers
		stopped := 0

		d.scaleLock.Lock()
		for id, worker := range d.Workers {
			if stopped >= toStop {
				break
			}
			if atomic.LoadInt32(&worker.Busy) == 0 {
				stopped++
				go func(wid int) { d.stopWorker(wid) }(id)
			}
		}
		d.scaleLock.Unlock()
		d.Logger.Info("Scaling down: %d workers\n", stopped)
	}
}

func FeedDispatcher(d *Dispatcher) {
	backoff := 500 * time.Millisecond
	const maxBackoff = 30 * time.Second

	for {
		// Drain retry queue first (higher priority than main queue)
		var retryRecord models.PushNotificationRecord
		retryRec, retryErr := retryRecord.PopFromRetryQueue(d.DB.Redis)
		if retryErr == nil {
			d.JobQueue <- Job{Notification: retryRec}
			atomic.AddInt64(&d.JobQueueCount, 1)
			backoff = 500 * time.Millisecond
			continue
		}

		queueLen := d.JobQueueCount
		atomic.StoreInt64(&d.Metrics.QueuedJobs, queueLen)

		var notificationRecord models.PushNotificationRecord
		rec, err := notificationRecord.PopFromQueue(d.DB.Redis)
		if err != nil {
			if strings.Contains(err.Error(), "redis: nil") {
				time.Sleep(backoff)
				continue
			}
			d.Logger.Error("error popping queue: %v", err)
			if backoff < maxBackoff {
				backoff = min(backoff*2, maxBackoff)
			}
			time.Sleep(backoff)
			continue
		}

		backoff = 500 * time.Millisecond
		d.Logger.Info("Feeding new job to dispatcher")
		d.JobQueue <- Job{Notification: rec}
		atomic.AddInt64(&d.JobQueueCount, 1)
		d.Logger.Info(PrintMetrics(d.Metrics))
	}
}


