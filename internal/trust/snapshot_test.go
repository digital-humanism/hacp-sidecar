package trust

import (
	"crypto/ed25519"
	"errors"
	"testing"
	"time"
)

func testPublicKey(fill byte) ed25519.PublicKey {
	key := make(ed25519.PublicKey, ed25519.PublicKeySize)
	for i := range key {
		key[i] = fill
	}
	return key
}

func TestNewSnapshotValid(t *testing.T) {
	loadedAt := time.Unix(123, 0).UTC()
	snapshot, err := NewSnapshot(
		7,
		loadedAt,
		"test",
		[]KeyBinding{
			{KeyID: "key-a", PublicKey: testPublicKey(1)},
			{KeyID: "key-b", PublicKey: testPublicKey(2)},
		},
	)
	if err != nil {
		t.Fatalf("NewSnapshot() error = %v", err)
	}
	if snapshot.Revision() != 7 {
		t.Fatalf("Revision() = %d, want 7", snapshot.Revision())
	}
	if snapshot.LoadedAt() != loadedAt {
		t.Fatalf("LoadedAt() = %v, want %v", snapshot.LoadedAt(), loadedAt)
	}
	if snapshot.Source() != "test" {
		t.Fatalf("Source() = %q, want test", snapshot.Source())
	}
	if snapshot.KeyCount() != 2 {
		t.Fatalf("KeyCount() = %d, want 2", snapshot.KeyCount())
	}
	if snapshot.Fingerprint() == "" {
		t.Fatal("Fingerprint() is empty")
	}
}

func TestNewSnapshotRejectsEmpty(t *testing.T) {
	_, err := NewSnapshot(1, time.Time{}, "", nil)
	if !errors.Is(err, ErrEmptyTrustSnapshot) {
		t.Fatalf("error = %v, want ErrEmptyTrustSnapshot", err)
	}
}

func TestNewSnapshotRejectsEmptyKeyID(t *testing.T) {
	_, err := NewSnapshot(
		1,
		time.Time{},
		"",
		[]KeyBinding{{KeyID: "", PublicKey: testPublicKey(1)}},
	)
	if !errors.Is(err, ErrInvalidKeyID) {
		t.Fatalf("error = %v, want ErrInvalidKeyID", err)
	}
}

func TestNewSnapshotRejectsInvalidPublicKeyLength(t *testing.T) {
	_, err := NewSnapshot(
		1,
		time.Time{},
		"",
		[]KeyBinding{{KeyID: "key-a", PublicKey: ed25519.PublicKey{1, 2, 3}}},
	)
	if !errors.Is(err, ErrInvalidPublicKey) {
		t.Fatalf("error = %v, want ErrInvalidPublicKey", err)
	}
}

func TestNewSnapshotNormalizesIdenticalDuplicate(t *testing.T) {
	key := testPublicKey(1)
	snapshot, err := NewSnapshot(
		1,
		time.Time{},
		"",
		[]KeyBinding{
			{KeyID: "key-a", PublicKey: key},
			{KeyID: "key-a", PublicKey: append(ed25519.PublicKey(nil), key...)},
		},
	)
	if err != nil {
		t.Fatalf("NewSnapshot() error = %v", err)
	}
	if snapshot.KeyCount() != 1 {
		t.Fatalf("KeyCount() = %d, want 1", snapshot.KeyCount())
	}
}

func TestNewSnapshotRejectsConflictingDuplicate(t *testing.T) {
	_, err := NewSnapshot(
		1,
		time.Time{},
		"",
		[]KeyBinding{
			{KeyID: "key-a", PublicKey: testPublicKey(1)},
			{KeyID: "key-a", PublicKey: testPublicKey(2)},
		},
	)
	if !errors.Is(err, ErrConflictingKeyBinding) {
		t.Fatalf("error = %v, want ErrConflictingKeyBinding", err)
	}
}

func TestSnapshotFingerprintIndependentOfBindingOrder(t *testing.T) {
	a, err := NewSnapshot(
		1,
		time.Time{},
		"",
		[]KeyBinding{
			{KeyID: "key-a", PublicKey: testPublicKey(1)},
			{KeyID: "key-b", PublicKey: testPublicKey(2)},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewSnapshot(
		1,
		time.Time{},
		"",
		[]KeyBinding{
			{KeyID: "key-b", PublicKey: testPublicKey(2)},
			{KeyID: "key-a", PublicKey: testPublicKey(1)},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatalf("fingerprints differ: %s != %s", a.Fingerprint(), b.Fingerprint())
	}
}
