package controlplane

import (
	"sync"
	"time"

	"hacp-sidecar/internal/evaluate"
)

// RevocationStoreAdapter bridges the distributed control-plane write path
// with the existing evaluator RevocationStore read path.
//
// Snapshot replacement is implemented by constructing a fresh
// evaluate.InMemoryRevocationStore and atomically swapping it in.
//
// This avoids exposing or mutating the internal maps of the evaluator store.
type RevocationStoreAdapter struct {
	mu sync.RWMutex

	store *evaluate.InMemoryRevocationStore
}

// NewRevocationStoreAdapter creates an adapter backed by a real
// evaluate.InMemoryRevocationStore.
func NewRevocationStoreAdapter() *RevocationStoreAdapter {
	return &RevocationStoreAdapter{
		store: evaluate.NewInMemoryRevocationStore(),
	}
}

// -----------------------------------------------------------------------------
// evaluate.RevocationStore read side
// -----------------------------------------------------------------------------

func (a *RevocationStoreAdapter) IsKeyRevoked(id string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.IsKeyRevoked(id)
}

func (a *RevocationStoreAdapter) IsTokenRevoked(id string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.IsTokenRevoked(id)
}

func (a *RevocationStoreAdapter) IsEnvelopeRevoked(id string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.IsEnvelopeRevoked(id)
}

func (a *RevocationStoreAdapter) LastUpdated() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.LastUpdated()
}

// -----------------------------------------------------------------------------
// MutableRevocationStore write side
// -----------------------------------------------------------------------------

func (a *RevocationStoreAdapter) RevokeKey(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.store.RevokeKey(id)
}

func (a *RevocationStoreAdapter) RevokeToken(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.store.RevokeToken(id)
}

func (a *RevocationStoreAdapter) RevokeEnvelope(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.store.RevokeEnvelope(id)
}

// ReplaceRevocations atomically replaces local distributed revocation state.
//
// A fresh evaluator store is populated first. Only after the complete
// snapshot has been materialized is it made visible to readers.
//
// Readers therefore see either:
//
//	old complete state
//
// or:
//
//	new complete state
//
// but never a partially loaded snapshot.
func (a *RevocationStoreAdapter) ReplaceRevocations(
	keys []string,
	tokens []string,
	envelopes []string,
) error {
	next := evaluate.NewInMemoryRevocationStore()

	for _, id := range keys {
		next.RevokeKey(id)
	}

	for _, id := range tokens {
		next.RevokeToken(id)
	}

	for _, id := range envelopes {
		next.RevokeEnvelope(id)
	}

	a.mu.Lock()
	a.store = next
	a.mu.Unlock()

	return nil
}

// Compile-time contracts.
//
// If either interface changes, compilation fails here rather than later
// during integration.
var _ MutableRevocationStore = (*RevocationStoreAdapter)(nil)
var _ evaluate.RevocationStore = (*RevocationStoreAdapter)(nil)
