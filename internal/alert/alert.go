// Package alert owns typed rules, deduplicated shared occurrences, and member read state.
package alert

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
)

var ErrInvalid = errors.New("invalid alert")

type Limits struct {
	Rules       int
	Occurrences int
}

var DefaultLimits = Limits{Rules: 500, Occurrences: 10_000}

func (l Limits) ValidateVolume(ruleCount, occurrenceCount int) error {
	if l.Rules <= 0 || l.Occurrences <= 0 || ruleCount < 0 || occurrenceCount < 0 ||
		ruleCount > l.Rules || occurrenceCount > l.Occurrences {
		return fmt.Errorf("%w: workspace alert quota exceeded", ErrInvalid)
	}
	return nil
}

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type State string

const (
	StateOpen         State = "open"
	StateAcknowledged State = "acknowledged"
	StateResolved     State = "resolved"
	StateDismissed    State = "dismissed"
)

type Operator string

const (
	GreaterThan Operator = "gt"
	LessThan    Operator = "lt"
	Equal       Operator = "eq"
)

var knownSignals = map[string]bool{
	"metric.release_frequency": true, "metric.active_contributors": true,
	"metric.median_issue_first_response": true, "metric.median_pr_merge_time": true,
	"trend.increase": true, "trend.decrease": true, "forecast.warning": true,
	"recommendation.not_recommended": true, "recommendation.conditional": true,
}

