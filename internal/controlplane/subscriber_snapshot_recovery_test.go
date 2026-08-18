package controlplane

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// snapshotRecoveryTestServer orchestrates this deterministic recovery path:
//
//	revision 1 delivered
//	→ forced disconnect
//	→ reconnect(after_revision=1)
//	→ test commits revisions 2 and 3
//	→ journal compacts through revision 2
//	→ second Watch sees replay unavailable
//	→ ResetRequired
//	→ Subscriber loads snapshot revision 3
//	→ third Watch(after_revision=3)
//	→ revision 4 delivered live
type snapshotRecoveryTestServer struct {
	controlplanev1.UnimplementedControlPlaneServer

	journal *Journal

	watchCalls atomic.Int32

	firstWatchStarted  chan struct{}
	secondWatchRequest chan uint64
	releaseSecond      chan struct{}
	thirdWatchRequest  chan uint64
}

func newSnapshotRecoveryTestServer(
	journal *Journal,
) *snapshotRecoveryTestServer {
	return &snapshotRecoveryTestServer{
		journal:            journal,
		firstWatchStarted:  make(chan struct{}),
		secondWatchRequest: make(chan uint64, 1),
		releaseSecond:      make(chan struct{}),
		thirdWatchRequest:  make(chan uint64, 1),
	}
}

func (s *snapshotRecoveryTestServer) GetRevocationSnapshot(
	ctx context.Context,
	req *controlplanev1.GetRevocationSnapshotRequest,
) (*controlplanev1.RevocationSnapshot, error) {
	return NewServer(s.journal).GetRevocationSnapshot(ctx, req)
}

func (s *snapshotRecoveryTestServer) WatchRevocations(
	req *controlplanev1.WatchRevocationsRequest,
	stream grpc.ServerStreamingServer[controlplanev1.WatchRevocationsResponse],
) error {
	call := s.watchCalls.Add(1)

	switch call {
	case 1:
		// -------------------------------------------------------------
		// First stream.
		//
		// Deliver exactly one event successfully, then force disconnect.
		// -------------------------------------------------------------

		close(s.firstWatchStarted)

		lastRevision := req.GetAfterRevision()

		for {
			events, notify, err := s.journal.EventsAfter(lastRevision)
			if err != nil {
				return err
			}

			if len(events) > 0 {
				event := events[0]

				if err := stream.Send(
					&controlplanev1.WatchRevocationsResponse{
						Sequence: 1,
						Payload: &controlplanev1.WatchRevocationsResponse_Event{
							Event: event,
						},
					},
				); err != nil {
					return err
				}

				// Event was successfully delivered.
				// Now simulate transport failure.
				return status.Error(
					codes.Unavailable,
					"forced disconnect after revision 1",
				)
			}

			select {
			case <-stream.Context().Done():
				return stream.Context().Err()

			case <-notify:
			}
		}

	case 2:
		// -------------------------------------------------------------
		// First reconnect.
		//
		// Subscriber should still believe revision 1 is its durable
		// cursor.
		// -------------------------------------------------------------

		s.secondWatchRequest <- req.GetAfterRevision()

		// Block until the test:
		//
		//   commits revisions 2 and 3
		//   compacts replay through revision 2
		//
		// Then the normal server must return ResetRequired.
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()

		case <-s.releaseSecond:
		}

		return NewServer(s.journal).WatchRevocations(req, stream)

	case 3:
		// -------------------------------------------------------------
		// After ResetRequired the Subscriber must have loaded a complete
		// snapshot at revision 3.
		//
		// Therefore the next durable cursor must be 3.
		// -------------------------------------------------------------

		s.thirdWatchRequest <- req.GetAfterRevision()

		return NewServer(s.journal).WatchRevocations(req, stream)

	default:
		return NewServer(s.journal).WatchRevocations(req, stream)
	}
}

