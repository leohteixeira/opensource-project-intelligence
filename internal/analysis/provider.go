package analysis

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidProviderConfig = errors.New("analysis: invalid provider configuration")
	ErrProviderQuota         = errors.New("analysis: provider quota exhausted")
)

type ProviderConfig struct {
	Provider       string
	Model          string
	Capabilities   []string
	MaxConcurrency int
	CostCurrency   string
	InputCost      float64
	OutputCost     float64
	Enabled        bool
	Revision       int64
}

func (config ProviderConfig) Validate() error {
	if strings.TrimSpace(config.Provider) == "" || strings.TrimSpace(config.Model) == "" ||
		config.MaxConcurrency <= 0 || config.MaxConcurrency > 64 || config.Revision <= 0 ||
		config.InputCost < 0 || config.OutputCost < 0 ||
		(config.InputCost > 0 || config.OutputCost > 0) && strings.TrimSpace(config.CostCurrency) == "" {
		return ErrInvalidProviderConfig
	}
	seen := make(map[string]struct{}, len(config.Capabilities))
	for _, capability := range config.Capabilities {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			return ErrInvalidProviderConfig
		}
		if _, exists := seen[capability]; exists {
			return ErrInvalidProviderConfig
		}
		seen[capability] = struct{}{}
	}
	return nil
}

type ProviderSnapshot struct {
	Provider       string   `json:"provider"`
	Model          string   `json:"model"`
	Capabilities   []string `json:"capabilities"`
	MaxConcurrency int      `json:"max_concurrency"`
	Enabled        bool     `json:"enabled"`
	Revision       int64    `json:"revision"`
	inputCost      float64
	outputCost     float64
	currency       string
}

type ProviderUsage struct {
	Runs         int64   `json:"runs"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Cost         float64 `json:"cost"`
	Currency     string  `json:"currency,omitempty"`
}

type ProviderStatus struct {
	Configured   bool          `json:"configured"`
	Enabled      bool          `json:"enabled"`
	Health       string        `json:"health"`
	Provider     string        `json:"provider,omitempty"`
	Model        string        `json:"model,omitempty"`
	Capabilities []string      `json:"capabilities"`
	Revision     int64         `json:"revision,omitempty"`
	Usage        ProviderUsage `json:"usage"`
	Redacted     bool          `json:"redacted"`
}

type ProviderManager struct {
	mu     sync.RWMutex
	active *ProviderSnapshot
	health string
	usage  ProviderUsage
	gate   chan struct{}
}

func NewProviderManager(config *ProviderConfig) (*ProviderManager, error) {
	manager := &ProviderManager{health: "unavailable"}
	if config == nil {
		return manager, nil
	}
	if err := manager.Activate(*config); err != nil {
		return nil, err
	}
	return manager, nil
}

func (manager *ProviderManager) Activate(config ProviderConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	snapshot := &ProviderSnapshot{Provider: strings.TrimSpace(config.Provider), Model: strings.TrimSpace(config.Model),
		Capabilities: append([]string(nil), config.Capabilities...), MaxConcurrency: config.MaxConcurrency,
		Enabled: config.Enabled, Revision: config.Revision, inputCost: config.InputCost,
		outputCost: config.OutputCost, currency: strings.TrimSpace(config.CostCurrency)}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.active = snapshot
	manager.gate = make(chan struct{}, config.MaxConcurrency)
	if config.Enabled {
		manager.health = "healthy"
	} else {
		manager.health = "disabled"
	}
	manager.usage.Currency = snapshot.currency
	return nil
}

func (manager *ProviderManager) Snapshot() (ProviderSnapshot, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if manager.active == nil || !manager.active.Enabled {
		return ProviderSnapshot{}, ErrProviderUnavailable
	}
	value := *manager.active
	value.Capabilities = append([]string(nil), manager.active.Capabilities...)
	return value, nil
}

// Acquire freezes one provider configuration for the complete run and bounds
// concurrency. Release records only aggregate usage and never prompt content.
func (manager *ProviderManager) Acquire(ctx context.Context) (ProviderSnapshot, func(UsageRecord), error) {
	snapshot, err := manager.Snapshot()
	if err != nil {
		return ProviderSnapshot{}, nil, err
	}
	manager.mu.RLock()
	gate := manager.gate
	manager.mu.RUnlock()
	select {
	case gate <- struct{}{}:
	case <-ctx.Done():
		return ProviderSnapshot{}, nil, ctx.Err()
	default:
		return ProviderSnapshot{}, nil, ErrProviderQuota
	}
	var once sync.Once
	release := func(usage UsageRecord) {
		once.Do(func() {
			<-gate
			manager.mu.Lock()
			manager.usage.Runs++
			manager.usage.InputTokens += usage.InputTokens
			manager.usage.OutputTokens += usage.OutputTokens
			manager.usage.Cost += float64(usage.InputTokens)*snapshot.inputCost/1_000_000 +
				float64(usage.OutputTokens)*snapshot.outputCost/1_000_000
			manager.mu.Unlock()
		})
	}
	return snapshot, release, nil
}

func (manager *ProviderManager) MarkDegraded() {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.active != nil && manager.active.Enabled {
		manager.health = "degraded"
	}
}

func (manager *ProviderManager) Status() ProviderStatus {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	status := ProviderStatus{Health: manager.health, Capabilities: []string{}, Usage: manager.usage, Redacted: true}
	if manager.active != nil {
		status.Configured = true
		status.Enabled = manager.active.Enabled
		status.Provider = manager.active.Provider
		status.Model = manager.active.Model
		status.Capabilities = append([]string(nil), manager.active.Capabilities...)
		status.Revision = manager.active.Revision
	}
	return status
}

// FinishInterrupted converts a process-shutdown interruption to a terminal run.
func FinishInterrupted(run Run, now time.Time) (Run, error) {
	if run.State != StateRunning {
		return Run{}, ErrInvalidRun
	}
	return run.Cancel(now)
}
