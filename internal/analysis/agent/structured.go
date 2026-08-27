package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

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