func TestSubscriberResetRequiredRecoversFromSnapshot(t *testing.T) {
	journal := NewJournal()

	server := newSnapshotRecoveryTestServer(journal)

	listener := bufconn.Listen(testBufferSize)

	grpcServer := grpc.NewServer()

	controlplanev1.RegisterControlPlaneServer(
		grpcServer,
		server,
	)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := grpc.NewClient(
		"passthrough:///hacp-control-plane",
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
		grpc.WithContextDialer(
			func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			},
		),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	defer conn.Close()

	store := newTestRevocationStore()

	subscriber := NewSubscriber(
		controlplanev1.NewControlPlaneClient(conn),
		store,
		"sidecar-snapshot-recovery-test",
	)

	// Keep reconnect deterministic and fast.
	subscriber.reconnectInitialBackoff = time.Millisecond
	subscriber.reconnectMaxBackoff = time.Millisecond

	errCh := make(chan error, 1)

	go func() {
		errCh <- subscriber.Run(ctx)
	}()

	// -------------------------------------------------------------
	// Wait for initial Watch(after_revision=0).
	// -------------------------------------------------------------

	select {
	case <-server.firstWatchStarted:

	case <-time.After(time.Second):
		t.Fatal("initial WatchRevocations did not start")
	}

	// -------------------------------------------------------------
	// Revision 1.
	//
	// This is successfully delivered before the forced disconnect.
	// -------------------------------------------------------------

	first, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_KEY,
		"key-revision-1",
	)
	if err != nil {
		t.Fatalf("revision 1 revoke: %v", err)
	}
	if !created {
		t.Fatal("revision 1 did not create event")
	}

	if first.Revision != 1 {
		t.Fatalf(
			"revision = %d, want 1",
			first.Revision,
		)
	}

	requireEventually(t, func() bool {
		return subscriber.LastSeenRevision() == 1 &&
			store.HasKey("key-revision-1")
	})

	// -------------------------------------------------------------
	// Subscriber reconnects from durable cursor 1.
	// -------------------------------------------------------------

	var secondAfter uint64

	select {
	case secondAfter = <-server.secondWatchRequest:

	case <-time.After(time.Second):
		t.Fatal("subscriber did not perform first reconnect")
	}

	if secondAfter != 1 {
		t.Fatalf(
			"first reconnect after_revision = %d, want 1",
			secondAfter,
		)
	}

	// -------------------------------------------------------------
	// While reconnect is blocked, control-plane advances.
	// -------------------------------------------------------------

	second, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		"token-revision-2",
	)
	if err != nil {
		t.Fatalf("revision 2 revoke: %v", err)
	}
	if !created {
		t.Fatal("revision 2 did not create event")
	}

	third, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_ENVELOPE,
		"envelope-revision-3",
	)
	if err != nil {
		t.Fatalf("revision 3 revoke: %v", err)
	}
	if !created {
		t.Fatal("revision 3 did not create event")
	}

	if second.Revision != 2 {
		t.Fatalf(
			"second revision = %d, want 2",
			second.Revision,
		)
	}

	if third.Revision != 3 {
		t.Fatalf(
			"third revision = %d, want 3",
			third.Revision,
		)
	}

	// -------------------------------------------------------------
	// Make replay from revision 1 impossible.
	//
	// Snapshot still contains all distributed revocation state.
	// Replay history now begins at revision 3.
	// -------------------------------------------------------------

	journal.CompactThrough(2)

	if got := journal.OldestAvailableRevision(); got != 3 {
		t.Fatalf(
			"oldest available revision = %d, want 3",
			got,
		)
	}

	// Let Watch(after_revision=1) continue.
	//
	// Normal server logic must now emit ResetRequired.
	close(server.releaseSecond)

	// -------------------------------------------------------------
	// Subscriber receives ResetRequired and executes:
	//
	// GetRevocationSnapshot
	// → ReplaceRevocations
	// → last_seen_revision = 3
	// → Watch(after_revision=3)
	// -------------------------------------------------------------

	var thirdAfter uint64

	select {
	case thirdAfter = <-server.thirdWatchRequest:

	case <-time.After(time.Second):
		t.Fatal(
			"subscriber did not resume after snapshot recovery",
		)
	}

	if thirdAfter != 3 {
		t.Fatalf(
			"post-snapshot after_revision = %d, want 3",
			thirdAfter,
		)
	}

	if subscriber.LastSeenRevision() != 3 {
		t.Fatalf(
			"last_seen_revision = %d after snapshot, want 3",
			subscriber.LastSeenRevision(),
		)
	}

	// -------------------------------------------------------------
	// Snapshot must contain the COMPLETE state at revision 3.
	// -------------------------------------------------------------

	if !store.HasKey("key-revision-1") {
		t.Fatal(
			"snapshot recovery lost revision 1 key revoke",
		)
	}

	if !store.HasToken("token-revision-2") {
		t.Fatal(
			"snapshot recovery did not materialize revision 2 token revoke",
		)
	}

	if !store.HasEnvelope("envelope-revision-3") {
		t.Fatal(
			"snapshot recovery did not materialize revision 3 envelope revoke",
		)
	}

	// -------------------------------------------------------------
	// Revision 4 proves that snapshot recovery resumed normal live
	// streaming.
	// -------------------------------------------------------------

	fourth, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		"token-revision-4",
	)
	if err != nil {
		t.Fatalf("revision 4 revoke: %v", err)
	}
	if !created {
		t.Fatal("revision 4 did not create event")
	}

	if fourth.Revision != 4 {
		t.Fatalf(
			"fourth revision = %d, want 4",
			fourth.Revision,
		)
	}

	requireEventually(t, func() bool {
		return subscriber.LastSeenRevision() == 4 &&
			store.HasToken("token-revision-4")
	})

	// -------------------------------------------------------------
	// Final convergence assertions.
	// -------------------------------------------------------------

	if !store.HasKey("key-revision-1") {
		t.Fatal("revision 1 state missing at final convergence")
	}

	if !store.HasToken("token-revision-2") {
		t.Fatal("revision 2 state missing at final convergence")
	}

	if !store.HasEnvelope("envelope-revision-3") {
		t.Fatal("revision 3 state missing at final convergence")
	}

	if !store.HasToken("token-revision-4") {
		t.Fatal("revision 4 state missing at final convergence")
	}

	cancel()

	select {
	case <-errCh:

	case <-time.After(time.Second):
		t.Fatal(
			"subscriber did not stop after context cancellation",
		)
	}
}
