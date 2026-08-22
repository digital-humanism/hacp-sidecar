package trust

import (
	"testing"
	"time"
)

func TestAtomicTrustStoreObservability(t *testing.T) {
	loadedAt := time.Unix(1234, 0).UTC()
	snapshot, err := NewSnapshot(
		11,
		loadedAt,
		"file:test.json",
		[]KeyBinding{{KeyID: "key-a", PublicKey: testPublicKey(1)}},
	)
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewAtomicTrustStore(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	revision, err := store.Revision()
	if err != nil {
		t.Fatal(err)
	}
	if revision != 11 {
		t.Fatalf("Revision() = %d, want 11", revision)
	}

	loaded, err := store.LoadedAt()
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Equal(loadedAt) {
		t.Fatalf("LoadedAt() = %v, want %v", loaded, loadedAt)
	}

	source, err := store.Source()
	if err != nil {
		t.Fatal(err)
	}
	if source != "file:test.json" {
		t.Fatalf("Source() = %q, want file:test.json", source)
	}

	keyCount, err := store.KeyCount()
	if err != nil {
		t.Fatal(err)
	}
	if keyCount != 1 {
		t.Fatalf("KeyCount() = %d, want 1", keyCount)
	}

	fingerprint, err := store.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint == "" {
		t.Fatal("Fingerprint() is empty")
	}
}
