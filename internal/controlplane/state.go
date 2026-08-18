package controlplane

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrHeartbeatRevisionMismatch = errors.New(
	"heartbeat revision does not match materialized revision",
)

// ControlState describes the health/freshness of distributed control state.
//
// It is intentionally separate from RevocationStore:
//
//   - RevocationStore answers: "what is revoked?"
//   - ControlState answers:    "can this local knowledge still be trusted?"
//
// A temporary transport disconnect does not immediately make state stale.
// Previously synchronized state remains usable until maxStaleness expires.
//
// An explicit unsafe condition, however, fails closed immediately.
type ControlState struct {
	mu sync.RWMutex

	lastSeenRevision uint64
	lastUpdate       time.Time

	connected bool
	unsafe    bool

	maxStaleness time.Duration
}

func NewControlState(maxStaleness time.Duration) *ControlState {
	if maxStaleness <= 0 {
		maxStaleness = 5 * time.Second
	}

	return &ControlState{
		maxStaleness: maxStaleness,
	}
}

// LastSeenRevision is the highest completely materialized control-plane
// revision.
func (s *ControlState) LastSeenRevision() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastSeenRevision
}

func (s *ControlState) LastUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastUpdate
}

func (s *ControlState) Connected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.connected
}

func (s *ControlState) Unsafe() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe
}

// MarkConnected records transport connectivity.
//
// Connectivity alone does NOT refresh freshness because merely opening a TCP
// or gRPC connection proves nothing about the completeness of control state.
func (s *ControlState) MarkConnected() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connected = true
}

func (s *ControlState) MarkDisconnected() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connected = false
}

// MarkSnapshot records successful atomic materialization of a complete
// snapshot.
func (s *ControlState) MarkSnapshot(
	revision uint64,
	now time.Time,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastSeenRevision = revision
	s.lastUpdate = now
	s.unsafe = false
}

// MarkEvent records successful materialization of one ordered revision.
//
// Revision ordering itself remains enforced by Subscriber.applyEvent().
func (s *ControlState) MarkEvent(
	revision uint64,
	now time.Time,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastSeenRevision = revision
	s.lastUpdate = now
	s.unsafe = false
}

// MarkHeartbeat records evidence that the control-plane stream is alive and
// synchronized.
//
// A heartbeat MUST describe exactly the highest revision already materialized
// locally. A heartbeat claiming a different revision cannot safely refresh
// freshness.
//
// IMPORTANT:
//
// heartbeat does not advance lastSeenRevision because it carries no revocation
// mutation.
func (s *ControlState) MarkHeartbeat(
	currentRevision uint64,
	now time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if currentRevision != s.lastSeenRevision {
		s.unsafe = true

		return fmt.Errorf(
			"%w: heartbeat=%d last_seen=%d",
			ErrHeartbeatRevisionMismatch,
			currentRevision,
			s.lastSeenRevision,
		)
	}

	s.lastUpdate = now
	s.unsafe = false

	return nil
}

// MarkUnsafe immediately invalidates the locally held distributed control
// state.
//
// Typical reasons:
//   - revision gap
//   - malformed control event
//   - unknown revocation kind
//   - protocol invariant violation
func (s *ControlState) MarkUnsafe() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.unsafe = true
}

// IsFresh reports whether local distributed control state may currently be
// trusted.
//
// Rules:
//
//	explicit unsafe
//	    -> false
//
//	no valid snapshot/event/heartbeat ever observed
//	    -> false
//
//	age <= maxStaleness
//	    -> true
//
//	age > maxStaleness
//	    -> false
//
// Connectivity is deliberately not part of this decision.
func (s *ControlState) IsFresh(now time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unsafe {
		return false
	}

	if s.lastUpdate.IsZero() {
		return false
	}

	// Defensive handling for clocks that move backwards.
	if now.Before(s.lastUpdate) {
		return true
	}

	return now.Sub(s.lastUpdate) <= s.maxStaleness
}
