package evaluate

import (
	"crypto/ed25519"
	"time"

	"hacp-sidecar/internal/wire"
)

// KeyResolver resolves signer_key_id to Ed25519 public key
type KeyResolver interface {
	ResolveKey(keyID string) (ed25519.PublicKey, error)
}

// RevocationStore tracks revoked keys, envelopes, and tokens
type RevocationStore interface {
	IsKeyRevoked(keyID string) bool
	IsEnvelopeRevoked(envelopeID string) bool
	IsTokenRevoked(tokenID string) bool
	LastUpdated() time.Time
}

// BudgetLedger tracks token consumption and budget limits
type BudgetLedger interface {
	// Consume attempts to consume one use of a token
	// Returns false if budget exhausted or token replayed
	Consume(tokenID string, maxUses int) bool
}

// ScopeGuard validates scope containment and boundary matrix
type ScopeGuard interface {
	// MatchRequestConstraints checks if request matches token constraints
	MatchRequestConstraints(constraints *wire.Constraints, req *RequestContext) bool

	// CheckBoundary evaluates boundary matrix
	CheckBoundary(scope *wire.ScopeGrant, req *RequestContext) bool
}

// ProvenanceWriter records provenance events
type ProvenanceWriter interface {
	// Accept records a provenance event before forwarding
	// Must be called before forwarding the request
	Accept(env *wire.IntentEnvelope, tok *wire.DecisionToken, req *RequestContext) error
}

// RequestContext holds information about the incoming request
type RequestContext struct {
	Method      string
	Path        string
	ToolName    string
	PayloadHash string // SHA-256 hex of request body
	Timestamp   time.Time
	RequestID   string
	LatencyNs   int64
}

// Decision represents the result of evaluation
type Decision struct {
	Allow      bool
	ReasonCode string
	Error      error
}

// Reason codes from hacp-spec/error-model.md
const (
	ReasonInvalidEnvelope  = "INVALID_ENVELOPE"
	ReasonInvalidAction    = "INVALID_ACTION"
	ReasonKeyRevoked       = "KEY_REVOKED"
	ReasonSignatureFailure = "SIGNATURE_FAILURE"
	ReasonEnvelopeRevoked  = "ENVELOPE_REVOKED"
	ReasonTokenRevoked     = "TOKEN_REVOKED"
	ReasonEnvelopeExpired  = "ENVELOPE_EXPIRED"
	ReasonTokenExpired     = "TOKEN_EXPIRED"
	ReasonScopeExceeded    = "SCOPE_EXCEEDED"
	ReasonBudgetExhausted  = "BUDGET_EXHAUSTED"
	ReasonTraceabilityFail = "TRACEABILITY_FAILURE"
	ReasonPolicyDenied     = "POLICY_DENIED"
)
