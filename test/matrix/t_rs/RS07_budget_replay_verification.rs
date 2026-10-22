// RS07 — Rust B9 budget/replay ADDITIONAL VERIFICATION (verification-only).
//
// VERIFICATION ONLY: NO production code change; NO new production semantics.
// Target (compile location): hacp-spec/hacp-rs/tests/budget_replay_verification.rs
// The include_str! path "../../vectors/core_inv1_001_golden.json" resolves relative
// to hacp-rs/tests/ at compile time (same convention as tracked tests
// tests/authorization_v2_signed_deny_red.rs, tests/authorization_v2_envelope_signature_red.rs).
//
// RUST B9 SEMANTIC BOUNDARY (source-backed, NOT imported from Go):
//   - Actual Rust budget behavior lives in hacp-rs/src/evaluate.rs Step 9
//     "Bounded autonomy (INV-7)" (lines ~442-460):
//         envelope.autonomy_budget.max_actions  vs  context.current_action_count
//         current >= max_actions  =>  DENY + reason "BUDGET_EXHAUSTED" (fail-closed).
//   - DecisionToken `max_uses`, default maxUses=1, and token replay/reuse semantics are
//     ABSENT in the Rust implementation (grep "max_uses" over hacp-rs/src and hacp-rs/tests
//     returns no match). Those Go-level S12 semantics are deliberately NOT mirrored here.
//     See RUST_MATRIX_RECONCILIATION.md (B9 row) and RUST_VERIFICATION_CONTRACTS.md (RS07).
//
// This closes the B9 gap: budget behavior EXISTS in the Rust production path but had NO test.

use base64::engine::general_purpose::URL_SAFE_NO_PAD;
use base64::Engine;
use ed25519_dalek::{Signer, SigningKey};
use serde_json::Value;
use sha2::{Digest, Sha256};

use hacp_rs::evaluate;
use hacp_rs::jcs::canonicalize;

const TEST_PUBKEY_HEX: &str =
    "9d17f1bbcc0845865e670f526413fb7a510380798fe300b6c98e28f3a3b0fdb3";

const CONFORMANCE_SEED_SOURCE: &[u8] = b"hacp-conformance-v0.9-key-001";

fn conformance_signing_key() -> SigningKey {
    let seed_digest = Sha256::digest(CONFORMANCE_SEED_SOURCE);
    let seed: [u8; 32] = seed_digest.into();
    SigningKey::from_bytes(&seed)
}

/// Load the golden vector, inject envelope.autonomy_budget.max_actions, re-sign the
/// IntentEnvelope with the conformance key (canonical-minus-signature form, matching
/// evaluate.rs:179-186 `copy_without(envelope,"signature")` + `canonicalize` + verify),
/// set context.current_action_count, and run evaluate().
fn eval_with_budget(max_actions: i64, current_action_count: i64) -> evaluate::EvalResult {
    let mut vector: Value = serde_json::from_str(include_str!(
        "../../vectors/core_inv1_001_golden.json"
    ))
    .expect("CORE-INV1-001 golden vector must parse");

    let signing_key = conformance_signing_key();

    {
        let inputs = vector
            .get_mut("inputs")
            .expect("golden vector must contain inputs");
        let envelope = inputs
            .get_mut("intent_envelope")
            .expect("golden inputs must contain intent_envelope");
        // Pin the conformance signing identity (same as tracked tests).
        assert_eq!(
            hex::encode(signing_key.verifying_key().as_bytes()),
            TEST_PUBKEY_HEX,
            "derived conformance signing identity must match key-ed25519-test-001"
        );

        {
            let envelope_object = envelope
                .as_object_mut()
                .expect("IntentEnvelope must be an object");

            envelope_object.insert(
                "autonomy_budget".to_owned(),
                serde_json::json!({ "max_actions": max_actions }),
            );
            envelope_object.remove("signature");
        }

        let payload = canonicalize(envelope).expect("modified IntentEnvelope must canonicalize");
        let signature = signing_key.sign(&payload);

        envelope
            .as_object_mut()
            .expect("IntentEnvelope must be an object")
            .insert(
                "signature".to_owned(),
                Value::String(URL_SAFE_NO_PAD.encode(signature.to_bytes())),
            );
    }

    let mut context = vector
        .get("policy_context")
        .expect("golden vector must contain policy_context")
        .clone();
    context["current_action_count"] = Value::from(current_action_count);

    let inputs = vector
        .get("inputs")
        .expect("golden vector must contain inputs")
        .clone();

    evaluate::evaluate(&inputs, &context, TEST_PUBKEY_HEX)
}

// B9 — max_actions exhaustion is fail-closed (current == max_actions is the boundary).
#[test]
fn rs07_b9_budget_exhaustion_is_fail_closed() {
    let result = eval_with_budget(1, 1);
    assert_eq!(
        result.decision, "DENY",
        "exhausted autonomy budget (current_action_count == max_actions) must deny"
    );
    assert_eq!(
        result.reason_codes,
        vec!["BUDGET_EXHAUSTED".to_string()],
        "exhausted autonomy budget must be fail-closed with BUDGET_EXHAUSTED"
    );
}

// B9 — boundary: the counter equal to max_actions is exhausted (source: `current >= max_actions`).
#[test]
fn rs07_b9_replay_counter_at_limit_is_exhausted() {
    let result = eval_with_budget(3, 3);
    assert_eq!(result.decision, "DENY");
    assert_eq!(result.reason_codes, vec!["BUDGET_EXHAUSTED".to_string()]);
}

// B9 — under the limit the budget must NOT deny (fail-closed only when exhausted).
#[test]
fn rs07_b9_budget_not_exhausted_allows() {
    let result = eval_with_budget(2, 1);
    assert!(
        !result.reason_codes.contains(&"BUDGET_EXHAUSTED".to_string()),
        "current_action_count below max_actions must not be budget-denied"
    );
    assert_eq!(
        result.decision, "ALLOW",
        "valid golden action under the autonomy budget limit must allow"
    );
}
