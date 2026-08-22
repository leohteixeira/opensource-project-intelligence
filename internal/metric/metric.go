package metric

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/contributor"
	"github.com/leohteixeira/opensource-project-intelligence/internal/issue"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/timewindow"
	"github.com/leohteixeira/opensource-project-intelligence/internal/pullrequest"
	"github.com/leohteixeira/opensource-project-intelligence/internal/release"
)

var ErrInvalid = errors.New("invalid metric request")

type Status string

const (
	StatusAvailable        Status = "available"
	StatusUnknown          Status = "unknown"
	StatusNotApplicable    Status = "not_applicable"
	StatusInsufficientData Status = "insufficient_data"
	StatusIncomparable     Status = "incomparable"
	StatusStale            Status = "stale"
	StatusUnavailable      Status = "unavailable"
)

type Definition struct {
	Name            string        `json:"name"`
	Version         string        `json:"version"`
	Unit            string        `json:"unit"`
	DefaultWindow   time.Duration `json:"-"`
	Formula         string        `json:"formula"`
	Eligibility     string        `json:"eligibility"`
	MissingDataRule string        `json:"missing_data_rule"`
}

var catalog = []Definition{
	{Name: "release_frequency", Version: "v1", Unit: "releases", DefaultWindow: 90 * 24 * time.Hour, Formula: "count(stable releases published in [from,to))", Eligibility: "published and not draft or prerelease", MissingDataRule: "insufficient_data when release coverage is incomplete"},
	{Name: "active_contributors", Version: "v1", Unit: "people", DefaultWindow: 30 * 24 * time.Hour, Formula: "count(distinct eligible contributor identities)", Eligibility: "human non-merge default-branch commits", MissingDataRule: "insufficient_data when commit evidence is absent"},
	{Name: "issues_opened_closed", Version: "v1", Unit: "issues", DefaultWindow: 30 * 24 * time.Hour, Formula: "count(opened), count(closed) in [from,to)", Eligibility: "canonical issues", MissingDataRule: "insufficient_data when issue coverage is incomplete"},
	{Name: "median_issue_first_response", Version: "v1", Unit: "hours", DefaultWindow: 30 * 24 * time.Hour, Formula: "median(first qualifying response - opened_at)", Eligibility: "issues opened in window; public non-bot member response excluding opener", MissingDataRule: "unanswered issues are censored and reduce coverage"},
	{Name: "median_pr_merge_time", Version: "v1", Unit: "hours", DefaultWindow: 30 * 24 * time.Hour, Formula: "median(merged_at - ready_for_review)", Eligibility: "pull requests merged in window", MissingDataRule: "created_at fallback is counted in coverage"},
	{Name: "backlog_change", Version: "v1", Unit: "issues", DefaultWindow: 30 * 24 * time.Hour, Formula: "open_at_end - open_at_start", Eligibility: "issue state events including reopen", MissingDataRule: "insufficient_data without boundary history"},
	{Name: "top_three_author_share", Version: "v1", Unit: "ratio", DefaultWindow: 90 * 24 * time.Hour, Formula: "commits by top three resolved-or-distinct authors / eligible commits", Eligibility: "human non-merge default-branch commits", MissingDataRule: "unresolved accounts remain distinct and reduce resolution coverage"},
}

func Catalog() []Definition { return slices.Clone(catalog) }

func DefinitionByName(name string) (Definition, bool) {
	for _, definition := range catalog {
		if definition.Name == name {
			return definition, true
		}
	}
	return Definition{}, false
}

type Window struct {
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Cutoff time.Time `json:"cutoff"`
}

func (w Window) Validate() error {
	if w.From.IsZero() || w.To.IsZero() || w.Cutoff.IsZero() || !w.From.Before(w.To) || w.To.After(w.Cutoff) {
		return fmt.Errorf("%w: window must be ordered and end at or before cutoff", ErrInvalid)
	}
	if w.From.Location() != time.UTC || w.To.Location() != time.UTC || w.Cutoff.Location() != time.UTC {
		return fmt.Errorf("%w: window and cutoff must use UTC", ErrInvalid)
	}
	return nil
}

func PresetWindow(value string, cutoff time.Time) (Window, error) {
	days := map[string]int{"30d": 30, "90d": 90, "180d": 180, "365d": 365}[value]
	if days == 0 {
		return Window{}, fmt.Errorf("%w: window must be 30d, 90d, 180d, or 365d", ErrInvalid)
	}
	cutoff = cutoff.UTC()
	return Window{From: cutoff.AddDate(0, 0, -days), To: cutoff, Cutoff: cutoff}, nil
}

