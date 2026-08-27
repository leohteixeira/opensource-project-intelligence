// Package worker coordinates bounded, leased durable jobs.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
	gh "github.com/leohteixeira/opensource-project-intelligence/internal/platform/github"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/jobstore"
)

const (
	defaultLeaseTTL   = 30 * time.Second
	defaultPurgeBatch = 100
)

type DurableStore interface {
	Claim(context.Context, string, time.Duration) (*jobstore.Lease, error)
	ClaimJob(context.Context, int64, string, time.Duration) (*jobstore.Lease, error)
	Heartbeat(context.Context, jobstore.Lease, time.Duration) (jobstore.Lease, error)
	CheckCancelled(context.Context, jobstore.Lease) (bool, error)
	Checkpoint(context.Context, jobstore.Lease, int64, job.Checkpoint) (jobstore.Lease, error)
	MarkCollectionSuccess(context.Context, jobstore.Lease) error
	PurgeProject(context.Context, jobstore.Lease, int) (bool, jobstore.Lease, error)
	Complete(context.Context, jobstore.Lease) error
	Fail(context.Context, jobstore.Lease, string, time.Duration) error
}

type Runner struct {
	store        DurableStore
	holder       string
	logger       *slog.Logger
	ttl          time.Duration
	github       GitHubClient
	source       GitHubSourceStore
	pages        GitHubPageStore
	intelligence IntelligenceMaterializer
	exports      ExportProcessor
}

type IntelligenceMaterializer interface {
	MaterializeProject(context.Context, int64, time.Time) error
}

type ExportProcessor interface {
	Process(context.Context, jobstore.Lease) error
}

type GitHubClient interface {
	Repository(context.Context, string, string) (gh.Repository, error)
	Issues(context.Context, string, string, int) ([]gh.Issue, error)
	PullRequests(context.Context, string, string, int) ([]gh.PullRequest, error)
	Releases(context.Context, string, string, int) ([]gh.Release, error)
	Commits(context.Context, string, string, string, int) ([]gh.Commit, error)
}

type GitHubSourceStore interface {
	GitHubSource(context.Context, int64) (jobstore.GitHubSource, error)
	MarkSourceUnavailable(context.Context, int64, string) error
}

type GitHubPageStore interface {
	CommitGitHubIssues(context.Context, int64, int64, int64, []gh.Issue, string, time.Time, time.Time) error
	CommitGitHubPullRequests(context.Context, int64, int64, int64, []gh.PullRequest, string, time.Time, time.Time) error
	CommitGitHubReleases(context.Context, int64, int64, int64, []gh.Release, string, time.Time, time.Time) error
	CommitGitHubCommits(context.Context, int64, int64, int64, []gh.Commit, string, time.Time, time.Time) error
}

func New(store DurableStore, holder string, logger *slog.Logger) (*Runner, error) {
	if store == nil || holder == "" || logger == nil {
		return nil, errors.New("durable store, holder, and logger are required")
	}
	return &Runner{store: store, holder: holder, logger: logger, ttl: defaultLeaseTTL}, nil
}

func NewWithGitHub(
	store DurableStore,
	holder string,
	logger *slog.Logger,
	client GitHubClient,
	sources GitHubSourceStore,
	pages GitHubPageStore,
) (*Runner, error) {
	runner, err := New(store, holder, logger)
	if err != nil {
		return nil, err
	}
	if client == nil || sources == nil || pages == nil {
		return nil, errors.New("GitHub client, source store, and page store are required")
	}
	runner.github, runner.source, runner.pages = client, sources, pages
	return runner, nil
}

// UseIntelligence installs the deterministic post-collection materializer. It is optional so the
// core durable runner remains usable in focused collection tests and recovery tooling.
func (r *Runner) UseIntelligence(value IntelligenceMaterializer) error {
	if value == nil {
		return errors.New("intelligence materializer is required")
	}
	r.intelligence = value
	return nil
}

