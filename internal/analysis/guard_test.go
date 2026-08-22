package analysis_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis"
)

// UT-263: agent tools are explicit capability-scoped adapters.
func TestUT263AgentToolsAreExplicitlyAllowlisted(t *testing.T) {
	t.Parallel()
	guard, err := analysis.NewGuard("project.read", "evidence.read")
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.AuthorizeTool("project.read"); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"sql.execute", "filesystem.read", "http.request"} {
		if !errors.Is(guard.AuthorizeTool(tool), analysis.ErrToolNotAllowed) {
			t.Fatalf("tool %q escaped the allowlist", tool)
		}
	}
}

// UT-264: confirmations are single-use, action/version/expiry-bound grants.
func TestUT264ConfirmationIsSingleUseActionVersionAndExpiryBound(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 22, 7, 0, 0, 0, time.UTC)
	base := analysis.Confirmation{Action: "project.sync", Version: 4, ExpiresAt: now.Add(time.Minute)}
	for _, value := range []struct {
		confirmation analysis.Confirmation
		action       string
		version      int64
		at           time.Time
	}{
		{base, "project.archive", 4, now},
		{base, "project.sync", 5, now},
		{base, "project.sync", 4, base.ExpiresAt},
	} {
		if _, err := value.confirmation.Consume(value.action, value.version, value.at); !errors.Is(err, analysis.ErrConfirmation) {
			t.Fatalf("mismatched confirmation returned %v", err)
		}
	}
	used, err := base.Consume("project.sync", 4, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := used.Consume("project.sync", 4, now); !errors.Is(err, analysis.ErrConfirmation) {
		t.Fatal("used confirmation was reusable")
	}
}

// UT-265: forbidden proposal classes fail before confirmation is possible.
func TestUT265ForbiddenProposalFailsBeforeConfirmation(t *testing.T) {
	t.Parallel()
	for _, action := range []string{"membership", "credential", "policy", "archive", "deletion"} {
		if !errors.Is(analysis.AuthorizeProposal(action), analysis.ErrActionNotAllowed) {
			t.Fatalf("proposal %q reached confirmation", action)
		}
	}
}

// IT-127: persisted confirmation authority survives a process restart exactly.
func TestIT127PersistedConfirmationRecoversWithIdenticalAuthority(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 22, 7, 0, 0, 0, time.UTC)
	before := analysis.Confirmation{Action: "project.sync", Version: 9, ExpiresAt: now.Add(time.Hour)}
	encoded, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	var recovered analysis.Confirmation
	if err := json.Unmarshal(encoded, &recovered); err != nil {
		t.Fatal(err)
	}
	if recovered != before {
		t.Fatalf("recovered confirmation = %#v, want %#v", recovered, before)
	}
	if _, err := recovered.Consume("project.sync", 9, now.Add(time.Minute)); err != nil {
		t.Fatalf("recovered unexpired confirmation failed: %v", err)
	}
}

// IT-128: every durable execution budget dimension terminates independently.
func TestIT128EveryAgentBudgetDimensionTerminatesExecution(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 22, 7, 0, 0, 0, time.UTC)
	budget := analysis.Budget{MaxSteps: 3, MaxOutput: 1024, MaxCost: 20, Deadline: now.Add(time.Minute)}
	for _, usage := range []analysis.Usage{
		{Steps: 4, Output: 1, Cost: 1, Now: now},
		{Steps: 1, Output: 1025, Cost: 1, Now: now},
		{Steps: 1, Output: 1, Cost: 21, Now: now},
		{Steps: 1, Output: 1, Cost: 1, Now: now.Add(2 * time.Minute)},
	} {
		if !errors.Is(budget.Check(usage), analysis.ErrBudgetExceeded) {
			t.Fatalf("usage %#v escaped its budget", usage)
		}
	}
}
