package collector_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/collector"
)

func TestRunBoundedValidatesLimit(t *testing.T) {
	t.Parallel()

	err := collector.RunBounded(t.Context(), 0, []collector.Task{
		func(context.Context) error { return nil },
	})
	if err == nil {
		t.Fatal("RunBounded() returned no error, want a validation failure")
	}
}

func TestRunBoundedRunsEveryTask(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		taskCount int
		limit     int
	}{
		"no tasks":               {taskCount: 0, limit: 4},
		"fewer tasks than limit": {taskCount: 2, limit: 4},
		"more tasks than limit":  {taskCount: 32, limit: 4},
		"serial":                 {taskCount: 8, limit: 1},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var executed atomic.Int64

			tasks := make([]collector.Task, tc.taskCount)
			for index := range tasks {
				tasks[index] = func(context.Context) error {
					executed.Add(1)
					return nil
				}
			}

			if err := collector.RunBounded(t.Context(), tc.limit, tasks); err != nil {
				t.Fatalf("RunBounded() returned an unexpected error: %v", err)
			}
			if got := int(executed.Load()); got != tc.taskCount {
				t.Errorf("executed %d tasks, want %d", got, tc.taskCount)
			}
		})
	}
}

func TestRunBoundedNeverExceedsTheLimit(t *testing.T) {
	t.Parallel()

	const limit = 3

	var (
		mutex   sync.Mutex
		running int
		peak    int
	)

	tasks := make([]collector.Task, 64)
	for index := range tasks {
		tasks[index] = func(context.Context) error {
			mutex.Lock()
			running++
			if running > peak {
				peak = running
			}
			mutex.Unlock()

			mutex.Lock()
			running--
			mutex.Unlock()

			return nil
		}
	}

	if err := collector.RunBounded(t.Context(), limit, tasks); err != nil {
		t.Fatalf("RunBounded() returned an unexpected error: %v", err)
	}
	if peak > limit {
		t.Errorf("peak concurrency was %d, want at most %d", peak, limit)
	}
}

func TestRunBoundedJoinsEveryFailure(t *testing.T) {
	t.Parallel()

	first := errors.New("first failure")
	second := errors.New("second failure")

	err := collector.RunBounded(t.Context(), 2, []collector.Task{
		func(context.Context) error { return first },
		func(context.Context) error { return nil },
		func(context.Context) error { return second },
	})
	if err == nil {
		t.Fatal("RunBounded() returned no error, want the joined failures")
	}
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Errorf("RunBounded() error = %v, want it to wrap both failures", err)
	}
}

func TestRunBoundedStopsOnCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var executed atomic.Int64

	tasks := make([]collector.Task, 128)
	for index := range tasks {
		tasks[index] = func(context.Context) error {
			if executed.Add(1) == 1 {
				cancel()
			}
			return nil
		}
	}

	err := collector.RunBounded(ctx, 1, tasks)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunBounded() error = %v, want it to wrap context.Canceled", err)
	}
	if got := executed.Load(); got >= int64(len(tasks)) {
		t.Errorf("executed %d tasks, want the scheduler to stop early", got)
	}
}
