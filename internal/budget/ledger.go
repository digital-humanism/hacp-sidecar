package budget

import (
	"sync"
)

// Ledger tracks token consumption with atomic counters
type Ledger struct {
	mu       sync.Mutex
	consumed map[string]int
}

// NewLedger creates a new budget ledger
func NewLedger() *Ledger {
	return &Ledger{
		consumed: make(map[string]int),
	}
}

// Consume attempts to consume one use of a token.
// Returns false if the token has reached its max_uses limit (budget exhausted or replayed).
func (l *Ledger) Consume(tokenID string, maxUses int) bool {
	if maxUses <= 0 {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.consumed[tokenID]
	if current >= maxUses {
		return false
	}

	l.consumed[tokenID] = current + 1
	return true
}

// Reset clears all counters (for testing)
func (l *Ledger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.consumed = make(map[string]int)
}

// Count returns the current consumption count for a token (for testing)
func (l *Ledger) Count(tokenID string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.consumed[tokenID]
}
