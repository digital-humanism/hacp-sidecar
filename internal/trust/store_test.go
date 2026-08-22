package trust

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"sync"
	"testing"
	"time"
)

func mustSnapshot(t *testing.T, revision uint64, bindings ...KeyBinding) TrustSnapshot {
	t.Helper()
	snapshot, err := NewSnapshot(revision, time.Unix(int64(revision), 0).UTC(), "test", bindings)
	if err != nil {
		t.Fatalf("NewSnapshot() error = %v", err)
	}
	return snapshot
}

func TestAtomicTrustStoreResolveKnownAndUnknownKey(t *testing.T) {
	want := testPublicKey(1)
	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		1,
		KeyBinding{KeyID: "key-a", PublicKey: want},
	))
	if err != nil {
		t.Fatalf("NewAtomicTrustStore() error = %v", err)
	}

	got, err := store.ResolveKey("key-a")
	if err != nil {
		t.Fatalf("ResolveKey(known) error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("ResolveKey(known) = %x, want %x", got, want)
	}

	_, err = store.ResolveKey("missing")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("ResolveKey(missing) error = %v, want ErrKeyNotFound", err)
	}

	_, err = store.ResolveKey("")
	if !errors.Is(err, ErrInvalidKeyID) {
		t.Fatalf("ResolveKey(empty) error = %v, want ErrInvalidKeyID", err)
	}
}

func TestAtomicTrustStoreResolveKeyReturnsDefensiveCopy(t *testing.T) {
	original := testPublicKey(1)
	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		1,
		KeyBinding{KeyID: "key-a", PublicKey: original},
	))
	if err != nil {
		t.Fatal(err)
	}

	first, err := store.ResolveKey("key-a")
	if err != nil {
		t.Fatal(err)
	}
	first[0] ^= 0xff

	second, err := store.ResolveKey("key-a")
	if err != nil {
		t.Fatal(err)
	}
	if second[0] != original[0] {
		t.Fatalf("active key mutated through returned slice: got %x want %x", second[0], original[0])
	}
}

func TestAtomicTrustStoreReplaceIsAtomicAndPreservesOldStateOnFailure(t *testing.T) {
	oldKey := testPublicKey(1)
	newKey := testPublicKey(2)

	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		1,
		KeyBinding{KeyID: "key-a", PublicKey: oldKey},
	))
	if err != nil {
		t.Fatal(err)
	}

	validNext := mustSnapshot(
		t,
		2,
		KeyBinding{KeyID: "key-a", PublicKey: newKey},
	)
	if err := store.Replace(validNext); err != nil {
		t.Fatalf("Replace(validNext) error = %v", err)
	}

	got, err := store.ResolveKey("key-a")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, newKey) {
		t.Fatalf("after valid replace got %x, want %x", got, newKey)
	}

	invalid := TrustSnapshot{
		revision:    3,
		keys:        map[string]ed25519.PublicKey{"key-a": {1, 2, 3}},
		fingerprint: "invalid",
	}
	if err := store.Replace(invalid); !errors.Is(err, ErrInvalidTrustSnapshot) {
		t.Fatalf("Replace(invalid) error = %v, want ErrInvalidTrustSnapshot", err)
	}

	got, err = store.ResolveKey("key-a")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, newKey) {
		t.Fatalf("failed replacement changed active state: got %x want %x", got, newKey)
	}
}

func TestAtomicTrustStoreRejectsRevisionRollback(t *testing.T) {
	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		2,
		KeyBinding{KeyID: "key-a", PublicKey: testPublicKey(1)},
	))
	if err != nil {
		t.Fatal(err)
	}

	err = store.Replace(mustSnapshot(
		t,
		1,
		KeyBinding{KeyID: "key-a", PublicKey: testPublicKey(1)},
	))
	if !errors.Is(err, ErrTrustRevisionRollback) {
		t.Fatalf("Replace(rollback) error = %v, want ErrTrustRevisionRollback", err)
	}
}

func TestAtomicTrustStoreSameRevisionSameContentIsIdempotent(t *testing.T) {
	snapshot := mustSnapshot(
		t,
		1,
		KeyBinding{KeyID: "key-a", PublicKey: testPublicKey(1)},
	)
	store, err := NewAtomicTrustStore(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Replace(snapshot); err != nil {
		t.Fatalf("Replace(identical) error = %v", err)
	}
}

func TestAtomicTrustStoreRejectsSameRevisionDifferentContent(t *testing.T) {
	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		1,
		KeyBinding{KeyID: "key-a", PublicKey: testPublicKey(1)},
	))
	if err != nil {
		t.Fatal(err)
	}

	err = store.Replace(mustSnapshot(
		t,
		1,
		KeyBinding{KeyID: "key-a", PublicKey: testPublicKey(2)},
	))
	if !errors.Is(err, ErrTrustRevisionConflict) {
		t.Fatalf("Replace(conflict) error = %v, want ErrTrustRevisionConflict", err)
	}
}

func TestAtomicTrustStoreConcurrentReadersObserveCompleteSnapshots(t *testing.T) {
	oldKey := testPublicKey(1)
	newKey := testPublicKey(2)

	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		1,
		KeyBinding{KeyID: "key-a", PublicKey: oldKey},
	))
	if err != nil {
		t.Fatal(err)
	}

	const readers = 32
	const iterations = 200

	start := make(chan struct{})
	errCh := make(chan error, readers)
	var wg sync.WaitGroup

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				got, err := store.ResolveKey("key-a")
				if err != nil {
					errCh <- err
					return
				}
				if !bytes.Equal(got, oldKey) && !bytes.Equal(got, newKey) {
					errCh <- errors.New("reader observed partial or unexpected key state")
					return
				}
			}
		}()
	}

	close(start)
	if err := store.Replace(mustSnapshot(
		t,
		2,
		KeyBinding{KeyID: "key-a", PublicKey: newKey},
	)); err != nil {
		t.Fatal(err)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}
