package analysis

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/knowledge"
)

var (
	ErrInvalidRun          = errors.New("analysis: invalid run")
	ErrSchema              = errors.New("analysis: output schema rejected")
	ErrEvidence            = errors.New("analysis: output evidence rejected")
	ErrSelection           = errors.New("analysis: run cannot be selected")
	ErrProviderUnavailable = errors.New("analysis: model provider unavailable")
)

type State string

const (
	StateQueued    State = "queued"
	StateRunning   State = "running"
	StateSucceeded State = "succeeded"
	StateFailed    State = "failed"
	StateCancelled State = "cancelled"
)

type UsageRecord struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Cost         float64 `json:"cost"`
	Currency     string  `json:"currency"`
}

type Claim struct {
	Text      string               `json:"text"`
	Citations []knowledge.Citation `json:"citations"`
}

type Output struct {
	Summary string  `json:"summary"`
	Claims  []Claim `json:"claims"`
}

type Run struct {
	ID               int64                `json:"id,string"`
	SeriesID         int64                `json:"series_id,string"`
	ProjectID        int64                `json:"project_id,string"`
	ParentRunID      *int64               `json:"parent_run_id,omitempty,string"`
	Kind             string               `json:"kind"`
	State            State                `json:"state"`
	PromptVersion    string               `json:"prompt_version"`
	SchemaVersion    string               `json:"schema_version"`
	RetrievalVersion string               `json:"retrieval_version"`
	Provider         string               `json:"provider"`
	Model            string               `json:"model"`
	Language         string               `json:"language"`
	RequestedBy      int64                `json:"requested_by,string"`
	Reason           string               `json:"reason,omitempty"`
	Cutoff           time.Time            `json:"cutoff"`
	CreatedAt        time.Time            `json:"created_at"`
	StartedAt        *time.Time           `json:"started_at,omitempty"`
	FinishedAt       *time.Time           `json:"finished_at,omitempty"`
	Output           json.RawMessage      `json:"output,omitempty"`
	Evidence         []knowledge.Citation `json:"evidence"`
	Usage            UsageRecord          `json:"usage"`
	FailureCode      string               `json:"failure_code,omitempty"`
}

func NewRun(value Run) (Run, error) {
	if value.ID <= 0 || value.SeriesID <= 0 || value.ProjectID <= 0 ||
		strings.TrimSpace(value.Kind) == "" || strings.TrimSpace(value.PromptVersion) == "" ||
		strings.TrimSpace(value.SchemaVersion) == "" || strings.TrimSpace(value.RetrievalVersion) == "" ||
		strings.TrimSpace(value.Provider) == "" || strings.TrimSpace(value.Model) == "" ||
		strings.TrimSpace(value.Language) == "" || value.RequestedBy <= 0 ||
		value.Cutoff.IsZero() || value.CreatedAt.IsZero() {
		return Run{}, ErrInvalidRun
	}
	value.Provider = strings.TrimSpace(value.Provider)
	value.Model = strings.TrimSpace(value.Model)
	value.Language = strings.TrimSpace(value.Language)
	value.Reason = strings.TrimSpace(value.Reason)
	value.State = StateQueued
	value.Output = nil
	value.Evidence = []knowledge.Citation{}
	return value, nil
}

func (run Run) Start(now time.Time) (Run, error) {
	if run.State != StateQueued || now.Before(run.CreatedAt) {
		return Run{}, ErrInvalidRun
	}
	value := cloneRun(run)
	value.State, value.StartedAt = StateRunning, timePointer(now.UTC())
	return value, nil
}

// Succeed validates the structured schema and every immutable evidence reference before publishing
// a new value; the receiver remains unchanged.
func (run Run) Succeed(raw json.RawMessage, accessible map[int64]knowledge.Citation,
	usage UsageRecord, now time.Time) (Run, error) {
	if run.State != StateRunning || run.StartedAt == nil || now.Before(*run.StartedAt) || !json.Valid(raw) {
		return Run{}, ErrSchema
	}
	evidence, err := validateOutput(raw, accessible)
	if err != nil {
		return Run{}, err
	}
	if err := usage.Validate(); err != nil {
		return Run{}, err
	}
	value := cloneRun(run)
	value.State, value.FinishedAt = StateSucceeded, timePointer(now.UTC())
	value.Output, value.Evidence, value.Usage = slices.Clone(raw), evidence, usage
	return value, nil
}

