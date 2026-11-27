package cronjobs

import "errors"

// ErrCronJobNotRunning is returned when stopping a cron job that is not running.
var ErrCronJobNotRunning = errors.New("cronjob is not running")
