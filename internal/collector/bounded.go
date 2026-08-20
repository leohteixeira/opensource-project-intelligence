package collector

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Task is a single unit of collection work.
//
// A task must be idempotent: the scheduler may run it again after a transient
// failure, and the worker may restart from the last checkpoint.
type Task func(ctx context.Context) error

// RunBounded runs every task with at most limit of them in flight.
//
// Concurrency is bounded explicitly rather than spawning one goroutine per
// item, so that a project with thousands of issues cannot exhaust the process
// or the source's rate limit. The context is honoured between tasks: once it is
// cancelled, no further task is started and the cancellation cause is returned
// alongside whatever the running tasks reported.
//
// The returned error joins the failures of every task that failed, so a single
// broken repository does not hide the others.
func RunBounded(ctx context.Context, limit int, tasks []Task) error {
	if limit < 1 {
		return fmt.Errorf("collector: limit must be at least 1, got %d", limit)
	}
	if len(tasks) == 0 {
		return nil
	}

	semaphore := make(chan struct{}, limit)
	errs := make([]error, len(tasks))

	var wait sync.WaitGroup

	for index, task := range tasks {
		select {
		case <-ctx.Done():
			wait.Wait()
			return errors.Join(append(compact(errs), ctx.Err())...)
		case semaphore <- struct{}{}:
		}

		wait.Add(1)

		go func() {
			defer wait.Done()
			defer func() { <-semaphore }()

			if err := task(ctx); err != nil {
				errs[index] = fmt.Errorf("collector: task %d failed: %w", index, err)
			}
		}()
	}

	wait.Wait()

	return errors.Join(compact(errs)...)
}

func compact(errs []error) []error {
	compacted := make([]error, 0, len(errs))

	for _, err := range errs {
		if err != nil {
			compacted = append(compacted, err)
		}
	}

	return compacted
}
