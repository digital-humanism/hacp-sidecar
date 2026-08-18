package controlplane

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"
)

// MutableRevocationStore is the local write-side contract required by
// distributed revocation delivery.
type MutableRevocationStore interface {
	RevokeKey(string)
	RevokeToken(string)
	RevokeEnvelope(string)

	ReplaceRevocations(
		keys []string,
		tokens []string,
		envelopes []string,
	) error
}

var (
	ErrRevisionGap           = errors.New("revocation revision gap")
	ErrUnknownRevocationKind = errors.New("unknown revocation kind")
	ErrSnapshotRequired      = errors.New("revocation snapshot required")
)

// Subscriber consumes distributed revocation state from ControlPlane,
// materializes it into the local RevocationStore, and tracks distributed
// control-state freshness separately through ControlState.
type Subscriber struct {
	client controlplanev1.ControlPlaneClient
	store  MutableRevocationStore
	state  *ControlState

	sidecarID string

	mu               sync.RWMutex
	lastSeenRevision uint64

	reconnectInitialBackoff time.Duration
	reconnectMaxBackoff     time.Duration
}

// NewSubscriber creates a distributed revocation subscriber.
//
// A default ControlState is created automatically.
// Tests or runtime wiring may replace it with SetControlState().
func NewSubscriber(
	client controlplanev1.ControlPlaneClient,
	store MutableRevocationStore,
	sidecarID string,
) *Subscriber {
	if client == nil {
		panic("controlplane: nil client")
	}
	if store == nil {
		panic("controlplane: nil revocation store")
	}

	return &Subscriber{
		client:                  client,
		store:                   store,
		state:                   NewControlState(5 * time.Second),
		sidecarID:               sidecarID,
		reconnectInitialBackoff: 100 * time.Millisecond,
		reconnectMaxBackoff:     5 * time.Second,
	}
}

// SetControlState replaces the freshness state used by this subscriber.
//
// Intended for runtime dependency wiring and deterministic tests.
func (s *Subscriber) SetControlState(state *ControlState) {
	if state == nil {
		panic("controlplane: nil control state")
	}

	s.state = state
}

// ControlState returns the subscriber's distributed control-state tracker.
func (s *Subscriber) ControlState() *ControlState {
	return s.state
}

// LastSeenRevision returns the highest revision successfully materialized
// into the local revocation store.
func (s *Subscriber) LastSeenRevision() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastSeenRevision
}

func (s *Subscriber) setLastSeenRevision(revision uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastSeenRevision = revision
}

// Run performs:
//
//	snapshot
//	→ watch
//	→ reconnect/replay
//	→ snapshot recovery when replay is unavailable
func (s *Subscriber) Run(ctx context.Context) error {
	if err := s.loadSnapshot(ctx); err != nil {
		return fmt.Errorf("load revocation snapshot: %w", err)
	}

	var backoff time.Duration

	for {
		connected, err := s.watch(ctx)

		if ctx.Err() != nil {
			if s.state != nil {
				s.state.MarkDisconnected()
			}

			return ctx.Err()
		}

		if errors.Is(err, ErrSnapshotRequired) {
			if s.state != nil {
				s.state.MarkUnsafe()
			}

			if err := s.loadSnapshot(ctx); err != nil {
				return fmt.Errorf(
					"recover revocation snapshot: %w",
					err,
				)
			}

			backoff = 0
			continue
		}

		if connected {
			backoff = 0
		}

		if err == nil {
			continue
		}

		if s.state != nil {
			s.state.MarkDisconnected()
		}

		backoff = nextReconnectBackoff(
			backoff,
			s.reconnectInitialBackoff,
			s.reconnectMaxBackoff,
		)

		if err := waitForReconnect(ctx, backoff); err != nil {
			return err
		}
	}
}

func nextReconnectBackoff(
	current time.Duration,
	initial time.Duration,
	max time.Duration,
) time.Duration {
	if initial <= 0 {
		initial = 100 * time.Millisecond
	}

	if max < initial {
		max = initial
	}

	if current <= 0 {
		return initial
	}

	if current >= max {
		return max
	}

	next := current * 2
	if next > max {
		return max
	}

	return next
}

func waitForReconnect(
	ctx context.Context,
	delay time.Duration,
) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-timer.C:
		return nil
	}
}

