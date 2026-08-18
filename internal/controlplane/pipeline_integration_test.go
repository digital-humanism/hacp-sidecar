package controlplane

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
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

const distributedTestPublicKeyHex = "9d17f1bbcc0845865e670f526413fb7a510380798fe300b6c98e28f3a3b0fdb3"

type distributedVector struct {
	TestID string `json:"test_id"`

	Inputs struct {
		IntentEnvelope json.RawMessage `json:"intent_envelope"`
		ProposedAction json.RawMessage `json:"proposed_action"`
		DecisionToken  json.RawMessage `json:"decision_token"`
	} `json:"inputs"`

	PolicyContext struct {
		Clock              int64    `json:"clock"`
		HumanRequired      bool     `json:"human_required"`
		HumanRequiredVerbs []string `json:"human_required_verbs"`
		ConsequenceClass   string   `json:"consequence_class"`
		RiskClass          string   `json:"risk_class"`
	} `json:"policy_context"`

	Expected struct {
		Outcome     string   `json:"outcome"`
		ReasonCodes []string `json:"reason_codes"`
	} `json:"expected"`
}

func TestPipelineDistributedTokenRevocation(t *testing.T) {
	// -----------------------------------------------------------------
	// 1. Load canonical HACP-Core v0.9.2 golden vector.
	// -----------------------------------------------------------------

	vector := loadDistributedGoldenVector(t)

	if vector.Expected.Outcome != "ALLOW" {
		t.Fatalf(
			"golden vector expected outcome = %q, want ALLOW",
			vector.Expected.Outcome,
		)
	}

	env, err := wire.ParseIntentEnvelope(vector.Inputs.IntentEnvelope)
	if err != nil {
		t.Fatalf("ParseIntentEnvelope: %v", err)
	}

	tok, err := wire.ParseDecisionToken(vector.Inputs.DecisionToken)
	if err != nil {
		t.Fatalf("ParseDecisionToken: %v", err)
	}

	if tok == nil {
		t.Fatal("golden vector decision_token is nil")
	}

	if tok.TokenID == "" {
		t.Fatal("golden vector token_id is empty")
	}

	// -----------------------------------------------------------------
	// 2. Start in-memory distributed control-plane over real gRPC.
	// -----------------------------------------------------------------

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

	// -----------------------------------------------------------------
	// 3. Construct the REAL evaluator pipeline.
	//
	// This mirrors cmd/hacp-conformance-runner.
	// -----------------------------------------------------------------

	keyResolver := wire.NewStaticKeyResolver()

	if err := keyResolver.AddKeyFromHex(
		"key-ed25519-test-001",
		distributedTestPublicKeyHex,
	); err != nil {
		t.Fatalf("AddKeyFromHex: %v", err)
	}

	revocation := NewRevocationStoreAdapter()

	budgetLedger := budget.NewLedger()
	scopeGuard := evaluate.NewDefaultScopeGuard()
	provLog := provenance.NewNoopWriter()

	pipeline := evaluate.NewPipeline(
		keyResolver,
		revocation,
		budgetLedger,
		scopeGuard,
		provLog,
	)

	policy := &evaluate.PolicyContext{
		Clock:            vector.PolicyContext.Clock,
		HumanRequired:    vector.PolicyContext.HumanRequired,
		ConsequenceClass: vector.PolicyContext.ConsequenceClass,
		RiskClass:        vector.PolicyContext.RiskClass,
	}

	reqCtx := &evaluate.RequestContext{
		Method:         "EVALUATE",
		Path:           "/gate-e/distributed-revocation",
		RequestID:      "gate-e-token-revocation",
		Timestamp:      time.Now(),
		Clock:          vector.PolicyContext.Clock,
		ProposedAction: vector.Inputs.ProposedAction,
		Policy:         policy,
	}

	// -----------------------------------------------------------------
	// 4. Baseline security outcome MUST be ALLOW.
	// -----------------------------------------------------------------

	before := pipeline.Evaluate(
		context.Background(),
		env,
		tok,
		reqCtx,
	)

	if !before.Allow {
		t.Fatalf(
			"baseline decision = %q allow=%v reason=%q error=%v, want ALLOW",
			before.Outcome,
			before.Allow,
			before.ReasonCode,
			before.Error,
		)
	}

	// -----------------------------------------------------------------
	// 5. Start sidecar revocation subscriber.
	//
	// Initial snapshot is revision 0.
	// Any revoke racing with snapshot->stream transition is recovered by
	// WatchRevocations(after_revision), so no sleep synchronization is
	// required.
	// -----------------------------------------------------------------

	subscriber := NewSubscriber(
		controlplanev1.NewControlPlaneClient(conn),
		revocation,
		"sidecar-gate-e-01",
	)

	subscriberErr := make(chan error, 1)

	go func() {
		subscriberErr <- subscriber.Run(ctx)
	}()

	// -----------------------------------------------------------------
	// 6. Commit distributed token revocation.
	// -----------------------------------------------------------------

	event, created, err := journal.Revoke(
		controlplanev1.RevocationKind_REVOCATION_KIND_TOKEN,
		tok.TokenID,
	)
	if err != nil {
		t.Fatalf("distributed RevokeToken: %v", err)
	}

	if !created {
		t.Fatal("distributed token revoke did not create revision")
	}

	if event.Revision != 1 {
		t.Fatalf(
			"revocation revision = %d, want 1",
			event.Revision,
		)
	}

	// -----------------------------------------------------------------
	// 7. Wait for deterministic convergence.
	// -----------------------------------------------------------------

	requireEventually(t, func() bool {
		return subscriber.LastSeenRevision() == event.Revision &&
			revocation.IsTokenRevoked(tok.TokenID)
	})

	// -----------------------------------------------------------------
	// 8. Evaluate the EXACT SAME signed authority again.
	//
	// Expected transition:
	//
	//     ALLOW
	//       ↓
	// distributed revoke
	//       ↓
	//     DENY / TOKEN_REVOKED
	// -----------------------------------------------------------------

	after := pipeline.Evaluate(
		context.Background(),
		env,
		tok,
		reqCtx,
	)

	if after.Allow {
		t.Fatalf(
			"decision after distributed revoke = %q allow=true, want DENY",
			after.Outcome,
		)
	}

	if after.ReasonCode != evaluate.ReasonTokenRevoked {
		t.Fatalf(
			"reason after distributed revoke = %q, want %q (error=%v)",
			after.ReasonCode,
			evaluate.ReasonTokenRevoked,
			after.Error,
		)
	}

	// -----------------------------------------------------------------
	// 9. Shutdown subscriber cleanly.
	// -----------------------------------------------------------------

	cancel()

	select {
	case <-subscriberErr:
	case <-time.After(time.Second):
		t.Fatal("subscriber did not stop after context cancellation")
	}
}

func loadDistributedGoldenVector(t *testing.T) distributedVector {
	t.Helper()

	// This test file lives at:
	//
	//   hacp-sidecar/internal/controlplane/pipeline_integration_test.go
	//
	// Both repositories are siblings under ...\GitHub\Dev:
	//
	//   hacp-sidecar
	//   hacp-spec
	//
	// runtime.Caller makes the test independent of the process working
	// directory used by `go test`.
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	vectorPath := filepath.Clean(
		filepath.Join(
			filepath.Dir(currentFile),
			"..",
			"..",
			"..",
			"hacp-spec",
			"vectors",
			"core_inv5_001_golden.json",
		),
	)

	raw, err := os.ReadFile(vectorPath)
	if err != nil {
		t.Fatalf(
			"read canonical vector %q: %v",
			vectorPath,
			err,
		)
	}

	var vector distributedVector

	if err := json.Unmarshal(raw, &vector); err != nil {
		t.Fatalf(
			"parse canonical vector %q: %v",
			vectorPath,
			err,
		)
	}

	if vector.TestID != "CORE-INV5-001" {
		t.Fatalf(
			"vector test_id = %q, want CORE-INV5-001",
			vector.TestID,
		)
	}

	return vector
}