// UseExports installs the durable, bounded export processor.
func (r *Runner) UseExports(value ExportProcessor) error {
	if value == nil {
		return errors.New("export processor is required")
	}
	r.exports = value
	return nil
}

// RunOne claims at most one Job. A nil result means the queue was empty.
func (r *Runner) RunOne(ctx context.Context) (bool, error) {
	lease, err := r.store.Claim(ctx, r.holder, r.ttl)
	if err != nil {
		return false, err
	}
	if lease == nil {
		return false, nil
	}
	return r.runLease(ctx, *lease)
}

// RunJob executes broker-selected durable work. A nil lease is an idempotent
// success: PostgreSQL proves the notification is delayed, in-flight elsewhere,
// or already terminal.
func (r *Runner) RunJob(ctx context.Context, jobID int64) (bool, error) {
	lease, err := r.store.ClaimJob(ctx, jobID, r.holder, r.ttl)
	if err != nil {
		return false, err
	}
	if lease == nil {
		return false, nil
	}
	return r.runLease(ctx, *lease)
}

func (r *Runner) runLease(ctx context.Context, lease jobstore.Lease) (bool, error) {
	err := r.execute(ctx, lease)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		// Let the lease expire. Another worker resumes from the last committed
		// checkpoint after graceful shutdown or process loss.
		return true, err
	}
	if finishErr := r.store.Fail(ctx, lease, failureCode(err), retryDelay(err)); finishErr != nil {
		return true, errors.Join(err, finishErr)
	}
	return true, err
}

func (r *Runner) execute(ctx context.Context, lease jobstore.Lease) error {
	if lease.Job.Kind != "project_purge" {
		cancelled, err := r.store.CheckCancelled(ctx, lease)
		if err != nil {
			return err
		}
		if cancelled {
			return nil
		}
	}

	switch lease.Job.Kind {
	case "project_initial_sync", "project_sync", "project_history":
		if r.github != nil {
			return r.collectGitHub(ctx, lease)
		}
		if err := r.store.MarkCollectionSuccess(ctx, lease); err != nil {
			return err
		}
		checkpoint := job.Checkpoint{
			Scope: lease.Job.Progress.Unit, Cursor: "complete", CoverageTo: time.Now().UTC(),
			Version: lease.Job.Version + 1,
		}
		lease, err := r.store.Checkpoint(ctx, lease, 1, checkpoint)
		if err != nil {
			return err
		}
		if err := r.materialize(ctx, lease.Job.ProjectID); err != nil {
			return err
		}
		return r.store.Complete(ctx, lease)
	case "project_purge":
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			finished, updated, err := r.store.PurgeProject(ctx, lease, defaultPurgeBatch)
			if err != nil {
				return err
			}
			lease = updated
			if finished {
				return r.store.Complete(ctx, lease)
			}
			lease, err = r.store.Heartbeat(ctx, lease, r.ttl)
			if err != nil {
				return err
			}
		}
	case "association_recalculation", "source_recalculation":
		if err := r.materialize(ctx, lease.Job.ProjectID); err != nil {
			return err
		}
		return r.store.Complete(ctx, lease)
	case "project_transition":
		return r.store.Complete(ctx, lease)
	case "export":
		if r.exports == nil {
			return errors.New("export processor is unavailable")
		}
		if err := r.exports.Process(ctx, lease); err != nil {
			return err
		}
		return r.store.Complete(ctx, lease)
	default:
		return fmt.Errorf("unsupported job kind %q", lease.Job.Kind)
	}
}

