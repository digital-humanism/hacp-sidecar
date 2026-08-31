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

func TestConformanceRunnerEnforcesQuantityAndDestinationBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		scopeExtra  map[string]any
		actionExtra map[string]any
		wantReason  string
	}{
		{
			name: "quantity_above_granted_ceiling",
			scopeExtra: map[string]any{
				"max_quantity": 100,
			},
			actionExtra: map[string]any{
				"quantity": 101,
			},
			wantReason: "SCOPE_EXCEEDED",
		},
		{
			name: "destination_outside_allowlist",
			scopeExtra: map[string]any{
				"destinations": []string{"s3://approved-bucket"},
			},
			actionExtra: map[string]any{
				"destination": "s3://other-bucket",
			},
			wantReason: "BOUNDARY_CROSSING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("ed25519.GenerateKey() error = %v", err)
			}

			const keyID = "quantity-destination-boundary-test-key"
			const envelopeID = "quantity-destination-boundary-envelope"

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

			scopeGrant := map[string]any{
				"verbs":            []string{"read"},
				"resource_classes": []string{"customer_record"},
				"audiences":        []string{"internal"},
				"reversibility":    []string{"reversible"},
				"externality":      []string{"internal"},
				"data_classes":     []string{"internal"},
			}

			for k, v := range tt.scopeExtra {
				scopeGrant[k] = v
			}

			unsignedEnvelope := map[string]any{
				"hacp_version":     "0.9",
				"envelope_id":      envelopeID,
				"principal":        "quantity-destination-system",
				"principal_kind":   "system",
				"intent_statement": "Quantity/destination boundary conformance",
				"scope":            scopeGrant,
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

			proposedActionMap := map[string]any{
				"hacp_version":   "0.9",
				"action_id":      "quantity-destination-boundary-action",
				"envelope_id":    envelopeID,
				"verb":           "read",
				"resource_class": "customer_record",
				"resource_id":    "crm://acct/4411",
				"audience":       "internal",
				"reversibility":  "reversible",
				"externality":    "internal",
				"data_class":     "internal",
				"proposed_at":    int64(1786000100),
			}

			for k, v := range tt.actionExtra {
				proposedActionMap[k] = v
			}

			proposedAction, err := json.Marshal(proposedActionMap)
			if err != nil {
				t.Fatalf("json.Marshal(proposed action) error = %v", err)
			}

			input, err := json.Marshal(map[string]any{
				"intent_envelope": json.RawMessage(signedEnvelope),
				"proposed_action": json.RawMessage(proposedAction),
				"decision_token":  nil,
				"policy_context": map[string]any{
					"clock": int64(1786000300),
				},
			})
			if err != nil {
				t.Fatalf("json.Marshal(input) error = %v", err)
			}

			resp := runner.HandleRequest(Request{
				ProtocolVersion: ProtocolVersion,
				Operation:       "evaluate",
				VectorID:        "CORE-INV2-QUANTITY-DESTINATION-RED",
				Input:           input,
			})

			if resp.Decision != "DENY" {
				t.Fatalf(
					"decision = %q, want DENY (reason_codes=%v)",
					resp.Decision,
					resp.ReasonCodes,
				)
			}

			if len(resp.ReasonCodes) != 1 ||
				resp.ReasonCodes[0] != tt.wantReason {

				t.Fatalf(
					"reason_codes = %v, want [%q]",
					resp.ReasonCodes,
					tt.wantReason,
				)
			}
		})
	}
}