func validateOutput(raw json.RawMessage,
	accessible map[int64]knowledge.Citation) ([]knowledge.Citation, error) {
	if !json.Valid(raw) {
		return nil, ErrSchema
	}
	var output Output
	if err := json.Unmarshal(raw, &output); err != nil || strings.TrimSpace(output.Summary) == "" || len(output.Claims) == 0 {
		return nil, ErrSchema
	}
	evidence := make([]knowledge.Citation, 0)
	for _, claim := range output.Claims {
		if strings.TrimSpace(claim.Text) == "" || len(claim.Citations) == 0 {
			return nil, ErrEvidence
		}
		for _, citation := range claim.Citations {
			stored, exists := accessible[citation.ChunkID]
			if !exists || stored != citation {
				return nil, ErrEvidence
			}
			if !slices.Contains(evidence, citation) {
				evidence = append(evidence, citation)
			}
		}
	}
	return evidence, nil
}

// ValidateForPersistence revalidates the complete immutable aggregate at the storage boundary.
// Run is intentionally a serializable value type, so callers need not have constructed it through
// the transition methods before handing it to an adapter.
func (run Run) ValidateForPersistence() error {
	if run.ID <= 0 || run.SeriesID <= 0 || run.ProjectID <= 0 ||
		strings.TrimSpace(run.Kind) == "" || strings.TrimSpace(run.PromptVersion) == "" ||
		strings.TrimSpace(run.SchemaVersion) == "" || strings.TrimSpace(run.RetrievalVersion) == "" ||
		strings.TrimSpace(run.Provider) == "" || strings.TrimSpace(run.Model) == "" ||
		strings.TrimSpace(run.Language) == "" || run.RequestedBy <= 0 || run.Cutoff.IsZero() ||
		run.CreatedAt.IsZero() || (run.ParentRunID != nil && (*run.ParentRunID <= 0 || *run.ParentRunID == run.ID)) {
		return ErrInvalidRun
	}
	if err := run.Usage.Validate(); err != nil {
		return err
	}
	if run.StartedAt != nil && run.StartedAt.Before(run.CreatedAt) ||
		run.FinishedAt != nil && (run.StartedAt == nil || run.FinishedAt.Before(*run.StartedAt)) {
		return ErrInvalidRun
	}

	hasOutput := len(run.Output) > 0
	hasEvidence := len(run.Evidence) > 0
	switch run.State {
	case StateQueued:
		if run.StartedAt != nil || run.FinishedAt != nil || hasOutput || hasEvidence || run.FailureCode != "" {
			return ErrInvalidRun
		}
	case StateRunning:
		if run.StartedAt == nil || run.FinishedAt != nil || hasOutput || hasEvidence || run.FailureCode != "" {
			return ErrInvalidRun
		}
	case StateSucceeded:
		if run.StartedAt == nil || run.FinishedAt == nil || !hasOutput || !hasEvidence || run.FailureCode != "" {
			return ErrEvidence
		}
		accessible := make(map[int64]knowledge.Citation, len(run.Evidence))
		for _, citation := range run.Evidence {
			if citation.SnapshotID <= 0 || citation.ChunkID <= 0 || citation.StartOffset < 0 ||
				citation.EndOffset <= citation.StartOffset {
				return ErrEvidence
			}
			if prior, exists := accessible[citation.ChunkID]; exists && prior != citation {
				return ErrEvidence
			}
			accessible[citation.ChunkID] = citation
		}
		evidence, err := validateOutput(run.Output, accessible)
		if err != nil {
			return err
		}
		if !slices.Equal(evidence, run.Evidence) {
			return ErrEvidence
		}
	case StateFailed:
		if run.StartedAt == nil || run.FinishedAt == nil || hasOutput || hasEvidence ||
			strings.TrimSpace(run.FailureCode) == "" {
			return ErrInvalidRun
		}
	case StateCancelled:
		if run.StartedAt == nil || run.FinishedAt == nil || hasOutput || hasEvidence || run.FailureCode != "" {
			return ErrInvalidRun
		}
	default:
		return ErrInvalidRun
	}
	return nil
}

func (usage UsageRecord) Validate() error {
	if usage.InputTokens < 0 || usage.OutputTokens < 0 || usage.Cost < 0 {
		return ErrInvalidRun
	}
	return nil
}

func (run Run) Fail(code string, now time.Time) (Run, error) {
	if run.State != StateRunning || strings.TrimSpace(code) == "" || run.StartedAt == nil || now.Before(*run.StartedAt) {
		return Run{}, ErrInvalidRun
	}
	value := cloneRun(run)
	value.State, value.FinishedAt, value.FailureCode = StateFailed, timePointer(now.UTC()), strings.TrimSpace(code)
	return value, nil
}

