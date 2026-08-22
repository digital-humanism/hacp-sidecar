package trust

import (
	"crypto/ed25519"
	"fmt"
	"sync"
	"sync/atomic"
)

type storeState struct {
	snapshot TrustSnapshot
}

// AtomicTrustStore exposes read-only signer resolution to the evaluation
// pipeline while allowing validated trust snapshots to be atomically replaced
// by an administrative caller.
type AtomicTrustStore struct {
	replaceMu sync.Mutex
	state     atomic.Pointer[storeState]
}

// NewAtomicTrustStore creates a ready store from a validated non-empty snapshot.
func NewAtomicTrustStore(snapshot TrustSnapshot) (*AtomicTrustStore, error) {
	if err := snapshot.validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTrustSnapshot, err)
	}

	store := &AtomicTrustStore{}
	store.state.Store(&storeState{snapshot: cloneSnapshot(snapshot)})
	return store, nil
}

// ResolveKey implements the existing pipeline-facing KeyResolver behavior.
//
// The returned key is a defensive copy and cannot mutate active trust state.
func (s *AtomicTrustStore) ResolveKey(keyID string) (ed25519.PublicKey, error) {
	if keyID == "" {
		return nil, ErrInvalidKeyID
	}

	current := s.state.Load()
	if current == nil {
		return nil, ErrTrustNotReady
	}

	publicKey, ok := current.snapshot.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, keyID)
	}
	return clonePublicKey(publicKey), nil
}

// Replace validates and atomically activates a complete trust snapshot.
//
// Readers observe either the old complete snapshot or the new complete
// snapshot. They never observe a partially-applied trust state.
func (s *AtomicTrustStore) Replace(snapshot TrustSnapshot) error {
	if err := snapshot.validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTrustSnapshot, err)
	}

	s.replaceMu.Lock()
	defer s.replaceMu.Unlock()

	current := s.state.Load()
	if current != nil {
		currentSnapshot := current.snapshot

		switch {
		case snapshot.revision < currentSnapshot.revision:
			return fmt.Errorf(
				"%w: current=%d candidate=%d",
				ErrTrustRevisionRollback,
				currentSnapshot.revision,
				snapshot.revision,
			)
		case snapshot.revision == currentSnapshot.revision &&
			snapshot.fingerprint != currentSnapshot.fingerprint:
			return fmt.Errorf(
				"%w: revision=%d",
				ErrTrustRevisionConflict,
				snapshot.revision,
			)
		case snapshot.revision == currentSnapshot.revision &&
			snapshot.fingerprint == currentSnapshot.fingerprint:
			return nil
		}
	}

	s.state.Store(&storeState{snapshot: cloneSnapshot(snapshot)})
	return nil
}

// Ready reports whether a validated active snapshot exists.
func (s *AtomicTrustStore) Ready() bool {
	return s.state.Load() != nil
}

// Revision returns the active snapshot revision.
func (s *AtomicTrustStore) Revision() (uint64, error) {
	current := s.state.Load()
	if current == nil {
		return 0, ErrTrustNotReady
	}
	return current.snapshot.revision, nil
}

// Fingerprint returns the active snapshot fingerprint.
func (s *AtomicTrustStore) Fingerprint() (string, error) {
	current := s.state.Load()
	if current == nil {
		return "", ErrTrustNotReady
	}
	return current.snapshot.fingerprint, nil
}

// KeyCount returns the number of active signer bindings.
func (s *AtomicTrustStore) KeyCount() (int, error) {
	current := s.state.Load()
	if current == nil {
		return 0, ErrTrustNotReady
	}
	return len(current.snapshot.keys), nil
}
