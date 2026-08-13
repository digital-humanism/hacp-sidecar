package evaluate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hacp-sidecar/internal/wire"
)

// Pipeline implements the normative HACP evaluation order.
type Pipeline struct {
	KeyResolver   KeyResolver
	Revocation    RevocationStore
	BudgetLedger  BudgetLedger
	ScopeGuard    ScopeGuard
	ProvenanceLog ProvenanceWriter

	// Configuration.
	MaxClockSkewSeconds      int64
	MaxRevocationStalenessMs int64
}

// NewPipeline creates a new evaluation pipeline.
func NewPipeline(
	keyResolver KeyResolver,
	revocation RevocationStore,
	budgetLedger BudgetLedger,
	scopeGuard ScopeGuard,
	provenanceLog ProvenanceWriter,
) *Pipeline {
	return &Pipeline{
		KeyResolver:              keyResolver,
		Revocation:               revocation,
		BudgetLedger:             budgetLedger,
		ScopeGuard:               scopeGuard,
		ProvenanceLog:            provenanceLog,
		MaxClockSkewSeconds:      5,
		MaxRevocationStalenessMs: 5000,
	}
}

// Evaluate executes HACP evaluation.
//
// The pipeline supports three major paths:
//
//  1. Envelope + DecisionToken
//     -> normal token-bound evaluation.
//
//  2. Envelope without DecisionToken + autonomy_budget
//     -> autonomous system action.
//
//  3. Envelope without DecisionToken + human/checkpoint requirement
//     -> CHECKPOINT rather than INVALID_ENVELOPE.
func (p *Pipeline) Evaluate(
	ctx context.Context,
	env *wire.IntentEnvelope,
	tok *wire.DecisionToken,
	req *RequestContext,
) Decision {

	_ = ctx // reserved for future cancellation / tracing

	// ============================================================
	// Step 1: Basic request validation
	// ============================================================

	if env == nil {
		return DenyDecision(
			ReasonInvalidEnvelope,
			errors.New("nil intent envelope"),
		)
	}

	if req == nil {
		return DenyDecision(
			ReasonInvalidAction,
			errors.New("nil request context"),
		)
	}

	// Checkpoint state is evaluated before credential verification.
	// An unresolved checkpoint must remain paused and must never
	// progress into execution authorization.
	if req.Checkpoint != nil {
		cp := req.Checkpoint
		now := req.EffectiveClock()

		switch cp.State {

		case CheckpointStateOpen:
			if cp.ExpiresAt > 0 && now > cp.ExpiresAt {
				return DenyDecision(
					ReasonCheckpointExpired,
					errors.New("checkpoint expired"),
				)
			}

			return CheckpointDecision(
				ReasonCheckpointOpen,
				errors.New("checkpoint is still open"),
			)

		case CheckpointStateResolvedDeny:
			return DenyDecision(
				ReasonCheckpointDenied,
				errors.New("checkpoint resolved with denial"),
			)

		case CheckpointStateExpired:
			return DenyDecision(
				ReasonCheckpointExpired,
				errors.New("checkpoint expired"),
			)

		case CheckpointStateResolvedAllow:
			if cp.ResolvedByKind == "system" {
				return DenyDecision(
					ReasonSelfApprovalDenied,
					errors.New(
						"system principal may not resolve a human checkpoint",
					),
				)
			}

			// Human-approved checkpoint continues through normal validation.
		}
	}

	// ============================================================
	// Step 2: Resolve and validate envelope signer
	// ============================================================

	envKey, err := p.KeyResolver.ResolveKey(env.SignerKeyID)
	if err != nil || envKey == nil {
		return DenyDecision(
			ReasonSignatureFailure,
			fmt.Errorf(
				"envelope key not found: signer_key_id=%s",
				env.SignerKeyID,
			),
		)
	}

	// Key revocation MUST be checked before signature verification.
	if p.Revocation.IsKeyRevoked(env.SignerKeyID) {
		return DenyDecision(
			ReasonKeyRevoked,
			fmt.Errorf(
				"envelope signer key revoked: %s",
				env.SignerKeyID,
			),
		)
	}

	// ============================================================
	// Step 3: Verify envelope signature
	// ============================================================

	if !wire.VerifyEd25519(
		envKey,
		env.CanonicalPayload(),
		env.Signature,
	) {
		return DenyDecision(
			ReasonSignatureFailure,
			errors.New("envelope signature invalid"),
		)
	}

	// ============================================================
	// Step 4: Envelope revocation
	// ============================================================

	if p.Revocation.IsEnvelopeRevoked(env.EnvelopeID) {
		return DenyDecision(
			ReasonEnvelopeRevoked,
			fmt.Errorf(
				"envelope revoked: %s",
				env.EnvelopeID,
			),
		)
	}

	// ============================================================
	// Step 5: Envelope expiry
	// ============================================================

	now := req.EffectiveClock()
	if now <= 0 {
		now = time.Now().Unix()
	}

	skew := p.MaxClockSkewSeconds

	if now > env.ExpiresAt+skew {
		return DenyDecision(
			ReasonEnvelopeExpired,
			fmt.Errorf(
				"envelope expired: now=%d expires_at=%d",
				now,
				env.ExpiresAt,
			),
		)
	}

	// ============================================================
	// Step 6: Existing checkpoint state
	// ============================================================

	//
	// Checkpoint state takes precedence over starting a new
	// autonomous or token-less action.
	//
	if req.Checkpoint != nil {
		cp := req.Checkpoint

		// Explicit EXPIRED state.
		if cp.State == CheckpointStateExpired {
			return DenyDecision(
				ReasonCheckpointExpired,
				errors.New("checkpoint expired"),
			)
		}

		// Time-based expiry.
		if cp.ExpiresAt > 0 && now > cp.ExpiresAt+skew {
			return DenyDecision(
				ReasonCheckpointExpired,
				fmt.Errorf(
					"checkpoint expired: now=%d expires_at=%d",
					now,
					cp.ExpiresAt,
				),
			)
		}

		// Explicit checkpoint rejection.
		if cp.State == CheckpointStateResolvedDeny {
			return DenyDecision(
				ReasonCheckpointDenied,
				errors.New("checkpoint resolved with denial"),
			)
		}

		// OPEN means execution is still waiting for authority.
		if cp.State == CheckpointStateOpen {
			return CheckpointDecision(
				ReasonCheckpointOpen,
				errors.New("checkpoint is still open"),
			)
		}

		// System principals must not approve their own checkpoint.
		if cp.State == CheckpointStateResolvedAllow &&
			cp.ResolvedByKind == "system" {

			return DenyDecision(
				ReasonSelfApprovalDenied,
				errors.New(
					"system principal may not resolve a human checkpoint",
				),
			)
		}

		// APPROVED by a human is allowed to continue through
		// normal token verification below.
	}

	// ============================================================
	// Step 7: No DecisionToken
	// ============================================================

	if tok == nil {

		// --------------------------------------------------------
		// 7A: Explicit human-required policy
		// --------------------------------------------------------

		if req.HumanRequired() {
			return CheckpointDecision(
				ReasonHumanRequired,
				errors.New(
					"explicit human authorization required",
				),
			)
		}

		// --------------------------------------------------------
		// 7B: System autonomous execution
		// --------------------------------------------------------

		if env.PrincipalKind == "system" {

			if env.AutonomyBudget == nil {
				return CheckpointDecision(
					ReasonHumanRequired,
					errors.New(
						"system principal has no autonomy budget",
					),
				)
			}

			if env.AutonomyBudget.MaxActions <= 0 {
				return DenyDecision(
					ReasonBudgetExhausted,
					errors.New(
						"autonomy budget max_actions is zero",
					),
				)
			}

			// Scope containment still applies to autonomous actions.
			if !p.ScopeGuard.CheckBoundary(
				env.Scope,
				req,
			) {
				return DenyDecision(
					ReasonScopeExceeded,
					errors.New(
						"autonomous action exceeds envelope scope",
					),
				)
			}

			// Consume one autonomous action.
			if !p.BudgetLedger.ConsumeAutonomy(
				env.EnvelopeID,
				env.AutonomyBudget.MaxActions,
			) {
				return DenyDecision(
					ReasonBudgetExhausted,
					fmt.Errorf(
						"autonomy budget exhausted for envelope %s",
						env.EnvelopeID,
					),
				)
			}

			// Provenance MUST be accepted before forwarding.
			if err := p.ProvenanceLog.Accept(
				env,
				nil,
				req,
			); err != nil {
				return DenyDecision(
					ReasonTraceabilityFail,
					fmt.Errorf(
						"provenance acceptance failed: %w",
						err,
					),
				)
			}

			return AllowDecision()
		}

		// --------------------------------------------------------
		// 7C: Human principal without token
		// --------------------------------------------------------

		//
		// A human envelope alone is not sufficient to create a
		// cryptographically bound ALLOW decision.
		//
		return DenyDecision(
			ReasonPolicyDenied,
			errors.New(
				"decision token required for non-autonomous action",
			),
		)
	}

	// ============================================================
	// Step 8: DecisionToken signer / key revocation
	// ============================================================

	tokKey, err := p.KeyResolver.ResolveKey(tok.SignerKeyID)
	if err != nil || tokKey == nil {
		return DenyDecision(
			ReasonSignatureFailure,
			fmt.Errorf(
				"token key not found: signer_key_id=%s",
				tok.SignerKeyID,
			),
		)
	}

	if p.Revocation.IsKeyRevoked(tok.SignerKeyID) {
		return DenyDecision(
			ReasonKeyRevoked,
			fmt.Errorf(
				"token signer key revoked: %s",
				tok.SignerKeyID,
			),
		)
	}

	// ============================================================
	// Step 9: DecisionToken signature
	// ============================================================

	if !wire.VerifyEd25519(
		tokKey,
		tok.CanonicalPayload(),
		tok.Signature,
	) {
		return DenyDecision(
			ReasonSignatureFailure,
			errors.New("decision token signature invalid"),
		)
	}

	// ============================================================
	// Step 10: Token revocation
	// ============================================================

	if p.Revocation.IsTokenRevoked(tok.TokenID) {
		return DenyDecision(
			ReasonTokenRevoked,
			fmt.Errorf(
				"decision token revoked: %s",
				tok.TokenID,
			),
		)
	}

	// ============================================================
	// Step 11: Token expiry
	// ============================================================

	if now > tok.ExpiresAt+skew {
		return DenyDecision(
			ReasonTokenExpired,
			fmt.Errorf(
				"token expired: now=%d expires_at=%d",
				now,
				tok.ExpiresAt,
			),
		)
	}

	// ============================================================
	// Step 12: Token ↔ Envelope binding
	// ============================================================

	if tok.EnvelopeID != env.EnvelopeID {
		return DenyDecision(
			ReasonEnvelopeBinding,
			fmt.Errorf(
				"token envelope_id mismatch: token=%s envelope=%s",
				tok.EnvelopeID,
				env.EnvelopeID,
			),
		)
	}

	// ============================================================
	// Step 13: Signed token decision
	// ============================================================

	switch tok.Decision {

	case "DENY":
		return DenyDecision(
			ReasonPolicyDenied,
			errors.New("decision token explicitly denies action"),
		)

	case "CHECKPOINT":
		return CheckpointDecision(
			ReasonHumanRequired,
			errors.New(
				"decision token requires human checkpoint",
			),
		)

	case "ALLOW":
		// Continue with binding / constraints.

	default:
		return DenyDecision(
			ReasonPolicyDenied,
			fmt.Errorf(
				"unsupported token decision: %s",
				tok.Decision,
			),
		)
	}

	// ============================================================
	// Step 14: Action Hash Binding
	// ============================================================

	if len(req.ProposedAction) == 0 {
		return DenyDecision(
			ReasonInvalidAction,
			errors.New("missing proposed action"),
		)
	}

	canonicalAction, err := wire.CanonicalizeJSON(
		req.ProposedAction,
	)
	if err != nil {
		return DenyDecision(
			ReasonInvalidAction,
			fmt.Errorf(
				"action canonicalization failed: %w",
				err,
			),
		)
	}

	computedActionHash := wire.SHA256Hex(
		canonicalAction,
	)

	if tok.ActionHash != computedActionHash {
		return DenyDecision(
			ReasonSignatureFailure,
			fmt.Errorf(
				"action_hash mismatch: expected=%s token=%s",
				computedActionHash,
				tok.ActionHash,
			),
		)
	}

	// ============================================================
	// Step 15: Request binding
	// ============================================================

	if !p.ScopeGuard.MatchRequestConstraints(
		tok.Constraints,
		req,
	) {
		return DenyDecision(
			ReasonScopeExceeded,
			errors.New(
				"request does not match token constraints",
			),
		)
	}

	// ============================================================
	// Step 16: Scope containment / boundary matrix
	// ============================================================

	if !p.ScopeGuard.CheckBoundary(
		env.Scope,
		req,
	) {
		return DenyDecision(
			ReasonScopeExceeded,
			errors.New(
				"boundary crossing / scope exceeded",
			),
		)
	}

	// ============================================================
	// Step 17: Token usage / replay protection
	// ============================================================

	maxUses := 1

	if tok.Constraints != nil &&
		tok.Constraints.MaxUses != nil {

		maxUses = *tok.Constraints.MaxUses
	}

	if !p.BudgetLedger.ConsumeToken(
		tok.TokenID,
		maxUses,
	) {
		return DenyDecision(
			ReasonBudgetExhausted,
			fmt.Errorf(
				"token budget exhausted or replayed: token_id=%s max_uses=%d",
				tok.TokenID,
				maxUses,
			),
		)
	}

	// ============================================================
	// Step 18: Provenance acceptance
	// ============================================================

	if err := p.ProvenanceLog.Accept(
		env,
		tok,
		req,
	); err != nil {
		return DenyDecision(
			ReasonTraceabilityFail,
			fmt.Errorf(
				"provenance acceptance failed: %w",
				err,
			),
		)
	}

	// ============================================================
	// Final decision
	// ============================================================

	return AllowDecision()
}

// deny is retained temporarily for compatibility with existing code
// that may still call the old helper.
//
// New code should prefer DenyDecision().
func deny(reason string, err error) Decision {
	return DenyDecision(reason, err)
}
