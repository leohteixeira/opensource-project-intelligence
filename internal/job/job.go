// Package job owns the durable work state machine independent of delivery infrastructure.
package job

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

var (
	ErrInvalid          = errors.New("invalid job")
	ErrConflict         = errors.New("job state conflict")
	ErrLeaseUnavailable = errors.New("job lease unavailable")
)

type State string

const (
	Queued    State = "queued"
	Running   State = "running"
	Succeeded State = "succeeded"
	Failed    State = "failed"
	Cancelled State = "cancelled"
)

type Progress struct {
	Completed   int64  `json:"completed"`
	Total       *int64 `json:"total,omitempty"`
	TotalStatus string `json:"total_status,omitempty"`
	Unit        string `json:"unit"`
}

type Checkpoint struct {
	Scope      string    `json:"scope"`
	Cursor     string    `json:"cursor"`
	CoverageTo time.Time `json:"coverage_to"`
	Version    int64     `json:"version"`
}

type Job struct {
	ID                int64       `json:"id,string"`
	ProjectID         int64       `json:"project_id,string,omitempty"`
	Kind              string      `json:"kind"`
	State             State       `json:"state"`
	Progress          Progress    `json:"progress"`
	Checkpoint        *Checkpoint `json:"checkpoint,omitempty"`
	RequestedFrom     *time.Time  `json:"requested_from,omitempty"`
	RequestedTo       *time.Time  `json:"requested_to,omitempty"`
	Version           int64       `json:"version"`
	CoalescedRequests int64       `json:"coalesced_requests"`
	Cancellable       bool        `json:"cancellable"`
	LeaseHolder       string      `json:"-"`
	LeaseExpiresAt    *time.Time  `json:"-"`
	CreatedAt         time.Time   `json:"created_at"`
	StartedAt         *time.Time  `json:"started_at,omitempty"`
	UpdatedAt         time.Time   `json:"updated_at"`
	FinishedAt        *time.Time  `json:"finished_at,omitempty"`
	Failure           string      `json:"failure,omitempty"`
}

func New(id, projectID int64, kind, unit string, total *int64, cancellable bool, now time.Time) (Job, error) {
	if id <= 0 || strings.TrimSpace(kind) == "" || strings.TrimSpace(unit) == "" ||
		total != nil && *total < 0 {
		return Job{}, ErrInvalid
	}
	progress := Progress{Unit: unit, Total: total}
	if total == nil {
		progress.TotalStatus = "unknown"
	}
	now = now.UTC()
	return Job{
		ID: id, ProjectID: projectID, Kind: kind, State: Queued, Progress: progress,
		Version: 1, Cancellable: cancellable, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (j Job) Transition(to State, now time.Time, failure string) (Job, error) {
	if terminal(j.State) {
		if j.State == to {
			return j, nil
		}
		return Job{}, fmt.Errorf("%w: terminal state %s", ErrConflict, j.State)
	}
	allowed := map[State][]State{
		Queued:  {Running, Failed, Cancelled},
		Running: {Succeeded, Failed, Cancelled},
	}
	if !slices.Contains(allowed[j.State], to) {
		return Job{}, fmt.Errorf("%w: transition %s to %s", ErrConflict, j.State, to)
	}
	now = now.UTC()
	j.State = to
	j.Version++
	j.UpdatedAt = now
	if to == Running && j.StartedAt == nil {
		j.StartedAt = &now
	}
	if terminal(to) {
		j.FinishedAt = &now
		j.LeaseHolder = ""
		j.LeaseExpiresAt = nil
	}
	if to == Failed {
		j.Failure = strings.TrimSpace(failure)
	}
	return j, nil
}

func (j Job) Cancel(now time.Time) (Job, error) {
	if !j.Cancellable {
		return Job{}, fmt.Errorf("%w: job is not cancellable", ErrConflict)
	}
	return j.Transition(Cancelled, now, "")
}

func (j Job) Claim(holder string, now time.Time, ttl time.Duration) (Job, error) {
	if strings.TrimSpace(holder) == "" || ttl <= 0 || terminal(j.State) {
		return Job{}, ErrInvalid
	}
	now = now.UTC()
	if j.LeaseExpiresAt != nil && j.LeaseExpiresAt.After(now) && j.LeaseHolder != holder {
		return Job{}, ErrLeaseUnavailable
	}
	expires := now.Add(ttl)
	j.LeaseHolder = holder
	j.LeaseExpiresAt = &expires
	j.Version++
	j.UpdatedAt = now
	if j.State == Queued {
		var err error
		j, err = j.Transition(Running, now, "")
		if err != nil {
			return Job{}, err
		}
	}
	return j, nil
}

func (j Job) CommitPage(holder string, completed int64, checkpoint Checkpoint, now time.Time) (Job, error) {
	if j.State != Running || j.LeaseHolder != holder || j.LeaseExpiresAt == nil ||
		!j.LeaseExpiresAt.After(now) || completed < j.Progress.Completed ||
		j.Progress.Total != nil && completed > *j.Progress.Total || checkpoint.Version <= 0 {
		return Job{}, ErrConflict
	}
	checkpoint.CoverageTo = checkpoint.CoverageTo.UTC()
	if j.Checkpoint != nil && checkpoint.Version <= j.Checkpoint.Version {
		return Job{}, fmt.Errorf("%w: checkpoint must advance", ErrConflict)
	}
	j.Progress.Completed = completed
	j.Checkpoint = &checkpoint
	j.Version++
	j.UpdatedAt = now.UTC()
	return j, nil
}

func (j Job) Coalesce() Job {
	if !terminal(j.State) {
		j.CoalescedRequests++
		j.Version++
	}
	return j
}

func terminal(state State) bool {
	return state == Succeeded || state == Failed || state == Cancelled
}
