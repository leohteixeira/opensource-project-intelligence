// Package adoption models source-contextual registry adoption and public advisory evidence.
package adoption

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

var ErrInvalid = errors.New("adoption: invalid evidence")

type Status string

const (
	StatusAvailable    Status = "available"
	StatusUnknown      Status = "unknown"
	StatusIncomparable Status = "incomparable"
	StatusStale        Status = "stale"
)

// Snapshot is one immutable registry observation. Unit and Population define its comparison
// boundary; callers must not combine observations across either boundary.
type Snapshot struct {
	ID         int64      `json:"id,string"`
	ProjectID  int64      `json:"project_id,string"`
	SourceID   int64      `json:"source_id,string"`
	Registry   string     `json:"registry"`
	Package    string     `json:"package"`
	Unit       string     `json:"unit"`
	Population string     `json:"population"`
	Value      *float64   `json:"value,omitempty"`
	Status     Status     `json:"status"`
	WindowFrom time.Time  `json:"window_from"`
	WindowTo   time.Time  `json:"window_to"`
	ObservedAt time.Time  `json:"observed_at"`
	EvidenceID int64      `json:"evidence_id,string"`
	StaleAt    *time.Time `json:"stale_at,omitempty"`
}

func NewSnapshot(value Snapshot) (Snapshot, error) {
	value.Registry = strings.ToLower(strings.TrimSpace(value.Registry))
	value.Package = strings.TrimSpace(value.Package)
	value.Unit = strings.TrimSpace(value.Unit)
	value.Population = strings.TrimSpace(value.Population)
	if value.ID <= 0 || value.ProjectID <= 0 || value.SourceID <= 0 || value.EvidenceID <= 0 ||
		value.Registry == "" || value.Package == "" || value.Unit == "" ||
		!value.WindowFrom.Before(value.WindowTo) || value.WindowTo.After(value.ObservedAt) {
		return Snapshot{}, ErrInvalid
	}
	if value.Value == nil {
		value.Status = StatusUnknown
	} else if *value.Value < 0 || value.Population == "" {
		return Snapshot{}, ErrInvalid
	} else if value.Status == "" {
		value.Status = StatusAvailable
	}
	if value.StaleAt != nil && !value.StaleAt.After(value.ObservedAt) {
		return Snapshot{}, ErrInvalid
	}
	return value, nil
}

// Comparable reports whether two observations have identical source semantics.
func Comparable(left, right Snapshot) bool {
	return left.Registry == right.Registry && left.Package == right.Package &&
		left.Unit == right.Unit && left.Population == right.Population
}

// ComparisonStatus prevents registry units or population contexts from being normalized together.
func ComparisonStatus(values []Snapshot) Status {
	if len(values) == 0 {
		return StatusUnknown
	}
	first := values[0]
	for _, value := range values[1:] {
		if !Comparable(first, value) {
			return StatusIncomparable
		}
	}
	for _, value := range values {
		if value.Value == nil {
			return StatusUnknown
		}
		if value.StaleAt != nil {
			return StatusStale
		}
	}
	return StatusAvailable
}

// MergeHistory combines paginated observations, rejecting duplicate samples and preserving a
// stable newest-first order.
func MergeHistory(pages ...[]Snapshot) ([]Snapshot, error) {
	seen := make(map[string]struct{})
	values := make([]Snapshot, 0)
	for _, page := range pages {
		for _, value := range page {
			key := fmt.Sprintf("%d/%s/%s/%s/%s/%s", value.SourceID, value.Registry, value.Package,
				value.Unit, value.Population, value.ObservedAt.UTC().Format(time.RFC3339Nano))
			if _, exists := seen[key]; exists {
				return nil, fmt.Errorf("%w: duplicate registry sample", ErrInvalid)
			}
			seen[key] = struct{}{}
			values = append(values, value)
		}
	}
	slices.SortFunc(values, func(left, right Snapshot) int {
		if compared := right.ObservedAt.Compare(left.ObservedAt); compared != 0 {
			return compared
		}
		switch {
		case left.ID < right.ID:
			return -1
		case left.ID > right.ID:
			return 1
		default:
			return 0
		}
	})
	return values, nil
}

type AdvisoryState string

const (
	AdvisoryPublished AdvisoryState = "published"
	AdvisoryWithdrawn AdvisoryState = "withdrawn"
)

// Advisory is immutable public security evidence, not a vulnerability-scanner finding.
type Advisory struct {
	ID          int64         `json:"id,string"`
	ProjectID   int64         `json:"project_id,string"`
	SourceID    int64         `json:"source_id,string"`
	ExternalID  string        `json:"external_id"`
	Severity    string        `json:"severity"`
	Summary     string        `json:"summary"`
	State       AdvisoryState `json:"state"`
	PublishedAt time.Time     `json:"published_at"`
	WithdrawnAt *time.Time    `json:"withdrawn_at,omitempty"`
	EvidenceID  int64         `json:"evidence_id,string"`
}

type Security struct {
	Status        Status     `json:"status"`
	Qualification string     `json:"qualification"`
	CoverageNote  string     `json:"coverage_note"`
	Advisories    []Advisory `json:"advisories"`
}

// SummarizeSecurity distinguishes complete public evidence containing no advisories from absent
// or incomplete evidence. Both are unknown for safety; only the explanation differs.
func SummarizeSecurity(sourceObserved, coverageComplete bool, advisories []Advisory) Security {
	result := Security{Status: StatusUnknown, Qualification: "public_advisory_evidence_only",
		Advisories: slices.Clone(advisories)}
	switch {
	case !sourceObserved:
		result.CoverageNote = "no public advisory source was observed"
	case !coverageComplete:
		result.CoverageNote = "public advisory coverage is incomplete"
	case len(advisories) == 0:
		result.CoverageNote = "no public advisories were found; this is not a safety claim"
	default:
		result.Status = StatusAvailable
		result.CoverageNote = "public advisories were observed; this is not a vulnerability scan"
	}
	slices.SortFunc(result.Advisories, func(left, right Advisory) int {
		if compared := right.PublishedAt.Compare(left.PublishedAt); compared != 0 {
			return compared
		}
		return strings.Compare(left.ExternalID, right.ExternalID)
	})
	return result
}
