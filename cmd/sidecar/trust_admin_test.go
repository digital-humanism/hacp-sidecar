package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hacp-sidecar/internal/trust"
)

func adminTestStore(t *testing.T, revision uint64, keyID string, fill byte) *trust.AtomicTrustStore {
	t.Helper()
	key := bytes.Repeat([]byte{fill}, 32)
	snapshot, err := trust.NewSnapshot(
		revision,
		time.Unix(int64(revision), 0).UTC(),
		"test",
		[]trust.KeyBinding{{KeyID: keyID, PublicKey: key}},
	)
	if err != nil {
		t.Fatal(err)
	}
	store, err := trust.NewAtomicTrustStore(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func writeAdminTrustFile(t *testing.T, revision uint64, keyID string, fillHex string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trust.json")
	body := `{"revision":` + fmtUint(revision) + `,"keys":[{"key_id":"` + keyID + `","public_key_hex":"` + fillHex + `"}]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func fmtUint(v uint64) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

func repeatHexPair(pair string) string {
	out := ""
	for i := 0; i < 32; i++ {
		out += pair
	}
	return out
}

func TestTrustAdminGetState(t *testing.T) {
	store := adminTestStore(t, 1, "old-key", 1)
	req := httptest.NewRequest(http.MethodGet, "/trust", nil)
	rec := httptest.NewRecorder()

	newTrustAdminHandler(store, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var got trustStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Revision != 1 || got.KeyCount != 1 || got.Fingerprint == "" {
		t.Fatalf("unexpected state: %+v", got)
	}
}

func TestTrustAdminReloadActivatesNewSnapshot(t *testing.T) {
	store := adminTestStore(t, 1, "old-key", 1)
	path := writeAdminTrustFile(t, 2, "new-key", repeatHexPair("02"))

	req := httptest.NewRequest(http.MethodPost, "/trust/reload", nil)
	rec := httptest.NewRecorder()
	newTrustAdminHandler(store, path).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	if _, err := store.ResolveKey("old-key"); !errors.Is(err, trust.ErrKeyNotFound) {
		t.Fatalf("old key error = %v, want ErrKeyNotFound", err)
	}
	if _, err := store.ResolveKey("new-key"); err != nil {
		t.Fatalf("new key did not activate: %v", err)
	}
}

func TestTrustAdminReloadRejectsRollbackAndPreservesState(t *testing.T) {
	store := adminTestStore(t, 3, "active-key", 3)
	path := writeAdminTrustFile(t, 2, "old-key", repeatHexPair("02"))

	req := httptest.NewRequest(http.MethodPost, "/trust/reload", nil)
	rec := httptest.NewRecorder()
	newTrustAdminHandler(store, path).ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", rec.Code, rec.Body.String())
	}
	if _, err := store.ResolveKey("active-key"); err != nil {
		t.Fatalf("active state was not preserved: %v", err)
	}
}

func TestTrustAdminReloadUnavailableWithoutTrustFile(t *testing.T) {
	store := adminTestStore(t, 1, "key-a", 1)

	req := httptest.NewRequest(http.MethodPost, "/trust/reload", nil)
	rec := httptest.NewRecorder()
	newTrustAdminHandler(store, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestTrustAdminMethodRestrictions(t *testing.T) {
	store := adminTestStore(t, 1, "key-a", 1)
	handler := newTrustAdminHandler(store, "")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/trust", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /trust status = %d, want 405", rec.Code)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/trust/reload", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /trust/reload status = %d, want 405", rec.Code)
	}
}

func TestValidateTrustAdminAddressAllowsLoopbackOnly(t *testing.T) {
	for _, addr := range []string{
		"127.0.0.1:9081",
		"localhost:9081",
		"[::1]:9081",
	} {
		if err := validateTrustAdminAddress(addr); err != nil {
			t.Fatalf("validateTrustAdminAddress(%q) error = %v", addr, err)
		}
	}

	for _, addr := range []string{
		":9081",
		"0.0.0.0:9081",
		"192.168.1.10:9081",
		"example.com:9081",
		"127.0.0.1",
	} {
		if err := validateTrustAdminAddress(addr); !errors.Is(err, errUnsafeTrustAdminAddress) {
			t.Fatalf("validateTrustAdminAddress(%q) error = %v, want errUnsafeTrustAdminAddress", addr, err)
		}
	}
}

func TestStartTrustAdminServerDisabledByDefault(t *testing.T) {
	t.Setenv("HACP_TRUST_ADMIN_ADDR", "")
	store := adminTestStore(t, 1, "key-a", 1)

	server, err := startTrustAdminServer(store, "")
	if err != nil {
		t.Fatal(err)
	}
	if server != nil {
		t.Fatal("server enabled without HACP_TRUST_ADMIN_ADDR")
	}
}

func TestStartTrustAdminServerRejectsNonLoopback(t *testing.T) {
	t.Setenv("HACP_TRUST_ADMIN_ADDR", "0.0.0.0:9081")
	store := adminTestStore(t, 1, "key-a", 1)

	_, err := startTrustAdminServer(store, "")
	if !errors.Is(err, errUnsafeTrustAdminAddress) {
		t.Fatalf("error = %v, want errUnsafeTrustAdminAddress", err)
	}
}