// ResolveWindow accepts a preset or an explicit half-open UTC date interval formatted from/to.
func ResolveWindow(value string, cutoff time.Time) (Window, error) {
	if preset, err := PresetWindow(value, cutoff); err == nil {
		return preset, nil
	}
	bounds := strings.Split(value, "/")
	if len(bounds) != 2 {
		return Window{}, fmt.Errorf("%w: custom window must be YYYY-MM-DD/YYYY-MM-DD", ErrInvalid)
	}
	from, fromErr := time.Parse("2006-01-02", bounds[0])
	to, toErr := time.Parse("2006-01-02", bounds[1])
	window := Window{From: from.UTC(), To: to.UTC(), Cutoff: cutoff.UTC()}
	if fromErr != nil || toErr != nil || window.Validate() != nil {
		return Window{}, fmt.Errorf("%w: invalid custom interval", ErrInvalid)
	}
	return window, nil
}

type Coverage struct {
	Eligible int     `json:"eligible"`
	Observed int     `json:"observed"`
	Ratio    float64 `json:"ratio"`
	Note     string  `json:"note,omitempty"`
}

type Factor struct {
	Name       string   `json:"name"`
	Value      *float64 `json:"value,omitempty"`
	Unit       string   `json:"unit"`
	EvidenceID int64    `json:"evidence_id,string,omitempty"`
}

type Snapshot struct {
	ID           int64      `json:"id,string"`
	ProjectID    int64      `json:"project_id,string"`
	Definition   Definition `json:"definition"`
	Window       Window     `json:"window"`
	Status       Status     `json:"status"`
	Value        *float64   `json:"value,omitempty"`
	Coverage     Coverage   `json:"coverage"`
	Factors      []Factor   `json:"factors"`
	Repositories []int64    `json:"repository_ids"`
	StaleReason  string     `json:"stale_reason,omitempty"`
}

func NewSnapshot(projectID int64, definition Definition, window Window, status Status, value *float64, coverage Coverage, factors []Factor, repositories []int64) (Snapshot, error) {
	if projectID <= 0 || strings.TrimSpace(definition.Name) == "" || window.Validate() != nil {
		return Snapshot{}, ErrInvalid
	}
	if status == StatusAvailable && (value == nil || math.IsNaN(*value) || math.IsInf(*value, 0)) {
		return Snapshot{}, fmt.Errorf("%w: available results require a finite value", ErrInvalid)
	}
	if status != StatusAvailable && value != nil {
		return Snapshot{}, fmt.Errorf("%w: non-available results cannot carry a numeric value", ErrInvalid)
	}
	if coverage.Eligible > 0 {
		coverage.Ratio = float64(coverage.Observed) / float64(coverage.Eligible)
	}
	return Snapshot{ProjectID: projectID, Definition: definition, Window: window, Status: status, Value: value, Coverage: coverage, Factors: slices.Clone(factors), Repositories: slices.Clone(repositories)}, nil
}

var HealthDimensionNames = []string{"Activity", "Community", "Maintenance", "Concentration", "Stability", "Security", "Adoption"}

type Dimension struct {
	Name    string   `json:"name"`
	Status  Status   `json:"status"`
	Score   *float64 `json:"score,omitempty"`
	Weight  float64  `json:"weight"`
	Version string   `json:"version"`
	Factors []Factor `json:"factors"`
}

type Health struct {
	ProjectID     int64       `json:"project_id,string"`
	Window        Window      `json:"window"`
	Version       string      `json:"version"`
	Dimensions    []Dimension `json:"dimensions"`
	Overall       *float64    `json:"overall,omitempty"`
	OverallStatus Status      `json:"overall_status"`
}

func CalculateHealth(projectID int64, window Window, version string, dimensions []Dimension) (Health, error) {
	if projectID <= 0 || window.Validate() != nil || len(dimensions) != len(HealthDimensionNames) {
		return Health{}, ErrInvalid
	}
	byName := make(map[string]Dimension, len(dimensions))
	for _, dimension := range dimensions {
		byName[dimension.Name] = dimension
	}
	ordered := make([]Dimension, 0, len(HealthDimensionNames))
	total := 0.0
	calculable := true
	for _, name := range HealthDimensionNames {
		dimension, ok := byName[name]
		if !ok {
			return Health{}, fmt.Errorf("%w: missing health dimension %s", ErrInvalid, name)
		}
		dimension.Weight = 1.0 / 7.0
		if dimension.Status != StatusAvailable || dimension.Score == nil {
			calculable = false
		} else {
			total += *dimension.Score * dimension.Weight
		}
		ordered = append(ordered, dimension)
	}
	health := Health{ProjectID: projectID, Window: window, Version: version, Dimensions: ordered, OverallStatus: StatusInsufficientData}
	if calculable {
		health.Overall = &total
		health.OverallStatus = StatusAvailable
	}
	return health, nil
}

func Median(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	middle := len(ordered) / 2
	if len(ordered)%2 == 1 {
		return ordered[middle], true
	}
	return (ordered[middle-1] + ordered[middle]) / 2, true
}

func AsTimeWindow(window Window) timewindow.Window {
	return timewindow.Window{From: window.From, To: window.To}
}