func (r *Runner) collectGitHub(ctx context.Context, lease jobstore.Lease) error {
	target, err := r.source.GitHubSource(ctx, lease.Job.ProjectID)
	if err != nil {
		return err
	}
	repository, err := r.github.Repository(ctx, target.Owner, target.Repository)
	if err != nil {
		if errors.Is(err, gh.ErrNotPublic) {
			if updateErr := r.source.MarkSourceUnavailable(ctx, target.SourceID, "not_public"); updateErr != nil {
				return errors.Join(err, updateErr)
			}
		}
		return err
	}
	if strings.TrimSpace(repository.DefaultBranch) == "" {
		return errors.New("GitHub repository omitted its default branch")
	}
	completed := lease.Job.Progress.Completed
	lease, completed, err = r.collectGitHubIssues(ctx, lease, target, completed)
	if err != nil {
		return err
	}
	lease, completed, err = r.collectGitHubPullRequests(ctx, lease, target, completed)
	if err != nil {
		return err
	}
	lease, completed, err = r.collectGitHubReleases(ctx, lease, target, completed)
	if err != nil {
		return err
	}
	lease, _, err = r.collectGitHubCommits(ctx, lease, target, repository.DefaultBranch, completed)
	if err != nil {
		return err
	}
	if err := r.materialize(ctx, lease.Job.ProjectID); err != nil {
		return err
	}
	return r.store.Complete(ctx, lease)
}

func (r *Runner) materialize(ctx context.Context, projectID int64) error {
	if r.intelligence == nil {
		return nil
	}
	cutoff := time.Now().UTC().Truncate(24 * time.Hour)
	if err := r.intelligence.MaterializeProject(ctx, projectID, cutoff); err != nil {
		return fmt.Errorf("materialize deterministic intelligence: %w", err)
	}
	return nil
}

func (r *Runner) collectGitHubIssues(
	ctx context.Context,
	lease jobstore.Lease,
	target jobstore.GitHubSource,
	completed int64,
) (jobstore.Lease, int64, error) {
	for page := target.Pages["github_issues"]; ; page++ {
		if err := r.collectionAllowed(ctx, lease); err != nil {
			return lease, completed, err
		}
		issues, err := r.github.Issues(ctx, target.Owner, target.Repository, page)
		if err != nil {
			return lease, completed, err
		}
		next := nextCursor(page, len(issues))
		coverageTo := time.Now().UTC()
		if err := r.pages.CommitGitHubIssues(ctx, target.ProjectID, target.RepositoryID,
			target.SourceID, issues, next, target.CoverageFrom, coverageTo); err != nil {
			return lease, completed, err
		}
		completed += int64(len(issues))
		checkpoint := job.Checkpoint{Scope: "github_issues", Cursor: next,
			CoverageTo: coverageTo, Version: lease.Job.Version + 1}
		lease, err = r.store.Checkpoint(ctx, lease, completed, checkpoint)
		if err != nil {
			return lease, completed, err
		}
		if next == "complete" {
			return lease, completed, nil
		}
		lease, err = r.store.Heartbeat(ctx, lease, r.ttl)
		if err != nil {
			return lease, completed, err
		}
	}
}

func (r *Runner) collectGitHubPullRequests(
	ctx context.Context,
	lease jobstore.Lease,
	target jobstore.GitHubSource,
	completed int64,
) (jobstore.Lease, int64, error) {
	for page := target.Pages["github_pull_requests"]; ; page++ {
		if err := r.collectionAllowed(ctx, lease); err != nil {
			return lease, completed, err
		}
		values, err := r.github.PullRequests(ctx, target.Owner, target.Repository, page)
		if err != nil {
			return lease, completed, err
		}
		next, coverageTo := nextCursor(page, len(values)), time.Now().UTC()
		if err := r.pages.CommitGitHubPullRequests(ctx, target.ProjectID, target.RepositoryID,
			target.SourceID, values, next, target.CoverageFrom, coverageTo); err != nil {
			return lease, completed, err
		}
		completed += int64(len(values))
		lease, err = r.store.Checkpoint(ctx, lease, completed, job.Checkpoint{
			Scope: "github_pull_requests", Cursor: next, CoverageTo: coverageTo,
			Version: lease.Job.Version + 1,
		})
		if err != nil || next == "complete" {
			return lease, completed, err
		}
		lease, err = r.store.Heartbeat(ctx, lease, r.ttl)
		if err != nil {
			return lease, completed, err
		}
	}
}

