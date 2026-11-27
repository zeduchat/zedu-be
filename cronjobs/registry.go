package cronjobs

import (
	"strings"
	"time"
)

// RegisterCronJob adds or replaces a cron job definition. It lets tests
// drive the scheduler with a job that does not touch the database.
func RegisterCronJob(jobName string, cronJob CronJob, interval time.Duration) {
	cronJobs[strings.ToLower(jobName)] = CronJobObject{CronJob: cronJob, Interval: interval}
}

// CronJobInterval returns the configured interval of a cron job.
func CronJobInterval(jobName string) (time.Duration, bool) {
	cronJob, ok := cronJobs[strings.ToLower(jobName)]
	return cronJob.Interval, ok
}
