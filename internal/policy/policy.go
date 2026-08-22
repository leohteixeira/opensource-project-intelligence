// Package policy owns typed, immutable adoption policies and deterministic recommendations.
package policy

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
)

var ErrInvalid = errors.New("invalid policy")

type Outcome string

const (
	Recommended      Outcome = "recommended"
	Conditional      Outcome = "conditional"
	NotRecommended   Outcome = "not_recommended"
	InsufficientData Outcome = "insufficient_data"
)

func (o Outcome) Valid() bool {
	return o == Recommended || o == Conditional || o == NotRecommended || o == InsufficientData
}

type State string

const (
	StateDraft      State = "draft"
	StateActive     State = "active"
	StateSuperseded State = "superseded"
	StateRetired    State = "retired"
)

type Operator string

const (
	GreaterThan        Operator = "gt"
	GreaterThanOrEqual Operator = "gte"
	LessThan           Operator = "lt"
	LessThanOrEqual    Operator = "lte"
	Equal              Operator = "eq"
)

type Rule struct {
	MetricName       string   `json:"metric_name"`
	MetricVersion    string   `json:"metric_version"`
	Operator         Operator `json:"operator"`
	Threshold        float64  `json:"threshold"`
	Weight           float64  `json:"weight"`
	Required         bool     `json:"required"`
	RequiredEvidence string   `json:"required_evidence"`
	OnFailure        Outcome  `json:"on_failure"`
	Label            string   `json:"label"`
}

type Version struct {
	ID          int64              `json:"id,string,omitempty"`
	FamilyID    int64              `json:"policy_id,string,omitempty"`
	Version     int                `json:"version"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Owner       string             `json:"owner"`
	State       State              `json:"state"`
	Rules       []Rule             `json:"rules"`
	RadarMap    map[Outcome]string `json:"radar_mapping"`
	CreatedBy   int64              `json:"created_by,string,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	ActivatedAt *time.Time         `json:"activated_at,omitempty"`
	RetiredAt   *time.Time         `json:"retired_at,omitempty"`
	Revision    int64              `json:"revision"`
}

var radarRings = map[string]bool{"adopt": true, "trial": true, "assess": true, "hold": true, "unplaced": true}

func (v Version) Validate() error {
	if v.Version <= 0 || strings.TrimSpace(v.Name) == "" || strings.TrimSpace(v.Owner) == "" ||
		len(v.Rules) == 0 || len(v.Rules) > 100 {
		return fmt.Errorf("%w: identity, owner, and one to 100 rules are required", ErrInvalid)
	}
	total := 0.0
	bounds := make(map[string][2]*float64)
	for index, rule := range v.Rules {
		definition, known := metric.DefinitionByName(rule.MetricName)
		if !known || definition.Version != rule.MetricVersion || !validOperator(rule.Operator) ||
			math.IsNaN(rule.Threshold) || math.IsInf(rule.Threshold, 0) || rule.Weight <= 0 || rule.Weight > 1 ||
			(rule.OnFailure != Conditional && rule.OnFailure != NotRecommended) || strings.TrimSpace(rule.Label) == "" ||
			rule.Required && strings.TrimSpace(rule.RequiredEvidence) == "" {
			return fmt.Errorf("%w: rule %d is not a typed catalog rule", ErrInvalid, index)
		}
		total += rule.Weight
		pair := bounds[rule.MetricName]
		threshold := rule.Threshold
		switch rule.Operator {
		case GreaterThan, GreaterThanOrEqual:
			if pair[0] == nil || threshold > *pair[0] {
				pair[0] = &threshold
			}
		case LessThan, LessThanOrEqual:
			if pair[1] == nil || threshold < *pair[1] {
				pair[1] = &threshold
			}
		}
		bounds[rule.MetricName] = pair
	}
	if math.Abs(total-1) > 1e-9 {
		return fmt.Errorf("%w: rule weights must sum to one", ErrInvalid)
	}
	for name, pair := range bounds {
		if pair[0] != nil && pair[1] != nil && *pair[0] > *pair[1] {
			return fmt.Errorf("%w: contradictory bounds for %s", ErrInvalid, name)
		}
	}
	for _, outcome := range []Outcome{Recommended, Conditional, NotRecommended, InsufficientData} {
		if ring, ok := v.RadarMap[outcome]; !ok || !radarRings[ring] {
			return fmt.Errorf("%w: explicit radar mapping for %s is required", ErrInvalid, outcome)
		}
	}
	return nil
}

func validOperator(value Operator) bool {
	return value == GreaterThan || value == GreaterThanOrEqual || value == LessThan ||
		value == LessThanOrEqual || value == Equal
}

type Fact struct {
	MetricName    string          `json:"metric_name"`
	MetricVersion string          `json:"metric_version"`
	Status        metric.Status   `json:"status"`
	Value         *float64        `json:"value,omitempty"`
	Coverage      metric.Coverage `json:"coverage"`
	EvidenceIDs   []int64         `json:"evidence_ids"`
	SnapshotID    int64           `json:"snapshot_id,string"`
}

