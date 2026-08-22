package trust

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeTrustFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trust.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadSnapshotFileValid(t *testing.T) {
	path := writeTrustFile(t, `{
		"revision": 7,
		"keys": [
			{
				"key_id": "key-a",
				"public_key_hex": "0101010101010101010101010101010101010101010101010101010101010101"
			},
			{
				"key_id": "key-b",
				"public_key_hex": "0202020202020202020202020202020202020202020202020202020202020202"
			}
		]
	}`)

	snapshot, err := LoadSnapshotFile(path)
	if err != nil {
		t.Fatalf("LoadSnapshotFile() error = %v", err)
	}
	if snapshot.Revision() != 7 {
		t.Fatalf("Revision() = %d, want 7", snapshot.Revision())
	}
	if snapshot.KeyCount() != 2 {
		t.Fatalf("KeyCount() = %d, want 2", snapshot.KeyCount())
	}
	if snapshot.Source() != "file:"+path {
		t.Fatalf("Source() = %q, want %q", snapshot.Source(), "file:"+path)
	}
}

func TestLoadSnapshotFileRejectsMissingFile(t *testing.T) {
	_, err := LoadSnapshotFile(filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, ErrTrustSourceUnavailable) {
		t.Fatalf("error = %v, want ErrTrustSourceUnavailable", err)
	}
}

func TestLoadSnapshotFileRejectsUnknownField(t *testing.T) {
	path := writeTrustFile(t, `{
		"revision": 1,
		"unexpected": true,
		"keys": [{
			"key_id": "key-a",
			"public_key_hex": "0101010101010101010101010101010101010101010101010101010101010101"
		}]
	}`)

	_, err := LoadSnapshotFile(path)
	if !errors.Is(err, ErrInvalidTrustSnapshot) {
		t.Fatalf("error = %v, want ErrInvalidTrustSnapshot", err)
	}
}

func TestLoadSnapshotFileRejectsZeroRevision(t *testing.T) {
	path := writeTrustFile(t, `{
		"revision": 0,
		"keys": [{
			"key_id": "key-a",
			"public_key_hex": "0101010101010101010101010101010101010101010101010101010101010101"
		}]
	}`)

	_, err := LoadSnapshotFile(path)
	if !errors.Is(err, ErrInvalidTrustSnapshot) {
		t.Fatalf("error = %v, want ErrInvalidTrustSnapshot", err)
	}
}

func TestLoadSnapshotFileRejectsConflictingBinding(t *testing.T) {
	path := writeTrustFile(t, `{
		"revision": 1,
		"keys": [
			{
				"key_id": "key-a",
				"public_key_hex": "0101010101010101010101010101010101010101010101010101010101010101"
			},
			{
				"key_id": "key-a",
				"public_key_hex": "0202020202020202020202020202020202020202020202020202020202020202"
			}
		]
	}`)

	_, err := LoadSnapshotFile(path)
	if !errors.Is(err, ErrConflictingKeyBinding) {
		t.Fatalf("error = %v, want ErrConflictingKeyBinding", err)
	}
}

func TestLoadSnapshotFileNormalizesIdenticalDuplicate(t *testing.T) {
	path := writeTrustFile(t, `{
		"revision": 1,
		"keys": [
			{
				"key_id": "key-a",
				"public_key_hex": "0101010101010101010101010101010101010101010101010101010101010101"
			},
			{
				"key_id": "key-a",
				"public_key_hex": "0101010101010101010101010101010101010101010101010101010101010101"
			}
		]
	}`)

	snapshot, err := LoadSnapshotFile(path)
	if err != nil {
		t.Fatalf("LoadSnapshotFile() error = %v", err)
	}
	if snapshot.KeyCount() != 1 {
		t.Fatalf("KeyCount() = %d, want 1", snapshot.KeyCount())
	}

	store, err := NewAtomicTrustStore(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.ResolveKey("key-a")
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Repeat([]byte{1}, 32)
	if !bytes.Equal(got, want) {
		t.Fatalf("resolved key = %x, want %x", got, want)
	}
}
