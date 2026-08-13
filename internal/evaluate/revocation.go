package evaluate

import (
	"sync"
	"time"
)

// InMemoryRevocationStore implements RevocationStore with in-memory denylists
type InMemoryRevocationStore struct {
	mu               sync.RWMutex
	revokedKeys      map[string]bool
	revokedEnvelopes map[string]bool
	revokedTokens    map[string]bool
	lastUpdated      time.Time
}

// NewInMemoryRevocationStore creates a new in-memory revocation store
func NewInMemoryRevocationStore() *InMemoryRevocationStore {
	return &InMemoryRevocationStore{
		revokedKeys:      make(map[string]bool),
		revokedEnvelopes: make(map[string]bool),
		revokedTokens:    make(map[string]bool),
		lastUpdated:      time.Now(),
	}
}

// RevokeKey marks a key as revoked
func (s *InMemoryRevocationStore) RevokeKey(keyID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokedKeys[keyID] = true
	s.lastUpdated = time.Now()
}

// RevokeEnvelope marks an envelope as revoked
func (s *InMemoryRevocationStore) RevokeEnvelope(envelopeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokedEnvelopes[envelopeID] = true
	s.lastUpdated = time.Now()
}

// RevokeToken marks a token as revoked
func (s *InMemoryRevocationStore) RevokeToken(tokenID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokedTokens[tokenID] = true
	s.lastUpdated = time.Now()
}

// IsKeyRevoked checks if a key is revoked
func (s *InMemoryRevocationStore) IsKeyRevoked(keyID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revokedKeys[keyID]
}

// IsEnvelopeRevoked checks if an envelope is revoked
func (s *InMemoryRevocationStore) IsEnvelopeRevoked(envelopeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revokedEnvelopes[envelopeID]
}

// IsTokenRevoked checks if a token is revoked
func (s *InMemoryRevocationStore) IsTokenRevoked(tokenID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revokedTokens[tokenID]
}

// LastUpdated returns the timestamp of the last revocation update
func (s *InMemoryRevocationStore) LastUpdated() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdated
}

// Clear removes all revocations (for testing)
func (s *InMemoryRevocationStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokedKeys = make(map[string]bool)
	s.revokedEnvelopes = make(map[string]bool)
	s.revokedTokens = make(map[string]bool)
	s.lastUpdated = time.Now()
}
