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

func TestDistributedTokenRevocationPropagation(t *testing.T) {
	// -----------------------------------------------------------------
	// Control-plane
	// -----------------------------------------------------------------

	journal := NewJournal()

	listener := bufconn.Listen(testBufferSize)

	grpcServer := grpc.NewServer()

	controlplanev1.RegisterControlPlaneServer(
		grpcServer,
		NewServer(journal),
	)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	// -----------------------------------------------------------------
	// Sidecar gRPC client
	// -----------------------------------------------------------------

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

	// -----------------------------------------------------------------
	// REAL evaluator revocation store
	// -----------------------------------------------------------------

	store := NewRevocationStoreAdapter()

	const tokenID = "token-distributed-001"

	// Before distributed revocation this authority is not revoked.
	//
	// At the Pipeline token-revocation gate this corresponds to:
	//
	//     not revoked -> evaluation may continue
	//
	if store.IsTokenRevoked(tokenID) {
		t.Fatal("token unexpectedly revoked before control-plane event")
	}

	// -----------------------------------------------------------------
	// Subscriber
	// -----------------------------------------------------------------

	subscriber := NewSubscriber(
		controlplanev1.NewControlPlaneClient(conn),
		store,
		"sidecar-distributed-test-01",
	)

	errCh := make(chan error, 1)

	go func() {
		errCh <- subscriber.Run(ctx)
	}()

	// Wait until initial revision-0 snapshot is installed and stream is up.
	//
	// A small synchronization revoke is deliberately avoided here because
	// that would change the revision/state being tested.
	time.Sleep(20 * time.Millisecond)

	// -----------------------------------------------------------------
	// Distributed revoke
	// -----------------------------------------------------------------

	event, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		tokenID,
	)
	if err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	if !created {
		t.Fatal("distributed revoke did not create revision")
	}

	if event.Revision != 1 {
		t.Fatalf(
			"event revision = %d, want 1",
			event.Revision,
		)
	}

	// -----------------------------------------------------------------
	// Convergence
	// -----------------------------------------------------------------

	requireEventually(t, func() bool {
		return subscriber.LastSeenRevision() == event.Revision &&
			store.IsTokenRevoked(tokenID)
	})

	// The real evaluate.InMemoryRevocationStore is now updated through:
	//
	//	control-plane
	//	    ↓
	//	gRPC
	//	    ↓
	//	Subscriber
	//	    ↓
	//	RevocationStoreAdapter
	//	    ↓
	//	evaluate.InMemoryRevocationStore
	//
	if !store.IsTokenRevoked(tokenID) {
		t.Fatal("token was not materialized into evaluator revocation store")
	}

	cancel()

	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("subscriber did not stop after context cancellation")
	}
}
