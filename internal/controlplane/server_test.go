package controlplane

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const testBufferSize = 1024 * 1024

func TestControlPlaneReplayAndLiveDelivery(t *testing.T) {
	fixedTime := time.UnixMilli(1_786_000_000_000)

	journal := newJournalWithClock(func() time.Time {
		return fixedTime
	})

	// Revision 1 exists before the sidecar subscribes.
	first, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_KEY,
		"key-001",
	)
	if err != nil {
		t.Fatalf("Revoke(key): %v", err)
	}

	if !created {
		t.Fatal("expected key revocation to create revision")
	}

	if first.Revision != 1 {
		t.Fatalf(
			"first revision = %d, want 1",
			first.Revision,
		)
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

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
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

	//
	// Snapshot at revision 1.
	//

	snapshot, err := client.GetRevocationSnapshot(
		ctx,
		&controlplanev1.GetRevocationSnapshotRequest{
			SidecarId: "sidecar-test-01",
		},
	)
	if err != nil {
		t.Fatalf("GetRevocationSnapshot: %v", err)
	}

	if snapshot.Revision != 1 {
		t.Fatalf(
			"snapshot revision = %d, want 1",
			snapshot.Revision,
		)
	}

	if len(snapshot.Entries) != 1 {
		t.Fatalf(
			"snapshot entries = %d, want 1",
			len(snapshot.Entries),
		)
	}

	if snapshot.Entries[0].Kind !=
		controlplanev1.RevocationKind_REVOCATION_KIND_KEY {
		t.Fatalf(
			"snapshot kind = %s, want KEY",
			snapshot.Entries[0].Kind,
		)
	}

	if snapshot.Entries[0].SubjectId != "key-001" {
		t.Fatalf(
			"snapshot subject = %q, want key-001",
			snapshot.Entries[0].SubjectId,
		)
	}

	//
	// Subscribe from revision 0.
	//
	// Revision 1 must therefore be replayed.
	//

	stream, err := client.WatchRevocations(
		ctx,
		&controlplanev1.WatchRevocationsRequest{
			SidecarId:     "sidecar-test-01",
			AfterRevision: 0,
		},
	)
	if err != nil {
		t.Fatalf("WatchRevocations: %v", err)
	}

	replayed, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv(replay): %v", err)
	}

	if replayed.Sequence != 1 {
		t.Fatalf(
			"replay sequence = %d, want 1",
			replayed.Sequence,
		)
	}

	replayedEvent := replayed.GetEvent()
	if replayedEvent == nil {
		t.Fatal("replay response does not contain event")
	}

	if replayedEvent.Revision != 1 {
		t.Fatalf(
			"replay revision = %d, want 1",
			replayedEvent.Revision,
		)
	}

	if replayedEvent.Kind !=
		controlplanev1.RevocationKind_REVOCATION_KIND_KEY {
		t.Fatalf(
			"replay kind = %s, want KEY",
			replayedEvent.Kind,
		)
	}

	if replayedEvent.SubjectId != "key-001" {
		t.Fatalf(
			"replay subject = %q, want key-001",
			replayedEvent.SubjectId,
		)
	}

	//
	// Commit revision 2 while stream is active.
	//

	second, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		"token-001",
	)
	if err != nil {
		t.Fatalf("Revoke(token): %v", err)
	}

	if !created {
		t.Fatal("expected token revocation to create revision")
	}

	if second.Revision != 2 {
		t.Fatalf(
			"second revision = %d, want 2",
			second.Revision,
		)
	}

	//
	// Existing stream must receive revision 2 live.
	//

	live, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv(live): %v", err)
	}

	if live.Sequence != 2 {
		t.Fatalf(
			"live sequence = %d, want 2",
			live.Sequence,
		)
	}

	liveEvent := live.GetEvent()
	if liveEvent == nil {
		t.Fatal("live response does not contain event")
	}

	if liveEvent.Revision != 2 {
		t.Fatalf(
			"live revision = %d, want 2",
			liveEvent.Revision,
		)
	}

	if liveEvent.Kind !=
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN {
		t.Fatalf(
			"live kind = %s, want TOKEN",
			liveEvent.Kind,
		)
	}

	if liveEvent.SubjectId != "token-001" {
		t.Fatalf(
			"live subject = %q, want token-001",
			liveEvent.SubjectId,
		)
	}
}

