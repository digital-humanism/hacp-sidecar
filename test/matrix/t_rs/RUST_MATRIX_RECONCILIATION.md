# RUST MATRIX RECONCILIATION — bounded, Rust applicability only

Date: 2026-10-17
Status: `INTERNAL / PREPARATION / NOT TEST EVIDENCE / NOT ACCEPTANCE EVIDENCE / NOT ADMISSION AUTHORIZATION`
Authoritative matrix: `hacp-sidecar/docs/.restricted/engineering/hacp-rs/SIDECAR-A2_CLOSURE_TABLE.md` (36 rows: A1–A8, B1–B28).
Closed preparation audit: `Result_Audit/SIDECAR_MATRIX_FINAL_PREPARATION_AUDIT_RU_2026-10-17.md`.
Scope rule (§11 of the task): **DO NOT blindly mirror Go.** Rust applicability is determined by actual Rust semantics and Rust verification surfaces, not by mechanically transferring Go rows.

## Classification model

- `RUST_APPLICABILITY`: `RUST_APPLICABLE` (the row covers Rust-owned or Rust-implemented semantics and requires Rust evidence) | `RUST_NOT_APPLICABLE` (the row covers Go/TS/Python runtime and is not reproduced in the Rust-only execution contour).
- `COVERAGE_QUALITY` (for RUST_APPLICABLE only): `ALREADY_COVERED` (an existing RS surface/test is available) | `REQUIRES_ADDITIVE_PREPARATION` (behavior exists, but no test exists; a new RS artifact is required).

Claim boundary: **Core ≠ HC2; GO ≠ RUST; NO SYNTHETIC GO+RUST TOTAL.** The remaining rows are reproduced by the Go execution contour (SECTION I of the combined manifest). Rust Core/HC2 conformance (RS01/RS02) provides decision-level coverage of related semantics but does NOT substitute for Go runtime rows (no blind mirroring).

## Reconciliation table

