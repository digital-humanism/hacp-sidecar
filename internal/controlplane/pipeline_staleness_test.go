package controlplane

import (
	"context"
	"testing"
	"time"

	"hacp-sidecar/internal/budget"
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/wire"
)

func TestPipelineFailsClosedWhenControlStateStale(t *testing.T) {
	vector := loadDistributedGoldenVector(t)

	env, err := wire.ParseIntentEnvelope(
		vector.Inputs.IntentEnvelope,
	)
	if err != nil {
		t.Fatalf("ParseIntentEnvelope: %v", err)
	}

	tok, err := wire.ParseDecisionToken(
		vector.Inputs.DecisionToken,
	)
	if err != nil {
		t.Fatalf("ParseDecisionToken: %v", err)
	}

	keyResolver := wire.NewStaticKeyResolver()

	if err := keyResolver.AddKeyFromHex(
		"key-ed25519-test-001",
		distributedTestPublicKeyHex,
	); err != nil {
		t.Fatalf("AddKeyFromHex: %v", err)
	}

	revocation := NewRevocationStoreAdapter()

	pipeline := evaluate.NewPipeline(
		keyResolver,
		revocation,
		budget.NewLedger(),
		evaluate.NewDefaultScopeGuard(),
		provenance.NewNoopWriter(),
	)

	// Deterministic evaluation clock.
	evaluationTime := time.Unix(
		vector.PolicyContext.Clock,
		0,
	)

	controlState := NewControlState(
		5 * time.Second,
	)

	pipeline.ControlState = controlState

	policy := &evaluate.PolicyContext{
		Clock:            vector.PolicyContext.Clock,
		HumanRequired:    vector.PolicyContext.HumanRequired,
		ConsequenceClass: vector.PolicyContext.ConsequenceClass,
		RiskClass:        vector.PolicyContext.RiskClass,
	}

	reqCtx := &evaluate.RequestContext{
		Method:         "EVALUATE",
		Path:           "/gate-e/control-state-staleness",
		RequestID:      "gate-e-staleness",
		Timestamp:      evaluationTime,
		Clock:          vector.PolicyContext.Clock,
		ProposedAction: vector.Inputs.ProposedAction,
		Policy:         policy,
	}

	// -------------------------------------------------------------
	// Fresh distributed control state.
	// -------------------------------------------------------------

	controlState.MarkSnapshot(
		0,
		evaluationTime,
	)

	before := pipeline.Evaluate(
		context.Background(),
		env,
		tok,
		reqCtx,
	)

	if !before.Allow {
		t.Fatalf(
			"fresh-state decision = %q reason=%q error=%v, want ALLOW",
			before.Outcome,
			before.ReasonCode,
			before.Error,
		)
	}

	// -------------------------------------------------------------
	// Move evaluation clock beyond max staleness.
	//
	// No new snapshot/event/heartbeat has been observed.
	// -------------------------------------------------------------

	staleTime := evaluationTime.Add(6 * time.Second)

	reqCtx.Clock = staleTime.Unix()
	reqCtx.Timestamp = staleTime

	// Keep every clock source used by EffectiveClock() aligned.
	reqCtx.Policy.Clock = staleTime.Unix()

	after := pipeline.Evaluate(
		context.Background(),
		env,
		tok,
		reqCtx,
	)

	if after.Allow {
		t.Fatal(
			"stale control state produced ALLOW, want DENY",
		)
	}

	if after.ReasonCode != evaluate.ReasonControlStateStale {
		t.Fatalf(
			"reason = %q, want %q",
			after.ReasonCode,
			evaluate.ReasonControlStateStale,
		)
	}
}

func TestPipelineWithoutControlStateGuardPreservesStandaloneBehavior(
	t *testing.T,
) {
	vector := loadDistributedGoldenVector(t)

	env, err := wire.ParseIntentEnvelope(
		vector.Inputs.IntentEnvelope,
	)
	if err != nil {
		t.Fatalf("ParseIntentEnvelope: %v", err)
	}

	tok, err := wire.ParseDecisionToken(
		vector.Inputs.DecisionToken,
	)
	if err != nil {
		t.Fatalf("ParseDecisionToken: %v", err)
	}

	keyResolver := wire.NewStaticKeyResolver()

	if err := keyResolver.AddKeyFromHex(
		"key-ed25519-test-001",
		distributedTestPublicKeyHex,
	); err != nil {
		t.Fatalf("AddKeyFromHex: %v", err)
	}

	pipeline := evaluate.NewPipeline(
		keyResolver,
		NewRevocationStoreAdapter(),
		budget.NewLedger(),
		evaluate.NewDefaultScopeGuard(),
		provenance.NewNoopWriter(),
	)

	// Intentionally:
	//
	// pipeline.ControlState == nil
	//
	// Standalone/conformance semantics must remain unchanged.

	policy := &evaluate.PolicyContext{
		Clock:            vector.PolicyContext.Clock,
		HumanRequired:    vector.PolicyContext.HumanRequired,
		ConsequenceClass: vector.PolicyContext.ConsequenceClass,
		RiskClass:        vector.PolicyContext.RiskClass,
	}

	reqCtx := &evaluate.RequestContext{
		Method:         "EVALUATE",
		Path:           "/gate-e/no-control-state",
		RequestID:      "gate-e-no-control-state",
		Clock:          vector.PolicyContext.Clock,
		ProposedAction: vector.Inputs.ProposedAction,
		Policy:         policy,
	}

	got := pipeline.Evaluate(
		context.Background(),
		env,
		tok,
		reqCtx,
	)

	if !got.Allow {
		t.Fatalf(
			"standalone decision = %q reason=%q error=%v, want ALLOW",
			got.Outcome,
			got.ReasonCode,
			got.Error,
		)
	}
}