// Cancel records an interrupted running attempt as a terminal immutable value.
func (run Run) Cancel(now time.Time) (Run, error) {
	if run.State != StateRunning || run.StartedAt == nil || now.Before(*run.StartedAt) {
		return Run{}, ErrInvalidRun
	}
	value := cloneRun(run)
	value.State, value.FinishedAt = StateCancelled, timePointer(now.UTC())
	return value, nil
}

func cloneRun(value Run) Run {
	value.Output = slices.Clone(value.Output)
	value.Evidence = slices.Clone(value.Evidence)
	if value.ParentRunID != nil {
		copied := *value.ParentRunID
		value.ParentRunID = &copied
	}
	if value.StartedAt != nil {
		copied := *value.StartedAt
		value.StartedAt = &copied
	}
	if value.FinishedAt != nil {
		copied := *value.FinishedAt
		value.FinishedAt = &copied
	}
	return value
}

func timePointer(value time.Time) *time.Time { return &value }

type Feedback struct {
	ID        int64     `json:"id,string"`
	RunID     int64     `json:"run_id,string"`
	ActorID   int64     `json:"actor_id,string"`
	Rating    string    `json:"rating"`
	Note      string    `json:"note"`
	RequestID string    `json:"request_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (value Feedback) Validate(run Run) error {
	if value.ID <= 0 || value.RunID != run.ID || value.ActorID <= 0 || run.State != StateSucceeded ||
		!slices.Contains([]string{"faithful", "partial", "not_faithful", "correct", "incorrect"}, value.Rating) ||
		strings.TrimSpace(value.Note) == "" || strings.TrimSpace(value.RequestID) == "" || value.CreatedAt.IsZero() {
		return ErrInvalidRun
	}
	return nil
}

// AppendFeedback validates an attributed feedback event and makes exact request replays
// idempotent without allowing a replay to change immutable content.
func AppendFeedback(history []Feedback, run Run, value Feedback) ([]Feedback, error) {
	if err := value.Validate(run); err != nil {
		return nil, err
	}
	for _, prior := range history {
		if prior.RequestID != value.RequestID || prior.ActorID != value.ActorID || prior.RunID != value.RunID {
			continue
		}
		if prior == value {
			return slices.Clone(history), nil
		}
		return nil, fmt.Errorf("%w: feedback replay differs", ErrInvalidRun)
	}
	return append(slices.Clone(history), value), nil
}

type Selection struct {
	ID         int64     `json:"id,string"`
	SeriesID   int64     `json:"series_id,string"`
	RunID      int64     `json:"run_id,string"`
	ActorID    int64     `json:"actor_id,string"`
	Version    int64     `json:"version"`
	RequestID  string    `json:"request_id"`
	SelectedAt time.Time `json:"selected_at"`
}

// Select appends a reversible selection event. Only successful runs in the same series are valid.
func Select(history []Selection, run Run, value Selection) ([]Selection, error) {
	if run.State != StateSucceeded || value.ID <= 0 || value.RunID != run.ID ||
		value.SeriesID != run.SeriesID || value.ActorID <= 0 || value.SelectedAt.IsZero() ||
		strings.TrimSpace(value.RequestID) == "" {
		return nil, ErrSelection
	}
	if len(history) == 0 {
		if value.Version != 1 {
			return nil, ErrSelection
		}
	} else if value.Version != history[len(history)-1].Version+1 {
		return nil, ErrSelection
	}
	for _, prior := range history {
		if prior.RequestID == value.RequestID {
			if prior == value {
				return slices.Clone(history), nil
			}
			return nil, fmt.Errorf("%w: request replay differs", ErrSelection)
		}
	}
	return append(slices.Clone(history), value), nil
}

type Query struct {
	ProjectID      int64
	Question       string
	Cutoff         time.Time
	SourceIDs      []int64
	MaxResults     int
	MaxOutputBytes int64
}

func (query Query) Validate() error {
	if query.ProjectID <= 0 || strings.TrimSpace(query.Question) == "" || query.Cutoff.IsZero() ||
		query.MaxResults < 1 || query.MaxResults > 100 || query.MaxOutputBytes < 1 || query.MaxOutputBytes > 1<<20 {
		return ErrInvalidRun
	}
	if len(query.Question) > 4_096 || strings.ContainsRune(query.Question, '\x00') || len(query.SourceIDs) > 32 {
		return ErrInvalidRun
	}
	seen := make(map[int64]struct{}, len(query.SourceIDs))
	for _, sourceID := range query.SourceIDs {
		if sourceID <= 0 {
			return ErrInvalidRun
		}
		if _, exists := seen[sourceID]; exists {
			return ErrInvalidRun
		}
		seen[sourceID] = struct{}{}
	}
	return nil
}