func (s *Subscriber) loadSnapshot(ctx context.Context) error {
	snapshot, err := s.client.GetRevocationSnapshot(
		ctx,
		&controlplanev1.GetRevocationSnapshotRequest{
			SidecarId: s.sidecarID,
		},
	)
	if err != nil {
		return err
	}

	var keys []string
	var tokens []string
	var envelopes []string

	for _, entry := range snapshot.GetEntries() {
		switch entry.GetKind() {
		case controlplanev1.RevocationKind_REVOCATION_KIND_KEY:
			keys = append(keys, entry.GetSubjectId())

		case controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN:
			tokens = append(tokens, entry.GetSubjectId())

		case controlplanev1.RevocationKind_REVOCATION_KIND_ENVELOPE,
			controlplanev1.RevocationKind_REVOCATION_KIND_PARENT_ENVELOPE:

			envelopes = append(envelopes, entry.GetSubjectId())

		default:
			if s.state != nil {
				s.state.MarkUnsafe()
			}

			return fmt.Errorf(
				"%w in snapshot: %s",
				ErrUnknownRevocationKind,
				entry.GetKind(),
			)
		}
	}

	if err := s.store.ReplaceRevocations(
		keys,
		tokens,
		envelopes,
	); err != nil {
		if s.state != nil {
			s.state.MarkUnsafe()
		}

		return err
	}

	s.setLastSeenRevision(snapshot.GetRevision())

	if s.state != nil {
		s.state.MarkSnapshot(
			snapshot.GetRevision(),
			time.Now(),
		)
	}

	return nil
}

// watch establishes one WatchRevocations stream.
//
// connected is true once the gRPC stream has been established.
func (s *Subscriber) watch(
	ctx context.Context,
) (connected bool, err error) {
	stream, err := s.client.WatchRevocations(
		ctx,
		&controlplanev1.WatchRevocationsRequest{
			SidecarId:     s.sidecarID,
			AfterRevision: s.LastSeenRevision(),
		},
	)
	if err != nil {
		return false, err
	}

	if s.state != nil {
		s.state.MarkConnected()
	}

	for {
		response, err := stream.Recv()
		if err != nil {
			if s.state != nil {
				s.state.MarkDisconnected()
			}

			return true, err
		}

		if response.GetResetRequired() != nil {
			if s.state != nil {
				s.state.MarkUnsafe()
			}

			return true, ErrSnapshotRequired
		}

		if heartbeat := response.GetHeartbeat(); heartbeat != nil {
			if s.state != nil {
				if err := s.state.MarkHeartbeat(
					heartbeat.GetCurrentRevision(),
					time.Now(),
				); err != nil {
					return true, err
				}
			}

			continue
		}

		event := response.GetEvent()
		if event == nil {
			continue
		}

		if err := s.applyEvent(event); err != nil {
			if s.state != nil {
				s.state.MarkUnsafe()
			}

			return true, err
		}
	}
}

func (s *Subscriber) applyEvent(
	event *controlplanev1.RevocationEvent,
) error {
	last := s.LastSeenRevision()
	revision := event.GetRevision()

	// Duplicate / already materialized.
	if revision <= last {
		return nil
	}

	// Missing revision means the local distributed state can no longer
	// be proven complete.
	if revision != last+1 {
		if s.state != nil {
			s.state.MarkUnsafe()
		}

		return fmt.Errorf(
			"%w: last_seen_revision=%d received_revision=%d",
			ErrRevisionGap,
			last,
			revision,
		)
	}

	switch event.GetKind() {
	case controlplanev1.RevocationKind_REVOCATION_KIND_KEY:
		s.store.RevokeKey(event.GetSubjectId())

	case controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN:
		s.store.RevokeToken(event.GetSubjectId())

	case controlplanev1.RevocationKind_REVOCATION_KIND_ENVELOPE,
		controlplanev1.RevocationKind_REVOCATION_KIND_PARENT_ENVELOPE:

		s.store.RevokeEnvelope(event.GetSubjectId())

	default:
		if s.state != nil {
			s.state.MarkUnsafe()
		}

		return fmt.Errorf(
			"%w: %s",
			ErrUnknownRevocationKind,
			event.GetKind(),
		)
	}

	// Cursor advances only AFTER successful local materialization.
	s.setLastSeenRevision(revision)

	if s.state != nil {
		s.state.MarkEvent(
			revision,
			time.Now(),
		)
	}

	return nil
}
