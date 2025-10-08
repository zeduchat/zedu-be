package riverqueueBg

import (
	"github.com/riverqueue/river"

	"github.com/hngprojects/telex_be/utility"
)

func registerWorkers(logger *utility.Logger) *river.Workers {
	workers := river.NewWorkers()

	// add more workers here
	river.AddWorker(workers, &AgentJobWorker{Logger: logger})
	return workers
}
