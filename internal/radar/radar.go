// Package radar derives technology placement from immutable recommendations and attributed overrides.
package radar

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/policy"
)

type RingCount struct {
	Ring  Ring `json:"ring"`
	Count int  `json:"count"`
}

// FilterAndCount keeps total off-screen ring counts visible while returning a bounded filtered page.
func FilterAndCount(values []Placement, ring Ring, limit, offset int) ([]Placement, []RingCount, error) {
	if ring != "" && !ring.Valid() || limit <= 0 || limit > 200 || offset < 0 {
		return nil, nil, ErrInvalid
	}
	counts := map[Ring]int{RingAdopt: 0, RingTrial: 0, RingAssess: 0, RingHold: 0, RingUnplaced: 0}
	filtered := make([]Placement, 0, len(values))
	for _, value := range values {
		counts[value.Effective]++
		if ring == "" || value.Effective == ring {
			filtered = append(filtered, value)
		}
	}
	if offset >= len(filtered) {
		filtered = nil
	} else {
		filtered = filtered[offset:min(len(filtered), offset+limit)]
	}
	ordered := []RingCount{{RingAdopt, counts[RingAdopt]}, {RingTrial, counts[RingTrial]},
		{RingAssess, counts[RingAssess]}, {RingHold, counts[RingHold]}, {RingUnplaced, counts[RingUnplaced]}}
	return filtered, ordered, nil
}

var ErrInvalid = errors.New("invalid radar placement")

type Ring string

const (
	RingAdopt    Ring = "adopt"
	RingTrial    Ring = "trial"
	RingAssess   Ring = "assess"
	RingHold     Ring = "hold"
	RingUnplaced Ring = "unplaced"
)

func (r Ring) Valid() bool {
	return r == RingAdopt || r == RingTrial || r == RingAssess || r == RingHold || r == RingUnplaced
}

type Override struct {
	ID        int64      `json:"id,string,omitempty"`
	Ring      Ring       `json:"ring"`
	Reason    string     `json:"reason"`
	Owner     string     `json:"owner"`
	ActorID   int64      `json:"actor_id,string"`
	CreatedAt time.Time  `json:"created_at"`
	ReviewOn  time.Time  `json:"review_on"`
	RemovedAt *time.Time `json:"removed_at,omitempty"`
	Revision  int64      `json:"revision"`
}

type Placement struct {
	ProjectID       int64             `json:"project_id,string"`
	ProjectState    string            `json:"project_state"`
	Evaluation      policy.Evaluation `json:"recommendation"`
	Suggested       Ring              `json:"suggested_ring"`
	Effective       Ring              `json:"effective_ring"`
	Override        *Override         `json:"override,omitempty"`
	OverrideExpired bool              `json:"override_expired"`
	ReviewOverdue   bool              `json:"review_overdue"`
	SuggestionStale bool              `json:"suggestion_stale"`
	Historical      bool              `json:"historical"`
}

func Derive(evaluation policy.Evaluation, mapping map[policy.Outcome]string) (Ring, error) {
	if !evaluation.Outcome.Valid() || evaluation.PolicyVersion <= 0 {
		return "", fmt.Errorf("%w: a versioned policy result is required", ErrInvalid)
	}
	ring, ok := mapping[evaluation.Outcome]
	if !ok || !Ring(ring).Valid() {
		return "", fmt.Errorf("%w: outcome has no explicit ring mapping", ErrInvalid)
	}
	return Ring(ring), nil
}

func NewOverride(principal access.Principal, ring Ring, reason, owner string, reviewOn, now time.Time) (Override, error) {
	if err := access.Authorize(principal, access.ActionProjectWrite); err != nil {
		return Override{}, err
	}
	if !ring.Valid() || ring == RingUnplaced || strings.TrimSpace(reason) == "" || strings.TrimSpace(owner) == "" ||
		reviewOn.Location() != time.UTC || !reviewOn.After(now) {
		return Override{}, fmt.Errorf("%w: ring, reason, owner, and a future UTC review date are required", ErrInvalid)
	}
	return Override{Ring: ring, Reason: strings.TrimSpace(reason), Owner: strings.TrimSpace(owner),
		ActorID: principal.ActorID, CreatedAt: now, ReviewOn: reviewOn, Revision: 1}, nil
}

func Resolve(projectID int64, projectState string, evaluation policy.Evaluation,
	mapping map[policy.Outcome]string, override *Override, now time.Time) (Placement, error) {
	suggested, err := Derive(evaluation, mapping)
	if err != nil || projectID <= 0 {
		return Placement{}, errors.Join(ErrInvalid, err)
	}
	result := Placement{ProjectID: projectID, ProjectState: projectState, Evaluation: evaluation,
		Suggested: suggested, Effective: suggested, Historical: projectState == "archived"}
	if result.Historical {
		return result, nil
	}
	if override != nil {
		copy := *override
		result.Override = &copy
		result.OverrideExpired = copy.RemovedAt != nil || !copy.ReviewOn.After(now)
		result.ReviewOverdue = !copy.ReviewOn.After(now)
		if !result.OverrideExpired {
			result.Effective = copy.Ring
		}
	}
	return result, nil
}

func RemoveOverride(value Override, expectedRevision int64, now time.Time) (Override, error) {
	if value.Revision != expectedRevision || value.RemovedAt != nil {
		return Override{}, access.ErrVersionConflict
	}
	value.RemovedAt, value.Revision = &now, value.Revision+1
	return value, nil
}
