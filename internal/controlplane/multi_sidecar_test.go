package controlplane

import (
	"context"
	"net"
	"testing"
	"time"

	controlplanev1 "hacp-sidecar/gen/controlplane/v1"
	"hacp-sidecar/internal/budget"
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/wire"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestTwoSidecarsConvergeOnDistributedRevocation(t *testing.T) {
	// -------------------------------------------------------------
	// 1. Canonical valid authority.
	// -------------------------------------------------------------

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

	if tok == nil {
		t.Fatal("golden decision token is nil")
	}

	// -------------------------------------------------------------
	// 2. One distributed control-plane.
	// -------------------------------------------------------------

	journal := NewJournal()

	listener := bufconn.Listen(testBufferSize)

	grpcServer := grpc.NewServer()

	controlplanev1.RegisterControlPlaneServer(
		grpcServer,
		NewServer(journal),
	)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := grpc.NewClient(
		"passthrough:///hacp-control-plane",
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
		grpc.WithContextDialer(
			func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			},
		),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	defer conn.Close()

	client := controlplanev1.NewControlPlaneClient(conn)

	// -------------------------------------------------------------
	// 3. Two completely independent sidecar-local stores.
	// -------------------------------------------------------------

	storeA := NewRevocationStoreAdapter()
	storeB := NewRevocationStoreAdapter()

	stateA := NewControlState(5 * time.Second)
	stateB := NewControlState(5 * time.Second)

	subscriberA := NewSubscriber(
		client,
		storeA,
		"sidecar-A",
	)
	subscriberA.SetControlState(stateA)

	subscriberB := NewSubscriber(
		client,
		storeB,
		"sidecar-B",
	)
	subscriberB.SetControlState(stateB)

	// Keep reconnect paths fast if the test environment causes a transient
	// stream interruption.
	subscriberA.reconnectInitialBackoff = time.Millisecond
	subscriberA.reconnectMaxBackoff = time.Millisecond

	subscriberB.reconnectInitialBackoff = time.Millisecond
	subscriberB.reconnectMaxBackoff = time.Millisecond

	errA := make(chan error, 1)
	errB := make(chan error, 1)

	go func() {
		errA <- subscriberA.Run(ctx)
	}()

	go func() {
		errB <- subscriberB.Run(ctx)
	}()

	// Both sidecars must have loaded the initial revision-0 snapshot.
	requireEventually(t, func() bool {
		return !stateA.LastUpdate().IsZero() &&
			!stateB.LastUpdate().IsZero()
	})

	if subscriberA.LastSeenRevision() != 0 {
		t.Fatalf(
			"sidecar A initial revision = %d, want 0",
			subscriberA.LastSeenRevision(),
		)
	}

	if subscriberB.LastSeenRevision() != 0 {
		t.Fatalf(
			"sidecar B initial revision = %d, want 0",
			subscriberB.LastSeenRevision(),
		)
	}

	// -------------------------------------------------------------
	// 4. Construct two independent evaluator pipelines.
	//
	// Same authority and same distributed control-plane, but separate:
	//
	//   RevocationStore
	//   BudgetLedger
	//   ControlState
	//
	// exactly as separate sidecars would have.
	// -------------------------------------------------------------

	keyResolverA := wire.NewStaticKeyResolver()

	if err := keyResolverA.AddKeyFromHex(
		"key-ed25519-test-001",
		distributedTestPublicKeyHex,
	); err != nil {
		t.Fatalf("sidecar A AddKeyFromHex: %v", err)
	}

	keyResolverB := wire.NewStaticKeyResolver()

	if err := keyResolverB.AddKeyFromHex(
		"key-ed25519-test-001",
		distributedTestPublicKeyHex,
	); err != nil {
		t.Fatalf("sidecar B AddKeyFromHex: %v", err)
	}

	pipelineA := evaluate.NewPipeline(
		keyResolverA,
		storeA,
		budget.NewLedger(),
		evaluate.NewDefaultScopeGuard(),
		provenance.NewNoopWriter(),
	)

	pipelineB := evaluate.NewPipeline(
		keyResolverB,
		storeB,
		budget.NewLedger(),
		evaluate.NewDefaultScopeGuard(),
		provenance.NewNoopWriter(),
	)

	policyA := &evaluate.PolicyContext{
		Clock:            vector.PolicyContext.Clock,
		HumanRequired:    vector.PolicyContext.HumanRequired,
		ConsequenceClass: vector.PolicyContext.ConsequenceClass,
		RiskClass:        vector.PolicyContext.RiskClass,
	}

	policyB := &evaluate.PolicyContext{
		Clock:            vector.PolicyContext.Clock,
		HumanRequired:    vector.PolicyContext.HumanRequired,
		ConsequenceClass: vector.PolicyContext.ConsequenceClass,
		RiskClass:        vector.PolicyContext.RiskClass,
	}

	reqA := &evaluate.RequestContext{
		Method:         "EVALUATE",
		Path:           "/gate-e/multi-sidecar/A",
		RequestID:      "gate-e-sidecar-A",
		Clock:          vector.PolicyContext.Clock,
		ProposedAction: vector.Inputs.ProposedAction,
		Policy:         policyA,
	}

	reqB := &evaluate.RequestContext{
		Method:         "EVALUATE",
		Path:           "/gate-e/multi-sidecar/B",
		RequestID:      "gate-e-sidecar-B",
		Clock:          vector.PolicyContext.Clock,
		ProposedAction: vector.Inputs.ProposedAction,
		Policy:         policyB,
	}

	// -------------------------------------------------------------
	// 5. Both independent sidecars initially ALLOW.
	// -------------------------------------------------------------

	beforeA := pipelineA.Evaluate(
		context.Background(),
		env,
		tok,
		reqA,
	)

	if !beforeA.Allow {
		t.Fatalf(
			"sidecar A baseline = %q reason=%q error=%v, want ALLOW",
			beforeA.Outcome,
			beforeA.ReasonCode,
			beforeA.Error,
		)
	}

	beforeB := pipelineB.Evaluate(
		context.Background(),
		env,
		tok,
		reqB,
	)

	if !beforeB.Allow {
		t.Fatalf(
			"sidecar B baseline = %q reason=%q error=%v, want ALLOW",
			beforeB.Outcome,
			beforeB.ReasonCode,
			beforeB.Error,
		)
	}

	// -------------------------------------------------------------
	// 6. One distributed revoke.
	// -------------------------------------------------------------

	event, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		tok.TokenID,
	)
	if err != nil {
		t.Fatalf("distributed revoke: %v", err)
	}

	if !created {
		t.Fatal(
			"distributed revoke did not create revision",
		)
	}

	if event.Revision != 1 {
		t.Fatalf(
			"distributed revision = %d, want 1",
			event.Revision,
		)
	}

	// -------------------------------------------------------------
	// 7. Both sidecars must converge to the exact same durable revision
	// and materialize the same security state.
	// -------------------------------------------------------------

	requireEventually(t, func() bool {
		return subscriberA.LastSeenRevision() == event.Revision &&
			subscriberB.LastSeenRevision() == event.Revision &&
			storeA.IsTokenRevoked(tok.TokenID) &&
			storeB.IsTokenRevoked(tok.TokenID)
	})

	if stateA.LastSeenRevision() != event.Revision {
		t.Fatalf(
			"sidecar A control-state revision = %d, want %d",
			stateA.LastSeenRevision(),
			event.Revision,
		)
	}

	if stateB.LastSeenRevision() != event.Revision {
		t.Fatalf(
			"sidecar B control-state revision = %d, want %d",
			stateB.LastSeenRevision(),
			event.Revision,
		)
	}

	// -------------------------------------------------------------
	// 8. Same authority must now produce the same security decision
	// independently on both sidecars.
	//
	// Token revocation is evaluated before token budget consumption, so the
	// second evaluation must terminate as TOKEN_REVOKED rather than replay /
	// budget exhaustion.
	// -------------------------------------------------------------

	afterA := pipelineA.Evaluate(
		context.Background(),
		env,
		tok,
		reqA,
	)

	afterB := pipelineB.Evaluate(
		context.Background(),
		env,
		tok,
		reqB,
	)

	if afterA.Allow {
		t.Fatal(
			"sidecar A returned ALLOW after distributed revoke",
		)
	}

	if afterB.Allow {
		t.Fatal(
			"sidecar B returned ALLOW after distributed revoke",
		)
	}

	if afterA.ReasonCode != evaluate.ReasonTokenRevoked {
		t.Fatalf(
			"sidecar A reason = %q, want %q",
			afterA.ReasonCode,
			evaluate.ReasonTokenRevoked,
		)
	}

	if afterB.ReasonCode != evaluate.ReasonTokenRevoked {
		t.Fatalf(
			"sidecar B reason = %q, want %q",
			afterB.ReasonCode,
			evaluate.ReasonTokenRevoked,
		)
	}

	// -------------------------------------------------------------
	// 9. Formal convergence assertions.
	// -------------------------------------------------------------

	if subscriberA.LastSeenRevision() !=
		subscriberB.LastSeenRevision() {

		t.Fatalf(
			"sidecar revisions diverged: A=%d B=%d",
			subscriberA.LastSeenRevision(),
			subscriberB.LastSeenRevision(),
		)
	}

	if afterA.ReasonCode != afterB.ReasonCode {
		t.Fatalf(
			"sidecar decisions diverged: A=%q B=%q",
			afterA.ReasonCode,
			afterB.ReasonCode,
		)
	}

	cancel()

	select {
	case <-errA:
	case <-time.After(time.Second):
		t.Fatal(
			"sidecar A subscriber did not stop",
		)
	}

	select {
	case <-errB:
	case <-time.After(time.Second):
		t.Fatal(
			"sidecar B subscriber did not stop",
		)
	}
}
