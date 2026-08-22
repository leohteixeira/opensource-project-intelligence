package release

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis"
)

var ErrInvalid = errors.New("release: invalid intelligence")

type State string

const (
	StatePublished State = "published"
	StateWithdrawn State = "withdrawn"
)

type Intelligence struct {
	ID            int64      `json:"id,string"`
	ProjectID     int64      `json:"project_id,string"`
	RepositoryID  int64      `json:"repository_id,string"`
	SourceID      int64      `json:"source_id,string"`
	ExternalID    string     `json:"external_id"`
	Tag           string     `json:"tag"`
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	Language      string     `json:"language"`
	URL           string     `json:"url"`
	PublishedAt   time.Time  `json:"published_at"`
	Prerelease    bool       `json:"prerelease"`
	State         State      `json:"state"`
	WithdrawnAt   *time.Time `json:"withdrawn_at,omitempty"`
	EvidenceID    int64      `json:"evidence_id,string"`
	ChangelogID   *int64     `json:"changelog_id,omitempty,string"`
	AnalysisRunID *int64     `json:"analysis_run_id,omitempty,string"`
}

func NewIntelligence(value Intelligence) (Intelligence, error) {
	value.ExternalID = strings.TrimSpace(value.ExternalID)
	value.Tag = strings.TrimSpace(value.Tag)
	if value.ID <= 0 || value.ProjectID <= 0 || value.RepositoryID <= 0 || value.SourceID <= 0 ||
		value.EvidenceID <= 0 || value.ExternalID == "" || value.Tag == "" ||
		!strings.HasPrefix(value.URL, "https://") || value.PublishedAt.IsZero() {
		return Intelligence{}, ErrInvalid
	}
	if value.State == "" {
		value.State = StatePublished
	}
	if value.State == StateWithdrawn && (value.WithdrawnAt == nil || value.WithdrawnAt.Before(value.PublishedAt)) {
		return Intelligence{}, ErrInvalid
	}
	return value, nil
}

// MergeHistory rejects duplicate provider identities and returns immutable newest-first history.
func MergeHistory(pages ...[]Intelligence) ([]Intelligence, error) {
	seen := make(map[string]struct{})
	values := make([]Intelligence, 0)
	for _, page := range pages {
		for _, value := range page {
			key := strings.Join([]string{value.ExternalID, value.Tag, value.URL}, "\x00")
			if _, exists := seen[key]; exists {
				return nil, ErrInvalid
			}
			seen[key] = struct{}{}
			values = append(values, value)
		}
	}
	slices.SortFunc(values, func(left, right Intelligence) int {
		if compared := right.PublishedAt.Compare(left.PublishedAt); compared != 0 {
			return compared
		}
		if left.ID < right.ID {
			return -1
		}
		if left.ID > right.ID {
			return 1
		}
		return 0
	})
	return values, nil
}

type View struct {
	Release        Intelligence  `json:"release"`
	Analysis       *analysis.Run `json:"analysis,omitempty"`
	AnalysisStatus string        `json:"analysis_status"`
	CoverageNote   string        `json:"coverage_note"`
}

// Present keeps canonical release metadata available when the model or changelog is absent.
func Present(value Intelligence, run *analysis.Run, providerAvailable bool) View {
	view := View{Release: value, AnalysisStatus: "pending", CoverageNote: "release metadata available"}
	if value.ChangelogID == nil {
		view.CoverageNote = "release metadata available; no changelog evidence"
	}
	if run != nil {
		copy := *run
		view.Analysis = &copy
		view.AnalysisStatus = string(run.State)
	} else if !providerAvailable {
		view.AnalysisStatus = "unavailable"
	}
	return view
}
