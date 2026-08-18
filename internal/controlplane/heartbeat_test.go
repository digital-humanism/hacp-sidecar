package controlplane

import (
	"context"
	"net"
	"testing"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestWatchRevocationsDeliversHeartbeat(t *testing.T) {
	journal := NewJournal()

	// Establish revision 1 before the stream starts.
	event, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_KEY,
		"heartbeat-key",
	)
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if !created {
		t.Fatal("revoke did not create revision")
	}

	if event.Revision != 1 {
		t.Fatalf(
			"revision = %d, want 1",
			event.Revision,
		)
	}

	listener := bufconn.Listen(testBufferSize)

	grpcServer := grpc.NewServer()

	controlplanev1.RegisterControlPlaneServer(
		grpcServer,
		NewServerWithHeartbeat(
			journal,
			5*time.Millisecond,
		),
	)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
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

	client := controlplanev1.NewControlPlaneClient(conn)

	stream, err := client.WatchRevocations(
		ctx,
		&controlplanev1.WatchRevocationsRequest{
			SidecarId:     "heartbeat-test-sidecar",
			AfterRevision: 0,
		},
	)
	if err != nil {
		t.Fatalf("WatchRevocations: %v", err)
	}

	// -------------------------------------------------------------
	// First response must be revision 1.
	// -------------------------------------------------------------

	first, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv(event): %v", err)
	}

	firstEvent := first.GetEvent()
	if firstEvent == nil {
		t.Fatal("first response is not an event")
	}

	if firstEvent.Revision != 1 {
		t.Fatalf(
			"event revision = %d, want 1",
			firstEvent.Revision,
		)
	}

	if first.Sequence != 1 {
		t.Fatalf(
			"event sequence = %d, want 1",
			first.Sequence,
		)
	}

	// -------------------------------------------------------------
	// Once caught up, next response should be heartbeat.
	// -------------------------------------------------------------

	second, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv(heartbeat): %v", err)
	}

	heartbeat := second.GetHeartbeat()
	if heartbeat == nil {
		t.Fatal("second response is not a heartbeat")
	}

	if heartbeat.CurrentRevision != 1 {
		t.Fatalf(
			"heartbeat current_revision = %d, want 1",
			heartbeat.CurrentRevision,
		)
	}

	if heartbeat.ServerTimeMs <= 0 {
		t.Fatalf(
			"heartbeat server_time_ms = %d, want > 0",
			heartbeat.ServerTimeMs,
		)
	}

	if second.Sequence != 2 {
		t.Fatalf(
			"heartbeat sequence = %d, want 2",
			second.Sequence,
		)
	}
}

func TestSubscriberHeartbeatRefreshesControlState(t *testing.T) {
	journal := NewJournal()

	listener := bufconn.Listen(testBufferSize)

	grpcServer := grpc.NewServer()

	controlplanev1.RegisterControlPlaneServer(
		grpcServer,
		NewServerWithHeartbeat(
			journal,
			5*time.Millisecond,
		),
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

	state := NewControlState(
		100 * time.Millisecond,
	)

	subscriber := NewSubscriber(
		controlplanev1.NewControlPlaneClient(conn),
		store,
		"subscriber-heartbeat-test",
	)

	subscriber.SetControlState(state)

	errCh := make(chan error, 1)

	go func() {
		errCh <- subscriber.Run(ctx)
	}()

	// Initial snapshot revision 0 must establish freshness.
	requireEventually(t, func() bool {
		return !state.LastUpdate().IsZero()
	})

	initialUpdate := state.LastUpdate()

	// A real heartbeat over gRPC must advance lastUpdate without changing
	// lastSeenRevision.
	requireEventually(t, func() bool {
		return state.LastUpdate().After(initialUpdate)
	})

	if state.LastSeenRevision() != 0 {
		t.Fatalf(
			"heartbeat changed revision to %d, want 0",
			state.LastSeenRevision(),
		)
	}

	if !state.IsFresh(time.Now()) {
		t.Fatal(
			"control state is stale after valid heartbeat",
		)
	}

	if !state.Connected() {
		t.Fatal(
			"subscriber control state is not marked connected",
		)
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

type invalidHeartbeatServer struct {
	controlplanev1.UnimplementedControlPlaneServer
}

func (s *invalidHeartbeatServer) GetRevocationSnapshot(
	context.Context,
	*controlplanev1.GetRevocationSnapshotRequest,
) (*controlplanev1.RevocationSnapshot, error) {
	return &controlplanev1.RevocationSnapshot{
		Revision:      0,
		GeneratedAtMs: time.Now().UnixMilli(),
	}, nil
}

func (s *invalidHeartbeatServer) WatchRevocations(
	_ *controlplanev1.WatchRevocationsRequest,
	stream grpc.ServerStreamingServer[controlplanev1.WatchRevocationsResponse],
) error {
	if err := stream.Send(
		&controlplanev1.WatchRevocationsResponse{
			Sequence: 1,
			Payload: &controlplanev1.WatchRevocationsResponse_Heartbeat{
				Heartbeat: &controlplanev1.Heartbeat{
					// Local snapshot is revision 0.
					// Claiming revision 1 without delivering event 1 is invalid.
					CurrentRevision: 1,
					ServerTimeMs:    time.Now().UnixMilli(),
				},
			},
		},
	); err != nil {
		return err
	}

	<-stream.Context().Done()
	return stream.Context().Err()
}

func TestSubscriberInvalidHeartbeatMarksControlStateUnsafe(
	t *testing.T,
) {
	listener := bufconn.Listen(testBufferSize)

	grpcServer := grpc.NewServer()

	controlplanev1.RegisterControlPlaneServer(
		grpcServer,
		&invalidHeartbeatServer{},
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
	state := NewControlState(5 * time.Second)

	subscriber := NewSubscriber(
		controlplanev1.NewControlPlaneClient(conn),
		store,
		"invalid-heartbeat-sidecar",
	)

	subscriber.SetControlState(state)

	// Keep retries cheap; the assertion concerns the first invalid heartbeat.
	subscriber.reconnectInitialBackoff = time.Millisecond
	subscriber.reconnectMaxBackoff = time.Millisecond

	go func() {
		_ = subscriber.Run(ctx)
	}()

	requireEventually(t, func() bool {
		return state.Unsafe()
	})

	if state.IsFresh(time.Now()) {
		t.Fatal(
			"invalid heartbeat left control state fresh",
		)
	}

	if state.LastSeenRevision() != 0 {
		t.Fatalf(
			"invalid heartbeat changed revision to %d, want 0",
			state.LastSeenRevision(),
		)
	}

	cancel()
}
