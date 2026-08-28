package evaluate

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"

	"hacp-sidecar/internal/wire"
)

const (
	keyID      = "ar3-token-precedence-key"
	actionHash = "ar3-precedence-action-hash"
)

func TestDecisionTokenSignatureFailurePrecedesSignedDenyDecision(
	t *testing.T,
) {
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

	validDenyToken := signedPrecedenceToken(
		t,
		privateKey,
		keyID,
		env.EnvelopeID,
		actionHash,
	)

	req := &RequestContext{
		Clock: 1500,
	}

	t.Run("authenticated DENY token reaches policy decision", func(t *testing.T) {
		decision := pipeline.Evaluate(
			context.Background(),
			env,
			validDenyToken,
			req,
		)

		if decision.Outcome != OutcomeDeny {
			t.Fatalf(
				"Outcome = %q, want %q",
				decision.Outcome,
				OutcomeDeny,
			)
		}

		if decision.ReasonCode != ReasonPolicyDenied {
			t.Fatalf(
				"ReasonCode = %q, want %q",
				decision.ReasonCode,
				ReasonPolicyDenied,
			)
		}
	})

	t.Run("invalid signature precedes DENY semantics", func(t *testing.T) {
		corrupted := *validDenyToken

		corrupted.Signature = append(
			[]byte(nil),
			validDenyToken.Signature...,
		)

		if len(corrupted.Signature) == 0 {
			t.Fatal("DecisionToken signature is empty")
		}

		corrupted.Signature[0] ^= 0xff

		decision := pipeline.Evaluate(
			context.Background(),
			env,
			&corrupted,
			req,
		)

		if decision.Outcome != OutcomeDeny {
			t.Fatalf(
				"Outcome = %q, want %q",
				decision.Outcome,
				OutcomeDeny,
			)
		}

		if decision.ReasonCode != ReasonSignatureFailure {
			t.Fatalf(
				"ReasonCode = %q, want %q",
				decision.ReasonCode,
				ReasonSignatureFailure,
			)
		}
	})
}

func signedPrecedenceEnvelope(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	keyID string,
) *wire.IntentEnvelope {
	t.Helper()

	unsigned := map[string]any{
		"hacp_version":     "0.9",
		"envelope_id":      "ar3-precedence-envelope",
		"principal":        "ar3-human",
		"principal_kind":   "human",
		"intent_statement": "AR-3 DecisionToken precedence fixture",
		"scope": map[string]any{
			"verbs": []string{
				"read",
			},
			"resource_classes": []string{
				"document",
			},
			"audiences": []string{
				"internal",
			},
			"reversibility": []string{
				"reversible",
			},
			"externality": []string{
				"internal",
			},
			"data_classes": []string{
				"internal",
			},
		},
		"issued_at":     int64(1000),
		"expires_at":    int64(2000),
		"signer_key_id": keyID,
	}

	signedJSON := signPrecedenceObject(
		t,
		privateKey,
		unsigned,
	)

	env, err := wire.ParseIntentEnvelope(signedJSON)
	if err != nil {
		t.Fatalf(
			"ParseIntentEnvelope() error = %v",
			err,
		)
	}

	return env
}

func signedPrecedenceToken(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	keyID string,
	envelopeID string,
	actionHash string,
) *wire.DecisionToken {
	t.Helper()

	unsigned := map[string]any{
		"hacp_version":  "0.9",
		"token_id":      "ar3-precedence-token",
		"envelope_id":   envelopeID,
		"action_hash":   actionHash,
		"policy_digest": "ar3-precedence-policy",
		"principal":     "ar3-human",
		"signer_key_id": keyID,
		"issued_at":     int64(1000),
		"expires_at":    int64(2000),
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

func signPrecedenceObject(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	unsigned map[string]any,
) []byte {
	t.Helper()

	unsignedJSON, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatalf(
			"json.Marshal(unsigned) error = %v",
			err,
		)
	}

	canonical, err := wire.CanonicalizeJSON(unsignedJSON)
	if err != nil {
		t.Fatalf(
			"CanonicalizeJSON(unsigned) error = %v",
			err,
		)
	}

	unsigned["signature"] = wire.Base64URLEncode(
		ed25519.Sign(
			privateKey,
			canonical,
		),
	)

	signedJSON, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatalf(
			"json.Marshal(signed) error = %v",
			err,
		)
	}

	return signedJSON
}