func TestJournalDuplicateRevocationIsIdempotent(t *testing.T) {
	journal := NewJournal()

	first, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		"token-duplicate",
	)
	if err != nil {
		t.Fatalf("first Revoke: %v", err)
	}

	if !created {
		t.Fatal("first revoke should create revision")
	}

	if first.Revision != 1 {
		t.Fatalf(
			"first revision = %d, want 1",
			first.Revision,
		)
	}

	second, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		"token-duplicate",
	)
	if err != nil {
		t.Fatalf("duplicate Revoke: %v", err)
	}

	if created {
		t.Fatal("duplicate revoke unexpectedly created revision")
	}

	if second != nil {
		t.Fatal("duplicate revoke unexpectedly returned event")
	}

	if journal.Revision() != 1 {
		t.Fatalf(
			"revision after duplicate = %d, want 1",
			journal.Revision(),
		)
	}

	snapshot := journal.Snapshot()

	if len(snapshot.Entries) != 1 {
		t.Fatalf(
			"snapshot entries = %d, want 1",
			len(snapshot.Entries),
		)
	}
}

func TestJournalRejectsRevisionAhead(t *testing.T) {
	journal := NewJournal()

	_, _, err := journal.EventsAfter(1)
	if err == nil {
		t.Fatal("expected revision-ahead error")
	}

	if !errors.Is(err, ErrRevisionAhead) {
		t.Fatalf(
			"error = %v, want ErrRevisionAhead",
			err,
		)
	}
}

func TestJournalReplayUnavailableAfterCompaction(t *testing.T) {
	journal := NewJournal()

	for i := 0; i < 3; i++ {
		_, created, err := journal.Revoke(
			controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
			fmt.Sprintf("token-%d", i+1),
		)
		if err != nil {
			t.Fatalf("Revoke: %v", err)
		}
		if !created {
			t.Fatal("revoke did not create revision")
		}
	}

	// History:
	//
	// 1
	// 2
	// 3
	//
	// Remove replay history through revision 2.
	journal.CompactThrough(2)

	if got := journal.OldestAvailableRevision(); got != 3 {
		t.Fatalf(
			"oldest available revision = %d, want 3",
			got,
		)
	}

	// after_revision=1 would require revision 2,
	// but revision 2 has already been compacted.
	_, _, err := journal.EventsAfter(1)

	if !errors.Is(err, ErrReplayUnavailable) {
		t.Fatalf(
			"error = %v, want ErrReplayUnavailable",
			err,
		)
	}

	// after_revision=2 is still valid:
	// revision 3 remains available.
	events, _, err := journal.EventsAfter(2)
	if err != nil {
		t.Fatalf("EventsAfter(2): %v", err)
	}

	if len(events) != 1 {
		t.Fatalf(
			"events = %d, want 1",
			len(events),
		)
	}

	if events[0].Revision != 3 {
		t.Fatalf(
			"revision = %d, want 3",
			events[0].Revision,
		)
	}
}

func TestWatchRevocationsReturnsResetRequired(t *testing.T) {
	journal := NewJournal()

	for i := 1; i <= 3; i++ {
		_, _, err := journal.Revoke(
			controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
			fmt.Sprintf("token-%d", i),
		)
		if err != nil {
			t.Fatalf("Revoke: %v", err)
		}
	}

	journal.CompactThrough(2)

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
			SidecarId:     "stale-sidecar",
			AfterRevision: 1,
		},
	)
	if err != nil {
		t.Fatalf("WatchRevocations: %v", err)
	}

	response, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	reset := response.GetResetRequired()
	if reset == nil {
		t.Fatal("response does not contain ResetRequired")
	}

	if reset.OldestAvailableRevision != 3 {
		t.Fatalf(
			"oldest_available_revision = %d, want 3",
			reset.OldestAvailableRevision,
		)
	}

	if reset.CurrentRevision != 3 {
		t.Fatalf(
			"current_revision = %d, want 3",
			reset.CurrentRevision,
		)
	}
}
