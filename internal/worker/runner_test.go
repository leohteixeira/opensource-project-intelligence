package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
	gh "github.com/leohteixeira/opensource-project-intelligence/internal/platform/github"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/jobstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/worker"
)

type fakeStore struct {
	lease        *jobstore.Lease
	completed    bool
	failed       bool
	marked       bool
	checkpoints  int
	failureDelay time.Duration
}

func (f *fakeStore) Claim(context.Context, string, time.Duration) (*jobstore.Lease, error) {
	return f.lease, nil
}
func (f *fakeStore) ClaimJob(_ context.Context, id int64, _ string, _ time.Duration) (*jobstore.Lease, error) {
	if f.lease == nil || f.lease.Job.ID != id {
		return nil, nil
	}
	return f.lease, nil
}
func (f *fakeStore) Heartbeat(_ context.Context, lease jobstore.Lease, _ time.Duration) (jobstore.Lease, error) {
	return lease, nil
}
func (f *fakeStore) CheckCancelled(context.Context, jobstore.Lease) (bool, error) { return false, nil }
func (f *fakeStore) Checkpoint(_ context.Context, lease jobstore.Lease, _ int64, _ job.Checkpoint) (jobstore.Lease, error) {
	f.checkpoints++
	return lease, nil
}
func (f *fakeStore) MarkCollectionSuccess(context.Context, jobstore.Lease) error {
	f.marked = true
	return nil
}

type fakeGitHub struct {
	repository gh.Repository
	issues     []gh.Issue
	err        error
}

func (client fakeGitHub) Repository(context.Context, string, string) (gh.Repository, error) {
	return client.repository, client.err
}
func (client fakeGitHub) Issues(context.Context, string, string, int) ([]gh.Issue, error) {
	return client.issues, client.err
}
func (client fakeGitHub) PullRequests(context.Context, string, string, int) ([]gh.PullRequest, error) {
	return nil, client.err
}
func (client fakeGitHub) Releases(context.Context, string, string, int) ([]gh.Release, error) {
	return nil, client.err
}
func (client fakeGitHub) Commits(context.Context, string, string, string, int) ([]gh.Commit, error) {
	return nil, client.err
}

type fakeGitHubSource struct {
	target      jobstore.GitHubSource
	unavailable bool
}

func (source *fakeGitHubSource) GitHubSource(context.Context, int64) (jobstore.GitHubSource, error) {
	return source.target, nil
}
func (source *fakeGitHubSource) MarkSourceUnavailable(context.Context, int64, string) error {
	source.unavailable = true
	return nil
}

type fakeGitHubPages struct{ committed bool }

func (pages *fakeGitHubPages) CommitGitHubIssues(
	context.Context, int64, int64, int64, []gh.Issue, string, time.Time, time.Time,
) error {
	pages.committed = true
	return nil
}
func (pages *fakeGitHubPages) CommitGitHubPullRequests(
	context.Context, int64, int64, int64, []gh.PullRequest, string, time.Time, time.Time,
) error {
	pages.committed = true
	return nil
}
func (pages *fakeGitHubPages) CommitGitHubReleases(
	context.Context, int64, int64, int64, []gh.Release, string, time.Time, time.Time,
) error {
	pages.committed = true
	return nil
}
func (pages *fakeGitHubPages) CommitGitHubCommits(
	context.Context, int64, int64, int64, []gh.Commit, string, time.Time, time.Time,
) error {
	pages.committed = true
	return nil
}
func (f *fakeStore) PurgeProject(_ context.Context, lease jobstore.Lease, _ int) (bool, jobstore.Lease, error) {
	return true, lease, nil
}

func TestRunnerCommitsGitHubPageAndCheckpointBeforeCompleting(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := &fakeStore{lease: &jobstore.Lease{Job: job.Job{ID: 1, ProjectID: 2,
		Kind: "project_initial_sync", State: job.Running, Progress: job.Progress{Unit: "issues"}, Version: 2}}}
	sources := &fakeGitHubSource{target: jobstore.GitHubSource{ProjectID: 2, RepositoryID: 3,
		SourceID: 4, Owner: "open-source", Repository: "project", Pages: map[string]int{
			"github_issues": 1, "github_pull_requests": 1, "github_releases": 1, "github_commits": 1,
		},
		CoverageFrom: now.AddDate(0, 0, -180)}}
	pages := &fakeGitHubPages{}
	client := fakeGitHub{repository: gh.Repository{ExternalID: 10, DefaultBranch: "main"}, issues: []gh.Issue{{
		ExternalID: 11, CreatedAt: now, UpdatedAt: now, Raw: []byte(`{"id":11}`),
	}}}
	runner, err := worker.NewWithGitHub(store, "worker-1", slog.Default(), client, sources, pages)
	if err != nil {
		t.Fatal(err)
	}
	worked, err := runner.RunOne(context.Background())
	if err != nil || !worked || !pages.committed || store.checkpoints != 4 || !store.completed {
		t.Fatalf("UT-256 page=%v checkpoints=%d complete=%v worked=%v err=%v",
			pages.committed, store.checkpoints, store.completed, worked, err)
	}
}

