package agent

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
)

type memoryRecord struct {
	proposal        Proposal
	createKey       string
	createDigest    [32]byte
	confirmationKey string
}

// MemoryStore is a concurrency-safe adapter for tests and local degraded mode.
// Production uses the PostgreSQL adapter.
type MemoryStore struct {
	mu      sync.Mutex
	values  map[int64]memoryRecord
	creates map[string]int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{values: make(map[int64]memoryRecord), creates: make(map[string]int64)}
}

func (store *MemoryStore) Create(_ context.Context, proposal Proposal, key string, digest [32]byte) (Proposal, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	identity := createIdentity(proposal, key)
	if id, ok := store.creates[identity]; ok {
		record := store.values[id]
		if subtle.ConstantTimeCompare(record.createDigest[:], digest[:]) != 1 {
			return Proposal{}, false, ErrStateChanged
		}
		return record.proposal, true, nil
	}
	store.values[proposal.ID] = memoryRecord{proposal: proposal, createKey: key, createDigest: digest}
	store.creates[identity] = proposal.ID
	return proposal, false, nil
}

func (store *MemoryStore) Begin(_ context.Context, id int64, principal access.Principal, token [32]byte,
	key string, now time.Time) (Proposal, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, ok := store.values[id]
	if !ok {
		return Proposal{}, false, access.ErrNotFound
	}
	if record.proposal.WorkspaceID != principal.Workspace || record.proposal.ActorID != principal.ActorID {
		return Proposal{}, false, access.ErrPermissionDenied
	}
	if record.confirmationKey == key && (record.proposal.Status == Executed || record.proposal.Status == Failed) {
		return record.proposal, true, nil
	}
	if record.confirmationKey != "" || record.proposal.Status != AwaitingConfirmation {
		return Proposal{}, false, ErrAlreadyUsed
	}
	if !now.Before(record.proposal.ExpiresAt) {
		record.proposal.Status = Expired
		store.values[id] = record
		return Proposal{}, false, ErrExpired
	}
	if subtle.ConstantTimeCompare(record.proposal.TokenDigest[:], token[:]) != 1 {
		return Proposal{}, false, access.ErrPermissionDenied
	}
	record.confirmationKey = key
	record.proposal.Status = Executing
	record.proposal.ConsumedAt = &now
	store.values[id] = record
	return record.proposal, false, nil
}

func (store *MemoryStore) Finish(_ context.Context, id int64, status Status, result Result, at time.Time) (Proposal, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, ok := store.values[id]
	if !ok {
		return Proposal{}, access.ErrNotFound
	}
	if record.proposal.Status != Executing || status != Executed && status != Failed {
		return Proposal{}, errors.New("invalid assistant proposal completion")
	}
	record.proposal.Status = status
	record.proposal.Result = result
	if record.proposal.ConsumedAt == nil {
		record.proposal.ConsumedAt = &at
	}
	store.values[id] = record
	return record.proposal, nil
}

func createIdentity(proposal Proposal, key string) string {
	return fmt.Sprintf("%d:%d:%s", proposal.WorkspaceID, proposal.ActorID, key)
}
