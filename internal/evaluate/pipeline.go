package evaluate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hacp-sidecar/internal/wire"
)

// Pipeline implements the normative HACP evaluation order
type Pipeline struct {
	KeyResolver   KeyResolver
	Revocation    RevocationStore
	BudgetLedger  BudgetLedger
	ScopeGuard    ScopeGuard
	ProvenanceLog ProvenanceWriter

	// Configuration
	MaxClockSkewSeconds      int64
	MaxRevocationStalenessMs int64
}

// NewPipeline creates a new evaluation pipeline
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

// Evaluate executes the normative HACP verification order
// Per HACP-SPEC-0.9-draft.md §5.1 and wire/crypto-profile.md
func (p *Pipeline) Evaluate(ctx context.Context, env *wire.IntentEnvelope, tok *wire.DecisionToken, req *RequestContext) Decision {
	// Step 1: Schema / required fields (handled during wire parsing, but double-check critical fields)
	if env == nil || tok == nil {
		return deny(ReasonInvalidEnvelope, errors.New("nil envelope or token"))
	}

	// Step 2: Check token decision
	if tok.Decision != "ALLOW" {
		return deny(ReasonPolicyDenied, fmt.Errorf("token decision is %s", tok.Decision))
	}

	// Step 3: Resolve signer_key_id & check Key Revocation (BEFORE signature verification)
	envKey, err := p.KeyResolver.ResolveKey(env.SignerKeyID)
	if err != nil || envKey == nil {
		return deny(ReasonSignatureFailure, errors.New("envelope key not found"))
	}
	if p.Revocation.IsKeyRevoked(env.SignerKeyID) {
		return deny(ReasonKeyRevoked, errors.New("envelope signer key revoked"))
	}

	tokKey, err := p.KeyResolver.ResolveKey(tok.SignerKeyID)
	if err != nil || tokKey == nil {
		return deny(ReasonSignatureFailure, errors.New("token key not found"))
	}
	if p.Revocation.IsKeyRevoked(tok.SignerKeyID) {
		return deny(ReasonKeyRevoked, errors.New("token signer key revoked"))
	}

	// Step 4: Verify Signatures (only if keys are valid)
	if !wire.VerifyEd25519(envKey, env.CanonicalPayload(), env.Signature) {
		return deny(ReasonSignatureFailure, errors.New("envelope signature invalid"))
	}
	if !wire.VerifyEd25519(tokKey, tok.CanonicalPayload(), tok.Signature) {
		return deny(ReasonSignatureFailure, errors.New("token signature invalid"))
	}

	// Step 5: Envelope / Token Revocation Checks (AFTER signature)
	if p.Revocation.IsEnvelopeRevoked(env.EnvelopeID) {
		return deny(ReasonEnvelopeRevoked, errors.New("envelope revoked"))
	}
	if p.Revocation.IsTokenRevoked(tok.TokenID) {
		return deny(ReasonTokenRevoked, errors.New("token revoked"))
	}

	// Step 6: Expiry Checks with clock skew tolerance
	now := time.Now().Unix()
	skew := p.MaxClockSkewSeconds

	if now > env.ExpiresAt+skew {
		return deny(ReasonEnvelopeExpired, errors.New("envelope expired"))
	}
	if now > tok.ExpiresAt+skew {
		return deny(ReasonTokenExpired, errors.New("token expired"))
	}

	// Step 7: Action Hash Binding (Token MUST match Envelope)
	envActionHash := wire.SHA256Hex(env.CanonicalPayload())
	if tok.ActionHash != envActionHash {
		return deny(ReasonSignatureFailure, errors.New("action_hash mismatch"))
	}

	// Step 8: Request Binding (Constraints match)
	if !p.ScopeGuard.MatchRequestConstraints(tok.Constraints, req) {
		return deny(ReasonScopeExceeded, errors.New("request binding mismatch"))
	}

	// Step 9: Scope Containment (Boundary Matrix)
	if !p.ScopeGuard.CheckBoundary(env.Scope, req) {
		return deny(ReasonScopeExceeded, errors.New("boundary crossing / scope exceeded"))
	}

	// Step 10: Budget & Replay Protection
	maxUses := 1 // Default single-use
	if tok.Constraints != nil && tok.Constraints.MaxUses != nil {
		maxUses = *tok.Constraints.MaxUses
	}
	if !p.BudgetLedger.Consume(tok.TokenID, maxUses) {
		return deny(ReasonBudgetExhausted, errors.New("budget exhausted or token replayed"))
	}

	// Step 11: Provenance Ring Buffer Acceptance (BEFORE forwarding)
	if err := p.ProvenanceLog.Accept(env, tok, req); err != nil {
		return deny(ReasonTraceabilityFail, errors.New("provenance buffer full/unavailable"))
	}

	return Decision{Allow: true}
}

func deny(reason string, err error) Decision {
	return Decision{
		Allow:      false,
		ReasonCode: reason,
		Error:      err,
	}
}
