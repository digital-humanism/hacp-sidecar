package controlplane

import (
	"errors"
	"testing"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"
)

func TestSubscriberAppliesNextRevision(t *testing.T) {
	store := newTestRevocationStore()

	subscriber := &Subscriber{
		store:            store,
		sidecarID:        "sidecar-revision-test",
		lastSeenRevision: 5,
	}

	event := &controlplanev1.RevocationEvent{
		Revision:  6,
		EventId:   "revocation-00000000000000000006",
		Kind:      controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		SubjectId: "token-006",
	}

	if err := subscriber.applyEvent(event); err != nil {
		t.Fatalf("applyEvent: %v", err)
	}

	if got := subscriber.LastSeenRevision(); got != 6 {
		t.Fatalf(
			"last_seen_revision = %d, want 6",
			got,
		)
	}

	if !store.HasToken("token-006") {
		t.Fatal("token revocation was not applied to local store")
	}
}

func TestSubscriberDuplicateEventIsIgnored(t *testing.T) {
	store := newTestRevocationStore()

	subscriber := &Subscriber{
		store:            store,
		sidecarID:        "sidecar-duplicate-test",
		lastSeenRevision: 6,
	}

	// Same revision as the already materialized state.
	event := &controlplanev1.RevocationEvent{
		Revision:  6,
		EventId:   "duplicate-revision-6",
		Kind:      controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		SubjectId: "token-duplicate",
	}

	if err := subscriber.applyEvent(event); err != nil {
		t.Fatalf("duplicate applyEvent returned error: %v", err)
	}

	if got := subscriber.LastSeenRevision(); got != 6 {
		t.Fatalf(
			"last_seen_revision = %d, want 6",
			got,
		)
	}

	// A duplicate/old event must not mutate local state.
	if store.HasToken("token-duplicate") {
		t.Fatal("duplicate event unexpectedly mutated local revocation state")
	}
}

func TestSubscriberOlderOutOfOrderEventIsIgnored(t *testing.T) {
	store := newTestRevocationStore()

	subscriber := &Subscriber{
		store:            store,
		sidecarID:        "sidecar-old-event-test",
		lastSeenRevision: 10,
	}

	event := &controlplanev1.RevocationEvent{
		Revision:  8,
		EventId:   "old-revision-8",
		Kind:      controlplanev1.RevocationKind_REVOCATION_KIND_KEY,
		SubjectId: "key-old",
	}

	if err := subscriber.applyEvent(event); err != nil {
		t.Fatalf("old applyEvent returned error: %v", err)
	}

	if got := subscriber.LastSeenRevision(); got != 10 {
		t.Fatalf(
			"last_seen_revision = %d, want 10",
			got,
		)
	}

	if store.HasKey("key-old") {
		t.Fatal("old out-of-order event unexpectedly mutated local state")
	}
}

func TestSubscriberRevisionGapFailsClosed(t *testing.T) {
	store := newTestRevocationStore()

	subscriber := &Subscriber{
		store:            store,
		sidecarID:        "sidecar-gap-test",
		lastSeenRevision: 5,
	}

	// Revision 6 is missing.
	event := &controlplanev1.RevocationEvent{
		Revision:  7,
		EventId:   "revision-gap-7",
		Kind:      controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		SubjectId: "token-gap",
	}

	err := subscriber.applyEvent(event)
	if err == nil {
		t.Fatal("revision gap unexpectedly succeeded")
	}

	if !errors.Is(err, ErrRevisionGap) {
		t.Fatalf(
			"error = %v, want ErrRevisionGap",
			err,
		)
	}

	// The cursor MUST remain at the last completely materialized revision.
	if got := subscriber.LastSeenRevision(); got != 5 {
		t.Fatalf(
			"last_seen_revision = %d after gap, want 5",
			got,
		)
	}

	// Critically, revision 7 must not be partially applied.
	if store.HasToken("token-gap") {
		t.Fatal("gap event unexpectedly mutated local revocation state")
	}
}

func TestSubscriberUnknownRevocationKindFailsWithoutAdvancingRevision(
	t *testing.T,
) {
	store := newTestRevocationStore()

	subscriber := &Subscriber{
		store:            store,
		sidecarID:        "sidecar-unknown-kind-test",
		lastSeenRevision: 20,
	}

	event := &controlplanev1.RevocationEvent{
		Revision:  21,
		EventId:   "unknown-kind-21",
		Kind:      controlplanev1.RevocationKind_REVOCATION_KIND_UNSPECIFIED,
		SubjectId: "unknown-subject",
	}

	err := subscriber.applyEvent(event)
	if err == nil {
		t.Fatal("unknown revocation kind unexpectedly succeeded")
	}

	if !errors.Is(err, ErrUnknownRevocationKind) {
		t.Fatalf(
			"error = %v, want ErrUnknownRevocationKind",
			err,
		)
	}

	// Revision advances only after valid local materialization.
	if got := subscriber.LastSeenRevision(); got != 20 {
		t.Fatalf(
			"last_seen_revision = %d, want 20",
			got,
		)
	}
}

func TestSubscriberAllSupportedRevocationKinds(t *testing.T) {
	tests := []struct {
		name string
		kind controlplanev1.RevocationKind
		id   string
		want func(*testRevocationStore, string) bool
	}{
		{
			name: "key",
			kind: controlplanev1.RevocationKind_REVOCATION_KIND_KEY,
			id:   "key-001",
			want: func(store *testRevocationStore, id string) bool {
				return store.HasKey(id)
			},
		},
		{
			name: "token",
			kind: controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
			id:   "token-001",
			want: func(store *testRevocationStore, id string) bool {
				return store.HasToken(id)
			},
		},
		{
			name: "envelope",
			kind: controlplanev1.RevocationKind_REVOCATION_KIND_ENVELOPE,
			id:   "envelope-001",
			want: func(store *testRevocationStore, id string) bool {
				return store.HasEnvelope(id)
			},
		},
		{
			name: "parent envelope",
			kind: controlplanev1.RevocationKind_REVOCATION_KIND_PARENT_ENVELOPE,
			id:   "parent-envelope-001",
			want: func(store *testRevocationStore, id string) bool {
				return store.HasEnvelope(id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestRevocationStore()

			subscriber := &Subscriber{
				store:            store,
				sidecarID:        "sidecar-kind-test",
				lastSeenRevision: 100,
			}

			event := &controlplanev1.RevocationEvent{
				Revision:  101,
				EventId:   "revision-101",
				Kind:      tt.kind,
				SubjectId: tt.id,
			}

			if err := subscriber.applyEvent(event); err != nil {
				t.Fatalf("applyEvent: %v", err)
			}

			if got := subscriber.LastSeenRevision(); got != 101 {
				t.Fatalf(
					"last_seen_revision = %d, want 101",
					got,
				)
			}

			if !tt.want(store, tt.id) {
				t.Fatalf(
					"%s revocation %q was not materialized",
					tt.name,
					tt.id,
				)
			}
		})
	}
}
