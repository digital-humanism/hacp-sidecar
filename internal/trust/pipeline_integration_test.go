package trust_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/trust"
	"hacp-sidecar/internal/wire"
)

const integrationSignerKeyID = "ph1a-integration-key"

func TestAtomicTrustStorePipelineValidSignerPreservesExistingBehavior(t *testing.T) {
	publicKey, privateKey := generateIntegrationKey(t)
	store := newIntegrationTrustStore(t, integrationSignerKeyID, publicKey)
	revocations := evaluate.NewInMemoryRevocationStore()
	pipeline := evaluate.NewPipeline(store, revocations, nil, nil, nil)

	env := parseSignedIntegrationEnvelope(t, privateKey, integrationSignerKeyID)
	decision := pipeline.Evaluate(context.Background(), env, nil, integrationRequest())

	// A human envelope without a DecisionToken is rejected later by the
	// existing policy path. Reaching POLICY_DENIED proves that key resolution,
	// key revocation lookup, and Ed25519 verification all succeeded unchanged.
	if decision.Outcome != evaluate.OutcomeDeny {
		t.Fatalf("Outcome = %q, want %q", decision.Outcome, evaluate.OutcomeDeny)
	}
	if decision.ReasonCode != evaluate.ReasonPolicyDenied {
		t.Fatalf("ReasonCode = %q, want %q", decision.ReasonCode, evaluate.ReasonPolicyDenied)
	}
}

func TestAtomicTrustStorePipelineUnknownSignerFailsClosed(t *testing.T) {
	trustedPublicKey, _ := generateIntegrationKey(t)
	_, untrustedPrivateKey := generateIntegrationKey(t)

	store := newIntegrationTrustStore(t, "different-trusted-key", trustedPublicKey)
	pipeline := evaluate.NewPipeline(
		store,
		evaluate.NewInMemoryRevocationStore(),
		nil,
		nil,
		nil,
	)

	env := parseSignedIntegrationEnvelope(t, untrustedPrivateKey, integrationSignerKeyID)
	decision := pipeline.Evaluate(context.Background(), env, nil, integrationRequest())

	assertIntegrationDenyReason(t, decision, evaluate.ReasonSignatureFailure)
}

func TestAtomicTrustStorePipelineRevokedSignerFailsClosed(t *testing.T) {
	publicKey, privateKey := generateIntegrationKey(t)
	store := newIntegrationTrustStore(t, integrationSignerKeyID, publicKey)
	revocations := evaluate.NewInMemoryRevocationStore()
	revocations.RevokeKey(integrationSignerKeyID)
	pipeline := evaluate.NewPipeline(store, revocations, nil, nil, nil)

	env := parseSignedIntegrationEnvelope(t, privateKey, integrationSignerKeyID)
	decision := pipeline.Evaluate(context.Background(), env, nil, integrationRequest())

	assertIntegrationDenyReason(t, decision, evaluate.ReasonKeyRevoked)
}

func TestAtomicTrustStorePipelineBadSignatureFailsClosed(t *testing.T) {
	publicKey, privateKey := generateIntegrationKey(t)
	store := newIntegrationTrustStore(t, integrationSignerKeyID, publicKey)
	pipeline := evaluate.NewPipeline(
		store,
		evaluate.NewInMemoryRevocationStore(),
		nil,
		nil,
		nil,
	)

	env := parseSignedIntegrationEnvelope(t, privateKey, integrationSignerKeyID)
	if len(env.Signature) == 0 {
		t.Fatal("parsed envelope has empty signature")
	}
	env.Signature[0] ^= 0xff

	decision := pipeline.Evaluate(context.Background(), env, nil, integrationRequest())
	assertIntegrationDenyReason(t, decision, evaluate.ReasonSignatureFailure)
}

func generateIntegrationKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	return publicKey, privateKey
}

func newIntegrationTrustStore(
	t *testing.T,
	keyID string,
	publicKey ed25519.PublicKey,
) *trust.AtomicTrustStore {
	t.Helper()

	snapshot, err := trust.NewSnapshot(
		1,
		time.Unix(1, 0).UTC(),
		"ph1a-pipeline-integration-test",
		[]trust.KeyBinding{{KeyID: keyID, PublicKey: publicKey}},
	)
	if err != nil {
		t.Fatalf("trust.NewSnapshot() error = %v", err)
	}

	store, err := trust.NewAtomicTrustStore(snapshot)
	if err != nil {
		t.Fatalf("trust.NewAtomicTrustStore() error = %v", err)
	}
	return store
}

func parseSignedIntegrationEnvelope(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	signerKeyID string,
) *wire.IntentEnvelope {
	t.Helper()

	unsigned := map[string]any{
		"hacp_version":     "0.9",
		"envelope_id":      "ph1a-pipeline-envelope",
		"principal":        "ph1a-human",
		"principal_kind":   "human",
		"intent_statement": "PH-1A pipeline compatibility test",
		"scope": map[string]any{
			"verbs":            []string{"read"},
			"resource_classes": []string{"test"},
			"audiences":        []string{"internal"},
			"reversibility":    []string{"reversible"},
			"externality":      []string{"internal"},
			"data_classes":     []string{"internal"},
		},
		"issued_at":     int64(1000),
		"expires_at":    int64(2000),
		"signer_key_id": signerKeyID,
	}

	unsignedJSON, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatalf("json.Marshal(unsigned envelope) error = %v", err)
	}
	canonical, err := wire.CanonicalizeJSON(unsignedJSON)
	if err != nil {
		t.Fatalf("wire.CanonicalizeJSON(unsigned envelope) error = %v", err)
	}

	signature := ed25519.Sign(privateKey, canonical)
	unsigned["signature"] = wire.Base64URLEncode(signature)

	signedJSON, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatalf("json.Marshal(signed envelope) error = %v", err)
	}

	env, err := wire.ParseIntentEnvelope(signedJSON)
	if err != nil {
		t.Fatalf("wire.ParseIntentEnvelope() error = %v", err)
	}
	return env
}

func integrationRequest() *evaluate.RequestContext {
	return &evaluate.RequestContext{
		Clock: 1500,
	}
}

func assertIntegrationDenyReason(t *testing.T, decision evaluate.Decision, reason string) {
	t.Helper()
	if decision.Outcome != evaluate.OutcomeDeny {
		t.Fatalf("Outcome = %q, want %q", decision.Outcome, evaluate.OutcomeDeny)
	}
	if decision.ReasonCode != reason {
		t.Fatalf("ReasonCode = %q, want %q", decision.ReasonCode, reason)
	}
}
