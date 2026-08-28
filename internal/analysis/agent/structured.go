package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
)

// StructuredPlanner is the deterministic, provider-degraded planner. It only
// accepts the same typed draft returned by the ADK boundary and is useful when
// no model provider is configured. Unknown fields and trailing documents are
// rejected before a proposal can be persisted.
type StructuredPlanner struct{}

func (StructuredPlanner) Plan(_ context.Context, _ access.Principal, message string) (Draft, error) {
	decoder := json.NewDecoder(strings.NewReader(message))
	decoder.DisallowUnknownFields()
	var draft Draft
	if err := decoder.Decode(&draft); err != nil {
		return Draft{}, fmt.Errorf("%w: assistant output is not the typed action schema", ErrInvalid)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Draft{}, fmt.Errorf("%w: assistant output contains multiple documents", ErrInvalid)
	}
	if err := draft.Validate(); err != nil {
		return Draft{}, err
	}
	return draft, nil
}

// Limits is the provider-neutral finite envelope around one planner run.
// ADK/model adapters remain responsible for token and monetary accounting;
// this boundary guarantees timeout, output, and concurrent run limits before
// any typed proposal reaches persistence.
type Limits struct {
	MaxSteps       int
	Timeout        time.Duration
	MaxOutputBytes int
	MaxCostMicros  int
	Concurrency    int
}

type BoundedPlanner struct {
	next   Planner
	limits Limits
	gate   chan struct{}
}

func NewBoundedPlanner(next Planner, limits Limits) (*BoundedPlanner, error) {
	if next == nil || limits.MaxSteps < 1 || limits.MaxSteps > 64 || limits.Timeout <= 0 ||
		limits.Timeout > 10*time.Minute || limits.MaxOutputBytes < 1 || limits.MaxCostMicros < 1 ||
		limits.Concurrency < 1 || limits.Concurrency > 8 {
		return nil, ErrRunLimit
	}
	return &BoundedPlanner{next: next, limits: limits, gate: make(chan struct{}, limits.Concurrency)}, nil
}

func (planner *BoundedPlanner) Plan(
	ctx context.Context,
	principal access.Principal,
	message string,
) (Draft, error) {
	select {
	case planner.gate <- struct{}{}:
		defer func() { <-planner.gate }()
	case <-ctx.Done():
		return Draft{}, ctx.Err()
	default:
		return Draft{}, ErrRunLimit
	}
	runCtx, cancel := context.WithTimeout(ctx, planner.limits.Timeout)
	defer cancel()
	draft, err := planner.next.Plan(runCtx, principal, message)
	if err != nil {
		return Draft{}, err
	}
	encoded, err := json.Marshal(draft)
	if err != nil {
		return Draft{}, fmt.Errorf("encode bounded assistant output: %w", err)
	}
	if len(encoded) > planner.limits.MaxOutputBytes {
		return Draft{}, ErrRunLimit
	}
	return draft, nil
}