| MATRIX_ID | MATRIX_REQUIREMENT | RUST_APPLICABILITY | EXISTING_RUST_TEST / TOOL | COVERAGE_QUALITY | RUST_SURFACE_ID | NEW_ARTIFACT_REQUIRED | CLAIM_BOUNDARY |
|---|---|---|---|---|---|---|---|
| A1 | Go Core 38/38 | RUST_NOT_APPLICABLE | — (Go) | — | — | NO | Go-owned; see GO SECTION |
| A2 | TS Core 38/38 | RUST_NOT_APPLICABLE | — (TS) | — | — | NO | TS-owned |
| A3 | Python Core 38/38 | RUST_NOT_APPLICABLE | — (Python) | — | — | NO | Python-owned |
| A4 | Rust Core `36/38 direct + 2 adjudicated` | RUST_APPLICABLE | C procedure `docs/verification/rust-core-composition-reproduction.md` + `hacp-rs-runner` + `harness_runner.py` | ALREADY_COVERED | RS01 | NO | `38/38 ACCOUNTED = 36/38 DIRECT + 2/38 ADJUDICATED (CORE-INV1-002, CORE-RUNTIME-004)`; claiming `38/38 direct` is FORBIDDEN |
| A5 | Go HC2-55 55/55 | RUST_NOT_APPLICABLE | — (Go) | — | — | NO | Go-owned |
| A6 | TS HC2-55 55/55 | RUST_NOT_APPLICABLE | — (TS) | — | — | NO | TS-owned |
| A7 | Python HC2-55 55/55 | RUST_NOT_APPLICABLE | — (Python) | — | — | NO | Python-owned |
| A8 | Rust HC2-55 55/55 | RUST_APPLICABLE | `hacp-rs-runner` + `harness/enforcement_v2_runner.py` (S04 contract) | ALREADY_COVERED | RS02 | NO | Rust HC2 is a separate verdict family; do NOT combine with Core |
| B1 | Native Go regression | RUST_NOT_APPLICABLE | — (Go native) | — | — | NO | Go native; Rust equivalent = B18/RS03 |
| B2 | Protocol correctness | RUST_NOT_APPLICABLE | (Rust wire — decision-level in RS01/RS02) | — | — | NO | SIDECAR/Go runtime row; Rust provides decision-level coverage only |
| B3 | Semantic boundary | RUST_NOT_APPLICABLE | (evaluate — decision-level in RS01/RS02) | — | — | NO | Go runtime row |
| B4 | HC2 runtime (sidecar black-box) | RUST_NOT_APPLICABLE | — (sidecar Go runner ≠ Rust runner) | — | — | NO | Rust HC2 = A8/RS02, a different runner identity |
| B5 | Trust lifecycle | RUST_NOT_APPLICABLE | (trusted_keys in evaluate — decision-level) | — | — | NO | Go SIDECAR trust/revocation lifecycle |
| B6 | Signature / canonicalization | RUST_APPLICABLE | `tests/canonical_v2.rs`(7), `canonical_v2_r2.rs`(3), `ed25519_v2.rs`(8), `ed25519_v1_red.rs`, `authorization_v2_envelope_signature_red.rs`, `e13_family2_binding.rs`, `json_ingress_v2.rs`(4), `json_ingress_v1_red.rs` | ALREADY_COVERED (EXACT) | RS03 (package) | NO (RS06=NOT_APPLICABLE) | Direct coverage of JCS/canonical/signature/canonical-binding/duplicate/ingress |
| B7 | Revocation / expiration | RUST_NOT_APPLICABLE | (TOKEN_REVOKED/ENVELOPE_EXPIRED in evaluate — decision-level) | — | — | NO | Go runtime row |
| B8 | Binding / action constraints | RUST_NOT_APPLICABLE | (`e13_family2_binding`, action_hash — in RS03) | — | — | NO | Go pipeline binding row; Rust coverage is decision-level |
| B9 | Budget / replay | RUST_APPLICABLE | **NONE** (behavior exists: `evaluate.rs:442-460` `autonomy_budget.max_actions`; Rust has no `max_uses`) | REQUIRES_ADDITIVE_PREPARATION | RS07 | **YES** (RS07) | Rust semantics = `autonomy_budget`/`max_actions`/`current_action_count` → `BUDGET_EXHAUSTED`; Go `max_uses` is NOT transferred |
| B10 | Provenance | RUST_NOT_APPLICABLE | (Steps 12–13 in evaluate — decision-level) | — | — | NO | Go provenance row |
| B11 | Distributed control | RUST_NOT_APPLICABLE | — | — | — | NO | Go in-memory control plane |
| B12 | Wire / interoperability | RUST_NOT_APPLICABLE | — | — | — | NO | Go wire row |
| B13 | External Python↔Go E2E | RUST_NOT_APPLICABLE | — | — | — | NO | Python↔Go |
| B14 | Anti-bypass | RUST_NOT_APPLICABLE | — | — | — | NO | SIDECAR deployment (Docker) |
| B15 | Go HTTP reproduction | RUST_NOT_APPLICABLE | — (Go HTTP) | — | — | NO | Go HTTP = forward claim; ≠ Rust HTTP |
| B16 | Rust HTTP runtime (auth + local ALLOW/DENY) | RUST_APPLICABLE | `benchmarks/rust-http-reproduction/main.go` (S) + `hacp-rs/src/bin/hacp-rs-proxy.rs` (P) + `docs/verification/rust-http-local-reproduction.md` | ALREADY_COVERED | RS04 | NO | AUTHORIZATION + LOCAL RESPONSE; NO upstream-forwarding claim |
| B17 | Rust HTTP repository ownership | RUST_APPLICABLE | S-owned client + P-owned proxy; admitted `3018b6b` | ALREADY_COVERED | RS04 | NO | Procedure/client S-owned; ownership resolved |
| B18 | Rust package qualification (49 items) | RUST_APPLICABLE | `cargo +stable-x86_64-pc-windows-msvc test --locked` (hacp-rs) | ALREADY_COVERED | RS03 | NO | `48 ACTIVE PASS + 1 HISTORICAL IGNORED`; claim bound to exact P_HEAD/P_TREE |
| B19 | Timing / benchmarks | RUST_APPLICABLE | measured workload timing (rust-http tool) | ALREADY_COVERED | RS05 | NO | INFORMATIONAL; NO threshold; NO Go-vs-Rust ranking |
| B20 | Gate D applicability | RUST_NOT_APPLICABLE | — | — | — | NO | Threshold applies only when a normative contract exists |
| B21 | Source boundary (S/P/C; G/M prohibited) | RUST_NOT_APPLICABLE | — (governance) | — | — | NO | Shared governance gate |
| B22 | Mutation discipline | RUST_NOT_APPLICABLE | — (governance) | — | — | NO | Shared governance gate |
| B23 | Unresolved HOLD closure | RUST_NOT_APPLICABLE | — | — | — | NO | Go closure row |
| B24 | R9.2 evidence manifest | RUST_NOT_APPLICABLE | — | — | — | NO | Go evidence row |
| B25 | R9.2 results reconciliation | RUST_NOT_APPLICABLE | — | — | — | NO | Go evidence row |
| B26 | Internal RU closure report | RUST_NOT_APPLICABLE | — | — | — | NO | Go closure artifact |
| B27 | EN runtime/integration closure report | RUST_NOT_APPLICABLE | — | — | — | NO | Go closure artifact |
| B28 | Final SIDECAR-A2 declaration | RUST_NOT_APPLICABLE | — | — | — | NO | Go declaration |

