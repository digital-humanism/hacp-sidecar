package controlplane

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type testRevocationStore struct {
	mu sync.RWMutex

	keys      map[string]struct{}
	tokens    map[string]struct{}
	envelopes map[string]struct{}
}

func newTestRevocationStore() *testRevocationStore {
	return &testRevocationStore{
		keys:      make(map[string]struct{}),
		tokens:    make(map[string]struct{}),
		envelopes: make(map[string]struct{}),
	}
}

func (s *testRevocationStore) RevokeKey(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[id] = struct{}{}
}

func (s *testRevocationStore) RevokeToken(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[id] = struct{}{}
}

func (s *testRevocationStore) RevokeEnvelope(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.envelopes[id] = struct{}{}
}

func (s *testRevocationStore) ReplaceRevocations(
	keys []string,
	tokens []string,
	envelopes []string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys = make(map[string]struct{})
	s.tokens = make(map[string]struct{})
	s.envelopes = make(map[string]struct{})

	for _, id := range keys {
		s.keys[id] = struct{}{}
	}

	for _, id := range tokens {
		s.tokens[id] = struct{}{}
	}

	for _, id := range envelopes {
		s.envelopes[id] = struct{}{}
	}

	return nil
}

func (s *testRevocationStore) HasKey(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.keys[id]
	return ok
}

func (s *testRevocationStore) HasToken(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.tokens[id]
	return ok
}

func TestSubscriberSnapshotAndLiveRevocation(t *testing.T) {
	journal := NewJournal()

	// Snapshot starts at revision 1.
	_, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_KEY,
		"key-snapshot",
	)
	if err != nil {
		t.Fatalf("Revoke(snapshot): %v", err)
	}
	if !created {
		t.Fatal("snapshot revoke did not create revision")
	}

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
		"sidecar-test-01",
	)

	errCh := make(chan error, 1)

	go func() {
		errCh <- subscriber.Run(ctx)
	}()

	// Snapshot must converge first.
	requireEventually(t, func() bool {
		return subscriber.LastSeenRevision() == 1 &&
			store.HasKey("key-snapshot")
	})

	// Commit revision 2 while subscriber is watching.
	event, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		"token-live",
	)
	if err != nil {
		t.Fatalf("Revoke(live): %v", err)
	}
	if !created {
		t.Fatal("live revoke did not create revision")
	}

	if event.Revision != 2 {
		t.Fatalf(
			"event revision = %d, want 2",
			event.Revision,
		)
	}

	// The stream must materialize revision 2 into the local store.
	requireEventually(t, func() bool {
		return subscriber.LastSeenRevision() == 2 &&
			store.HasToken("token-live")
	})

	cancel()

	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("subscriber did not stop after context cancellation")
	}
}

func requireEventually(
	t *testing.T,
	condition func() bool,
) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("condition was not satisfied before deadline")
}

func (s *testRevocationStore) HasEnvelope(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.envelopes[id]
	return ok
}
