package trust

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"
)

// KeyBinding binds one signer_key_id to one Ed25519 public key.
type KeyBinding struct {
	KeyID     string
	PublicKey ed25519.PublicKey
}

// TrustSnapshot is a validated, immutable trust-state generation.
//
// The fields are intentionally unexported so callers cannot mutate active
// key material after validation. Use the accessor methods for observability.
type TrustSnapshot struct {
	revision    uint64
	loadedAt    time.Time
	source      string
	keys        map[string]ed25519.PublicKey
	fingerprint string
}

// NewSnapshot validates bindings and returns an immutable trust snapshot.
//
// Identical duplicate bindings are normalized. A signer_key_id bound to
// different public keys is rejected.
func NewSnapshot(
	revision uint64,
	loadedAt time.Time,
	source string,
	bindings []KeyBinding,
) (TrustSnapshot, error) {
	if len(bindings) == 0 {
		return TrustSnapshot{}, ErrEmptyTrustSnapshot
	}

	keys := make(map[string]ed25519.PublicKey, len(bindings))
	for _, binding := range bindings {
		if binding.KeyID == "" {
			return TrustSnapshot{}, ErrInvalidKeyID
		}
		if len(binding.PublicKey) != ed25519.PublicKeySize {
			return TrustSnapshot{}, fmt.Errorf(
				"%w: key %q has %d bytes, want %d",
				ErrInvalidPublicKey,
				binding.KeyID,
				len(binding.PublicKey),
				ed25519.PublicKeySize,
			)
		}

		next := clonePublicKey(binding.PublicKey)
		if existing, ok := keys[binding.KeyID]; ok {
			if !bytes.Equal(existing, next) {
				return TrustSnapshot{}, fmt.Errorf(
					"%w: %q",
					ErrConflictingKeyBinding,
					binding.KeyID,
				)
			}
			continue
		}
		keys[binding.KeyID] = next
	}

	snapshot := TrustSnapshot{
		revision: revision,
		loadedAt: loadedAt,
		source:   source,
		keys:     keys,
	}
	snapshot.fingerprint = fingerprintKeys(keys)

	if err := snapshot.validate(); err != nil {
		return TrustSnapshot{}, err
	}
	return snapshot, nil
}

// Revision returns the operational trust-state generation.
func (s TrustSnapshot) Revision() uint64 {
	return s.revision
}

// LoadedAt returns the time at which the snapshot was loaded by its source.
func (s TrustSnapshot) LoadedAt() time.Time {
	return s.loadedAt
}

// Source returns the administrative source identifier.
func (s TrustSnapshot) Source() string {
	return s.source
}

// Fingerprint returns a deterministic SHA-256 fingerprint of signer bindings.
func (s TrustSnapshot) Fingerprint() string {
	return s.fingerprint
}

// KeyCount returns the number of unique trusted signer IDs.
func (s TrustSnapshot) KeyCount() int {
	return len(s.keys)
}

func (s TrustSnapshot) validate() error {
	if len(s.keys) == 0 {
		return ErrEmptyTrustSnapshot
	}
	for keyID, publicKey := range s.keys {
		if keyID == "" {
			return ErrInvalidKeyID
		}
		if len(publicKey) != ed25519.PublicKeySize {
			return fmt.Errorf(
				"%w: key %q has %d bytes, want %d",
				ErrInvalidPublicKey,
				keyID,
				len(publicKey),
				ed25519.PublicKeySize,
			)
		}
	}
	if s.fingerprint == "" {
		return ErrInvalidTrustSnapshot
	}
	if want := fingerprintKeys(s.keys); want != s.fingerprint {
		return fmt.Errorf("%w: fingerprint mismatch", ErrInvalidTrustSnapshot)
	}
	return nil
}

func cloneSnapshot(s TrustSnapshot) TrustSnapshot {
	keys := make(map[string]ed25519.PublicKey, len(s.keys))
	for keyID, publicKey := range s.keys {
		keys[keyID] = clonePublicKey(publicKey)
	}
	return TrustSnapshot{
		revision:    s.revision,
		loadedAt:    s.loadedAt,
		source:      s.source,
		keys:        keys,
		fingerprint: s.fingerprint,
	}
}

func clonePublicKey(publicKey ed25519.PublicKey) ed25519.PublicKey {
	out := make(ed25519.PublicKey, len(publicKey))
	copy(out, publicKey)
	return out
}

func fingerprintKeys(keys map[string]ed25519.PublicKey) string {
	ids := make([]string, 0, len(keys))
	for keyID := range keys {
		ids = append(ids, keyID)
	}
	sort.Strings(ids)

	h := sha256.New()
	for _, keyID := range ids {
		h.Write([]byte(keyID))
		h.Write([]byte{0})
		h.Write(keys[keyID])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