func TestRunnerStopsCollectionWhenRepositoryBecomesPrivate(t *testing.T) {
	t.Parallel()
	store := &fakeStore{lease: &jobstore.Lease{Job: job.Job{ID: 1, ProjectID: 2,
		Kind: "project_sync", State: job.Running, Progress: job.Progress{Unit: "issues"}}}}
	sources := &fakeGitHubSource{target: jobstore.GitHubSource{ProjectID: 2, RepositoryID: 3,
		SourceID: 4, Owner: "private", Repository: "repo", Pages: map[string]int{
			"github_issues": 1, "github_pull_requests": 1, "github_releases": 1, "github_commits": 1,
		}}}
	runner, err := worker.NewWithGitHub(store, "worker-1", slog.Default(),
		fakeGitHub{err: gh.ErrNotPublic}, sources, &fakeGitHubPages{})
	if err != nil {
		t.Fatal(err)
	}
	worked, err := runner.RunOne(context.Background())
	if !worked || err == nil || !sources.unavailable || !store.failed || store.completed {
		t.Fatalf("UT-084 unavailable=%v failed=%v complete=%v worked=%v err=%v",
			sources.unavailable, store.failed, store.completed, worked, err)
	}
}
func (f *fakeStore) Complete(context.Context, jobstore.Lease) error { f.completed = true; return nil }
func (f *fakeStore) Fail(_ context.Context, _ jobstore.Lease, _ string, delay time.Duration) error {
	f.failed = true
	f.failureDelay = delay
	return nil
}

func TestRunnerCommitsCollectionBeforeCompleting(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := &fakeStore{lease: &jobstore.Lease{Job: job.Job{
		ID: 1, ProjectID: 2, Kind: "project_sync", State: job.Running,
		Progress: job.Progress{Unit: "sources"}, Version: 2, UpdatedAt: now,
	}}}
	runner, err := worker.New(store, "worker-1", slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	worked, err := runner.RunOne(context.Background())
	if err != nil || !worked || !store.marked || !store.completed || store.failed {
		t.Fatalf("UT-256 unexpected result: worked=%v marked=%v complete=%v failed=%v err=%v",
			worked, store.marked, store.completed, store.failed, err)
	}
}

// IT-143: provider reset delays are bounded and resume from the durable boundary.
func TestIT143RateLimitResetUsesBoundedDelayThenResumesWithoutDuplicates(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := &fakeStore{lease: &jobstore.Lease{Job: job.Job{ID: 143, ProjectID: 2,
		Kind: "project_sync", State: job.Running, Progress: job.Progress{Unit: "issues"}, Version: 2}}}
	sources := &fakeGitHubSource{target: jobstore.GitHubSource{ProjectID: 2, RepositoryID: 3,
		SourceID: 4, Owner: "open-source", Repository: "project", Pages: map[string]int{
			"github_issues": 1, "github_pull_requests": 1, "github_releases": 1, "github_commits": 1,
		}}}
	rateLimited := &gh.RateLimitError{Reset: now.Add(2 * time.Second)}
	first, err := worker.NewWithGitHub(store, "worker-rate-limit", slog.Default(),
		fakeGitHub{err: rateLimited}, sources, &fakeGitHubPages{})
	if err != nil {
		t.Fatal(err)
	}
	worked, runErr := first.RunOne(context.Background())
	if !worked || !errors.Is(runErr, gh.ErrRateLimited) || !store.failed ||
		store.failureDelay < time.Second || store.failureDelay > time.Hour || store.checkpoints != 0 {
		t.Fatalf("rate limit result worked=%t err=%v failed=%t delay=%s checkpoints=%d",
			worked, runErr, store.failed, store.failureDelay, store.checkpoints)
	}

	store.failed = false
	store.failureDelay = 0
	pages := &fakeGitHubPages{}
	second, err := worker.NewWithGitHub(store, "worker-after-reset", slog.Default(),
		fakeGitHub{repository: gh.Repository{ExternalID: 10, DefaultBranch: "main"}}, sources, pages)
	if err != nil {
		t.Fatal(err)
	}
	worked, runErr = second.RunOne(context.Background())
	if runErr != nil || !worked || !pages.committed || !store.completed || store.checkpoints != 4 {
		t.Fatalf("resumed result worked=%t err=%v committed=%t complete=%t checkpoints=%d",
			worked, runErr, pages.committed, store.completed, store.checkpoints)
	}
}
