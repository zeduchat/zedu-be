package notification_processor

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
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
	DB            storage.Database
	Logger        *utility.Logger
}

func NewDispatcher(startWorkers int, maxJobs int, db storage.Database, logger *utility.Logger) *Dispatcher {
	pool := make(chan WorkerSlot, startWorkers)
	return &Dispatcher{
		WorkerPool: pool,
		MaxWorkers: startWorkers,
		MaxJobs:    maxJobs,
		JobQueue:   make(chan Job),
		Workers:    make(map[int]*Worker),
		Metrics:    &Metrics{},
		DB:         db,
		Logger:     logger,
	}
}

func (d *Dispatcher) Run() {
	for i := 0; i < d.MaxWorkers; i++ {
		d.startWorker(i)
	}
	go d.dispatch()
}

func (d *Dispatcher) startWorker(id int) {
	d.scaleLock.Lock()
	defer d.scaleLock.Unlock()

	atomic.AddInt64(&d.Metrics.ActiveWorkers, 1)
	worker := NewWorker(id, d.WorkerPool, d.Metrics, d.Logger)
	worker.Start()
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

	if atomic.LoadInt32(&worker.JobCount) > 0 {
		// Still handling jobs
		select {
		case worker.StopChan <- true:
			d.Logger.Info("Worker %d marked to stop after jobs finish\n", id)
		default:
			// Already marked
		}
	} else {
		// Idle now, stop immediately
		worker.Stop()
		delete(d.Workers, id)
		atomic.AddInt64(&d.Metrics.ActiveWorkers, -1)
		d.Logger.Info("Worker %d stopped immediately\n", id)
		d.Logger.Info(PrintMetrics(d.Metrics))
	}
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.JobQueue:
			go func(job Job) {
				for {
					workerSlot := <-d.WorkerPool // get available worker info

					workerID := workerSlot.WorkerID
					jobChannel := workerSlot.JobChannel

					worker, ok := d.Workers[workerID]
					if !ok {
						d.Logger.Info("Worker %d disappeared\n", workerID)
						time.Sleep(100 * time.Millisecond)
						continue
					}

					if atomic.LoadInt32(&worker.JobCount) < int32(d.MaxJobs) {
						jobChannel <- job
						atomic.AddInt64(&d.JobQueueCount, -1)
						break
					} else {
						// Worker too busy, try another one
						time.Sleep(10 * time.Millisecond)
					}
				}
			}(job)
		}
	}
}

func (d *Dispatcher) ScaleWorkers(queueLength int) {

	desiredWorkers := queueLength / d.MaxJobs // 1 worker per Max jobs

	if desiredWorkers > d.MaxWorkers {
		desiredWorkers = d.MaxWorkers // Safety cap
	}
	if desiredWorkers < 5 {
		desiredWorkers = 5 // leave 5 workers default
	}

	currentWorkers := len(d.Workers)

	if desiredWorkers > currentWorkers {
		for i := currentWorkers; i < desiredWorkers; i++ {
			d.startWorker(i)
		}
		d.Logger.Info("Scaling up: %d workers\n", desiredWorkers)
	} else if desiredWorkers < currentWorkers {

		toStop := currentWorkers - desiredWorkers
		stopped := 0

		// Stop only free workers
		for id, worker := range d.Workers {
			if stopped >= toStop {
				break
			}
			if atomic.LoadInt32(&worker.Busy) == 0 {
				d.stopWorker(id)
				stopped++
			}
		}
		d.Logger.Info("Scaling down: %d workers\n", stopped)
	}
}

func FeedDispatcher(d *Dispatcher) {
	for {
		queueLen := d.JobQueueCount
		atomic.StoreInt64(&d.Metrics.QueuedJobs, int64(queueLen))
		d.ScaleWorkers(int(queueLen))

		rec, err := redis.PopFromNotificationQueue(d.DB.Redis)
		if err != nil {
			if strings.Contains(err.Error(), "empty queue") {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			d.Logger.Error("error popping queue: ", err.Error())
			time.Sleep(500 * time.Millisecond)
			continue
		}

		d.Logger.Info("Feeding new job to dispatcher")

		d.JobQueue <- Job{Notification: rec}
		atomic.AddInt64(&d.JobQueueCount, 1)
		d.Logger.Info(PrintMetrics(d.Metrics))
	}
}
