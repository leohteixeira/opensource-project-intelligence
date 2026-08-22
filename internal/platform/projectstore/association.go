package projectstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

func (s *Store) ListAssociations(
	ctx context.Context,
	principal access.Principal,
	projectID int64,
	filter Filter,
) ([]project.Association, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return nil, err
	}
	limit, offset := page(filter.Limit, filter.Offset)
	rows, err := s.pool.Query(ctx, `
		SELECT a.id,a.source_id,a.project_id,a.method,a.confidence,a.evidence_ids,
			a.decision_version,a.status,a.version
		FROM source_associations a JOIN projects p ON p.id=a.project_id
		WHERE a.project_id=$1 AND p.workspace_id=$2 AND ($3='' OR a.status=$3)
		ORDER BY a.status='unresolved' DESC,a.confidence,a.id LIMIT $4 OFFSET $5`,
		projectID, principal.Workspace, filter.State, limit+1, offset)
	if err != nil {
		return nil, fmt.Errorf("list source associations: %w", err)
	}
	defer rows.Close()
	values := make([]project.Association, 0, limit)
	for rows.Next() {
		var value project.Association
		if err := rows.Scan(&value.ID, &value.SourceID, &value.ProjectID, &value.Method,
			&value.Confidence, &value.Evidence, &value.DecisionVersion, &value.Status,
			&value.Version); err != nil {
			return nil, fmt.Errorf("scan source association: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Store) CorrectAssociation(
	ctx context.Context,
	principal access.Principal,
	projectID, associationID, targetProjectID int64,
	action, reason, requestID string,
) (jobID int64, changed bool, err error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return 0, false, err
	}
	correction := project.Correction{
		Action: action, ProjectID: targetProjectID, Reason: strings.TrimSpace(reason),
		ActorID: principal.ActorID, At: s.now().UTC(),
	}
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		var replayed bool
		err := tx.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM identity_corrections c
			JOIN projects p ON p.id=c.from_project_id
			WHERE c.association_id=$1 AND c.action=$2
				AND c.to_project_id IS NOT DISTINCT FROM $3
				AND c.from_project_id=$4 AND p.workspace_id=$5
		)`, associationID, action, nullableProject(targetProjectID), projectID,
			principal.Workspace).Scan(&replayed)
		if err != nil {
			return fmt.Errorf("check source correction replay: %w", err)
		}
		if replayed {
			return nil
		}
		var association project.Association
		err = tx.QueryRow(ctx, `
			SELECT id,source_id,project_id,method,confidence,evidence_ids,decision_version,status,version
			FROM source_associations WHERE id=$1 AND project_id=$2 FOR UPDATE`,
			associationID, projectID).Scan(&association.ID, &association.SourceID, &association.ProjectID,
			&association.Method, &association.Confidence, &association.Evidence,
			&association.DecisionVersion, &association.Status, &association.Version)
		if errors.Is(err, pgx.ErrNoRows) {
			return project.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("lock source association: %w", err)
		}
		if action == "reassign" {
			var targetState project.State
			err = tx.QueryRow(ctx, `SELECT state FROM projects WHERE id=$1 AND workspace_id=$2 FOR UPDATE`,
				targetProjectID, principal.Workspace).Scan(&targetState)
			if errors.Is(err, pgx.ErrNoRows) {
				return project.ErrNotFound
			}
			if err != nil {
				return fmt.Errorf("validate reassignment target: %w", err)
			}
			if targetState == project.StateDeleting || targetState == project.StateDeleted {
				return fmt.Errorf("%w: target project cannot accept sources", project.ErrConflict)
			}
		}
		updated, shouldRecalculate, err := association.Correct(correction)
		if err != nil {
			return err
		}
		if !shouldRecalculate {
			return nil
		}
		changed = true
		correctionID, err := s.ids.Next(ctx)
		if err != nil {
			return fmt.Errorf("issue correction ID: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO identity_corrections
			(id,association_id,action,from_project_id,to_project_id,reason,actor_id,request_id,created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, correctionID, associationID, action,
			projectID, nullableProject(targetProjectID), correction.Reason, principal.ActorID,
			requestID, correction.At)
		if err != nil {
			return mapConflict("record source correction", err)
		}
		_, err = tx.Exec(ctx, `UPDATE source_associations SET project_id=$1,status='corrected',
			version=$2,updated_at=now() WHERE id=$3`, updated.ProjectID, updated.Version, associationID)
		if err != nil {
			return fmt.Errorf("apply source correction: %w", err)
		}
		jobID, err = s.ids.Next(ctx)
		if err != nil {
			return fmt.Errorf("issue correction job ID: %w", err)
		}
		queued, err := newCorrectionJob(jobID, projectID, s.now())
		if err != nil {
			return err
		}
		if err := insertJob(ctx, tx, queued, principal, requestID,
			fmt.Sprintf("association:%d:%s:%d", associationID, action, targetProjectID)); err != nil {
			return err
		}
		if err := s.recordJobEvent(ctx, tx, queued); err != nil {
			return err
		}
		return s.writeOutbox(ctx, tx, projectID, jobID, "association.corrected", principal,
			requestID, map[string]any{"association_id": associationID, "job_id": jobID})
	})
	return jobID, changed, err
}

func newCorrectionJob(id, projectID int64, now time.Time) (value job.Job, err error) {
	return job.New(id, projectID, "association_recalculation", "snapshots", nil, true, now)
}
