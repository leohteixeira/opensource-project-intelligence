// Package ingestion atomically commits normalized public-source pages and checkpoints.
package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	gh "github.com/leohteixeira/opensource-project-intelligence/internal/platform/github"
)

type IDSource interface {
	Next(context.Context) (int64, error)
}

type Store struct {
	pool *pgxpool.Pool
	ids  IDSource
	now  func() time.Time
}

func New(pool *database.Pool, ids IDSource) *Store {
	return &Store{pool: pool.Unwrap(), ids: ids, now: time.Now}
}

// CommitGitHubIssues preserves each provider payload before mapping it to the
// canonical issue model. The page checkpoint advances in the same transaction;
// normalization or persistence failure leaves both facts and cursor unchanged.
func (s *Store) CommitGitHubIssues(
	ctx context.Context,
	projectID, repositoryID, sourceID int64,
	issues []gh.Issue,
	nextCursor string,
	coverageFrom, coverageTo time.Time,
) error {
	if projectID <= 0 || repositoryID <= 0 || sourceID <= 0 || nextCursor == "" {
		return fmt.Errorf("invalid GitHub issue page identity")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin GitHub issue page: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, issue := range issues {
		if issue.ExternalID <= 0 || issue.CreatedAt.IsZero() || issue.UpdatedAt.IsZero() ||
			!json.Valid(issue.Raw) {
			return fmt.Errorf("normalize GitHub issue %d: malformed provider data", issue.ExternalID)
		}
		digest := sha256.Sum256(issue.Raw)
		rawID, err := s.ids.Next(ctx)
		if err != nil {
			return fmt.Errorf("issue raw object ID: %w", err)
		}
		var storedRawID int64
		err = tx.QueryRow(ctx, `INSERT INTO raw_objects
			(id,project_id,source_id,external_type,external_id,observed_at,payload,digest)
			VALUES ($1,$2,$3,'github_issue',$4,$5,$6,$7)
			ON CONFLICT (source_id,external_type,external_id,digest) DO UPDATE
				SET observed_at=LEAST(raw_objects.observed_at,EXCLUDED.observed_at)
			RETURNING id`, rawID, projectID, sourceID, strconv.FormatInt(issue.ExternalID, 10),
			issue.UpdatedAt, issue.Raw, digest[:]).Scan(&storedRawID)
		if err != nil {
			return fmt.Errorf("retain GitHub issue evidence: %w", err)
		}
		canonicalID, err := s.ids.Next(ctx)
		if err != nil {
			return fmt.Errorf("issue canonical issue ID: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO canonical_issues
			(id,project_id,repository_id,source_id,external_id,number,title,state,created_at,
			 updated_at,closed_at,raw_object_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (source_id,external_id) DO UPDATE SET
				number=EXCLUDED.number,title=EXCLUDED.title,state=EXCLUDED.state,
				created_at=EXCLUDED.created_at,updated_at=EXCLUDED.updated_at,
				closed_at=EXCLUDED.closed_at,raw_object_id=EXCLUDED.raw_object_id
			WHERE canonical_issues.updated_at <= EXCLUDED.updated_at`,
			canonicalID, projectID, repositoryID, sourceID, strconv.FormatInt(issue.ExternalID, 10),
			issue.Number, issue.Title, issue.State, issue.CreatedAt, issue.UpdatedAt, issue.ClosedAt,
			storedRawID)
		if err != nil {
			return fmt.Errorf("upsert canonical GitHub issue: %w", err)
		}
	}
	checkpointID, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("issue checkpoint ID: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO sync_checkpoints
		(id,project_id,source_id,scope,cursor,coverage_from,coverage_to)
		VALUES ($1,$2,$3,'github_issues',$4,$5,$6)
		ON CONFLICT (source_id,scope) DO UPDATE SET cursor=EXCLUDED.cursor,
			coverage_from=LEAST(sync_checkpoints.coverage_from,EXCLUDED.coverage_from),
			coverage_to=GREATEST(sync_checkpoints.coverage_to,EXCLUDED.coverage_to),
			version=sync_checkpoints.version+1,updated_at=now()`,
		checkpointID, projectID, sourceID, nextCursor, coverageFrom.UTC(), coverageTo.UTC())
	if err != nil {
		return fmt.Errorf("advance GitHub issue checkpoint: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE sources SET coverage_from=LEAST(coverage_from,$1),
		coverage_to=GREATEST(coverage_to,$2),last_attempt_at=$3,last_success_at=$3,
		failure_code='',version=version+1,updated_at=$3 WHERE id=$4 AND project_id=$5`,
		coverageFrom.UTC(), coverageTo.UTC(), s.now().UTC(), sourceID, projectID)
	if err != nil {
		return fmt.Errorf("advance GitHub source coverage: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit GitHub issue page: %w", err)
	}
	return nil
}

func (s *Store) CommitGitHubPullRequests(
	ctx context.Context,
	projectID, repositoryID, sourceID int64,
	values []gh.PullRequest,
	nextCursor string,
	coverageFrom, coverageTo time.Time,
) error {
	if err := validPageIdentity(projectID, repositoryID, sourceID, nextCursor); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin GitHub pull-request page: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, value := range values {
		if value.ExternalID <= 0 || value.CreatedAt.IsZero() || !json.Valid(value.Raw) ||
			value.State != "open" && value.State != "closed" && value.State != "merged" {
			return fmt.Errorf("normalize GitHub pull request %d: malformed provider data", value.ExternalID)
		}
		rawID, err := s.retainRaw(ctx, tx, projectID, sourceID, "github_pull_request",
			strconv.FormatInt(value.ExternalID, 10), value.CreatedAt, value.Raw)
		if err != nil {
			return err
		}
		canonicalID, err := s.ids.Next(ctx)
		if err != nil {
			return fmt.Errorf("pull-request canonical ID: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO canonical_pull_requests
			(id,project_id,repository_id,source_id,external_id,number,title,state,created_at,
			 ready_at,merged_at,raw_object_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT(source_id,external_id) DO UPDATE SET
				number=EXCLUDED.number,title=EXCLUDED.title,state=EXCLUDED.state,
				created_at=EXCLUDED.created_at,ready_at=EXCLUDED.ready_at,
				merged_at=EXCLUDED.merged_at,raw_object_id=EXCLUDED.raw_object_id`,
			canonicalID, projectID, repositoryID, sourceID, strconv.FormatInt(value.ExternalID, 10),
			value.Number, value.Title, value.State, value.CreatedAt.UTC(), value.ReadyAt,
			value.MergedAt, rawID)
		if err != nil {
			return fmt.Errorf("upsert canonical GitHub pull request: %w", err)
		}
	}
	return s.finishPage(ctx, tx, projectID, sourceID, "github_pull_requests", nextCursor,
		coverageFrom, coverageTo)
}

func (s *Store) CommitGitHubReleases(
	ctx context.Context,
	projectID, repositoryID, sourceID int64,
	values []gh.Release,
	nextCursor string,
	coverageFrom, coverageTo time.Time,
) error {
	if err := validPageIdentity(projectID, repositoryID, sourceID, nextCursor); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin GitHub release page: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, value := range values {
		if value.ExternalID <= 0 || strings.TrimSpace(value.Tag) == "" || !json.Valid(value.Raw) {
			return fmt.Errorf("normalize GitHub release %d: malformed provider data", value.ExternalID)
		}
		observedAt := coverageTo
		if value.PublishedAt != nil {
			observedAt = *value.PublishedAt
		}
		rawID, err := s.retainRaw(ctx, tx, projectID, sourceID, "github_release",
			strconv.FormatInt(value.ExternalID, 10), observedAt, value.Raw)
		if err != nil {
			return err
		}
		canonicalID, err := s.ids.Next(ctx)
		if err != nil {
			return fmt.Errorf("release canonical ID: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO canonical_releases
			(id,project_id,repository_id,source_id,external_id,tag,draft,prerelease,published_at,raw_object_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT(source_id,external_id) DO UPDATE SET
				tag=EXCLUDED.tag,draft=EXCLUDED.draft,prerelease=EXCLUDED.prerelease,
				published_at=EXCLUDED.published_at,raw_object_id=EXCLUDED.raw_object_id`,
			canonicalID, projectID, repositoryID, sourceID, strconv.FormatInt(value.ExternalID, 10),
			value.Tag, value.Draft, value.Prerelease, value.PublishedAt, rawID)
		if err != nil {
			return fmt.Errorf("upsert canonical GitHub release: %w", err)
		}
	}
	return s.finishPage(ctx, tx, projectID, sourceID, "github_releases", nextCursor,
		coverageFrom, coverageTo)
}

func (s *Store) CommitGitHubCommits(
	ctx context.Context,
	projectID, repositoryID, sourceID int64,
	values []gh.Commit,
	nextCursor string,
	coverageFrom, coverageTo time.Time,
) error {
	if err := validPageIdentity(projectID, repositoryID, sourceID, nextCursor); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin GitHub commit page: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, value := range values {
		if strings.TrimSpace(value.ExternalID) == "" || strings.TrimSpace(value.SHA) == "" ||
			value.CommittedAt.IsZero() || !json.Valid(value.Raw) {
			return fmt.Errorf("normalize GitHub commit %q: malformed provider data", value.ExternalID)
		}
		rawID, err := s.retainRaw(ctx, tx, projectID, sourceID, "github_commit",
			value.ExternalID, value.CommittedAt, value.Raw)
		if err != nil {
			return err
		}
		canonicalID, err := s.ids.Next(ctx)
		if err != nil {
			return fmt.Errorf("commit canonical ID: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO canonical_commits
			(id,project_id,repository_id,source_id,external_id,sha,author_external_id,
			 committed_at,default_branch,merge_commit,raw_object_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			ON CONFLICT(repository_id,sha) DO UPDATE SET
				external_id=EXCLUDED.external_id,author_external_id=EXCLUDED.author_external_id,
				committed_at=EXCLUDED.committed_at,default_branch=EXCLUDED.default_branch,
				merge_commit=EXCLUDED.merge_commit,raw_object_id=EXCLUDED.raw_object_id`,
			canonicalID, projectID, repositoryID, sourceID, value.ExternalID, value.SHA,
			value.AuthorExternalID, value.CommittedAt.UTC(), value.DefaultBranch, value.MergeCommit, rawID)
		if err != nil {
			return fmt.Errorf("upsert canonical GitHub commit: %w", err)
		}
	}
	return s.finishPage(ctx, tx, projectID, sourceID, "github_commits", nextCursor,
		coverageFrom, coverageTo)
}

func validPageIdentity(projectID, repositoryID, sourceID int64, nextCursor string) error {
	if projectID <= 0 || repositoryID <= 0 || sourceID <= 0 || strings.TrimSpace(nextCursor) == "" {
		return fmt.Errorf("invalid GitHub page identity")
	}
	return nil
}

func (s *Store) retainRaw(
	ctx context.Context,
	tx pgx.Tx,
	projectID, sourceID int64,
	externalType, externalID string,
	observedAt time.Time,
	raw json.RawMessage,
) (int64, error) {
	digest := sha256.Sum256(raw)
	issuedID, err := s.ids.Next(ctx)
	if err != nil {
		return 0, fmt.Errorf("%s raw object ID: %w", externalType, err)
	}
	var storedID int64
	err = tx.QueryRow(ctx, `INSERT INTO raw_objects
		(id,project_id,source_id,external_type,external_id,observed_at,payload,digest)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(source_id,external_type,external_id,digest) DO UPDATE
			SET observed_at=LEAST(raw_objects.observed_at,EXCLUDED.observed_at)
		RETURNING id`, issuedID, projectID, sourceID, externalType, externalID,
		observedAt.UTC(), raw, digest[:]).Scan(&storedID)
	if err != nil {
		return 0, fmt.Errorf("retain %s evidence: %w", externalType, err)
	}
	return storedID, nil
}

func (s *Store) finishPage(
	ctx context.Context,
	tx pgx.Tx,
	projectID, sourceID int64,
	scope, nextCursor string,
	coverageFrom, coverageTo time.Time,
) error {
	checkpointID, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("%s checkpoint ID: %w", scope, err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO sync_checkpoints
		(id,project_id,source_id,scope,cursor,coverage_from,coverage_to)
		VALUES($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT(source_id,scope) DO UPDATE SET cursor=EXCLUDED.cursor,
			coverage_from=LEAST(sync_checkpoints.coverage_from,EXCLUDED.coverage_from),
			coverage_to=GREATEST(sync_checkpoints.coverage_to,EXCLUDED.coverage_to),
			version=sync_checkpoints.version+1,updated_at=now()`,
		checkpointID, projectID, sourceID, scope, nextCursor,
		coverageFrom.UTC(), coverageTo.UTC())
	if err != nil {
		return fmt.Errorf("advance %s checkpoint: %w", scope, err)
	}
	now := s.now().UTC()
	_, err = tx.Exec(ctx, `UPDATE sources SET coverage_from=LEAST(coverage_from,$1),
		coverage_to=GREATEST(coverage_to,$2),last_attempt_at=$3,last_success_at=$3,
		failure_code='',version=version+1,updated_at=$3 WHERE id=$4 AND project_id=$5`,
		coverageFrom.UTC(), coverageTo.UTC(), now, sourceID, projectID)
	if err != nil {
		return fmt.Errorf("advance GitHub source coverage: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s page: %w", scope, err)
	}
	return nil
}
