package main

import (
	"encoding/json"
	"testing"
)

func TestHC2ExactRequestTargetBinding(t *testing.T) {
	runner, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner() error: %v", err)
	}

	input := json.RawMessage(`{
"intent_envelope": {
"hacp_version": "0.9",
"envelope_id": "22222222-2222-2222-2222-222222222222",
"principal": "human_admin_01",
"principal_kind": "human",
"intent_statement": "HC2 HTTP path binding conformance",
"scope": {
"verbs": ["read"],
"resource_classes": ["http_resource"],
"audiences": ["internal"],
"reversibility": ["reversible"],
"externality": ["internal"],
"data_classes": ["internal"]
},
"issued_at": 1786000000,
"expires_at": 1786003600,
"signer_key_id": "key-ed25519-test-001",
"signature": "sowmKlhJyXIj8jISjhxptxa-FjhZp9xUAaqFcXrlZlJwoTpYppPssPu2Fkya2imkdJZcZbCgQPNILZ5ifXr-Cg"
},
"proposed_action": {
"hacp_version": "0.9",
"action_id": "11111111-1111-1111-1111-111111111111",
"envelope_id": "22222222-2222-2222-2222-222222222222",
"verb": "read",
"resource_class": "http_resource",
"resource_id": "https://example.invalid/resource",
"audience": "internal",
"reversibility": "reversible",
"externality": "internal",
"data_class": "internal",
"proposed_at": 1786000100
},
"decision_token": {
"hacp_version": "0.9",
"token_id": "33333333-3333-3333-3333-000000000001",
"envelope_id": "22222222-2222-2222-2222-222222222222",
"action_hash": "04242787c01210e7ac914224772c16e45f947f7f9bea602164a21509816830ba",
"policy_digest": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
"principal": "human_admin_01",
"signer_key_id": "key-ed25519-test-001",
"issued_at": 1786000200,
"expires_at": 1786003600,
"decision": "ALLOW",
"constraints": {
"path": "/a/b"
},
"signature": "wHru1OakD5NWvYAWEsOmQ6k7WyDImN25aX9tQ8NbTO_FtzQBEjQv-UaTqu-4htMx9IregjRK_G9gi60CPwmeAg"
},
"http_request": {
"method": "GET",
"request_target": "/a/b"
},
"policy_context": {
"clock": 1786000300,
"current_action_count": 0
}
}`)

	resp := runner.HandleRequest(Request{
		ProtocolVersion: ProtocolVersion,
		Operation:       "evaluate",
		VectorID:        "ENF-HC2-001",
		Input:           input,
	})

	if resp.Decision != "ALLOW" {
		t.Fatalf(
			"HC2 exact request-target binding: expected ALLOW, got %s (reason_codes=%v)",
			resp.Decision,
			resp.ReasonCodes,
		)
	}
}
