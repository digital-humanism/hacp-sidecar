package evaluate

import (
	"crypto/ed25519"
	"time"

	"hacp-sidecar/internal/wire"
)

// ============================================================
// Outcome
// ============================================================

// Outcome represents the normative HACP evaluation outcome.
type Outcome string

const (
	OutcomeAllow      Outcome = "ALLOW"
	OutcomeDeny       Outcome = "DENY"
	OutcomeCheckpoint Outcome = "CHECKPOINT"
)

// ============================================================
// Key resolution / revocation
// ============================================================

// KeyResolver resolves signer_key_id to Ed25519 public key.
type KeyResolver interface {
	ResolveKey(keyID string) (ed25519.PublicKey, error)
}

// RevocationStore tracks revoked keys, envelopes, and tokens.
type RevocationStore interface {
	IsKeyRevoked(keyID string) bool
	IsEnvelopeRevoked(envelopeID string) bool
	IsTokenRevoked(tokenID string) bool
	LastUpdated() time.Time
}

// ============================================================
// Budget
// ============================================================

// BudgetLedger tracks both:
//  1. DecisionToken usage / replay protection
//  2. IntentEnvelope autonomy-budget consumption
//
// Token and envelope budgets are intentionally separate because
// they have different normative meanings.
type BudgetLedger interface {
	// ConsumeToken attempts to consume one use of a DecisionToken.
	//
	// Returns false if maxUses has been reached.
	ConsumeToken(tokenID string, maxUses int) bool

	// ConsumeAutonomy attempts to consume one autonomous action
	// granted by an IntentEnvelope autonomy_budget.
	//
	// Returns false if maxActions has been reached.
	ConsumeAutonomy(envelopeID string, maxActions int) bool
}

// ============================================================
// Scope
// ============================================================

// ScopeGuard validates token constraints and envelope scope.
type ScopeGuard interface {
	// MatchRequestConstraints checks whether the request matches
	// DecisionToken constraints.
	MatchRequestConstraints(
		constraints *wire.Constraints,
		req *RequestContext,
	) bool

	// CheckBoundary evaluates envelope scope containment / boundary matrix.
	CheckBoundary(
		scope *wire.ScopeGrant,
		req *RequestContext,
	) bool
}

// ============================================================
// Provenance
// ============================================================

// ProvenanceWriter records provenance events.
type ProvenanceWriter interface {
	// Accept records a provenance event before forwarding.
	//
	// tok may be nil for envelope-only autonomous/checkpoint flows.
	Accept(
		env *wire.IntentEnvelope,
		tok *wire.DecisionToken,
		req *RequestContext,
	) error
}

// ============================================================
// Checkpoint
// ============================================================

// CheckpointState represents the lifecycle state of a checkpoint.
type CheckpointState string

const (
	CheckpointStateUnknown       CheckpointState = ""
	CheckpointStateOpen          CheckpointState = "OPEN"
	CheckpointStateResolvedAllow CheckpointState = "RESOLVED_ALLOW"
	CheckpointStateResolvedDeny  CheckpointState = "RESOLVED_DENY"
	CheckpointStateExpired       CheckpointState = "EXPIRED"
)

// CheckpointContext contains checkpoint information supplied by
// the runner / policy context.
//
// Not every evaluate operation has a checkpoint, therefore this
// structure is optional in RequestContext.
type CheckpointContext struct {
	ID string

	State CheckpointState

	// Principal that created/resolved the checkpoint, when known.
	ResolvedBy string

	// Kind of resolving principal: "human" or "system", when known.
	ResolvedByKind string

	// Unix timestamps.
	CreatedAt  int64
	ExpiresAt  int64
	ResolvedAt int64
}

// ============================================================
// Policy context
// ============================================================

// PolicyContext carries conformance/policy metadata that is not
// part of the signed IntentEnvelope or DecisionToken.
//
// Raw JSON can be retained by the runner if additional profile-specific
// fields are introduced later.
type PolicyContext struct {
	// Deterministic clock used by conformance vectors.
	// 0 means use the system clock.
	Clock int64

	// Whether the proposed action requires explicit human authority.
	HumanRequired bool

	HumanRequiredVerbs []string

	// Optional consequence/risk classification.
	ConsequenceClass string
	RiskClass        string
}