## B6 coverage check — detail (§12)

| B6 aspect | Rust test | Verdict |
|---|---|---|
| JCS / canonicalization | `canonical_v2.rs` (7), `canonical_v2_r2.rs` (RFC8785 appendix) | COVERED |
| signature verification | `ed25519_v2.rs` (8), `ed25519_v1_red.rs`, `authorization_v2_envelope_signature_red.rs` | COVERED |
| canonical form binding | `ed25519_v2.rs` (whitespace-variant rejection), `e13_family2_binding.rs` | COVERED |
| duplicate member handling | `json_ingress_v2.rs` (duplicate/nested-duplicate path) | COVERED |
| invalid JSON / ingress | `json_ingress_v1_red.rs`, `json_ingress_v2.rs` (valid-preserved) | COVERED |

`B6_RUST_COVERAGE=EXACT` → `NEW_RUST_B6_FILE_REQUIRED=NO` → **RS06=NOT_APPLICABLE**.

## B9 coverage check — detail (§13)

```text
RUST_BUDGET_BEHAVIOR_EXISTS=YES
  source: hacp-rs/src/evaluate.rs Step 9 "Bounded autonomy (INV-7)" (lines ~442-460)
  semantics: envelope.autonomy_budget.max_actions vs context.current_action_count
             current >= max_actions => DENY + reason "BUDGET_EXHAUSTED" (fail-closed)
  DecisionToken max_uses / default maxUses=1 / token replay-reuse: ABSENT in Rust
  (grep "max_uses" over hacp-rs/src + hacp-rs/tests = 0 matches)
RUST_BUDGET_TEST_COVERAGE=NONE
  (no tracked Rust test mentions budget/max_uses/replay/exhaust)
```

Behavior exists, but tests are absent → `REQUIRES_ADDITIVE_PREPARATION` → **RS07 — Rust autonomy budget verification** (minimal verification-only artifact, actual Rust semantics, NOT Go `max_uses`). File: `C:\Personal\GitHub\Dev\hacp-sidecar\test\matrix\t_rs\RS07_budget_replay_verification.rs` → target `hacp-spec/hacp-rs/tests/budget_replay_verification.rs`.

```text
B9 Rust facet = autonomy budget exhaustion only
RUST_TOKEN_REPLAY_SEMANTICS=NOT_IMPLEMENTED
RUST_DECISION_TOKEN_MAX_USES=NOT_IMPLEMENTED
RUST_DEFAULT_MAX_USES_1=NOT_IMPLEMENTED
PHYSICAL FILE NAME (!= SEMANTIC CLAIM NAME): RS07_budget_replay_verification.rs contains "replay"
  → authoritative semantic name: RS07 — Rust autonomy budget verification
  → token-replay semantics are NOT claimed for Rust
```

## Reconciliation summary

```text
TOTAL_MATRIX_ROWS=36

RUST_APPLICABLE_ROWS=8          (A4, A8, B6, B9, B16, B17, B18, B19)
RUST_NOT_APPLICABLE_ROWS=28

RUST_ALREADY_COVERED_ROWS=7     (A4→RS01, A8→RS02, B6→RS03, B16→RS04, B17→RS04, B18→RS03, B19→RS05)
RUST_ADDITIVE_PREPARATION_ROWS=1 (B9→RS07)

RUST_UNRESOLVED_ROWS=0
```

`CURRENT_RUST_REPRODUCTION_SCOPE_COMPLETE=YES` (all 36 rows classified; unmapped=0; unresolved=0).
