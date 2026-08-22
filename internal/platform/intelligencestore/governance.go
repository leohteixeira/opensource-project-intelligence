package intelligencestore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/alert"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/policy"
	"github.com/leohteixeira/opensource-project-intelligence/internal/radar"
	"github.com/leohteixeira/opensource-project-intelligence/internal/trend"
)

// SaveObserved publishes one immutable observed signal. Exact replays return the original row.
func (s *Store) SaveObserved(ctx context.Context, value trend.Observed) (trend.Observed, error) {
	if value.Kind != trend.KindObserved || value.ProjectID <= 0 || value.InputDigest == "" {
		return trend.Observed{}, trend.ErrInvalid
	}
	if value.ID == 0 {
		id, err := s.ids.Next(ctx)
		if err != nil {
			return trend.Observed{}, fmt.Errorf("trend signal ID: %w", err)
		}
		value.ID = id
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return trend.Observed{}, fmt.Errorf("begin observed signal: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	inserted := false
	err = tx.QueryRow(ctx, `INSERT INTO trend_signals
		(id,project_id,metric_name,metric_version,kind,status,method_version,window_from,window_to,
		 baseline_from,baseline_to,cutoff,magnitude,eligible_count,observed_count,coverage_note,input_digest,superseded_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NULLIF($18,0))
		ON CONFLICT(project_id,metric_name,metric_version,kind,method_version,window_from,window_to,cutoff,input_digest)
		DO NOTHING RETURNING true`, value.ID, value.ProjectID, value.MetricName, value.MetricVersion,
		value.Kind, value.Direction, value.MethodVersion, value.ObservationWindow.From,
		value.ObservationWindow.To, value.BaselineWindow.From, value.BaselineWindow.To,
		value.ObservationWindow.Cutoff, value.Magnitude, value.Coverage.Eligible,
		value.Coverage.Observed, value.Coverage.Note, value.InputDigest, value.SupersededBy).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.observedByKey(ctx, value)
	}
	if err != nil {
		return trend.Observed{}, fmt.Errorf("insert observed signal: %w", err)
	}
	if err := insertSignalEvidence(ctx, tx, value.ID, value.EvidenceIDs); err != nil {
		return trend.Observed{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return trend.Observed{}, fmt.Errorf("commit observed signal: %w", err)
	}
	return value, nil
}

func (s *Store) observedByKey(ctx context.Context, expected trend.Observed) (trend.Observed, error) {
	var value trend.Observed
	err := s.pool.QueryRow(ctx, `SELECT id,status,magnitude,eligible_count,observed_count,coverage_note,
		COALESCE(superseded_by,0),baseline_from,baseline_to FROM trend_signals
		WHERE project_id=$1 AND metric_name=$2 AND metric_version=$3 AND kind='observed'
		AND method_version=$4 AND window_from=$5 AND window_to=$6 AND cutoff=$7 AND input_digest=$8`,
		expected.ProjectID, expected.MetricName, expected.MetricVersion, expected.MethodVersion,
		expected.ObservationWindow.From, expected.ObservationWindow.To, expected.ObservationWindow.Cutoff,
		expected.InputDigest).Scan(&value.ID, &value.Direction, &value.Magnitude, &value.Coverage.Eligible,
		&value.Coverage.Observed, &value.Coverage.Note, &value.SupersededBy, &value.BaselineWindow.From,
		&value.BaselineWindow.To)
	if err != nil {
		return trend.Observed{}, fmt.Errorf("load observed signal replay: %w", err)
	}
	value.ProjectID, value.MetricName, value.MetricVersion, value.Kind = expected.ProjectID, expected.MetricName, expected.MetricVersion, trend.KindObserved
	value.MethodVersion, value.ObservationWindow, value.InputDigest, value.MinimumPoints = expected.MethodVersion, expected.ObservationWindow, expected.InputDigest, expected.MinimumPoints
	value.BaselineWindow.Cutoff = expected.ObservationWindow.Cutoff
	value.EvidenceIDs, err = loadSignalEvidence(ctx, s.pool, value.ID)
	if err != nil {
		return trend.Observed{}, err
	}
	want := expected
	want.ID = value.ID
	if !reflect.DeepEqual(normalizeObserved(want), normalizeObserved(value)) {
		return trend.Observed{}, fmt.Errorf("%w: observed signal key collision", trend.ErrInvalid)
	}
	return value, nil
}

func normalizeObserved(value trend.Observed) trend.Observed {
	value.EvidenceIDs = slices.Compact(slices.Sorted(slices.Values(value.EvidenceIDs)))
	value.ObservationWindow.From = value.ObservationWindow.From.UTC()
	value.ObservationWindow.To = value.ObservationWindow.To.UTC()
	value.ObservationWindow.Cutoff = value.ObservationWindow.Cutoff.UTC()
	value.BaselineWindow.From = value.BaselineWindow.From.UTC()
	value.BaselineWindow.To = value.BaselineWindow.To.UTC()
	value.BaselineWindow.Cutoff = value.BaselineWindow.Cutoff.UTC()
	return value
}

// SaveForecast publishes an immutable forecast without affecting observed signals on failure.
func (s *Store) SaveForecast(ctx context.Context, value trend.Forecast, inputDigest string,
	window metric.Window) (trend.Forecast, error) {
	if value.Kind != trend.KindForecast || value.ProjectID <= 0 || inputDigest == "" || window.Validate() != nil {
		return trend.Forecast{}, trend.ErrInvalid
	}
	if value.ID == 0 {
		id, err := s.ids.Next(ctx)
		if err != nil {
			return trend.Forecast{}, fmt.Errorf("forecast signal ID: %w", err)
		}
		value.ID = id
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return trend.Forecast{}, fmt.Errorf("begin forecast signal: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	inserted := false
	err = tx.QueryRow(ctx, `INSERT INTO trend_signals
		(id,project_id,metric_name,metric_version,kind,status,method_version,selected_model,
		 window_from,window_to,cutoff,horizon_days,predicted_value,interval_low,interval_high,confidence,
		 backtest_error,eligible_count,observed_count,coverage_note,input_digest,outcome_status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
		ON CONFLICT(project_id,metric_name,metric_version,kind,method_version,window_from,window_to,cutoff,input_digest)
		DO NOTHING RETURNING true`, value.ID, value.ProjectID, value.MetricName, value.MetricVersion,
		value.Kind, value.Status, value.ModelVersion, value.SelectedModel, window.From, window.To, window.Cutoff,
		value.HorizonDays, value.Predicted, value.IntervalLow, value.IntervalHigh, value.Confidence,
		value.BacktestError, value.Coverage.Eligible, value.Coverage.Observed, value.Coverage.Note,
		inputDigest, value.OutcomeStatus).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.forecastByKey(ctx, value, inputDigest, window)
	}
	if err != nil {
		return trend.Forecast{}, fmt.Errorf("insert forecast signal: %w", err)
	}
	if err := insertSignalEvidence(ctx, tx, value.ID, value.EvidenceIDs); err != nil {
		return trend.Forecast{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return trend.Forecast{}, fmt.Errorf("commit forecast signal: %w", err)
	}
	return value, nil
}

func (s *Store) forecastByKey(ctx context.Context, expected trend.Forecast, inputDigest string,
	window metric.Window) (trend.Forecast, error) {
	var value trend.Forecast
	err := s.pool.QueryRow(ctx, `SELECT id,status,horizon_days,predicted_value,interval_low,interval_high,
		confidence,backtest_error,selected_model,eligible_count,observed_count,coverage_note,
		outcome_status,COALESCE(superseded_by,0) FROM trend_signals
		WHERE project_id=$1 AND metric_name=$2 AND metric_version=$3 AND kind='forecast'
		AND method_version=$4 AND window_from=$5 AND window_to=$6 AND cutoff=$7 AND input_digest=$8`,
		expected.ProjectID, expected.MetricName, expected.MetricVersion, expected.ModelVersion,
		window.From, window.To, window.Cutoff, inputDigest).Scan(&value.ID, &value.Status,
		&value.HorizonDays, &value.Predicted, &value.IntervalLow, &value.IntervalHigh, &value.Confidence,
		&value.BacktestError, &value.SelectedModel, &value.Coverage.Eligible, &value.Coverage.Observed,
		&value.Coverage.Note, &value.OutcomeStatus, &value.SupersededBy)
	if err != nil {
		return trend.Forecast{}, fmt.Errorf("load forecast replay: %w", err)
	}
	value.ProjectID, value.MetricName, value.MetricVersion = expected.ProjectID, expected.MetricName, expected.MetricVersion
	value.Kind, value.ModelVersion = trend.KindForecast, expected.ModelVersion
	value.EvidenceIDs, err = loadSignalEvidence(ctx, s.pool, value.ID)
	if err != nil {
		return trend.Forecast{}, err
	}
	want := expected
	want.ID = value.ID
	// Explanations are attached after publication and are deliberately outside the immutable
	// statistical result stored by this operation.
	want.Explanation = ""
	if !reflect.DeepEqual(normalizeForecast(want), normalizeForecast(value)) {
		return trend.Forecast{}, fmt.Errorf("%w: forecast signal key collision", trend.ErrInvalid)
	}
	return value, nil
}

func normalizeForecast(value trend.Forecast) trend.Forecast {
	value.EvidenceIDs = slices.Compact(slices.Sorted(slices.Values(value.EvidenceIDs)))
	return value
}

func insertSignalEvidence(ctx context.Context, tx pgx.Tx, signalID int64, evidence []int64) error {
	for ordinal, id := range slices.Compact(slices.Sorted(slices.Values(evidence))) {
		if id <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO trend_signal_evidence(signal_id,ordinal,evidence_id)
			VALUES($1,$2,$3)`, signalID, ordinal, id); err != nil {
			return fmt.Errorf("insert signal evidence: %w", err)
		}
	}
	return nil
}

func loadSignalEvidence(ctx context.Context, query rowQuery, signalID int64) ([]int64, error) {
	rows, err := query.Query(ctx, `SELECT evidence_id FROM trend_signal_evidence WHERE signal_id=$1 ORDER BY ordinal`, signalID)
	if err != nil {
		return nil, fmt.Errorf("load signal evidence: %w", err)
	}
	defer rows.Close()
	values := make([]int64, 0)
	for rows.Next() {
		var value int64
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan signal evidence: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

// Signals returns a stable bounded signal history. The wire representation preserves kind-specific fields.
func (s *Store) Signals(ctx context.Context, principal access.Principal, projectID int64,
	kind trend.Kind, limit, offset int) ([]map[string]any, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return nil, err
	}
	if kind != trend.KindObserved && kind != trend.KindForecast || limit <= 0 || limit > 200 || offset < 0 {
		return nil, trend.ErrInvalid
	}
	rows, err := s.pool.Query(ctx, `SELECT id,metric_name,metric_version,status,method_version,selected_model,
		window_from,window_to,baseline_from,baseline_to,cutoff,magnitude,horizon_days,predicted_value,
		interval_low,interval_high,confidence,backtest_error,eligible_count,observed_count,coverage_note,
		outcome_status,COALESCE(superseded_by,0) FROM trend_signals
		WHERE project_id=$1 AND kind=$2 ORDER BY cutoff DESC,id DESC LIMIT $3 OFFSET $4`, projectID, kind, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list signals: %w", err)
	}
	defer rows.Close()
	values := make([]map[string]any, 0, limit)
	for rows.Next() {
		var id, superseded int64
		var metricName, metricVersion, status, method, selected, coverageNote, outcome string
		var windowFrom, windowTo, cutoff time.Time
		var baselineFrom, baselineTo *time.Time
		var magnitude, predicted, low, high, confidence, backtest *float64
		var horizon *int
		var eligible, observed int
		if err := rows.Scan(&id, &metricName, &metricVersion, &status, &method, &selected,
			&windowFrom, &windowTo, &baselineFrom, &baselineTo, &cutoff, &magnitude, &horizon, &predicted,
			&low, &high, &confidence, &backtest, &eligible, &observed, &coverageNote, &outcome, &superseded); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		evidence, loadErr := loadSignalEvidence(ctx, s.pool, id)
		if loadErr != nil {
			return nil, loadErr
		}
		values = append(values, map[string]any{"id": id, "project_id": projectID, "kind": kind,
			"metric_name": metricName, "metric_version": metricVersion, "status": status,
			"method_version": method, "selected_model": selected, "window_from": windowFrom,
			"window_to": windowTo, "baseline_from": baselineFrom, "baseline_to": baselineTo,
			"cutoff": cutoff, "magnitude": magnitude, "horizon_days": horizon, "predicted": predicted,
			"interval_low": low, "interval_high": high, "confidence": confidence,
			"backtest_error": backtest, "coverage": metric.Coverage{Eligible: eligible, Observed: observed, Note: coverageNote},
			"outcome_status": outcome, "evidence_ids": evidence, "superseded_by": superseded})
	}
	return values, rows.Err()
}

func (s *Store) CreatePolicy(ctx context.Context, principal access.Principal, value policy.Version,
	requestID string) (policy.Version, error) {
	if err := policy.CanGovern(principal); err != nil {
		return policy.Version{}, err
	}
	if value.State == "" {
		value.State = policy.StateDraft
	}
	if value.Version == 0 {
		value.Version = 1
	}
	if value.State != policy.StateDraft || value.Validate() != nil || strings.TrimSpace(requestID) == "" {
		return policy.Version{}, policy.ErrInvalid
	}
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	familyID, err := s.ids.Next(ctx)
	if err != nil {
		return policy.Version{}, fmt.Errorf("policy family ID: %w", err)
	}
	versionID, err := s.ids.Next(ctx)
	if err != nil {
		return policy.Version{}, fmt.Errorf("policy version ID: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return policy.Version{}, fmt.Errorf("begin policy creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO policy_families
		(id,workspace_id,name,description,owner,created_by,request_id,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, familyID, principal.Workspace, value.Name,
		value.Description, value.Owner, principal.ActorID, requestID, value.CreatedAt)
	if err != nil {
		var existingID int64
		if loadErr := s.pool.QueryRow(ctx, `SELECT id FROM policy_families WHERE workspace_id=$1 AND request_id=$2`,
			principal.Workspace, requestID).Scan(&existingID); loadErr == nil {
			return s.PolicyVersion(ctx, principal, existingID, 1)
		}
		return policy.Version{}, fmt.Errorf("insert policy family: %w", err)
	}
	value.ID, value.FamilyID, value.CreatedBy, value.Revision = versionID, familyID, principal.ActorID, 1
	if err := insertPolicyVersion(ctx, tx, value, requestID); err != nil {
		return policy.Version{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return policy.Version{}, fmt.Errorf("commit policy creation: %w", err)
	}
	return value, nil
}

func insertPolicyVersion(ctx context.Context, tx pgx.Tx, value policy.Version, requestID string) error {
	_, err := tx.Exec(ctx, `INSERT INTO policy_versions
		(id,policy_id,version,state,revision,created_by,request_id,created_at,activated_at,retired_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, value.ID, value.FamilyID, value.Version,
		value.State, value.Revision, value.CreatedBy, requestID, value.CreatedAt, value.ActivatedAt, value.RetiredAt)
	if err != nil {
		return fmt.Errorf("insert policy version: %w", err)
	}
	for ordinal, rule := range value.Rules {
		_, err = tx.Exec(ctx, `INSERT INTO policy_rules
			(policy_version_id,ordinal,metric_name,metric_version,operator,threshold,weight,required,
			 required_evidence,on_failure,label) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			value.ID, ordinal, rule.MetricName, rule.MetricVersion, rule.Operator, rule.Threshold,
			rule.Weight, rule.Required, rule.RequiredEvidence, rule.OnFailure, rule.Label)
		if err != nil {
			return fmt.Errorf("insert policy rule: %w", err)
		}
	}
	for outcome, ring := range value.RadarMap {
		if _, err = tx.Exec(ctx, `INSERT INTO policy_radar_mappings(policy_version_id,outcome,ring)
			VALUES($1,$2,$3)`, value.ID, outcome, ring); err != nil {
			return fmt.Errorf("insert policy radar mapping: %w", err)
		}
	}
	return nil
}

func (s *Store) CreatePolicyVersion(ctx context.Context, principal access.Principal, policyID int64,
	value policy.Version, requestID string, expectedLatest int) (policy.Version, error) {
	if err := policy.CanGovern(principal); err != nil {
		return policy.Version{}, err
	}
	if strings.TrimSpace(requestID) == "" || value.State != "" && value.State != policy.StateDraft {
		return policy.Version{}, policy.ErrInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return policy.Version{}, fmt.Errorf("begin policy version: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var name, description, owner string
	if err := tx.QueryRow(ctx, `SELECT name,description,owner FROM policy_families
		WHERE id=$1 AND workspace_id=$2 FOR UPDATE`, policyID, principal.Workspace).Scan(&name, &description, &owner); err != nil {
		return policy.Version{}, fmt.Errorf("lock policy family: %w", err)
	}
	var latest int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(version),0) FROM policy_versions WHERE policy_id=$1`, policyID).Scan(&latest); err != nil {
		return policy.Version{}, fmt.Errorf("load latest policy version: %w", err)
	}
	if existing, loadErr := loadPolicyVersion(ctx, tx, principal.Workspace, policyID, 0, requestID); loadErr == nil {
		return existing, nil
	}
	if latest != expectedLatest {
		return policy.Version{}, access.ErrVersionConflict
	}
	value.FamilyID, value.Version, value.Name, value.Description, value.Owner = policyID, latest+1, name, description, owner
	value.State, value.Revision, value.CreatedBy = policy.StateDraft, 1, principal.ActorID
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	if err := value.Validate(); err != nil {
		return policy.Version{}, err
	}
	value.ID, err = s.ids.Next(ctx)
	if err != nil {
		return policy.Version{}, fmt.Errorf("policy version ID: %w", err)
	}
	if err := insertPolicyVersion(ctx, tx, value, requestID); err != nil {
		return policy.Version{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return policy.Version{}, fmt.Errorf("commit policy version: %w", err)
	}
	return value, nil
}

func (s *Store) PolicyVersion(ctx context.Context, principal access.Principal, policyID int64,
	version int) (policy.Version, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return policy.Version{}, err
	}
	return loadPolicyVersion(ctx, s.pool, principal.Workspace, policyID, version, "")
}

type policyQuery interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadPolicyVersion(ctx context.Context, query policyQuery, workspaceID, policyID int64,
	version int, requestID string) (policy.Version, error) {
	var value policy.Version
	where, args := "v.policy_id=$1 AND f.workspace_id=$2 AND v.version=$3", []any{policyID, workspaceID, version}
	if requestID != "" {
		where, args = "v.policy_id=$1 AND f.workspace_id=$2 AND v.request_id=$3", []any{policyID, workspaceID, requestID}
	}
	err := query.QueryRow(ctx, `SELECT v.id,v.policy_id,v.version,f.name,f.description,f.owner,v.state,
		v.created_by,v.created_at,v.activated_at,v.retired_at,v.revision FROM policy_versions v
		JOIN policy_families f ON f.id=v.policy_id WHERE `+where, args...).Scan(&value.ID, &value.FamilyID,
		&value.Version, &value.Name, &value.Description, &value.Owner, &value.State, &value.CreatedBy,
		&value.CreatedAt, &value.ActivatedAt, &value.RetiredAt, &value.Revision)
	if err != nil {
		return policy.Version{}, fmt.Errorf("load policy version: %w", err)
	}
	rows, err := query.Query(ctx, `SELECT metric_name,metric_version,operator,threshold,weight,required,
		required_evidence,on_failure,label FROM policy_rules WHERE policy_version_id=$1 ORDER BY ordinal`, value.ID)
	if err != nil {
		return policy.Version{}, fmt.Errorf("load policy rules: %w", err)
	}
	value.Rules = make([]policy.Rule, 0)
	for rows.Next() {
		var rule policy.Rule
		if err := rows.Scan(&rule.MetricName, &rule.MetricVersion, &rule.Operator, &rule.Threshold,
			&rule.Weight, &rule.Required, &rule.RequiredEvidence, &rule.OnFailure, &rule.Label); err != nil {
			rows.Close()
			return policy.Version{}, fmt.Errorf("scan policy rule: %w", err)
		}
		value.Rules = append(value.Rules, rule)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return policy.Version{}, fmt.Errorf("iterate policy rules: %w", err)
	}
	rows.Close()
	rows, err = query.Query(ctx, `SELECT outcome,ring FROM policy_radar_mappings WHERE policy_version_id=$1`, value.ID)
	if err != nil {
		return policy.Version{}, fmt.Errorf("load radar mapping: %w", err)
	}
	defer rows.Close()
	value.RadarMap = make(map[policy.Outcome]string, 4)
	for rows.Next() {
		var outcome policy.Outcome
		var ring string
		if err := rows.Scan(&outcome, &ring); err != nil {
			return policy.Version{}, fmt.Errorf("scan radar mapping: %w", err)
		}
		value.RadarMap[outcome] = ring
	}
	return value, rows.Err()
}

func (s *Store) Policies(ctx context.Context, principal access.Principal, state string,
	limit, offset int) ([]policy.Version, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 || offset < 0 {
		return nil, policy.ErrInvalid
	}
	rows, err := s.pool.Query(ctx, `SELECT f.id,COALESCE(f.active_version,
		(SELECT max(version) FROM policy_versions WHERE policy_id=f.id)) FROM policy_families f
		WHERE f.workspace_id=$1 AND ($2='' OR EXISTS(SELECT 1 FROM policy_versions v
		WHERE v.policy_id=f.id AND v.state=$2)) ORDER BY lower(f.name),f.id LIMIT $3 OFFSET $4`,
		principal.Workspace, state, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list policy families: %w", err)
	}
	defer rows.Close()
	type identity struct {
		id      int64
		version int
	}
	identities := make([]identity, 0, limit)
	for rows.Next() {
		var item identity
		if err := rows.Scan(&item.id, &item.version); err != nil {
			return nil, fmt.Errorf("scan policy family: %w", err)
		}
		identities = append(identities, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate policy families: %w", err)
	}
	rows.Close()
	values := make([]policy.Version, 0, len(identities))
	for _, item := range identities {
		value, err := loadPolicyVersion(ctx, s.pool, principal.Workspace, item.id, item.version, "")
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (s *Store) ActivatePolicy(ctx context.Context, principal access.Principal, policyID int64,
	version int, expectedRevision int64, now time.Time) (policy.Version, error) {
	if err := policy.CanGovern(principal); err != nil {
		return policy.Version{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return policy.Version{}, fmt.Errorf("begin policy activation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	value, err := loadPolicyVersion(ctx, tx, principal.Workspace, policyID, version, "")
	if err != nil {
		return policy.Version{}, err
	}
	if value.Revision != expectedRevision {
		return policy.Version{}, access.ErrVersionConflict
	}
	active, err := policy.Activate(value, now)
	if err != nil {
		return policy.Version{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE policy_versions SET state='superseded',retired_at=$3,revision=revision+1
		WHERE policy_id=$1 AND state='active' AND version<>$2`, policyID, version, now); err != nil {
		return policy.Version{}, fmt.Errorf("supersede policy version: %w", err)
	}
	command, err := tx.Exec(ctx, `UPDATE policy_versions SET state='active',activated_at=$4,revision=$5
		WHERE policy_id=$1 AND version=$2 AND revision=$3 AND state='draft'`, policyID, version,
		expectedRevision, now, active.Revision)
	if err != nil {
		return policy.Version{}, fmt.Errorf("activate policy version: %w", err)
	}
	if command.RowsAffected() != 1 {
		return policy.Version{}, access.ErrVersionConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE policy_families SET active_version=$2 WHERE id=$1`, policyID, version); err != nil {
		return policy.Version{}, fmt.Errorf("select active policy version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return policy.Version{}, fmt.Errorf("commit policy activation: %w", err)
	}
	return active, nil
}

func (s *Store) ActivePolicy(ctx context.Context, principal access.Principal, selector string) (policy.Version, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return policy.Version{}, err
	}
	selector = strings.TrimSpace(selector)
	var policyID int64
	query := `SELECT id FROM policy_families WHERE workspace_id=$1 AND active_version IS NOT NULL`
	args := []any{principal.Workspace}
	if selector != "" && selector != "default" {
		query += ` AND (id::text=$2 OR lower(name)=lower($2))`
		args = append(args, selector)
	}
	query += ` ORDER BY created_at,id LIMIT 1`
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&policyID); err != nil {
		return policy.Version{}, fmt.Errorf("load active policy: %w", err)
	}
	var version int
	if err := s.pool.QueryRow(ctx, `SELECT active_version FROM policy_families WHERE id=$1`, policyID).Scan(&version); err != nil {
		return policy.Version{}, fmt.Errorf("load active policy version: %w", err)
	}
	return loadPolicyVersion(ctx, s.pool, principal.Workspace, policyID, version, "")
}

func (s *Store) SaveEvaluation(ctx context.Context, principal access.Principal,
	value policy.Evaluation) (policy.Evaluation, error) {
	if err := s.authorizeProject(ctx, principal, value.ProjectID); err != nil {
		return policy.Evaluation{}, err
	}
	if value.PolicyID <= 0 || value.PolicyVersion <= 0 || !value.Outcome.Valid() ||
		value.Window.Validate() != nil || strings.TrimSpace(value.InputDigest) == "" {
		return policy.Evaluation{}, policy.ErrInvalid
	}
	if value.ID == 0 {
		id, err := s.ids.Next(ctx)
		if err != nil {
			return policy.Evaluation{}, fmt.Errorf("recommendation evaluation ID: %w", err)
		}
		value.ID = id
	}
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return policy.Evaluation{}, fmt.Errorf("begin recommendation evaluation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var policyVersionID int64
	if err := tx.QueryRow(ctx, `SELECT v.id FROM policy_versions v JOIN policy_families f ON f.id=v.policy_id
		WHERE v.policy_id=$1 AND v.version=$2 AND f.workspace_id=$3`, value.PolicyID,
		value.PolicyVersion, principal.Workspace).Scan(&policyVersionID); err != nil {
		return policy.Evaluation{}, fmt.Errorf("resolve recommendation policy version: %w", err)
	}
	inserted := false
	err = tx.QueryRow(ctx, `INSERT INTO recommendation_evaluations
		(id,project_id,policy_id,policy_version_id,window_from,window_to,cutoff,outcome,created_at,input_digest,explanation)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT(project_id,policy_version_id,window_from,window_to,cutoff,input_digest)
		DO NOTHING RETURNING true`, value.ID, value.ProjectID, value.PolicyID, policyVersionID,
		value.Window.From, value.Window.To, value.Window.Cutoff, value.Outcome, value.CreatedAt,
		value.InputDigest, value.Explanation).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.Recommendation(ctx, principal, value.ProjectID, value.PolicyID, value.PolicyVersion, value.Window)
	}
	if err != nil {
		return policy.Evaluation{}, fmt.Errorf("insert recommendation evaluation: %w", err)
	}
	for ordinal, factor := range value.Factors {
		_, err = tx.Exec(ctx, `INSERT INTO recommendation_factors
			(evaluation_id,ordinal,rule_ordinal,metric_name,numeric_value,threshold,weight,matched,label)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, value.ID, ordinal, factor.RuleIndex,
			factor.MetricName, factor.Value, factor.Threshold, factor.Weight, factor.Matched, factor.Label)
		if err != nil {
			return policy.Evaluation{}, fmt.Errorf("insert recommendation factor: %w", err)
		}
	}
	decisive := make(map[string]bool, len(value.Decisive))
	for _, item := range value.Decisive {
		decisive[item] = true
	}
	for _, evidenceID := range slices.Compact(slices.Sorted(slices.Values(value.EvidenceIDs))) {
		if _, err = tx.Exec(ctx, `INSERT INTO recommendation_evidence(evaluation_id,evidence_id,decisive)
			VALUES($1,$2,$3)`, value.ID, evidenceID, len(decisive) > 0); err != nil {
			return policy.Evaluation{}, fmt.Errorf("insert recommendation evidence: %w", err)
		}
	}
	for kind, values := range map[string][]string{"missing": value.Missing, "stale": value.StaleInputs,
		"condition": value.Conditions, "decisive": value.Decisive} {
		for ordinal, item := range values {
			if _, err = tx.Exec(ctx, `INSERT INTO recommendation_gaps(evaluation_id,kind,ordinal,value)
				VALUES($1,$2,$3,$4)`, value.ID, kind, ordinal, item); err != nil {
				return policy.Evaluation{}, fmt.Errorf("insert recommendation %s: %w", kind, err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return policy.Evaluation{}, fmt.Errorf("commit recommendation evaluation: %w", err)
	}
	return value, nil
}

func (s *Store) Recommendation(ctx context.Context, principal access.Principal, projectID, policyID int64,
	version int, window metric.Window) (policy.Evaluation, error) {
	if err := s.authorizeProject(ctx, principal, projectID); err != nil {
		return policy.Evaluation{}, err
	}
	var value policy.Evaluation
	err := s.pool.QueryRow(ctx, `SELECT e.id,e.project_id,e.policy_id,v.version,f.owner,e.outcome,
		e.created_at,e.input_digest,e.explanation FROM recommendation_evaluations e
		JOIN policy_versions v ON v.id=e.policy_version_id JOIN policy_families f ON f.id=e.policy_id
		WHERE e.project_id=$1 AND e.policy_id=$2 AND v.version=$3 AND e.window_from=$4
		AND e.window_to=$5 AND e.cutoff=$6 AND f.workspace_id=$7 ORDER BY e.created_at DESC,e.id DESC LIMIT 1`,
		projectID, policyID, version, window.From, window.To, window.Cutoff, principal.Workspace).
		Scan(&value.ID, &value.ProjectID, &value.PolicyID, &value.PolicyVersion, &value.PolicyOwner,
			&value.Outcome, &value.CreatedAt, &value.InputDigest, &value.Explanation)
	if err != nil {
		return policy.Evaluation{}, fmt.Errorf("load recommendation: %w", err)
	}
	value.Window = window
	rows, err := s.pool.Query(ctx, `SELECT rule_ordinal,metric_name,numeric_value,threshold,weight,matched,label
		FROM recommendation_factors WHERE evaluation_id=$1 ORDER BY ordinal`, value.ID)
	if err != nil {
		return policy.Evaluation{}, fmt.Errorf("load recommendation factors: %w", err)
	}
	for rows.Next() {
		var item policy.Factor
		if err := rows.Scan(&item.RuleIndex, &item.MetricName, &item.Value, &item.Threshold,
			&item.Weight, &item.Matched, &item.Label); err != nil {
			rows.Close()
			return policy.Evaluation{}, fmt.Errorf("scan recommendation factor: %w", err)
		}
		value.Factors = append(value.Factors, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return policy.Evaluation{}, fmt.Errorf("iterate recommendation factors: %w", err)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT evidence_id FROM recommendation_evidence WHERE evaluation_id=$1 ORDER BY evidence_id`, value.ID)
	if err != nil {
		return policy.Evaluation{}, fmt.Errorf("load recommendation evidence: %w", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return policy.Evaluation{}, fmt.Errorf("scan recommendation evidence: %w", err)
		}
		value.EvidenceIDs = append(value.EvidenceIDs, id)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT kind,value FROM recommendation_gaps WHERE evaluation_id=$1
		ORDER BY kind,ordinal`, value.ID)
	if err != nil {
		return policy.Evaluation{}, fmt.Errorf("load recommendation gaps: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind, item string
		if err := rows.Scan(&kind, &item); err != nil {
			return policy.Evaluation{}, fmt.Errorf("scan recommendation gap: %w", err)
		}
		switch kind {
		case "missing":
			value.Missing = append(value.Missing, item)
		case "stale":
			value.StaleInputs = append(value.StaleInputs, item)
		case "condition":
			value.Conditions = append(value.Conditions, item)
		case "decisive":
			value.Decisive = append(value.Decisive, item)
		}
	}
	return value, rows.Err()
}

func (s *Store) SelectRadar(ctx context.Context, principal access.Principal, evaluationID int64,
	expectedRevision int64, now time.Time) error {
	if err := policy.CanSelect(principal); err != nil {
		return err
	}
	var projectID int64
	if err := s.pool.QueryRow(ctx, `SELECT e.project_id FROM recommendation_evaluations e
		JOIN projects p ON p.id=e.project_id WHERE e.id=$1 AND p.workspace_id=$2 AND p.state='active'`,
		evaluationID, principal.Workspace).Scan(&projectID); err != nil {
		return fmt.Errorf("resolve radar recommendation: %w", err)
	}
	command, err := s.pool.Exec(ctx, `INSERT INTO radar_selections(project_id,evaluation_id,selected_by,selected_at,revision)
		VALUES($1,$2,$3,$4,1) ON CONFLICT(project_id) DO UPDATE SET evaluation_id=EXCLUDED.evaluation_id,
		selected_by=EXCLUDED.selected_by,selected_at=EXCLUDED.selected_at,revision=radar_selections.revision+1
		WHERE radar_selections.revision=$5 OR radar_selections.evaluation_id=$2`, projectID, evaluationID,
		principal.ActorID, now, expectedRevision)
	if err != nil {
		return fmt.Errorf("select radar recommendation: %w", err)
	}
	if command.RowsAffected() != 1 {
		return access.ErrVersionConflict
	}
	return nil
}

func (s *Store) SaveRadarOverride(ctx context.Context, principal access.Principal, projectID int64,
	value radar.Override, requestID string) (radar.Override, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return radar.Override{}, err
	}
	if projectID <= 0 || value.ActorID != principal.ActorID || value.Revision != 1 || requestID == "" {
		return radar.Override{}, radar.ErrInvalid
	}
	if value.ID == 0 {
		id, err := s.ids.Next(ctx)
		if err != nil {
			return radar.Override{}, fmt.Errorf("radar override ID: %w", err)
		}
		value.ID = id
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return radar.Override{}, fmt.Errorf("begin radar override: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `UPDATE radar_overrides SET removed_at=$2,revision=revision+1
		WHERE project_id=$1 AND removed_at IS NULL`, projectID, value.CreatedAt); err != nil {
		return radar.Override{}, fmt.Errorf("retire current radar override: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO radar_overrides
		(id,project_id,ring,reason,owner,actor_id,review_on,revision,created_at,request_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, value.ID, projectID, value.Ring, value.Reason,
		value.Owner, value.ActorID, value.ReviewOn, value.Revision, value.CreatedAt, requestID)
	if err != nil {
		var existing radar.Override
		loadErr := s.pool.QueryRow(ctx, `SELECT id,ring,reason,owner,actor_id,created_at,review_on,
			removed_at,revision FROM radar_overrides WHERE project_id=$1 AND request_id=$2`, projectID, requestID).
			Scan(&existing.ID, &existing.Ring, &existing.Reason, &existing.Owner, &existing.ActorID,
				&existing.CreatedAt, &existing.ReviewOn, &existing.RemovedAt, &existing.Revision)
		if loadErr == nil {
			return existing, nil
		}
		return radar.Override{}, fmt.Errorf("insert radar override: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return radar.Override{}, fmt.Errorf("commit radar override: %w", err)
	}
	return value, nil
}

func (s *Store) RemoveRadarOverride(ctx context.Context, principal access.Principal, projectID,
	expectedRevision int64, now time.Time) error {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return err
	}
	command, err := s.pool.Exec(ctx, `UPDATE radar_overrides SET removed_at=$3,revision=revision+1
		WHERE project_id=$1 AND revision=$2 AND removed_at IS NULL`, projectID, expectedRevision, now)
	if err != nil {
		return fmt.Errorf("remove radar override: %w", err)
	}
	if command.RowsAffected() != 1 {
		return access.ErrVersionConflict
	}
	return nil
}

func (s *Store) Radar(ctx context.Context, principal access.Principal, now time.Time,
	limit, offset int) ([]radar.Placement, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 || offset < 0 {
		return nil, radar.ErrInvalid
	}
	rows, err := s.pool.Query(ctx, `SELECT rs.project_id,p.state,e.id,e.policy_id,v.version,f.owner,e.outcome,
		e.window_from,e.window_to,e.cutoff,e.created_at,e.input_digest,pv.id,
		o.id,o.ring,o.reason,o.owner,o.actor_id,o.created_at,o.review_on,o.removed_at,o.revision
		FROM radar_selections rs JOIN projects p ON p.id=rs.project_id
		JOIN recommendation_evaluations e ON e.id=rs.evaluation_id
		JOIN policy_versions v ON v.id=e.policy_version_id JOIN policy_families f ON f.id=e.policy_id
		JOIN policy_versions pv ON pv.id=e.policy_version_id
		LEFT JOIN radar_overrides o ON o.project_id=rs.project_id AND o.removed_at IS NULL
		WHERE p.workspace_id=$1 ORDER BY lower(p.name),p.id LIMIT $2 OFFSET $3`, principal.Workspace, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list radar placements: %w", err)
	}
	defer rows.Close()
	values := make([]radar.Placement, 0, limit)
	for rows.Next() {
		var projectID, evaluationID, policyID, policyVersionID int64
		var version int
		var projectState, owner string
		var outcome policy.Outcome
		var valueWindow metric.Window
		var createdAt time.Time
		var digest string
		var overrideID *int64
		var overrideRing, reason, overrideOwner *string
		var actor, revision *int64
		var overrideCreated, reviewOn, removedAt *time.Time
		if err := rows.Scan(&projectID, &projectState, &evaluationID, &policyID, &version, &owner, &outcome,
			&valueWindow.From, &valueWindow.To, &valueWindow.Cutoff, &createdAt, &digest, &policyVersionID,
			&overrideID, &overrideRing, &reason, &overrideOwner, &actor, &overrideCreated, &reviewOn,
			&removedAt, &revision); err != nil {
			return nil, fmt.Errorf("scan radar placement: %w", err)
		}
		mappingRows, loadErr := s.pool.Query(ctx, `SELECT outcome,ring FROM policy_radar_mappings
			WHERE policy_version_id=$1`, policyVersionID)
		if loadErr != nil {
			return nil, fmt.Errorf("load placement mapping: %w", loadErr)
		}
		mapping := make(map[policy.Outcome]string, 4)
		for mappingRows.Next() {
			var key policy.Outcome
			var ring string
			if err := mappingRows.Scan(&key, &ring); err != nil {
				mappingRows.Close()
				return nil, fmt.Errorf("scan placement mapping: %w", err)
			}
			mapping[key] = ring
		}
		mappingRows.Close()
		evaluation := policy.Evaluation{ID: evaluationID, ProjectID: projectID, PolicyID: policyID,
			PolicyVersion: version, PolicyOwner: owner, Window: valueWindow, Outcome: outcome,
			CreatedAt: createdAt, InputDigest: digest}
		var activeOverride *radar.Override
		if overrideID != nil {
			activeOverride = &radar.Override{ID: *overrideID, Ring: radar.Ring(*overrideRing), Reason: *reason,
				Owner: *overrideOwner, ActorID: *actor, CreatedAt: *overrideCreated, ReviewOn: *reviewOn,
				RemovedAt: removedAt, Revision: *revision}
		}
		placement, resolveErr := radar.Resolve(projectID, projectState, evaluation, mapping, activeOverride, now)
		if resolveErr != nil {
			return nil, resolveErr
		}
		values = append(values, placement)
	}
	return values, rows.Err()
}

func (s *Store) CreateAlertRule(ctx context.Context, principal access.Principal,
	value alert.Rule) (alert.Rule, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return alert.Rule{}, err
	}
	value.Version, value.CreatedBy = 1, principal.ActorID
	if value.UpdatedAt.IsZero() {
		value.UpdatedAt = time.Now().UTC()
	}
	if err := value.Validate(); err != nil {
		return alert.Rule{}, err
	}
	if value.ID == 0 {
		id, err := s.ids.Next(ctx)
		if err != nil {
			return alert.Rule{}, fmt.Errorf("alert rule ID: %w", err)
		}
		value.ID = id
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return alert.Rule{}, fmt.Errorf("begin alert rule creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rules, occurrences, err := lockAlertWorkspaceVolume(ctx, tx, principal.Workspace)
	if err != nil {
		return alert.Rule{}, err
	}
	if err := alert.DefaultLimits.ValidateVolume(rules+1, occurrences); err != nil {
		return alert.Rule{}, err
	}
	if err := insertAlertRule(ctx, tx, principal.Workspace, value); err != nil {
		return alert.Rule{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return alert.Rule{}, fmt.Errorf("commit alert rule creation: %w", err)
	}
	return value, nil
}

type alertRuleExec interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertAlertRule(ctx context.Context, exec alertRuleExec, workspaceID int64, value alert.Rule) error {
	var projectID *int64
	if value.ProjectID > 0 {
		projectID = &value.ProjectID
	}
	_, err := exec.Exec(ctx, `INSERT INTO alert_rules
		(id,version,workspace_id,name,signal,operator,threshold,scope,project_id,severity,
		 cooldown_seconds,deduplication_seconds,enabled,created_by,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, value.ID, value.Version,
		workspaceID, value.Name, value.Signal, value.Operator, value.Threshold, value.Scope, projectID,
		value.Severity, int64(value.Cooldown/time.Second), int64(value.Deduplication/time.Second),
		value.Enabled, value.CreatedBy, value.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert alert rule: %w", err)
	}
	return nil
}

func (s *Store) UpdateAlertRule(ctx context.Context, principal access.Principal, value alert.Rule,
	expectedVersion int64) (alert.Rule, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return alert.Rule{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return alert.Rule{}, fmt.Errorf("begin alert rule update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var latest int64
	if err := tx.QueryRow(ctx, `SELECT version FROM alert_rules WHERE id=$1 AND workspace_id=$2
		ORDER BY version DESC LIMIT 1 FOR UPDATE`, value.ID, principal.Workspace).Scan(&latest); err != nil {
		return alert.Rule{}, fmt.Errorf("lock alert rule: %w", err)
	}
	if latest != expectedVersion {
		return alert.Rule{}, access.ErrVersionConflict
	}
	value.Version, value.CreatedBy = latest+1, principal.ActorID
	if err := value.Validate(); err != nil {
		return alert.Rule{}, err
	}
	if err := insertAlertRule(ctx, tx, principal.Workspace, value); err != nil {
		return alert.Rule{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return alert.Rule{}, fmt.Errorf("commit alert rule update: %w", err)
	}
	return value, nil
}

func (s *Store) AlertRule(ctx context.Context, principal access.Principal, id int64) (alert.Rule, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return alert.Rule{}, err
	}
	var value alert.Rule
	var cooldown, dedup int64
	err := s.pool.QueryRow(ctx, `SELECT id,version,name,signal,operator,threshold,scope,
		COALESCE(project_id,0),severity,cooldown_seconds,deduplication_seconds,enabled,created_by,updated_at
		FROM alert_rules WHERE id=$1 AND workspace_id=$2 ORDER BY version DESC LIMIT 1`, id,
		principal.Workspace).Scan(&value.ID, &value.Version, &value.Name, &value.Signal, &value.Operator,
		&value.Threshold, &value.Scope, &value.ProjectID, &value.Severity, &cooldown, &dedup,
		&value.Enabled, &value.CreatedBy, &value.UpdatedAt)
	if err != nil {
		return alert.Rule{}, fmt.Errorf("load alert rule: %w", err)
	}
	value.Cooldown, value.Deduplication = time.Duration(cooldown)*time.Second, time.Duration(dedup)*time.Second
	return value, nil
}

func (s *Store) EvaluateAlert(ctx context.Context, rule alert.Rule, fact alert.Fact) (*alert.Occurrence, error) {
	if rule.Validate() != nil {
		return nil, alert.ErrInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin alert evaluation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var workspaceID int64
	if err := tx.QueryRow(ctx, `SELECT w.id FROM projects p JOIN workspaces w ON w.id=p.workspace_id
		JOIN alert_rules r ON r.workspace_id=w.id AND r.id=$2 AND r.version=$3
		WHERE p.id=$1 FOR UPDATE OF w`, fact.ProjectID, rule.ID, rule.Version).Scan(&workspaceID); err != nil {
		return nil, fmt.Errorf("resolve alert workspace: %w", err)
	}
	existing, err := loadDedupOccurrence(ctx, tx, rule, fact)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	result, err := alert.Evaluate(rule, fact, existing)
	if err != nil || result == existing {
		return result, err
	}
	if result == nil {
		return nil, nil
	}
	if existing == nil {
		rules, occurrences, volumeErr := alertWorkspaceVolume(ctx, tx, workspaceID)
		if volumeErr != nil {
			return nil, volumeErr
		}
		if volumeErr = alert.DefaultLimits.ValidateVolume(rules, occurrences+1); volumeErr != nil {
			return nil, volumeErr
		}
		result.ID, err = s.ids.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("alert occurrence ID: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO alert_occurrences
			(id,rule_id,rule_version,project_id,signal_version,severity,state,window_from,window_to,
			 first_detected_at,last_detected_at,suppression_count,revision)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, result.ID, result.RuleID,
			result.RuleVersion, result.ProjectID, result.SignalVersion, result.Severity, result.State,
			result.WindowFrom, result.WindowTo, result.FirstDetectedAt, result.LastDetectedAt,
			result.SuppressionCount, result.Revision)
		if err != nil {
			return nil, fmt.Errorf("insert alert occurrence: %w", err)
		}
	} else {
		_, err = tx.Exec(ctx, `UPDATE alert_occurrences SET last_detected_at=$2,suppression_count=$3,
			revision=$4 WHERE id=$1`, result.ID, result.LastDetectedAt, result.SuppressionCount, result.Revision)
		if err != nil {
			return nil, fmt.Errorf("update deduplicated alert occurrence: %w", err)
		}
	}
	for _, evidenceID := range result.EvidenceIDs {
		if _, err = tx.Exec(ctx, `INSERT INTO alert_occurrence_evidence(occurrence_id,evidence_id)
			VALUES($1,$2) ON CONFLICT DO NOTHING`, result.ID, evidenceID); err != nil {
			return nil, fmt.Errorf("insert alert occurrence evidence: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit alert evaluation: %w", err)
	}
	return result, nil
}

func lockAlertWorkspaceVolume(ctx context.Context, tx pgx.Tx, workspaceID int64) (int, int, error) {
	if err := tx.QueryRow(ctx, `SELECT id FROM workspaces WHERE id=$1 FOR UPDATE`, workspaceID).
		Scan(&workspaceID); err != nil {
		return 0, 0, fmt.Errorf("lock alert workspace: %w", err)
	}
	return alertWorkspaceVolume(ctx, tx, workspaceID)
}

func alertWorkspaceVolume(ctx context.Context, query policyQuery, workspaceID int64) (int, int, error) {
	var rules, occurrences int
	if err := query.QueryRow(ctx, `SELECT count(DISTINCT id) FROM alert_rules WHERE workspace_id=$1`,
		workspaceID).Scan(&rules); err != nil {
		return 0, 0, fmt.Errorf("count alert rules: %w", err)
	}
	if err := query.QueryRow(ctx, `SELECT count(*) FROM alert_occurrences o
		JOIN projects p ON p.id=o.project_id WHERE p.workspace_id=$1`, workspaceID).
		Scan(&occurrences); err != nil {
		return 0, 0, fmt.Errorf("count alert occurrences: %w", err)
	}
	return rules, occurrences, nil
}

func loadDedupOccurrence(ctx context.Context, query policyQuery, rule alert.Rule,
	fact alert.Fact) (*alert.Occurrence, error) {
	var value alert.Occurrence
	err := query.QueryRow(ctx, `SELECT id,rule_id,rule_version,project_id,signal_version,severity,state,
		window_from,window_to,first_detected_at,last_detected_at,suppression_count,transition_reason,
		COALESCE(transitioned_by,0),revision FROM alert_occurrences WHERE rule_id=$1 AND rule_version=$2
		AND project_id=$3 AND first_detected_at >= $4 ORDER BY first_detected_at DESC,id DESC LIMIT 1 FOR UPDATE`,
		rule.ID, rule.Version, fact.ProjectID, fact.DetectedAt.Add(-rule.Deduplication)).Scan(&value.ID,
		&value.RuleID, &value.RuleVersion, &value.ProjectID, &value.SignalVersion, &value.Severity,
		&value.State, &value.WindowFrom, &value.WindowTo, &value.FirstDetectedAt, &value.LastDetectedAt,
		&value.SuppressionCount, &value.TransitionReason, &value.TransitionedBy, &value.Revision)
	if err != nil {
		return nil, err
	}
	rows, err := query.Query(ctx, `SELECT evidence_id FROM alert_occurrence_evidence WHERE occurrence_id=$1`, value.ID)
	if err != nil {
		return nil, fmt.Errorf("load alert evidence: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan alert evidence: %w", err)
		}
		value.EvidenceIDs = append(value.EvidenceIDs, id)
	}
	return &value, rows.Err()
}

type AlertItem struct {
	alert.Occurrence
	ReadAt *time.Time `json:"read_at,omitempty"`
}

func (s *Store) Alerts(ctx context.Context, principal access.Principal, state string,
	projectID int64, limit, offset int) ([]AlertItem, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 || offset < 0 {
		return nil, alert.ErrInvalid
	}
	rows, err := s.pool.Query(ctx, `SELECT o.id,o.rule_id,o.rule_version,o.project_id,o.signal_version,
		o.severity,o.state,o.window_from,o.window_to,o.first_detected_at,o.last_detected_at,
		o.suppression_count,o.transition_reason,COALESCE(o.transitioned_by,0),o.revision,m.read_at
		FROM alert_occurrences o JOIN projects p ON p.id=o.project_id
		LEFT JOIN alert_member_state m ON m.occurrence_id=o.id AND m.member_id=$2
		WHERE p.workspace_id=$1 AND ($3='' OR o.state=$3) AND ($4=0 OR o.project_id=$4)
		ORDER BY o.last_detected_at DESC,o.id DESC LIMIT $5 OFFSET $6`, principal.Workspace,
		principal.ActorID, state, projectID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()
	values := make([]AlertItem, 0, limit)
	for rows.Next() {
		var item AlertItem
		if err := rows.Scan(&item.ID, &item.RuleID, &item.RuleVersion, &item.ProjectID,
			&item.SignalVersion, &item.Severity, &item.State, &item.WindowFrom, &item.WindowTo,
			&item.FirstDetectedAt, &item.LastDetectedAt, &item.SuppressionCount,
			&item.TransitionReason, &item.TransitionedBy, &item.Revision, &item.ReadAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		values = append(values, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for index := range values {
		evidence, err := loadAlertEvidence(ctx, s.pool, values[index].ID)
		if err != nil {
			return nil, err
		}
		values[index].EvidenceIDs = evidence
	}
	return values, nil
}

func loadAlertEvidence(ctx context.Context, query policyQuery, occurrenceID int64) ([]int64, error) {
	rows, err := query.Query(ctx, `SELECT evidence_id FROM alert_occurrence_evidence
		WHERE occurrence_id=$1 ORDER BY evidence_id`, occurrenceID)
	if err != nil {
		return nil, fmt.Errorf("load alert evidence: %w", err)
	}
	defer rows.Close()
	values := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan alert evidence: %w", err)
		}
		values = append(values, id)
	}
	return values, rows.Err()
}

func (s *Store) MarkAlertRead(ctx context.Context, principal access.Principal, id int64,
	now time.Time) error {
	state, err := alert.MarkRead(principal, id, now)
	if err != nil {
		return err
	}
	command, err := s.pool.Exec(ctx, `INSERT INTO alert_member_state(occurrence_id,member_id,read_at)
		SELECT $1,$2,$3 FROM alert_occurrences o JOIN projects p ON p.id=o.project_id
		WHERE o.id=$1 AND p.workspace_id=$4 ON CONFLICT(occurrence_id,member_id)
		DO UPDATE SET read_at=EXCLUDED.read_at`, id, state.MemberID, now, principal.Workspace)
	if err != nil {
		return fmt.Errorf("mark alert read: %w", err)
	}
	if command.RowsAffected() != 1 {
		return access.ErrNotFound
	}
	return nil
}

func (s *Store) TransitionAlert(ctx context.Context, principal access.Principal, id int64,
	to alert.State, reason string, expectedRevision int64, now time.Time) (alert.Occurrence, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return alert.Occurrence{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return alert.Occurrence{}, fmt.Errorf("begin alert transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current alert.Occurrence
	err = tx.QueryRow(ctx, `SELECT o.id,o.rule_id,o.rule_version,o.project_id,o.signal_version,o.severity,
		o.state,o.window_from,o.window_to,o.first_detected_at,o.last_detected_at,o.suppression_count,
		o.transition_reason,COALESCE(o.transitioned_by,0),o.revision FROM alert_occurrences o
		JOIN projects p ON p.id=o.project_id WHERE o.id=$1 AND p.workspace_id=$2 FOR UPDATE`, id,
		principal.Workspace).Scan(&current.ID, &current.RuleID, &current.RuleVersion, &current.ProjectID,
		&current.SignalVersion, &current.Severity, &current.State, &current.WindowFrom, &current.WindowTo,
		&current.FirstDetectedAt, &current.LastDetectedAt, &current.SuppressionCount,
		&current.TransitionReason, &current.TransitionedBy, &current.Revision)
	if err != nil {
		return alert.Occurrence{}, fmt.Errorf("load alert transition: %w", err)
	}
	from := current.State
	updated, err := alert.Transition(principal, current, to, reason, expectedRevision)
	if err != nil {
		return alert.Occurrence{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE alert_occurrences SET state=$2,transition_reason=$3,
		transitioned_by=$4,revision=$5 WHERE id=$1`, id, updated.State, updated.TransitionReason,
		updated.TransitionedBy, updated.Revision)
	if err != nil {
		return alert.Occurrence{}, fmt.Errorf("update alert transition: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO alert_transitions
		(occurrence_id,revision,from_state,to_state,reason,actor_id,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, id, updated.Revision, from, updated.State,
		updated.TransitionReason, updated.TransitionedBy, now); err != nil {
		return alert.Occurrence{}, fmt.Errorf("insert alert transition history: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return alert.Occurrence{}, fmt.Errorf("commit alert transition: %w", err)
	}
	return updated, nil
}
