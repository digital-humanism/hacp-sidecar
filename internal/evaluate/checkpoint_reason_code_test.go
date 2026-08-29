package evaluate

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestCheckpointTimeoutReasonCode(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}

	env := signedPrecedenceEnvelope(
		t,
		privateKey,
		"checkpoint-timeout-test-key",
	)

	pipeline := NewPipeline(
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	req := &RequestContext{
		Clock: 1500,
		Checkpoint: &CheckpointContext{
			ID:        "checkpoint-timeout-test",
			State:     CheckpointStateOpen,
			ExpiresAt: 1400,
		},
	}

	decision := pipeline.Evaluate(
		context.Background(),
		env,
		nil,
		req,
	)

	if decision.Outcome != OutcomeDeny {
		t.Fatalf(
			"Outcome = %q, want %q",
			decision.Outcome,
			OutcomeDeny,
		)
	}

	if decision.ReasonCode != "CHECKPOINT_TIMEOUT" {
		t.Fatalf(
			"ReasonCode = %q, want %q",
			decision.ReasonCode,
			"CHECKPOINT_TIMEOUT",
		)
	}
}
