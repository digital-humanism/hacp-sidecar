package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"hacp-sidecar/internal/trust"
)

const (
	trustModeProduction = "production"
	trustModeTest       = "test"

	conformanceTestKeyID  = "key-ed25519-test-001"
	conformanceTestKeyHex = "9d17f1bbcc0845865e670f526413fb7a510380798fe300b6c98e28f3a3b0fdb3"
)

var (
	errMissingTrustConfig   = errors.New("production trust configuration is required")
	errAmbiguousTrustMode   = errors.New("test mode must not be combined with HACP_TRUST_KEYS_FILE")
	errUnsupportedTrustMode = errors.New("unsupported HACP_TRUST_MODE")
)

// loadStartupTrustStore constructs the sidecar's initial read-only resolver.
//
// Production mode is the default. It requires HACP_TRUST_KEYS_FILE.
// The published conformance key is available only through explicit test mode:
// HACP_TRUST_MODE=test.
func loadStartupTrustStore() (*trust.AtomicTrustStore, error) {
	mode := strings.TrimSpace(os.Getenv("HACP_TRUST_MODE"))
	if mode == "" {
		mode = trustModeProduction
	}

	trustFile := strings.TrimSpace(os.Getenv("HACP_TRUST_KEYS_FILE"))

	switch mode {
	case trustModeProduction:
		if trustFile == "" {
			return nil, errMissingTrustConfig
		}
		snapshot, err := trust.LoadSnapshotFile(trustFile)
		if err != nil {
			return nil, fmt.Errorf("load production trust snapshot: %w", err)
		}
		return trust.NewAtomicTrustStore(snapshot)

	case trustModeTest:
		if trustFile != "" {
			return nil, errAmbiguousTrustMode
		}
		raw, err := hex.DecodeString(conformanceTestKeyHex)
		if err != nil {
			return nil, fmt.Errorf("decode built-in conformance key: %w", err)
		}
		if len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf(
				"built-in conformance key has %d bytes, want %d",
				len(raw),
				ed25519.PublicKeySize,
			)
		}

		snapshot, err := trust.NewSnapshot(
			1,
			time.Now().UTC(),
			"builtin:conformance-test",
			[]trust.KeyBinding{{
				KeyID:     conformanceTestKeyID,
				PublicKey: ed25519.PublicKey(raw),
			}},
		)
		if err != nil {
			return nil, fmt.Errorf("build conformance trust snapshot: %w", err)
		}
		return trust.NewAtomicTrustStore(snapshot)

	default:
		return nil, fmt.Errorf("%w: %q", errUnsupportedTrustMode, mode)
	}
}