// Facts are canonical provider-neutral inputs to one immutable materialization.
type Facts struct {
	Releases       []release.Release
	Commits        []contributor.Commit
	Issues         []issue.Issue
	IssueClosedAt  map[int64]time.Time
	PullRequests   []pullrequest.PullRequest
	IssueEvents    []issue.StateEvent
	RepositoryIDs  []int64
	ReleaseCovered bool
	CommitCovered  bool
	IssueCovered   bool
	PRCovered      bool
}

// ComputeCatalog deterministically evaluates every definition in the closed catalog.
func ComputeCatalog(projectID int64, window Window, facts Facts) ([]Snapshot, error) {
	if projectID <= 0 || window.Validate() != nil {
		return nil, ErrInvalid
	}
	results := make([]Snapshot, 0, len(catalog))
	for _, definition := range catalog {
		status := StatusAvailable
		var value *float64
		coverage := Coverage{}
		factors := make([]Factor, 0, 3)
		switch definition.Name {
		case "release_frequency":
			stable := release.StableInWindow(facts.Releases, AsTimeWindow(window))
			coverage = Coverage{Eligible: len(facts.Releases), Observed: len(facts.Releases)}
			if !facts.ReleaseCovered {
				status = StatusInsufficientData
				coverage.Note = "release coverage is incomplete"
			} else {
				value = number(float64(len(stable)))
				factors = append(factors, Factor{Name: "stable_releases", Value: value, Unit: "releases"})
			}
		case "active_contributors", "top_three_author_share":
			summary := contributor.Aggregate(facts.Commits, AsTimeWindow(window))
			coverage = Coverage{Eligible: len(facts.Commits), Observed: len(facts.Commits), Ratio: summary.ResolutionCoverage,
				Note: "ratio additionally reports verified identity resolution"}
			if !facts.CommitCovered || summary.Status != "available" {
				status = StatusInsufficientData
			} else if definition.Name == "active_contributors" {
				value = number(float64(summary.Active))
				factors = append(factors, Factor{Name: "eligible_identities", Value: value, Unit: "people"})
			} else {
				value = summary.TopThreeShare
				factors = append(factors, Factor{Name: "top_three_share", Value: value, Unit: "ratio"})
			}
		case "issues_opened_closed":
			opened, closed := 0, 0
			for _, value := range facts.Issues {
				if AsTimeWindow(window).Contains(value.CreatedAt) {
					opened++
				}
			}
			for _, closedAt := range facts.IssueClosedAt {
				if AsTimeWindow(window).Contains(closedAt) {
					closed++
				}
			}
			coverage = Coverage{Eligible: len(facts.Issues), Observed: len(facts.Issues)}
			if !facts.IssueCovered {
				status = StatusInsufficientData
			} else {
				value = number(float64(opened + closed))
				factors = append(factors,
					Factor{Name: "opened", Value: number(float64(opened)), Unit: "issues"},
					Factor{Name: "closed", Value: number(float64(closed)), Unit: "issues"})
			}
		case "median_issue_first_response":
			durations := make([]float64, 0, len(facts.Issues))
			eligible := 0
			for _, value := range facts.Issues {
				if !AsTimeWindow(window).Contains(value.CreatedAt) {
					continue
				}
				eligible++
				if duration, ok := issue.FirstResponse(value, window.Cutoff); ok {
					durations = append(durations, duration.Hours())
				}
			}
			coverage = Coverage{Eligible: eligible, Observed: len(durations), Note: "unanswered issues are censored"}
			median, ok := Median(durations)
			if !facts.IssueCovered || !ok {
				status = StatusInsufficientData
			} else {
				value = number(median)
			}
		case "median_pr_merge_time":
			durations := make([]float64, 0, len(facts.PullRequests))
			fallbacks := 0
			for _, pull := range facts.PullRequests {
				if !AsTimeWindow(window).Contains(pull.MergedAt) {
					continue
				}
				duration, fallback := pullrequest.MergeDuration(pull)
				if fallback {
					fallbacks++
				}
				durations = append(durations, duration.Hours())
			}
			coverage = Coverage{Eligible: len(durations), Observed: len(durations), Note: fmt.Sprintf("%d created_at fallbacks", fallbacks)}
			median, ok := Median(durations)
			if !facts.PRCovered || !ok {
				status = StatusInsufficientData
			} else {
				value = number(median)
			}
		case "backlog_change":
			coverage = Coverage{Eligible: len(facts.IssueEvents), Observed: len(facts.IssueEvents)}
			if !facts.IssueCovered || len(facts.IssueEvents) == 0 {
				status = StatusInsufficientData
			} else {
				value = number(float64(issue.BacklogChange(facts.IssueEvents, AsTimeWindow(window))))
			}
		}
		snapshot, err := NewSnapshot(projectID, definition, window, status, value, coverage, factors, facts.RepositoryIDs)
		if err != nil {
			return nil, fmt.Errorf("compute %s: %w", definition.Name, err)
		}
		results = append(results, snapshot)
	}
	return results, nil
}

func number(value float64) *float64 { return &value }

// MaterializationAllowed prevents new intelligence on archived or deleting Projects.
func MaterializationAllowed(projectState string) bool {
	return projectState == "active" || projectState == "paused"
}
