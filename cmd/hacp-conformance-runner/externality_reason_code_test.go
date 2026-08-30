package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"

	"hacp-sidecar/internal/budget"
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/wire"
)

func TestConformanceRunnerReportsBoundaryCrossingForExternalityViolation(
	t *testing.T,
) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}

	const keyID = "externality-reason-test-key"

	keyResolver := wire.NewStaticKeyResolver()

	if err := keyResolver.AddKeyFromHex(
		keyID,
		hex.EncodeToString(publicKey),
	); err != nil {
		t.Fatalf("AddKeyFromHex() error = %v", err)
	}

	revocation := evaluate.NewInMemoryRevocationStore()
	budgetLedger := budget.NewLedger()

	runner := &Runner{
		pipeline: evaluate.NewPipeline(
			keyResolver,
			revocation,
			budgetLedger,
			evaluate.NewDefaultScopeGuard(),
			provenance.NewNoopWriter(),
		),
		revocation:   revocation,
		keyResolver:  keyResolver,
		budgetLedger: budgetLedger,
	}

	unsignedEnvelope := map[string]any{
		"hacp_version":     "0.9",
		"envelope_id":      "externality-reason-envelope",
		"principal":        "externality-reason-system",
		"principal_kind":   "system",
		"intent_statement": "Externality boundary reason-code conformance",
		"scope": map[string]any{
			"verbs":            []string{"notify"},
			"resource_classes": []string{"report"},
			"audiences":        []string{"internal"},
			"reversibility":    []string{"reversible"},
			"externality":      []string{"internal"},
			"data_classes":     []string{"internal"},
		},
		"autonomy_budget": map[string]any{
			"max_actions": 1,
		},
		"issued_at":     int64(1786000000),
		"expires_at":    int64(1786003600),
		"signer_key_id": keyID,
	}

	unsignedJSON, err := json.Marshal(unsignedEnvelope)
	if err != nil {
		t.Fatalf("json.Marshal(unsigned envelope) error = %v", err)
	}

	canonicalEnvelope, err := wire.CanonicalizeJSON(unsignedJSON)
	if err != nil {
		t.Fatalf("wire.CanonicalizeJSON(unsigned envelope) error = %v", err)
	}

	signature := ed25519.Sign(privateKey, canonicalEnvelope)
	unsignedEnvelope["signature"] = wire.Base64URLEncode(signature)

	signedEnvelope, err := json.Marshal(unsignedEnvelope)
	if err != nil {
		t.Fatalf("json.Marshal(signed envelope) error = %v", err)
	}

	proposedAction := json.RawMessage(`{
                "hacp_version":"0.9",
                "action_id":"externality-reason-action",
                "envelope_id":"externality-reason-envelope",
                "verb":"notify",
                "resource_class":"report",
                "resource_id":"rpt://q3",
                "audience":"internal",
                "reversibility":"reversible",
                "externality":"external",
                "data_class":"internal",
                "proposed_at":1786000100
        }`)

	input, err := json.Marshal(map[string]any{
		"intent_envelope": json.RawMessage(signedEnvelope),
		"proposed_action": proposedAction,
		"decision_token":  nil,
		"policy_context": map[string]any{
			"clock":                int64(1786000300),
			"current_action_count": 0,
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(input) error = %v", err)
	}

	resp := runner.HandleRequest(Request{
		ProtocolVersion: ProtocolVersion,
		Operation:       "evaluate",
		VectorID:        "CORE-INV2-004-EXTERNALITY-REASON",
		Input:           input,
	})

	if resp.Decision != "DENY" {
		t.Fatalf(
			"decision = %q, want %q (reason_codes=%v)",
			resp.Decision,
			"DENY",
			resp.ReasonCodes,
		)
	}

	const wantReason = "BOUNDARY_CROSSING"

	if len(resp.ReasonCodes) != 1 ||
		resp.ReasonCodes[0] != wantReason {
		t.Fatalf(
			"reason_codes = %v, want [%q]",
			resp.ReasonCodes,
			wantReason,
		)
	}
}
