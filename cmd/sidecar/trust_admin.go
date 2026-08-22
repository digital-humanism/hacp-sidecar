package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"hacp-sidecar/internal/trust"
)

var (
	errUnsafeTrustAdminAddress = errors.New("trust admin address must be loopback-only")
	errTrustReloadUnavailable  = errors.New("trust reload is unavailable without an explicit trust file")
)

type trustStateResponse struct {
	Revision    uint64    `json:"revision"`
	Fingerprint string    `json:"fingerprint"`
	KeyCount    int       `json:"key_count"`
	Source      string    `json:"source"`
	LoadedAt    time.Time `json:"loaded_at"`
}

func readTrustState(store *trust.AtomicTrustStore) (trustStateResponse, error) {
	if store == nil {
		return trustStateResponse{}, trust.ErrTrustNotReady
	}

	revision, err := store.Revision()
	if err != nil {
		return trustStateResponse{}, err
	}
	fingerprint, err := store.Fingerprint()
	if err != nil {
		return trustStateResponse{}, err
	}
	keyCount, err := store.KeyCount()
	if err != nil {
		return trustStateResponse{}, err
	}
	source, err := store.Source()
	if err != nil {
		return trustStateResponse{}, err
	}
	loadedAt, err := store.LoadedAt()
	if err != nil {
		return trustStateResponse{}, err
	}

	return trustStateResponse{
		Revision:    revision,
		Fingerprint: fingerprint,
		KeyCount:    keyCount,
		Source:      source,
		LoadedAt:    loadedAt,
	}, nil
}

func newTrustAdminHandler(store *trust.AtomicTrustStore, trustFile string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/trust", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		state, err := readTrustState(store)
		if err != nil {
			http.Error(w, "trust store not ready", http.StatusServiceUnavailable)
			return
		}
		writeTrustStateJSON(w, http.StatusOK, state)
	})

	mux.HandleFunc("/trust/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.TrimSpace(trustFile) == "" {
			http.Error(w, errTrustReloadUnavailable.Error(), http.StatusConflict)
			return
		}

		if err := trust.ReloadFromFile(store, trustFile); err != nil {
			http.Error(w, fmt.Sprintf("trust reload rejected: %v", err), http.StatusConflict)
			return
		}

		state, err := readTrustState(store)
		if err != nil {
			http.Error(w, "trust store not ready", http.StatusServiceUnavailable)
			return
		}
		writeTrustStateJSON(w, http.StatusOK, state)
	})

	return mux
}

func writeTrustStateJSON(w http.ResponseWriter, status int, state trustStateResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(state)
}

func validateTrustAdminAddress(addr string) error {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil || port == "" {
		return errUnsafeTrustAdminAddress
	}

	switch host {
	case "127.0.0.1", "::1", "localhost":
		return nil
	default:
		return errUnsafeTrustAdminAddress
	}
}

// startTrustAdminServer starts the optional loopback-only trust admin server.
//
// The admin surface is disabled unless HACP_TRUST_ADMIN_ADDR is explicitly set.
// Example: HACP_TRUST_ADMIN_ADDR=127.0.0.1:9081
func startTrustAdminServer(
	store *trust.AtomicTrustStore,
	trustFile string,
) (*http.Server, error) {
	addr := strings.TrimSpace(os.Getenv("HACP_TRUST_ADMIN_ADDR"))
	if addr == "" {
		return nil, nil
	}
	if err := validateTrustAdminAddress(addr); err != nil {
		return nil, err
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           newTrustAdminHandler(store, trustFile),
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		_ = server.Serve(listener)
	}()

	return server, nil
}
