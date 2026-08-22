package intelligencestore

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leohteixeira/opensource-project-intelligence/internal/contributor"
	"github.com/leohteixeira/opensource-project-intelligence/internal/issue"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/pullrequest"
	"github.com/leohteixeira/opensource-project-intelligence/internal/release"
)

// MaterializeProject freezes the supported preset windows at one UTC cutoff. Replays are safe:
// SaveMetricSet compares the complete immutable definition set before returning an existing result.
func (s *Store) MaterializeProject(ctx context.Context, projectID int64, cutoff time.Time) error {
	for _, preset := range []string{"30d", "90d", "180d", "365d"} {
		window, err := metric.PresetWindow(preset, cutoff.UTC())
		if err != nil {
			return err
		}
		if _, err := s.MaterializeWindow(ctx, projectID, window); err != nil {
			return err
		}
	}
	return nil
}

// MaterializeWindow reads canonical facts at one repeatable-read database snapshot and publishes
// the resulting closed metric definition set in a second atomic transaction. Facts after cutoff
// cannot enter the cohort, and a competing identical computation converges on the same keys.
func (s *Store) MaterializeWindow(ctx context.Context, projectID int64, window metric.Window) ([]metric.Snapshot, error) {
	if projectID <= 0 || window.Validate() != nil {
		return nil, metric.ErrInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin metric fact snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var state string
	if err := tx.QueryRow(ctx, `SELECT state FROM projects WHERE id=$1`, projectID).Scan(&state); err != nil {
		return nil, fmt.Errorf("load metric Project: %w", err)
	}
	if !metric.MaterializationAllowed(state) {
		return nil, fmt.Errorf("%w: Project state %s cannot materialize intelligence", metric.ErrInvalid, state)
	}
	facts, err := loadFacts(ctx, tx, projectID, window)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit metric fact snapshot: %w", err)
	}
	values, err := metric.ComputeCatalog(projectID, window, facts)
	if err != nil {
		return nil, err
	}
	return s.SaveMetricSet(ctx, values)
}

