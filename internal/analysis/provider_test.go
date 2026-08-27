package analysis

import (
	"context"
	"errors"
	"testing"
	"time"
)

func validProvider() ProviderConfig {
	return ProviderConfig{Provider: "vertex", Model: "gemini", Capabilities: []string{"analysis", "assistant"},
		MaxConcurrency: 1, CostCurrency: "USD", InputCost: 1, OutputCost: 2, Enabled: true, Revision: 1}
}

func TestUT205ProviderConfigurationValidatesBeforeActivation(t *testing.T) {
	manager, err := NewProviderManager(nil)
	if err != nil {
		t.Fatal(err)
	}
	config := validProvider()
	config.Model = ""
	if err := manager.Activate(config); !errors.Is(err, ErrInvalidProviderConfig) {
		t.Fatalf("Activate() = %v", err)
	}
	if manager.Status().Configured {
		t.Fatal("invalid config became active")
	}
}

func TestUT206NoProviderIsValidDegradedStartup(t *testing.T) {
	manager, err := NewProviderManager(nil)
	if err != nil {
		t.Fatal(err)
	}
	status := manager.Status()
	if status.Configured || status.Health != "unavailable" || !status.Redacted {
		t.Fatalf("status = %#v", status)
	}
	if _, _, err := manager.Acquire(context.Background()); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("Acquire() = %v", err)
	}
}

func TestUT207ProviderConfigurationIsFrozenPerRun(t *testing.T) {
	manager, _ := NewProviderManager(ptr(validProvider()))
	snapshot, release, err := manager.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	next := validProvider()
	next.Model = "gemini-next"
	next.Revision = 2
	if err := manager.Activate(next); err != nil {
		t.Fatal(err)
	}
	if snapshot.Model != "gemini" || snapshot.Revision != 1 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	release(UsageRecord{})
}

func TestUT208ConcurrencyQuotaDoesNotRetryStorm(t *testing.T) {
	manager, _ := NewProviderManager(ptr(validProvider()))
	_, release, _ := manager.Acquire(context.Background())
	for index := 0; index < 100; index++ {
		if _, _, err := manager.Acquire(context.Background()); !errors.Is(err, ErrProviderQuota) {
			t.Fatalf("Acquire %d = %v", index, err)
		}
	}
	release(UsageRecord{})
}

func TestUT209UsageCostAndHealthAreAggregateAndRedacted(t *testing.T) {
	manager, _ := NewProviderManager(ptr(validProvider()))
	_, release, _ := manager.Acquire(context.Background())
	release(UsageRecord{InputTokens: 1_000_000, OutputTokens: 500_000})
	manager.MarkDegraded()
	status := manager.Status()
	if status.Usage.Runs != 1 || status.Usage.Cost != 2 || status.Health != "degraded" || !status.Redacted {
		t.Fatalf("status = %#v", status)
	}
}

func TestUT210DisabledProviderAndInterruptedRunRemainTerminal(t *testing.T) {
	config := validProvider()
	config.Enabled = false
	manager, err := NewProviderManager(&config)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.Acquire(context.Background()); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("Acquire() = %v", err)
	}
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	run := Run{State: StateRunning, StartedAt: &now}
	cancelled, err := FinishInterrupted(run, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != StateCancelled || cancelled.FinishedAt == nil {
		t.Fatalf("run = %#v", cancelled)
	}
}

func ptr[T any](value T) *T { return &value }
