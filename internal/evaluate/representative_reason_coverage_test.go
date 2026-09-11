package evaluate

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"hacp-sidecar/internal/wire"
)

func TestDirectEnvelopeRevokedReasonCode(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}

	keyResolver := wire.NewStaticKeyResolver()
	if err := keyResolver.AddKey(keyID, publicKey); err != nil {
		t.Fatalf("AddKey() error = %v", err)
	}

	revocation := NewInMemoryRevocationStore()

	pipeline := NewPipeline(
		keyResolver,
		revocation,
		nil,
		nil,
		nil,
	)

	env := signedPrecedenceEnvelope(
		t,
		privateKey,
		keyID,
	)

	revocation.RevokeEnvelope(env.EnvelopeID)

	decision := pipeline.Evaluate(
		context.Background(),
		env,
		nil,
		&RequestContext{
			Clock: 1500,
		},
	)

	if decision.Outcome != OutcomeDeny {
		t.Fatalf(
			"Outcome = %q, want %q",
			decision.Outcome,
			OutcomeDeny,
		)
	}

	if decision.ReasonCode != ReasonEnvelopeRevoked {
		t.Fatalf(
			"ReasonCode = %q, want %q",
			decision.ReasonCode,
			ReasonEnvelopeRevoked,
		)
	}
}

func TestDecisionTokenExpiredReasonCode(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}

	keyResolver := wire.NewStaticKeyResolver()
	if err := keyResolver.AddKey(keyID, publicKey); err != nil {
		t.Fatalf("AddKey() error = %v", err)
	}

	pipeline := NewPipeline(
		keyResolver,
		NewInMemoryRevocationStore(),
		nil,
		nil,
		nil,
	)

	env := signedPrecedenceEnvelope(
		t,
		privateKey,
		keyID,
	)

	tok := signedExpiredReasonToken(
		t,
		privateKey,
		keyID,
		env.EnvelopeID,
		actionHash,
	)

	decision := pipeline.Evaluate(
		context.Background(),
		env,
		tok,
		&RequestContext{
			Clock: 1500,
		},
	)

	if decision.Outcome != OutcomeDeny {
		t.Fatalf(
			"Outcome = %q, want %q",
			decision.Outcome,
			OutcomeDeny,
		)
	}

	if decision.ReasonCode != ReasonTokenExpired {
		t.Fatalf(
			"ReasonCode = %q, want %q",
			decision.ReasonCode,
			ReasonTokenExpired,
		)
	}
}

func signedExpiredReasonToken(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	keyID string,
	envelopeID string,
	actionHash string,
) *wire.DecisionToken {
	t.Helper()

	unsigned := map[string]any{
		"hacp_version":  "0.9",
		"token_id":      "representative-expired-token",
		"envelope_id":   envelopeID,
		"action_hash":   actionHash,
		"policy_digest": "representative-reason-policy",
		"principal":     "ar3-human",
		"signer_key_id": keyID,
		"issued_at":     int64(1000),
		"expires_at":    int64(1400),
		"decision":      "DENY",
	}

	signedJSON := signPrecedenceObject(
		t,
		privateKey,
		unsigned,
	)

	tok, err := wire.ParseDecisionToken(signedJSON)
	if err != nil {
		t.Fatalf(
			"ParseDecisionToken() error = %v",
			err,
		)
	}

	return tok
}
