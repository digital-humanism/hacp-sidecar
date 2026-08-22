package trust

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

var ErrTrustSourceUnavailable = errors.New("trust source unavailable")

type fileTrustConfig struct {
	Revision uint64         `json:"revision"`
	Keys     []fileKeyEntry `json:"keys"`
}

type fileKeyEntry struct {
	KeyID        string `json:"key_id"`
	PublicKeyHex string `json:"public_key_hex"`
}

// LoadSnapshotFile loads and validates one complete production trust snapshot.
//
// The file format is an implementation/deployment format, not a normative HACP
// protocol object.
func LoadSnapshotFile(path string) (TrustSnapshot, error) {
	if path == "" {
		return TrustSnapshot{}, fmt.Errorf("%w: empty trust file path", ErrTrustSourceUnavailable)
	}

	f, err := os.Open(path)
	if err != nil {
		return TrustSnapshot{}, fmt.Errorf("%w: %v", ErrTrustSourceUnavailable, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()

	var cfg fileTrustConfig
	if err := decoder.Decode(&cfg); err != nil {
		return TrustSnapshot{}, fmt.Errorf("%w: %v", ErrInvalidTrustSnapshot, err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return TrustSnapshot{}, fmt.Errorf("%w: trailing JSON value", ErrInvalidTrustSnapshot)
		}
		return TrustSnapshot{}, fmt.Errorf("%w: %v", ErrInvalidTrustSnapshot, err)
	}

	if cfg.Revision == 0 {
		return TrustSnapshot{}, fmt.Errorf("%w: revision must be greater than zero", ErrInvalidTrustSnapshot)
	}
	if len(cfg.Keys) == 0 {
		return TrustSnapshot{}, ErrEmptyTrustSnapshot
	}

	bindings := make([]KeyBinding, 0, len(cfg.Keys))
	for _, entry := range cfg.Keys {
		if entry.KeyID == "" {
			return TrustSnapshot{}, ErrInvalidKeyID
		}

		raw, err := hex.DecodeString(entry.PublicKeyHex)
		if err != nil {
			return TrustSnapshot{}, fmt.Errorf(
				"%w: key %q: %v",
				ErrInvalidPublicKey,
				entry.KeyID,
				err,
			)
		}
		if len(raw) != ed25519.PublicKeySize {
			return TrustSnapshot{}, fmt.Errorf(
				"%w: key %q has %d bytes, want %d",
				ErrInvalidPublicKey,
				entry.KeyID,
				len(raw),
				ed25519.PublicKeySize,
			)
		}

		bindings = append(bindings, KeyBinding{
			KeyID:     entry.KeyID,
			PublicKey: ed25519.PublicKey(raw),
		})
	}

	return NewSnapshot(
		cfg.Revision,
		time.Now().UTC(),
		"file:"+path,
		bindings,
	)
}
