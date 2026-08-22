package job_test

import (
	"errors"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
)

func TestDurableJobRules(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	total := int64(2)
	value, err := job.New(1, 2, "sync", "pages", &total, true, now)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("UT-066 bounded admission remains queued", func(t *testing.T) {
		if value.State != job.Queued || value.Progress.Total == nil {
			t.Fatalf("got %#v", value)
		}
	})
	t.Run("UT-256 checkpoint advances only with page commit", func(t *testing.T) {
		claimed, claimErr := value.Claim("worker-1", now, time.Minute)
		if claimErr != nil {
			t.Fatal(claimErr)
		}
		committed, commitErr := claimed.CommitPage("worker-1", 1, job.Checkpoint{
			Scope: "issues", Cursor: "page-1", CoverageTo: now, Version: 1,
		}, now.Add(time.Second))
		if commitErr != nil || committed.Checkpoint == nil || committed.Progress.Completed != 1 {
			t.Fatalf("got %#v, %v", committed, commitErr)
		}
		if _, commitErr = committed.CommitPage("worker-1", 2, job.Checkpoint{Version: 1}, now.Add(2*time.Second)); !errors.Is(commitErr, job.ErrConflict) {
			t.Fatalf("stale checkpoint got %v", commitErr)
		}
	})
	t.Run("UT-257 terminal state never regresses", func(t *testing.T) {
		running, _ := value.Transition(job.Running, now, "")
		succeeded, _ := running.Transition(job.Succeeded, now.Add(time.Second), "")
		if _, transitionErr := succeeded.Transition(job.Running, now.Add(2*time.Second), ""); !errors.Is(transitionErr, job.ErrConflict) {
			t.Fatalf("got %v", transitionErr)
		}
	})
	t.Run("UT-258 purge cannot cancel", func(t *testing.T) {
		purge, createErr := job.New(3, 2, "purge", "objects", nil, false, now)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, cancelErr := purge.Cancel(now); !errors.Is(cancelErr, job.ErrConflict) {
			t.Fatalf("got %v", cancelErr)
		}
	})
}
