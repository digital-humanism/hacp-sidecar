package wire

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"
	"sync"
)

var ErrConflictingKeyBinding = errors.New("conflicting key binding")

// KeyResolver resolves signer_key_id to Ed25519 public keys
type KeyResolver interface {
	ResolveKey(keyID string) (ed25519.PublicKey, error)
}

// StaticKeyResolver is an in-memory key store
type StaticKeyResolver struct {
	mu   sync.RWMutex
	keys map[string]ed25519.PublicKey
}

// NewStaticKeyResolver creates an empty resolver
func NewStaticKeyResolver() *StaticKeyResolver {
	return &StaticKeyResolver{
		keys: make(map[string]ed25519.PublicKey),
	}
}

// AddKey registers a public key by its ID.
//
// Re-adding the same key for the same ID is idempotent.
// Rebinding an existing ID to different key material is rejected.
func (r *StaticKeyResolver) AddKey(keyID string, pubKey ed25519.PublicKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.keys[keyID]; ok {
		if bytes.Equal(existing, pubKey) {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrConflictingKeyBinding, keyID)
	}

	r.keys[keyID] = pubKey
	return nil
}

// AddKeyFromHex registers a public key from a hex-encoded string
func (r *StaticKeyResolver) AddKeyFromHex(keyID string, hexKey string) error {
	pubKey, err := ParseEd25519PublicKey(hexKey)
	if err != nil {
		return err
	}
	return r.AddKey(keyID, pubKey)
}

// ResolveKey looks up a key by ID
func (r *StaticKeyResolver) ResolveKey(keyID string) (ed25519.PublicKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pub, ok := r.keys[keyID]
	if !ok {
		return nil, errors.New("key not found: " + keyID)
	}
	return pub, nil
}
