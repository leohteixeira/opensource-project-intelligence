package analysis

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

var (
	ErrToolNotAllowed   = errors.New("agent tool not allowed")
	ErrActionNotAllowed = errors.New("action_not_allowed")
	ErrConfirmation     = errors.New("invalid human confirmation")
	ErrBudgetExceeded   = errors.New("agent budget exceeded")
)

// Guard is the provider-neutral authority boundary used before an agent tool
// is invoked. Tool names are explicit; arbitrary SQL, filesystems, and HTTP
// clients are never ambient capabilities.
type Guard struct {
	tools []string
}

func NewGuard(tools ...string) (*Guard, error) {
	if len(tools) == 0 {
		return nil, ErrToolNotAllowed
	}
	for _, tool := range tools {
		if strings.TrimSpace(tool) == "" {
			return nil, ErrToolNotAllowed
		}
	}
	return &Guard{tools: append([]string(nil), tools...)}, nil
}

func (guard *Guard) AuthorizeTool(tool string) error {
	if guard == nil || !slices.Contains(guard.tools, tool) {
		return fmt.Errorf("%w: %s", ErrToolNotAllowed, strings.TrimSpace(tool))
	}
	return nil
}

func AuthorizeProposal(action string) error {
	forbidden := []string{"membership", "credential", "policy", "archive", "deletion"}
	if slices.Contains(forbidden, strings.ToLower(strings.TrimSpace(action))) {
		return ErrActionNotAllowed
	}
	return nil
}

// Confirmation is persisted with an awaiting run. UsedAt must be committed in
// the same transaction as the approved mutation so a restart cannot reuse it.
type Confirmation struct {
	Action    string     `json:"action"`
	Version   int64      `json:"version"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

func (confirmation Confirmation) Consume(action string, version int64, now time.Time) (Confirmation, error) {
	if confirmation.UsedAt != nil || !now.Before(confirmation.ExpiresAt) ||
		confirmation.Action != action || confirmation.Version != version {
		return Confirmation{}, ErrConfirmation
	}
	usedAt := now.UTC()
	confirmation.UsedAt = &usedAt
	return confirmation, nil
}

type Budget struct {
	MaxSteps  int
	MaxOutput int64
	MaxCost   int64
	Deadline  time.Time
}

type Usage struct {
	Steps  int
	Output int64
	Cost   int64
	Now    time.Time
}

func (budget Budget) Check(usage Usage) error {
	if budget.MaxSteps <= 0 || budget.MaxOutput <= 0 || budget.MaxCost <= 0 || budget.Deadline.IsZero() {
		return fmt.Errorf("%w: invalid bounds", ErrBudgetExceeded)
	}
	switch {
	case usage.Steps > budget.MaxSteps:
		return fmt.Errorf("%w: steps", ErrBudgetExceeded)
	case usage.Output > budget.MaxOutput:
		return fmt.Errorf("%w: output", ErrBudgetExceeded)
	case usage.Cost > budget.MaxCost:
		return fmt.Errorf("%w: cost", ErrBudgetExceeded)
	case usage.Now.After(budget.Deadline):
		return fmt.Errorf("%w: duration", ErrBudgetExceeded)
	default:
		return nil
	}
}
