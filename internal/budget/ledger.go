package budget

import (
	"sync"
)

// Ledger tracks DecisionToken usage and envelope autonomy-budget
// consumption separately.
type Ledger struct {
	mu sync.Mutex

	tokenConsumed    map[string]int
	autonomyConsumed map[string]int
}

// NewLedger creates an empty budget ledger.
func NewLedger() *Ledger {
	return &Ledger{
		tokenConsumed:    make(map[string]int),
		autonomyConsumed: make(map[string]int),
	}
}

// ConsumeToken attempts to consume one use of a DecisionToken.
//
// Returns false if maxUses has already been reached.
func (l *Ledger) ConsumeToken(tokenID string, maxUses int) bool {
	if tokenID == "" || maxUses <= 0 {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.tokenConsumed[tokenID]

	if current >= maxUses {
		return false
	}

	l.tokenConsumed[tokenID] = current + 1
	return true
}

// ConsumeAutonomy attempts to consume one autonomous action
// granted by an IntentEnvelope autonomy_budget.
//
// Returns false if maxActions has already been reached.
func (l *Ledger) ConsumeAutonomy(
	envelopeID string,
	maxActions int,
) bool {
	if envelopeID == "" || maxActions <= 0 {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.autonomyConsumed[envelopeID]

	if current >= maxActions {
		return false
	}

	l.autonomyConsumed[envelopeID] = current + 1
	return true
}

// Reset clears all per-vector budget state.
func (l *Ledger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.tokenConsumed = make(map[string]int)
	l.autonomyConsumed = make(map[string]int)
}

// ResetTokens clears only DecisionToken usage counters.
func (l *Ledger) ResetTokens() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.tokenConsumed = make(map[string]int)
}

// ResetAutonomy clears only autonomy-budget counters.
func (l *Ledger) ResetAutonomy() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.autonomyConsumed = make(map[string]int)
}

// TokenCount returns current token consumption for diagnostics/tests.
func (l *Ledger) TokenCount(tokenID string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.tokenConsumed[tokenID]
}

// AutonomyCount returns current autonomy consumption for diagnostics/tests.
func (l *Ledger) AutonomyCount(envelopeID string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.autonomyConsumed[envelopeID]
}