type Rule struct {
	ID            int64         `json:"id,string,omitempty"`
	Version       int64         `json:"version"`
	Name          string        `json:"name"`
	Signal        string        `json:"signal"`
	Operator      Operator      `json:"operator"`
	Threshold     float64       `json:"threshold"`
	Scope         string        `json:"scope"`
	ProjectID     int64         `json:"project_id,string,omitempty"`
	Severity      Severity      `json:"severity"`
	Cooldown      time.Duration `json:"cooldown_seconds"`
	Deduplication time.Duration `json:"deduplication_seconds"`
	Enabled       bool          `json:"enabled"`
	CreatedBy     int64         `json:"created_by,string,omitempty"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

func (r Rule) Validate() error {
	if r.Version <= 0 || strings.TrimSpace(r.Name) == "" || !knownSignals[r.Signal] ||
		(r.Operator != GreaterThan && r.Operator != LessThan && r.Operator != Equal) ||
		math.IsNaN(r.Threshold) || math.IsInf(r.Threshold, 0) ||
		(r.Scope != "workspace" && r.Scope != "project") || r.Scope == "project" && r.ProjectID <= 0 ||
		(r.Severity != SeverityInfo && r.Severity != SeverityWarning && r.Severity != SeverityCritical) ||
		r.Cooldown < 0 || r.Deduplication <= 0 || r.Deduplication > 365*24*time.Hour {
		return fmt.Errorf("%w: rule fields are outside the typed catalog", ErrInvalid)
	}
	return nil
}

type Fact struct {
	ProjectID  int64     `json:"project_id,string"`
	Signal     string    `json:"signal"`
	Version    string    `json:"version"`
	Value      *float64  `json:"value,omitempty"`
	EvidenceID int64     `json:"evidence_id,string,omitempty"`
	WindowFrom time.Time `json:"window_from"`
	WindowTo   time.Time `json:"window_to"`
	DetectedAt time.Time `json:"detected_at"`
	Complete   bool      `json:"complete"`
	Archived   bool      `json:"archived"`
}

type Occurrence struct {
	ID               int64     `json:"id,string,omitempty"`
	RuleID           int64     `json:"rule_id,string"`
	RuleVersion      int64     `json:"rule_version"`
	ProjectID        int64     `json:"project_id,string"`
	SignalVersion    string    `json:"signal_version"`
	Severity         Severity  `json:"severity"`
	State            State     `json:"state"`
	WindowFrom       time.Time `json:"window_from"`
	WindowTo         time.Time `json:"window_to"`
	EvidenceIDs      []int64   `json:"evidence_ids"`
	FirstDetectedAt  time.Time `json:"first_detected_at"`
	LastDetectedAt   time.Time `json:"last_detected_at"`
	SuppressionCount int       `json:"suppression_count"`
	TransitionReason string    `json:"transition_reason,omitempty"`
	TransitionedBy   int64     `json:"transitioned_by,string,omitempty"`
	Revision         int64     `json:"revision"`
}

type MemberState struct {
	OccurrenceID int64      `json:"alert_id,string"`
	MemberID     int64      `json:"member_id,string"`
	ReadAt       *time.Time `json:"read_at,omitempty"`
}

func Evaluate(rule Rule, fact Fact, existing *Occurrence) (*Occurrence, error) {
	if rule.Validate() != nil || fact.ProjectID <= 0 || fact.Signal != rule.Signal ||
		fact.DetectedAt.IsZero() || !fact.WindowFrom.Before(fact.WindowTo) {
		return nil, ErrInvalid
	}
	if !rule.Enabled || fact.Archived || !fact.Complete || fact.Value == nil || fact.EvidenceID <= 0 ||
		!matches(*fact.Value, rule.Operator, rule.Threshold) {
		return existing, nil
	}
	if existing != nil {
		copy := *existing
		if copy.RuleID == rule.ID && copy.RuleVersion == rule.Version && copy.ProjectID == fact.ProjectID &&
			fact.DetectedAt.Sub(copy.FirstDetectedAt) <= rule.Deduplication {
			copy.LastDetectedAt = fact.DetectedAt
			copy.SuppressionCount++
			copy.Revision++
			if fact.EvidenceID > 0 && !contains(copy.EvidenceIDs, fact.EvidenceID) {
				copy.EvidenceIDs = append(copy.EvidenceIDs, fact.EvidenceID)
			}
			return &copy, nil
		}
	}
	return &Occurrence{RuleID: rule.ID, RuleVersion: rule.Version, ProjectID: fact.ProjectID,
		SignalVersion: fact.Version, Severity: rule.Severity, State: StateOpen,
		WindowFrom: fact.WindowFrom, WindowTo: fact.WindowTo, EvidenceIDs: []int64{fact.EvidenceID},
		FirstDetectedAt: fact.DetectedAt, LastDetectedAt: fact.DetectedAt, Revision: 1}, nil
}

func matches(value float64, operator Operator, threshold float64) bool {
	switch operator {
	case GreaterThan:
		return value > threshold
	case LessThan:
		return value < threshold
	case Equal:
		return value == threshold
	default:
		return false
	}
}

func Transition(principal access.Principal, value Occurrence, to State, reason string,
	expectedRevision int64) (Occurrence, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return Occurrence{}, err
	}
	if value.Revision != expectedRevision {
		return Occurrence{}, access.ErrVersionConflict
	}
	if strings.TrimSpace(reason) == "" || !allowedTransition(value.State, to) {
		return Occurrence{}, ErrInvalid
	}
	value.State, value.TransitionReason, value.TransitionedBy = to, strings.TrimSpace(reason), principal.ActorID
	value.Revision++
	return value, nil
}

func allowedTransition(from, to State) bool {
	if from == StateOpen {
		return to == StateAcknowledged || to == StateResolved || to == StateDismissed
	}
	if from == StateAcknowledged {
		return to == StateOpen || to == StateResolved || to == StateDismissed
	}
	return (from == StateResolved || from == StateDismissed) && to == StateOpen
}

func MarkRead(principal access.Principal, occurrenceID int64, at time.Time) (MemberState, error) {
	if err := access.Authorize(principal, access.ActionIntelligenceRead); err != nil {
		return MemberState{}, err
	}
	if occurrenceID <= 0 || at.IsZero() {
		return MemberState{}, ErrInvalid
	}
	return MemberState{OccurrenceID: occurrenceID, MemberID: principal.ActorID, ReadAt: &at}, nil
}

func contains(values []int64, expected int64) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
