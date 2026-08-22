package id_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/id"
)

type memoryLeaser struct {
	mu     sync.Mutex
	nodes  int
	leases map[uint16]id.Lease
}

func (l *memoryLeaser) Acquire(
	_ context.Context,
	holder string,
	now time.Time,
	ttl time.Duration,
) (id.Lease, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.leases == nil {
		l.leases = make(map[uint16]id.Lease)
	}
	for node := range l.nodes {
		candidate := uint16(node)
		lease, occupied := l.leases[candidate]
		if occupied && lease.ExpiresAt.After(now) {
			continue
		}
		lease = id.Lease{Node: candidate, Holder: holder, ExpiresAt: now.Add(ttl)}
		l.leases[candidate] = lease
		return lease, nil
	}
	return id.Lease{}, errors.New("no Snowflake node is available")
}

func (l *memoryLeaser) Renew(
	_ context.Context,
	lease id.Lease,
	now time.Time,
	ttl time.Duration,
) (id.Lease, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	current, exists := l.leases[lease.Node]
	if !exists || current.Holder != lease.Holder || !current.ExpiresAt.After(now) {
		return id.Lease{}, id.ErrLeaseLost
	}
	current.ExpiresAt = now.Add(ttl)
	l.leases[lease.Node] = current
	return current, nil
}

func TestUT225SnowflakeLeaseExclusivity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	leaser := &memoryLeaser{nodes: 1}
	results := make(chan error, 2)
	start := make(chan struct{})
	for _, holder := range []string{"api-a", "api-b"} {
		go func() {
			<-start
			_, err := id.New(context.Background(), leaser, id.Config{
				Holder:   holder,
				LeaseTTL: time.Minute,
				Now:      func() time.Time { return now },
			})
			results <- err
		}()
	}
	close(start)

	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful lease holders = %d, want exactly 1", successes)
	}
}

func TestUT226SnowflakeSequenceAndDecimalSerialization(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 21, 12, 0, 0, 123_000_000, time.UTC)
	generator, err := id.New(context.Background(), &memoryLeaser{nodes: 1}, id.Config{
		Holder:   "api-a",
		LeaseTTL: time.Minute,
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	first, err := generator.Next(context.Background())
	if err != nil {
		t.Fatalf("first Next() error = %v", err)
	}
	second, err := generator.Next(context.Background())
	if err != nil {
		t.Fatalf("second Next() error = %v", err)
	}
	if second != first+1 {
		t.Fatalf("IDs = %d, %d; want same-millisecond sequence ordering", first, second)
	}
	for _, value := range []int64{first, second} {
		decimal := id.Decimal(value)
		parsed, parseErr := strconv.ParseInt(decimal, 10, 64)
		if parseErr != nil || parsed != value {
			t.Errorf("Decimal(%d) = %q, parse = %d, %v", value, decimal, parsed, parseErr)
		}
	}
}

func TestUT227SnowflakeClockRegressionFailsClosed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	generator, err := id.New(context.Background(), &memoryLeaser{nodes: 1}, id.Config{
		Holder:              "api-a",
		LeaseTTL:            time.Minute,
		RegressionTolerance: 5 * time.Millisecond,
		Now:                 func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := generator.Next(context.Background()); err != nil {
		t.Fatalf("first Next() error = %v", err)
	}

	now = now.Add(-time.Second)
	if _, err := generator.Next(context.Background()); !errors.Is(err, id.ErrClockRegression) {
		t.Fatalf("Next() error = %v, want ErrClockRegression", err)
	}
}

func TestSnowflakeRenewsBeforeIssuing(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	generator, err := id.New(context.Background(), &memoryLeaser{nodes: 1}, id.Config{
		Holder: "api-a", LeaseTTL: 30 * time.Second, RenewalInterval: 10 * time.Second,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	now = now.Add(10 * time.Second)
	if _, err := generator.Next(context.Background()); err != nil {
		t.Fatalf("Next() after renewal interval error = %v", err)
	}
	now = now.Add(21 * time.Second)
	if _, err := generator.Next(context.Background()); err != nil {
		t.Fatalf("Next() after original lease expiry error = %v", err)
	}
}
