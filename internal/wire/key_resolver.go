package wire

import (
	"crypto/ed25519"
	"errors"
	"sync"
)

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

// AddKey registers a public key by its ID
func (r *StaticKeyResolver) AddKey(keyID string, pubKey ed25519.PublicKey) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys[keyID] = pubKey
}

// AddKeyFromHex registers a public key from a hex-encoded string
func (r *StaticKeyResolver) AddKeyFromHex(keyID string, hexKey string) error {
	pubKey, err := ParseEd25519PublicKey(hexKey)
	if err != nil {
		return err
	}
	r.AddKey(keyID, pubKey)
	return nil
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