type Factor struct {
	RuleIndex  int     `json:"rule_index"`
	MetricName string  `json:"metric_name"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	Weight     float64 `json:"weight"`
	Matched    bool    `json:"matched"`
	Label      string  `json:"label"`
}

type Evaluation struct {
	ID            int64         `json:"id,string,omitempty"`
	ProjectID     int64         `json:"project_id,string"`
	PolicyID      int64         `json:"policy_id,string"`
	PolicyVersion int           `json:"policy_version"`
	PolicyOwner   string        `json:"policy_owner"`
	Window        metric.Window `json:"window"`
	Outcome       Outcome       `json:"outcome"`
	Factors       []Factor      `json:"factors"`
	EvidenceIDs   []int64       `json:"evidence_ids"`
	Missing       []string      `json:"missing_data"`
	Conditions    []string      `json:"conditions"`
	Decisive      []string      `json:"decisive_factors"`
	StaleInputs   []string      `json:"stale_inputs"`
	Explanation   string        `json:"explanation,omitempty"`
	InputDigest   string        `json:"input_digest,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

func Evaluate(projectID int64, selected Version, window metric.Window, facts []Fact) (Evaluation, error) {
	if projectID <= 0 || selected.State != StateActive || selected.Validate() != nil || window.Validate() != nil {
		return Evaluation{}, fmt.Errorf("%w: only a compatible active policy can be evaluated", ErrInvalid)
	}
	byName := make(map[string]Fact, len(facts))
	for _, fact := range facts {
		byName[fact.MetricName] = fact
	}
	result := Evaluation{ProjectID: projectID, PolicyID: selected.FamilyID,
		PolicyVersion: selected.Version, PolicyOwner: selected.Owner, Window: window,
		Outcome: Recommended, Factors: []Factor{}, EvidenceIDs: []int64{}, Missing: []string{},
		Conditions: []string{}, Decisive: []string{}, StaleInputs: []string{}}
	for index, rule := range selected.Rules {
		fact, ok := byName[rule.MetricName]
		if !ok || fact.MetricVersion != rule.MetricVersion || fact.Status != metric.StatusAvailable || fact.Value == nil {
			if fact.Status == metric.StatusStale {
				result.StaleInputs = append(result.StaleInputs, rule.MetricName)
			}
			if rule.Required {
				result.Missing = append(result.Missing, rule.MetricName)
			}
			continue
		}
		matched := compare(*fact.Value, rule.Operator, rule.Threshold)
		result.Factors = append(result.Factors, Factor{RuleIndex: index, MetricName: rule.MetricName,
			Value: *fact.Value, Threshold: rule.Threshold, Weight: rule.Weight, Matched: matched, Label: rule.Label})
		result.EvidenceIDs = append(result.EvidenceIDs, fact.EvidenceIDs...)
		if !matched {
			result.Decisive = append(result.Decisive, rule.Label)
			if rule.OnFailure == NotRecommended {
				result.Outcome = NotRecommended
			} else if result.Outcome == Recommended {
				result.Outcome = Conditional
				result.Conditions = append(result.Conditions, rule.Label)
			}
		}
	}
	if len(result.Missing) > 0 {
		result.Outcome = InsufficientData
	}
	result.EvidenceIDs = compactIDs(result.EvidenceIDs)
	return result, nil
}

func compare(value float64, operator Operator, threshold float64) bool {
	switch operator {
	case GreaterThan:
		return value > threshold
	case GreaterThanOrEqual:
		return value >= threshold
	case LessThan:
		return value < threshold
	case LessThanOrEqual:
		return value <= threshold
	case Equal:
		return value == threshold
	default:
		return false
	}
}

func compactIDs(values []int64) []int64 {
	slices.Sort(values)
	return slices.Compact(values)
}

func CanSelect(principal access.Principal) error {
	return access.Authorize(principal, access.ActionProjectWrite)
}

func CanGovern(principal access.Principal) error {
	return access.Authorize(principal, access.ActionPolicyGovern)
}

func Activate(value Version, now time.Time) (Version, error) {
	if value.State != StateDraft || value.Validate() != nil || now.IsZero() {
		return Version{}, fmt.Errorf("%w: only a valid draft can be activated", ErrInvalid)
	}
	value.State, value.ActivatedAt, value.Revision = StateActive, &now, value.Revision+1
	return value, nil
}

func Retire(value Version, now time.Time) (Version, error) {
	if value.State != StateActive || now.IsZero() {
		return Version{}, fmt.Errorf("%w: only an active version can be retired", ErrInvalid)
	}
	value.State, value.RetiredAt, value.Revision = StateRetired, &now, value.Revision+1
	return value, nil
}

// PageVersions applies a stable bounded offset page without loading an unbounded history.
func PageVersions(values []Version, limit, offset int) ([]Version, bool, error) {
	if limit <= 0 || limit > 200 || offset < 0 {
		return nil, false, ErrInvalid
	}
	if offset >= len(values) {
		return []Version{}, false, nil
	}
	end := min(len(values), offset+limit)
	return slices.Clone(values[offset:end]), end < len(values), nil
}
