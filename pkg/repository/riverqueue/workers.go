package riverqueueBg

import (
	"context"
	"fmt"
	"sort"

	"github.com/riverqueue/river"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

type SortArgs struct {
	Strings []string `json:"strings"`
}

func (SortArgs) Kind() string { return "sort" }

type SortWorker struct {
	river.WorkerDefaults[SortArgs]
}

func (w *SortWorker) Work(ctx context.Context, job *river.Job[SortArgs]) error {
	sort.Strings(job.Args.Strings)
	fmt.Printf("Sorted strings: %v\n", job.Args.Strings)
	return nil
}

func (w *SortArgs) InsertSortJob(ctx context.Context) error {
	client := storage.DB.River
	_, err := client.Insert(ctx, w, &river.InsertOpts{
		MaxAttempts: 5,
		// ScheduledAt: time.Now().Add(time.Minute),
	})

	return err
}

func registerWorkers() *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, &SortWorker{})
	return workers
}
