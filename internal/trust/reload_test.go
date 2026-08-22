package trust

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func rotationTrustFile(t *testing.T, revision uint64, entries ...string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), fmt.Sprintf("trust-%d.json", revision))

	body := fmt.Sprintf("{\n  \"revision\": %d,\n  \"keys\": [\n%s\n  ]\n}\n",
		revision,
		joinRotationEntries(entries),
	)

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func joinRotationEntries(entries []string) string {
	out := ""
	for i, entry := range entries {
		if i > 0 {
			out += ",\n"
		}
		out += "    " + entry
	}
	return out
}

func rotationEntry(keyID string, fill byte) string {
	return fmt.Sprintf(
		`{"key_id":%q,"public_key_hex":%q}`,
		keyID,
		repeatedHexByte(fill),
	)
}

func repeatedHexByte(fill byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 64)
	for i := 0; i < 32; i++ {
		out[i*2] = digits[fill>>4]
		out[i*2+1] = digits[fill&0x0f]
	}
	return string(out)
}

func TestReloadFromFilePlannedRotationOldOnlyToOverlapToNewOnly(t *testing.T) {
	oldKey := testPublicKey(1)
	newKey := testPublicKey(2)

	initial := mustSnapshot(
		t,
		1,
		KeyBinding{KeyID: "old-key", PublicKey: oldKey},
	)
	store, err := NewAtomicTrustStore(initial)
	if err != nil {
		t.Fatal(err)
	}

	overlapFile := rotationTrustFile(
		t,
		2,
		rotationEntry("old-key", 1),
		rotationEntry("new-key", 2),
	)
	if err := ReloadFromFile(store, overlapFile); err != nil {
		t.Fatalf("ReloadFromFile(overlap) error = %v", err)
	}

	gotOld, err := store.ResolveKey("old-key")
	if err != nil {
		t.Fatalf("ResolveKey(old-key) during overlap error = %v", err)
	}
	if !bytes.Equal(gotOld, oldKey) {
		t.Fatalf("old key changed during overlap")
	}

	gotNew, err := store.ResolveKey("new-key")
	if err != nil {
		t.Fatalf("ResolveKey(new-key) during overlap error = %v", err)
	}
	if !bytes.Equal(gotNew, newKey) {
		t.Fatalf("new key mismatch during overlap")
	}

	newOnlyFile := rotationTrustFile(
		t,
		3,
		rotationEntry("new-key", 2),
	)
	if err := ReloadFromFile(store, newOnlyFile); err != nil {
		t.Fatalf("ReloadFromFile(new-only) error = %v", err)
	}

	if _, err := store.ResolveKey("old-key"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("ResolveKey(old-key) after retirement error = %v, want ErrKeyNotFound", err)
	}

	gotNew, err = store.ResolveKey("new-key")
	if err != nil {
		t.Fatalf("ResolveKey(new-key) after retirement error = %v", err)
	}
	if !bytes.Equal(gotNew, newKey) {
		t.Fatalf("new key mismatch after retirement")
	}
}

func TestReloadFromFileRejectsRevisionRollbackAndPreservesState(t *testing.T) {
	activeKey := testPublicKey(2)

	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		2,
		KeyBinding{KeyID: "active-key", PublicKey: activeKey},
	))
	if err != nil {
		t.Fatal(err)
	}

	rollbackFile := rotationTrustFile(
		t,
		1,
		rotationEntry("old-key", 1),
	)
	err = ReloadFromFile(store, rollbackFile)
	if !errors.Is(err, ErrTrustRevisionRollback) {
		t.Fatalf("ReloadFromFile(rollback) error = %v, want ErrTrustRevisionRollback", err)
	}

	got, err := store.ResolveKey("active-key")
	if err != nil {
		t.Fatalf("ResolveKey(active-key) error = %v", err)
	}
	if !bytes.Equal(got, activeKey) {
		t.Fatal("rollback changed active state")
	}
}

func TestReloadFromFileRejectsSameRevisionDifferentContentAndPreservesState(t *testing.T) {
	activeKey := testPublicKey(1)

	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		5,
		KeyBinding{KeyID: "active-key", PublicKey: activeKey},
	))
	if err != nil {
		t.Fatal(err)
	}

	conflictFile := rotationTrustFile(
		t,
		5,
		rotationEntry("different-key", 2),
	)
	err = ReloadFromFile(store, conflictFile)
	if !errors.Is(err, ErrTrustRevisionConflict) {
		t.Fatalf("ReloadFromFile(conflict) error = %v, want ErrTrustRevisionConflict", err)
	}

	got, err := store.ResolveKey("active-key")
	if err != nil {
		t.Fatalf("ResolveKey(active-key) error = %v", err)
	}
	if !bytes.Equal(got, activeKey) {
		t.Fatal("revision conflict changed active state")
	}
}

func TestReloadFromFileMalformedCandidatePreservesState(t *testing.T) {
	activeKey := testPublicKey(1)

	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		4,
		KeyBinding{KeyID: "active-key", PublicKey: activeKey},
	))
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(path, []byte(`{not-json`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ReloadFromFile(store, path); err == nil {
		t.Fatal("expected malformed reload error")
	}

	got, err := store.ResolveKey("active-key")
	if err != nil {
		t.Fatalf("ResolveKey(active-key) error = %v", err)
	}
	if !bytes.Equal(got, activeKey) {
		t.Fatal("malformed reload changed active state")
	}
}

func TestReloadFromFileConflictingBindingPreservesState(t *testing.T) {
	activeKey := testPublicKey(1)

	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		7,
		KeyBinding{KeyID: "active-key", PublicKey: activeKey},
	))
	if err != nil {
		t.Fatal(err)
	}

	path := rotationTrustFile(
		t,
		8,
		rotationEntry("same-id", 2),
		rotationEntry("same-id", 3),
	)

	err = ReloadFromFile(store, path)
	if !errors.Is(err, ErrConflictingKeyBinding) {
		t.Fatalf("ReloadFromFile(conflicting binding) error = %v, want ErrConflictingKeyBinding", err)
	}

	got, err := store.ResolveKey("active-key")
	if err != nil {
		t.Fatalf("ResolveKey(active-key) error = %v", err)
	}
	if !bytes.Equal(got, activeKey) {
		t.Fatal("conflicting candidate changed active state")
	}
}

func TestReloadFromFileIdenticalSnapshotIsIdempotent(t *testing.T) {
	key := testPublicKey(1)

	store, err := NewAtomicTrustStore(mustSnapshot(
		t,
		9,
		KeyBinding{KeyID: "key-a", PublicKey: key},
	))
	if err != nil {
		t.Fatal(err)
	}

	path := rotationTrustFile(
		t,
		9,
		rotationEntry("key-a", 1),
	)

	if err := ReloadFromFile(store, path); err != nil {
		t.Fatalf("ReloadFromFile(identical) error = %v", err)
	}

	revision, err := store.Revision()
	if err != nil {
		t.Fatal(err)
	}
	if revision != 9 {
		t.Fatalf("Revision() = %d, want 9", revision)
	}
}

func TestReloadFromFileNilStoreFailsClosed(t *testing.T) {
	path := rotationTrustFile(
		t,
		1,
		rotationEntry("key-a", 1),
	)

	err := ReloadFromFile(nil, path)
	if !errors.Is(err, ErrTrustNotReady) {
		t.Fatalf("ReloadFromFile(nil) error = %v, want ErrTrustNotReady", err)
	}
}
