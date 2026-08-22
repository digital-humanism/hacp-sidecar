package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeStartupTrustFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trust.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadStartupTrustStoreProductionRequiresExplicitConfig(t *testing.T) {
	t.Setenv("HACP_TRUST_MODE", "")
	t.Setenv("HACP_TRUST_KEYS_FILE", "")

	_, err := loadStartupTrustStore()
	if !errors.Is(err, errMissingTrustConfig) {
		t.Fatalf("error = %v, want errMissingTrustConfig", err)
	}
}

func TestLoadStartupTrustStoreProductionLoadsExplicitFile(t *testing.T) {
	path := writeStartupTrustFile(t, `{
		"revision": 3,
		"keys": [{
			"key_id": "production-key",
			"public_key_hex": "0303030303030303030303030303030303030303030303030303030303030303"
		}]
	}`)

	t.Setenv("HACP_TRUST_MODE", "production")
	t.Setenv("HACP_TRUST_KEYS_FILE", path)

	store, err := loadStartupTrustStore()
	if err != nil {
		t.Fatalf("loadStartupTrustStore() error = %v", err)
	}
	if !store.Ready() {
		t.Fatal("store is not ready")
	}
	got, err := store.ResolveKey("production-key")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bytes.Repeat([]byte{3}, 32)) {
		t.Fatalf("resolved key = %x", got)
	}
}

func TestLoadStartupTrustStoreMalformedProductionConfigFails(t *testing.T) {
	path := writeStartupTrustFile(t, `{not-json`)
	t.Setenv("HACP_TRUST_MODE", "production")
	t.Setenv("HACP_TRUST_KEYS_FILE", path)

	if _, err := loadStartupTrustStore(); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadStartupTrustStoreTestModeRequiresExplicitOptIn(t *testing.T) {
	t.Setenv("HACP_TRUST_MODE", "test")
	t.Setenv("HACP_TRUST_KEYS_FILE", "")

	store, err := loadStartupTrustStore()
	if err != nil {
		t.Fatalf("loadStartupTrustStore() error = %v", err)
	}

	got, err := store.ResolveKey(conformanceTestKeyID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 32 {
		t.Fatalf("test key length = %d, want 32", len(got))
	}
}

func TestLoadStartupTrustStoreTestModeRejectsTrustFile(t *testing.T) {
	t.Setenv("HACP_TRUST_MODE", "test")
	t.Setenv("HACP_TRUST_KEYS_FILE", "some-file.json")

	_, err := loadStartupTrustStore()
	if !errors.Is(err, errAmbiguousTrustMode) {
		t.Fatalf("error = %v, want errAmbiguousTrustMode", err)
	}
}

func TestLoadStartupTrustStoreRejectsUnsupportedMode(t *testing.T) {
	t.Setenv("HACP_TRUST_MODE", "legacy")
	t.Setenv("HACP_TRUST_KEYS_FILE", "")

	_, err := loadStartupTrustStore()
	if !errors.Is(err, errUnsupportedTrustMode) {
		t.Fatalf("error = %v, want errUnsupportedTrustMode", err)
	}
}
