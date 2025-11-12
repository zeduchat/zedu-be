package riverqueueBg

import (
	"github.com/riverqueue/river"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func registerWorkers(logger *utility.Logger, db *storage.Database) *river.Workers {
	workers := river.NewWorkers()

	// add more workers here
	river.AddWorker(workers, &AgentJobWorker{Logger: logger})
	river.AddWorker(workers, &SavedMessagesRemainderWorker{logger: logger, db: db.Postgresql})
	return workers
}