func loadFacts(ctx context.Context, tx pgx.Tx, projectID int64, window metric.Window) (metric.Facts, error) {
	facts := metric.Facts{
		IssueClosedAt: make(map[int64]time.Time),
	}
	rows, err := tx.Query(ctx, `SELECT id FROM repositories WHERE project_id=$1 ORDER BY id`, projectID)
	if err != nil {
		return facts, fmt.Errorf("load metric repositories: %w", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return facts, fmt.Errorf("scan metric repository: %w", err)
		}
		facts.RepositoryIDs = append(facts.RepositoryIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return facts, fmt.Errorf("iterate metric repositories: %w", err)
	}
	rows.Close()

	var eligibleSources, coveredSources int
	if err := tx.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE state='available' AND public
		AND coverage_from IS NOT NULL AND coverage_from <= $2
		AND coverage_to IS NOT NULL AND coverage_to >= $3)
		FROM sources WHERE project_id=$1 AND kind IN ('github','gitlab','gitea') AND state <> 'removed'`,
		projectID, window.From, window.To).Scan(&eligibleSources, &coveredSources); err != nil {
		return facts, fmt.Errorf("load metric source coverage: %w", err)
	}
	covered := eligibleSources > 0 && eligibleSources == coveredSources
	facts.ReleaseCovered, facts.CommitCovered, facts.IssueCovered, facts.PRCovered = covered, covered, covered, covered

	rows, err = tx.Query(ctx, `SELECT id,published_at,draft,prerelease FROM canonical_releases
		WHERE project_id=$1 AND published_at >= $2 AND published_at < $3 AND published_at <= $4
		ORDER BY published_at,id`, projectID, window.From, window.To, window.Cutoff)
	if err != nil {
		return facts, fmt.Errorf("load release metric facts: %w", err)
	}
	for rows.Next() {
		var value release.Release
		if err := rows.Scan(&value.ID, &value.PublishedAt, &value.Draft, &value.Prerelease); err != nil {
			rows.Close()
			return facts, fmt.Errorf("scan release metric fact: %w", err)
		}
		facts.Releases = append(facts.Releases, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return facts, fmt.Errorf("iterate release metric facts: %w", err)
	}
	rows.Close()

	rows, err = tx.Query(ctx, `SELECT c.id,c.author_external_id,c.committed_at,c.default_branch,c.merge_commit,
		COALESCE(a.bot,false),COALESCE(l.status,'unresolved'),COALESCE(l.identity_id,0)
		FROM canonical_commits c
		LEFT JOIN contributor_accounts a ON a.source_id=c.source_id AND a.external_id=c.author_external_id
		LEFT JOIN contributor_identity_links l ON l.account_id=a.id
		WHERE c.project_id=$1 AND c.committed_at >= $2 AND c.committed_at < $3 AND c.committed_at <= $4
		ORDER BY c.committed_at,c.id`, projectID, window.From, window.To, window.Cutoff)
	if err != nil {
		return facts, fmt.Errorf("load contributor metric facts: %w", err)
	}
	for rows.Next() {
		var value contributor.Commit
		var identityID int64
		if err := rows.Scan(&value.ID, &value.AccountID, &value.CommittedAt, &value.DefaultBranch,
			&value.MergeCommit, &value.Bot, &value.LinkStatus, &identityID); err != nil {
			rows.Close()
			return facts, fmt.Errorf("scan contributor metric fact: %w", err)
		}
		if identityID > 0 {
			value.IdentityID = strconv.FormatInt(identityID, 10)
		}
		facts.Commits = append(facts.Commits, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return facts, fmt.Errorf("iterate contributor metric facts: %w", err)
	}
	rows.Close()

	issues := make(map[int64]*issue.Issue)
	rows, err = tx.Query(ctx, `SELECT id,created_at,closed_at FROM canonical_issues
		WHERE project_id=$1 AND created_at < $2 AND created_at <= $3 ORDER BY created_at,id`,
		projectID, window.To, window.Cutoff)
	if err != nil {
		return facts, fmt.Errorf("load issue metric facts: %w", err)
	}
	for rows.Next() {
		var value issue.Issue
		var closedAt *time.Time
		if err := rows.Scan(&value.ID, &value.CreatedAt, &closedAt); err != nil {
			rows.Close()
			return facts, fmt.Errorf("scan issue metric fact: %w", err)
		}
		value.OpenerID = "opener"
		issues[value.ID] = &value
		facts.Issues = append(facts.Issues, value)
		facts.IssueEvents = append(facts.IssueEvents, issue.StateEvent{IssueID: value.ID, At: value.CreatedAt, State: "open"})
		if closedAt != nil && !closedAt.After(window.Cutoff) {
			facts.IssueClosedAt[value.ID] = *closedAt
			facts.IssueEvents = append(facts.IssueEvents, issue.StateEvent{IssueID: value.ID, At: *closedAt, State: "closed"})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return facts, fmt.Errorf("iterate issue metric facts: %w", err)
	}
	rows.Close()

	rows, err = tx.Query(ctx, `SELECT e.issue_id,e.actor_external_id,e.actor_is_opener,e.public,e.bot,
		e.recognized_member,e.occurred_at FROM issue_response_events e
		JOIN canonical_issues i ON i.id=e.issue_id
		WHERE i.project_id=$1 AND e.occurred_at <= $2 ORDER BY e.occurred_at,e.id`, projectID, window.Cutoff)
	if err != nil {
		return facts, fmt.Errorf("load issue response facts: %w", err)
	}
	for rows.Next() {
		var issueID int64
		var actor string
		var opener bool
		var response issue.Response
		if err := rows.Scan(&issueID, &actor, &opener, &response.Public, &response.Bot,
			&response.Member, &response.At); err != nil {
			rows.Close()
			return facts, fmt.Errorf("scan issue response fact: %w", err)
		}
		if opener {
			actor = "opener"
		}
		response.ActorID = actor
		if value := issues[issueID]; value != nil {
			value.Responses = append(value.Responses, response)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return facts, fmt.Errorf("iterate issue response facts: %w", err)
	}
	rows.Close()
	// Copy attached responses back without introducing map-order nondeterminism.
	for index := range facts.Issues {
		facts.Issues[index].Responses = issues[facts.Issues[index].ID].Responses
	}

	rows, err = tx.Query(ctx, `SELECT e.issue_id,e.occurred_at,e.state FROM issue_state_events e
		JOIN canonical_issues i ON i.id=e.issue_id
		WHERE i.project_id=$1 AND e.occurred_at <= $2 ORDER BY e.occurred_at,e.id`, projectID, window.Cutoff)
	if err != nil {
		return facts, fmt.Errorf("load issue state facts: %w", err)
	}
	for rows.Next() {
		var event issue.StateEvent
		if err := rows.Scan(&event.IssueID, &event.At, &event.State); err != nil {
			rows.Close()
			return facts, fmt.Errorf("scan issue state fact: %w", err)
		}
		facts.IssueEvents = append(facts.IssueEvents, event)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return facts, fmt.Errorf("iterate issue state facts: %w", err)
	}
	rows.Close()

	rows, err = tx.Query(ctx, `SELECT p.created_at,
		COALESCE((SELECT min(e.occurred_at) FROM pull_request_readiness_events e
			WHERE e.pull_request_id=p.id AND e.state='ready_for_review' AND e.occurred_at <= $4),p.ready_at),
		p.merged_at FROM canonical_pull_requests p
		WHERE p.project_id=$1 AND p.merged_at >= $2 AND p.merged_at < $3 AND p.merged_at <= $4
		ORDER BY p.merged_at,p.id`, projectID, window.From, window.To, window.Cutoff)
	if err != nil {
		return facts, fmt.Errorf("load pull request metric facts: %w", err)
	}
	for rows.Next() {
		var value pullrequest.PullRequest
		if err := rows.Scan(&value.CreatedAt, &value.ReadyAt, &value.MergedAt); err != nil {
			rows.Close()
			return facts, fmt.Errorf("scan pull request metric fact: %w", err)
		}
		facts.PullRequests = append(facts.PullRequests, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return facts, fmt.Errorf("iterate pull request metric facts: %w", err)
	}
	rows.Close()
	return facts, nil
}
