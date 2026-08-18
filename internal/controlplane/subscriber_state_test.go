package controlplane

import (
	"testing"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"
)

func TestSubscriberEventUpdatesControlState(t *testing.T) {
	store := newTestRevocationStore()

	subscriber := &Subscriber{
		store:            store,
		state:            NewControlState(5 * time.Second),
		sidecarID:        "state-event-test",
		lastSeenRevision: 5,
	}

	subscriber.state.MarkSnapshot(
		5,
		time.Now(),
	)

	event := &controlplanev1.RevocationEvent{
		Revision:  6,
		EventId:   "event-6",
		Kind:      controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		SubjectId: "token-6",
	}

	if err := subscriber.applyEvent(event); err != nil {
		t.Fatalf("applyEvent: %v", err)
	}

	if got := subscriber.LastSeenRevision(); got != 6 {
		t.Fatalf(
			"subscriber revision = %d, want 6",
			got,
		)
	}

	if got := subscriber.state.LastSeenRevision(); got != 6 {
		t.Fatalf(
			"control-state revision = %d, want 6",
			got,
		)
	}

	if !subscriber.state.IsFresh(time.Now()) {
		t.Fatal("event did not establish fresh control state")
	}
}

func TestSubscriberRevisionGapMarksControlStateUnsafe(t *testing.T) {
	store := newTestRevocationStore()

	state := NewControlState(5 * time.Second)
	state.MarkSnapshot(5, time.Now())

	subscriber := &Subscriber{
		store:            store,
		state:            state,
		sidecarID:        "state-gap-test",
		lastSeenRevision: 5,
	}

	event := &controlplanev1.RevocationEvent{
		Revision:  7,
		EventId:   "gap-7",
		Kind:      controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		SubjectId: "token-gap",
	}

	err := subscriber.applyEvent(event)
	if err == nil {
		t.Fatal("revision gap unexpectedly succeeded")
	}

	if !state.Unsafe() {
		t.Fatal("revision gap did not mark control state unsafe")
	}

	if state.IsFresh(time.Now()) {
		t.Fatal("unsafe state unexpectedly remained fresh")
	}

	if got := state.LastSeenRevision(); got != 5 {
		t.Fatalf(
			"control-state revision = %d, want 5",
			got,
		)
	}
}

func TestSubscriberUnknownKindMarksControlStateUnsafe(t *testing.T) {
	store := newTestRevocationStore()

	state := NewControlState(5 * time.Second)
	state.MarkSnapshot(10, time.Now())

	subscriber := &Subscriber{
		store:            store,
		state:            state,
		sidecarID:        "state-kind-test",
		lastSeenRevision: 10,
	}

	event := &controlplanev1.RevocationEvent{
		Revision:  11,
		EventId:   "unknown-11",
		Kind:      controlplanev1.RevocationKind_REVOCATION_KIND_UNSPECIFIED,
		SubjectId: "unknown",
	}

	err := subscriber.applyEvent(event)
	if err == nil {
		t.Fatal("unknown kind unexpectedly succeeded")
	}

	if !state.Unsafe() {
		t.Fatal("unknown kind did not mark control state unsafe")
	}

	if got := subscriber.LastSeenRevision(); got != 10 {
		t.Fatalf(
			"subscriber revision = %d, want 10",
			got,
		)
	}
}