func (r *Runner) collectGitHubReleases(
	ctx context.Context,
	lease jobstore.Lease,
	target jobstore.GitHubSource,
	completed int64,
) (jobstore.Lease, int64, error) {
	for page := target.Pages["github_releases"]; ; page++ {
		if err := r.collectionAllowed(ctx, lease); err != nil {
			return lease, completed, err
		}
		values, err := r.github.Releases(ctx, target.Owner, target.Repository, page)
		if err != nil {
			return lease, completed, err
		}
		next, coverageTo := nextCursor(page, len(values)), time.Now().UTC()
		if err := r.pages.CommitGitHubReleases(ctx, target.ProjectID, target.RepositoryID,
			target.SourceID, values, next, target.CoverageFrom, coverageTo); err != nil {
			return lease, completed, err
		}
		completed += int64(len(values))
		lease, err = r.store.Checkpoint(ctx, lease, completed, job.Checkpoint{
			Scope: "github_releases", Cursor: next, CoverageTo: coverageTo,
			Version: lease.Job.Version + 1,
		})
		if err != nil || next == "complete" {
			return lease, completed, err
		}
		lease, err = r.store.Heartbeat(ctx, lease, r.ttl)
		if err != nil {
			return lease, completed, err
		}
	}
}

func (r *Runner) collectGitHubCommits(
	ctx context.Context,
	lease jobstore.Lease,
	target jobstore.GitHubSource,
	defaultBranch string,
	completed int64,
) (jobstore.Lease, int64, error) {
	for page := target.Pages["github_commits"]; ; page++ {
		if err := r.collectionAllowed(ctx, lease); err != nil {
			return lease, completed, err
		}
		values, err := r.github.Commits(ctx, target.Owner, target.Repository, defaultBranch, page)
		if err != nil {
			return lease, completed, err
		}
		next, coverageTo := nextCursor(page, len(values)), time.Now().UTC()
		if err := r.pages.CommitGitHubCommits(ctx, target.ProjectID, target.RepositoryID,
			target.SourceID, values, next, target.CoverageFrom, coverageTo); err != nil {
			return lease, completed, err
		}
		completed += int64(len(values))
		lease, err = r.store.Checkpoint(ctx, lease, completed, job.Checkpoint{
			Scope: "github_commits", Cursor: next, CoverageTo: coverageTo,
			Version: lease.Job.Version + 1,
		})
		if err != nil || next == "complete" {
			return lease, completed, err
		}
		lease, err = r.store.Heartbeat(ctx, lease, r.ttl)
		if err != nil {
			return lease, completed, err
		}
	}
}

func (r *Runner) collectionAllowed(ctx context.Context, lease jobstore.Lease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cancelled, err := r.store.CheckCancelled(ctx, lease)
	if err != nil {
		return err
	}
	if cancelled {
		return job.ErrConflict
	}
	return nil
}

func nextCursor(page, count int) string {
	if count < 100 {
		return "complete"
	}
	return strconv.Itoa(page + 1)
}

func failureCode(err error) string {
	if errors.Is(err, gh.ErrRateLimited) {
		return "rate_limited"
	}
	if errors.Is(err, gh.ErrNotPublic) {
		return "not_public"
	}
	if errors.Is(err, job.ErrLeaseUnavailable) {
		return "lease_lost"
	}
	return "worker_error"
}

func retryDelay(err error) time.Duration {
	if errors.Is(err, gh.ErrRateLimited) {
		var limited *gh.RateLimitError
		if errors.As(err, &limited) && !limited.Reset.IsZero() {
			delay := time.Until(limited.Reset)
			if delay < time.Second {
				return time.Second
			}
			if delay > time.Hour {
				return time.Hour
			}
			return delay
		}
		return 15 * time.Minute
	}
	if errors.Is(err, gh.ErrNotPublic) {
		return -1
	}
	if errors.Is(err, job.ErrLeaseUnavailable) {
		return -1
	}
	return 0
}
