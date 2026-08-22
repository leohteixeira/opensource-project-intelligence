// Package id issues signed-positive Snowflake identifiers through an exclusive node lease.
package id

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

const (
	nodeBits     = 10
	sequenceBits = 12
	maxNode      = 1<<nodeBits - 1
	maxSequence  = 1<<sequenceBits - 1
)

var (
	// Epoch is the versioned UTC origin of every identifier.
	Epoch = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	// ErrLeaseLost reports that issuance cannot continue without an authoritative lease.
	ErrLeaseLost = errors.New("snowflake node lease is not valid")
	// ErrClockRegression reports a clock movement beyond the configured tolerance.
	ErrClockRegression = errors.New("clock regressed beyond tolerance")
)

// Lease is the authoritative grant for one Snowflake node.
type Lease struct {
	Node      uint16
	Holder    string
	ExpiresAt time.Time
}

// Leaser allocates a node exclusively. Implementations coordinate through PostgreSQL.
type Leaser interface {
	Acquire(context.Context, string, time.Time, time.Duration) (Lease, error)
	Renew(context.Context, Lease, time.Time, time.Duration) (Lease, error)
}

// Config contains deterministic generator dependencies and timing limits.
type Config struct {
	Holder              string
	LeaseTTL            time.Duration
	RenewalInterval     time.Duration
	RegressionTolerance time.Duration
	Now                 func() time.Time
	Sleep               func(context.Context, time.Duration) error
}

// Generator serializes issuance for one leased node.
type Generator struct {
	mu sync.Mutex

	lease           Lease
	leaser          Leaser
	leaseTTL        time.Duration
	renewAt         time.Time
	renewalInterval time.Duration
	now             func() time.Time
	sleep           func(context.Context, time.Duration) error
	tolerance       time.Duration
	lastMillisecond int64
	sequence        uint16
}

// New acquires one node and constructs a fail-closed generator.
func New(ctx context.Context, leaser Leaser, cfg Config) (*Generator, error) {
	if cfg.Holder == "" {
		return nil, errors.New("snowflake holder is required")
	}
	if cfg.LeaseTTL <= 0 {
		return nil, errors.New("snowflake lease TTL must be positive")
	}
	if cfg.RenewalInterval == 0 {
		cfg.RenewalInterval = cfg.LeaseTTL / 3
	}
	if cfg.RenewalInterval <= 0 || cfg.RenewalInterval >= cfg.LeaseTTL {
		return nil, errors.New("snowflake renewal interval must be positive and shorter than the lease TTL")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Sleep == nil {
		cfg.Sleep = sleepContext
	}

	now := cfg.Now().UTC()
	lease, err := leaser.Acquire(ctx, cfg.Holder, now, cfg.LeaseTTL)
	if err != nil {
		return nil, fmt.Errorf("acquire Snowflake node lease: %w", err)
	}
	if lease.Node > maxNode || lease.Holder != cfg.Holder || !lease.ExpiresAt.After(now) {
		return nil, fmt.Errorf("acquire Snowflake node lease: %w", ErrLeaseLost)
	}

	return &Generator{
		lease:           lease,
		leaser:          leaser,
		leaseTTL:        cfg.LeaseTTL,
		renewAt:         now.Add(cfg.RenewalInterval),
		renewalInterval: cfg.RenewalInterval,
		now:             cfg.Now,
		sleep:           cfg.Sleep,
		tolerance:       cfg.RegressionTolerance,
	}, nil
}

// Next returns a unique identifier and refuses issuance after lease loss or unsafe clock movement.
func (g *Generator) Next(ctx context.Context) (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now().UTC()
	if !now.Before(g.lease.ExpiresAt) {
		return 0, ErrLeaseLost
	}
	if !now.Before(g.renewAt) {
		lease, err := g.leaser.Renew(ctx, g.lease, now, g.leaseTTL)
		if err != nil {
			return 0, fmt.Errorf("renew Snowflake node lease: %w", err)
		}
		if lease.Node != g.lease.Node || lease.Holder != g.lease.Holder || !lease.ExpiresAt.After(now) {
			return 0, ErrLeaseLost
		}
		g.lease = lease
		g.renewAt = now.Add(g.renewalInterval)
	}
	millisecond := now.UnixMilli()
	if millisecond < g.lastMillisecond {
		regression := time.Duration(g.lastMillisecond-millisecond) * time.Millisecond
		if regression > g.tolerance {
			return 0, fmt.Errorf("%w: %s", ErrClockRegression, regression)
		}
		if err := g.sleep(ctx, regression); err != nil {
			return 0, fmt.Errorf("wait for clock recovery: %w", err)
		}
		now = g.now().UTC()
		if !now.Before(g.lease.ExpiresAt) {
			return 0, ErrLeaseLost
		}
		millisecond = now.UnixMilli()
		if millisecond < g.lastMillisecond {
			return 0, fmt.Errorf("%w: clock did not recover", ErrClockRegression)
		}
	}

	if millisecond == g.lastMillisecond {
		if g.sequence == maxSequence {
			waitUntil := time.UnixMilli(g.lastMillisecond + 1)
			if !waitUntil.Before(g.lease.ExpiresAt) {
				return 0, ErrLeaseLost
			}
			if err := g.sleep(ctx, waitUntil.Sub(now)); err != nil {
				return 0, fmt.Errorf("wait for Snowflake sequence: %w", err)
			}
			now = g.now().UTC()
			if !now.Before(g.lease.ExpiresAt) {
				return 0, ErrLeaseLost
			}
			millisecond = now.UnixMilli()
			if millisecond <= g.lastMillisecond {
				return 0, errors.New("Snowflake clock did not advance after sequence exhaustion")
			}
			g.lastMillisecond = millisecond
			g.sequence = 0
		} else {
			g.sequence++
		}
	} else {
		g.lastMillisecond = millisecond
		g.sequence = 0
	}

	delta := millisecond - Epoch.UnixMilli()
	if delta < 0 || delta >= 1<<41 {
		return 0, errors.New("Snowflake timestamp is outside the supported epoch")
	}

	value := delta<<(nodeBits+sequenceBits) |
		int64(g.lease.Node)<<sequenceBits |
		int64(g.sequence)
	if value <= 0 {
		return 0, errors.New("Snowflake identifier must be positive")
	}

	return value, nil
}

// Decimal serializes an identifier without JavaScript precision loss.
func Decimal(value int64) string {
	return strconv.FormatInt(value, 10)
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
