package cronjobs

import "sync"

var (
	jobMutexesLock sync.Mutex
	jobMutexes     = map[string]*sync.Mutex{}
)

// jobMutex returns the mutex shared by every scheduler of a job, so a job
// never runs concurrently with itself, even across restarts.
func jobMutex(jobName string) *sync.Mutex {
	jobMutexesLock.Lock()
	defer jobMutexesLock.Unlock()
	mutex, ok := jobMutexes[jobName]
	if !ok {
		mutex = &sync.Mutex{}
		jobMutexes[jobName] = mutex
	}
	return mutex
}