// ============================================================
// Request
// ============================================================

// RequestContext holds information about the incoming proposed action.
type RequestContext struct {
	Method      string
	Path        string
	ToolName    string
	PayloadHash string

	Timestamp time.Time
	RequestID string
	LatencyNs int64

	// Legacy-compatible deterministic clock.
	//
	// Keep this field for now so existing Pipeline code continues
	// to compile while Policy is introduced.
	Clock int64

	// Raw JSON bytes of proposed_action for JCS/action_hash binding.
	ProposedAction []byte

	// Parsed policy information.
	Policy *PolicyContext

	// Optional checkpoint lifecycle information.
	Checkpoint *CheckpointContext
}

// EffectiveClock returns the deterministic policy clock if available,
// otherwise falls back to the legacy Clock field.
func (r *RequestContext) EffectiveClock() int64 {
	if r == nil {
		return 0
	}

	if r.Policy != nil && r.Policy.Clock > 0 {
		return r.Policy.Clock
	}

	return r.Clock
}

// HumanRequired reports whether policy requires a human checkpoint.
func (r *RequestContext) HumanRequired() bool {
	if r == nil || r.Policy == nil {
		return false
	}

	return r.Policy.HumanRequired
}

// ============================================================
// Decision
// ============================================================

// Decision represents the normative result of evaluation.
//
// Outcome is the authoritative result.
//
// Allow is retained temporarily for compatibility with existing code:
//
//	OutcomeAllow      -> Allow=true
//	OutcomeDeny       -> Allow=false
//	OutcomeCheckpoint -> Allow=false
type Decision struct {
	Outcome Outcome

	Allow bool

	ReasonCode string
	Error      error
}

// AllowDecision creates an ALLOW decision.
func AllowDecision() Decision {
	return Decision{
		Outcome: OutcomeAllow,
		Allow:   true,
	}
}

// DenyDecision creates a DENY decision.
func DenyDecision(reason string, err error) Decision {
	return Decision{
		Outcome:    OutcomeDeny,
		Allow:      false,
		ReasonCode: reason,
		Error:      err,
	}
}

// CheckpointDecision creates a CHECKPOINT decision.
func CheckpointDecision(reason string, err error) Decision {
	return Decision{
		Outcome:    OutcomeCheckpoint,
		Allow:      false,
		ReasonCode: reason,
		Error:      err,
	}
}

// ============================================================
// Reason codes
// ============================================================

// Reason codes from hacp-spec/error-model.md and distributed control-plane
// runtime safety semantics.
const (
	ReasonInvalidEnvelope   = "INVALID_ENVELOPE"
	ReasonInvalidAction     = "INVALID_ACTION"
	ReasonKeyRevoked        = "KEY_REVOKED"
	ReasonSignatureFailure  = "SIGNATURE_FAILURE"
	ReasonEnvelopeRevoked   = "ENVELOPE_REVOKED"
	ReasonTokenRevoked      = "TOKEN_REVOKED"
	ReasonControlStateStale = "CONTROL_STATE_STALE"
	ReasonEnvelopeExpired   = "ENVELOPE_EXPIRED"
	ReasonTokenExpired      = "TOKEN_EXPIRED"
	ReasonScopeExceeded     = "SCOPE_EXCEEDED"
	ReasonBudgetExhausted   = "BUDGET_EXHAUSTED"
	ReasonTraceabilityFail  = "TRACEABILITY_FAILURE"
	ReasonPolicyDenied      = "POLICY_DENIED"

	// Checkpoint / human-authorization semantics.
	ReasonHumanRequired      = "HUMAN_REQUIRED"
	ReasonCheckpointOpen     = "CHECKPOINT_OPEN"
	ReasonCheckpointExpired  = "CHECKPOINT_EXPIRED"
	ReasonCheckpointDenied   = "CHECKPOINT_DENIED"
	ReasonCheckpointInvalid  = "CHECKPOINT_INVALID"
	ReasonSelfApprovalDenied = "SELF_APPROVAL_DENIED"

	// Binding failures.
	ReasonEnvelopeBinding  = "ENVELOPE_BINDING_FAILURE"
	ReasonPrincipalBinding = "PRINCIPAL_BINDING_FAILURE"
)
