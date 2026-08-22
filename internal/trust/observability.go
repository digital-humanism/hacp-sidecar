package trust

import "time"

// LoadedAt returns the active snapshot load timestamp.
func (s *AtomicTrustStore) LoadedAt() (time.Time, error) {
	current := s.state.Load()
	if current == nil {
		return time.Time{}, ErrTrustNotReady
	}
	return current.snapshot.loadedAt, nil
}

// Source returns the active snapshot source identifier.
func (s *AtomicTrustStore) Source() (string, error) {
	current := s.state.Load()
	if current == nil {
		return "", ErrTrustNotReady
	}
	return current.snapshot.source, nil
}
