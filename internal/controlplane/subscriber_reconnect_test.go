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

type reconnectTestServer struct {
	controlplanev1.UnimplementedControlPlaneServer

	journal *Journal

	watchCalls atomic.Int32

	firstWatchStarted  chan struct{}
	secondWatchRequest chan uint64
	releaseSecond      chan struct{}
}

func newReconnectTestServer(journal *Journal) *reconnectTestServer {
	return &reconnectTestServer{
		journal:            journal,
		firstWatchStarted:  make(chan struct{}),
		secondWatchRequest: make(chan uint64, 1),
		releaseSecond:      make(chan struct{}),
	}
}

func (s *reconnectTestServer) GetRevocationSnapshot(
	ctx context.Context,
	req *controlplanev1.GetRevocationSnapshotRequest,
) (*controlplanev1.RevocationSnapshot, error) {
	return NewServer(s.journal).GetRevocationSnapshot(ctx, req)
}

func (s *reconnectTestServer) WatchRevocations(
	req *controlplanev1.WatchRevocationsRequest,
	stream grpc.ServerStreamingServer[controlplanev1.WatchRevocationsResponse],
) error {
	call := s.watchCalls.Add(1)

	switch call {
	case 1:
		// First connection starts from revision 0.
		close(s.firstWatchStarted)

		lastRevision := req.GetAfterRevision()

		for {
			events, notify, err := s.journal.EventsAfter(lastRevision)
			if err != nil {
				return err
			}

			if len(events) > 0 {
				event := events[0]

				// IMPORTANT:
				// Send succeeds normally.
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

				// Only AFTER successful delivery do we terminate the stream.
				return status.Error(
					codes.Unavailable,
					"forced disconnect after first delivered event",
				)
			}

			select {
			case <-stream.Context().Done():
				return stream.Context().Err()

			case <-notify:
			}
		}

	case 2:
		// Capture the durable recovery cursor used by Subscriber.
		s.secondWatchRequest <- req.GetAfterRevision()

		// Prevent the second stream from reading the journal yet.
		// The test will commit revision 2 while we are blocked here.
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()

		case <-s.releaseSecond:
		}

		// Normal server semantics now replay everything > after_revision.
		return NewServer(s.journal).WatchRevocations(req, stream)

	default:
		return NewServer(s.journal).WatchRevocations(req, stream)
	}
}

func TestSubscriberReconnectReplaysMissedRevocation(t *testing.T) {
	journal := NewJournal()
	server := newReconnectTestServer(journal)

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
		"sidecar-reconnect-test",
	)

	subscriber.reconnectInitialBackoff = time.Millisecond
	subscriber.reconnectMaxBackoff = time.Millisecond

	errCh := make(chan error, 1)

	go func() {
		errCh <- subscriber.Run(ctx)
	}()

	// -------------------------------------------------------------
	// First Watch(after_revision=0) is established.
	// -------------------------------------------------------------

	select {
	case <-server.firstWatchStarted:
	case <-time.After(time.Second):
		t.Fatal("first WatchRevocations did not start")
	}

	// -------------------------------------------------------------
	// Commit revision 1.
	//
	// First stream delivers it successfully and then deliberately dies.
	// -------------------------------------------------------------

	first, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_KEY,
		"key-before-disconnect",
	)
	if err != nil {
		t.Fatalf("first revoke: %v", err)
	}
	if !created {
		t.Fatal("first revoke did not create revision")
	}

	requireEventually(t, func() bool {
		return subscriber.LastSeenRevision() == 1 &&
			store.HasKey("key-before-disconnect")
	})

	if first.Revision != 1 {
		t.Fatalf(
			"first revision = %d, want 1",
			first.Revision,
		)
	}

	// -------------------------------------------------------------
	// Subscriber must reconnect using revision 1.
	// -------------------------------------------------------------

	var afterRevision uint64

	select {
	case afterRevision = <-server.secondWatchRequest:
	case <-time.After(time.Second):
		t.Fatal("subscriber did not reconnect")
	}

	if afterRevision != 1 {
		t.Fatalf(
			"reconnect after_revision = %d, want 1",
			afterRevision,
		)
	}

	// -------------------------------------------------------------
	// Commit revision 2 WHILE the second Watch is connected but
	// deliberately blocked from reading the journal.
	//
	// Therefore revision 2 is definitely missed by the old stream and
	// must be recovered from journal history using after_revision=1.
	// -------------------------------------------------------------

	second, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		"token-during-disconnect",
	)
	if err != nil {
		t.Fatalf("second revoke: %v", err)
	}
	if !created {
		t.Fatal("second revoke did not create revision")
	}

	if second.Revision != 2 {
		t.Fatalf(
			"second revision = %d, want 2",
			second.Revision,
		)
	}

	// Allow second Watch to begin journal processing.
	close(server.releaseSecond)

	// Revision 2 must now be replayed.
	requireEventually(t, func() bool {
		return subscriber.LastSeenRevision() == 2 &&
			store.HasToken("token-during-disconnect")
	})

	// Revision 1 must remain materialized.
	if !store.HasKey("key-before-disconnect") {
		t.Fatal(
			"revision 1 state disappeared after reconnect recovery",
		)
	}

	// -------------------------------------------------------------
	// Revision 3 proves the recovered stream remains live.
	// -------------------------------------------------------------

	third, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_ENVELOPE,
		"envelope-after-reconnect",
	)
	if err != nil {
		t.Fatalf("third revoke: %v", err)
	}
	if !created {
		t.Fatal("third revoke did not create revision")
	}

	if third.Revision != 3 {
		t.Fatalf(
			"third revision = %d, want 3",
			third.Revision,
		)
	}

	requireEventually(t, func() bool {
		return subscriber.LastSeenRevision() == 3 &&
			store.HasEnvelope("envelope-after-reconnect")
	})

	cancel()

	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal(
			"subscriber did not stop after context cancellation",
		)
	}
}
