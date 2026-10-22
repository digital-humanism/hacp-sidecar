# WINDOWS EXECUTION MANIFEST — RS (Rust reproduction only)

Date: 2026-10-17
Status: `INTERNAL / PREPARATION / NOT TEST EVIDENCE / NOT ACCEPTANCE EVIDENCE / NOT ADMISSION AUTHORIZATION`
Authoritative execution manifest **for Rust reproduction only** (RS01–RS07). The Go execution contour is separate: `WINDOWS_EXECUTION_MANIFEST.md` (NOT superseded by this file).
Contract details: `RUST_VERIFICATION_CONTRACTS.md`. Matrix reconciliation: `RUST_MATRIX_RECONCILIATION.md`.

```text
THIS MANIFEST DOES NOT AUTHORIZE ADMISSION BEFORE VERIFICATION
RS PHASE 0 = LOCATE + IDENTITY CHECK ONLY
```

Fundamental sequence: PREPARE → LOCATE → STATIC IDENTITY CHECK → HUMAN WINDOWS EXECUTION → HUMAN REVIEW → SEPARATE ADMISSION DECISION. `VERIFY BEFORE ADMIT`; `ARTIFACT PRESENCE != ADMISSION`; the verification artifact remains in `t_rs`.

## Global Rust claim boundaries (mandatory)

```text
GO RESULT != RUST RESULT
CORE != HC2
NO SYNTHETIC GO+RUST TOTAL

RUST CORE:  38/38 ACCOUNTED = 36/38 DIRECT + 2/38 ADJUDICATED (CORE-INV1-002, CORE-RUNTIME-004)
            "RUST 38/38 DIRECT" is FORBIDDEN
RUST HC2:   55/55 PASS (separate verdict family)
RUST PACKAGE: 48 ACTIVE PASS + 1 HISTORICAL IGNORED (claim bound to exact P_HEAD/P_TREE, H2)
RUST HTTP:  AUTHORIZATION + LOCAL RESPONSE; NO upstream-forwarding claim
RUST TIMING: INFORMATIONAL; NO THRESHOLD; NO GO-vs-RUST LATENCY RANKING
HISTORICAL PASS != CURRENT CANDIDATE PASS
```

## RS execution order (sequential)

```text
RS PHASE 0A  LOCATE required Rust verification artifacts in t_rs only
             (explicitly: source artifact = C:\Personal\GitHub\Dev\hacp-sidecar\test\matrix\t_rs\RS07_budget_replay_verification.rs; no copying into hacp-spec;
              NO staging/commit)
RS PHASE 0B  SHA/path/status/identity validation of each verification artifact
             (Test-Path, Get-Item, Get-FileHash -Algorithm SHA256, git status --short -- <path>,
              git check-ignore -v -- <path>; gate: STAGED_FILES=0)

RS PHASE 1   RS01  Rust Core            (A4; strict 13/38 + reconciliation 36/38+2)
RS PHASE 2   RS02  Rust HC2-55          (A8; 55/55)
RS PHASE 3   RS03  Rust package qualification (B18; 48+1 ignored; B6 coverage)
RS PHASE 4   RS04  Rust HTTP runtime    (B16+B17; smoke + 100 warmup + 5000 measured)
RS PHASE 5   RS05  Rust timing capture  (B19-subset; evidence RS04; INFORMATIONAL)
RS PHASE 6   RS06  (NOT_APPLICABLE — B6 EXACT coverage) / RS07 (B9 autonomy budget verification)

POST-RUN     Aggregate evidence → Human result review
ADMISSION    Separate human decision (PENDING HUMAN DECISION; never automatic)
             After explicit GO: explicit paths only, no `git add .`/`git add -A`,
             signed commits, verify signature/HEAD/tree/status, no unrelated files.
```

Environment preparation (builds/installations are MUTATING and human-controlled) occurs within RS PHASES 1–5 according to the contracts; RS PHASE 0A/0B are limited to artifact location and identity validation.

---

# SURFACE RS01 — Rust Core reproduction (A4)
- MATRIX_ROWS: A4 | OWNER: P + S | LANG: Rust MSVC + Python | PATHS: `hacp-rs/`, `harness/`, `vectors/`, `docs/verification/rust-core-composition-reproduction.md`
- REQUIRED_FILES: existing files (C procedure + runner sources) | PREPARATION/PREFLIGHT/READINESS: `RUST_VERIFICATION_CONTRACTS.md` §RS01
- ENTRYPOINT: `cargo build --locked --release --bin hacp-rs-runner` → `python .\harness\harness_runner.py --runner .\hacp-rs\target\release\hacp-rs-runner.exe --vectors-dir .\vectors --manifest .\harness\conformance_manifest.json --implementation-name hacp-rs --implementation-version 0.1.0-1.1.0 --output console` → raw witness `cargo test --locked --test json_ingress_v1_red -- --nocapture`
- EXECUTION_ORDER: RS PHASE 1 | EXPECTED_RESULT: `BUILD_RC=0`; **`RESULTS: 13/38 passed` + `CORE_HARNESS_RC=1` (expected strict result)**; reconciliation `36/38 direct + 2 adjudicated`; witness `1 passed`
- PASS_CRITERIA: exact strict count + exactly 2 adjudicated | ENVIRONMENT_RED: pin/toolchain/protoc | BEHAVIORAL_RED: mismatch outside the adjudicated cases
- EVIDENCE: console, RC, SHA-256 runner, P identity | CLEANUP: harness terminates runner | GUARDS: clone clean | DEPENDENCIES: RS01 does not depend on Go

# SURFACE RS02 — Rust HC2-55 (A8)
- MATRIX_ROWS: A8 | OWNER: P | LANG: Rust + Python
- ENTRYPOINT: `python .\harness\enforcement_v2_runner.py --runner .\hacp-rs\target\release\hacp-rs-runner.exe --vectors-dir .\vectors\enforcement-v2 --implementation-name hacp-rs --output console`
- EXECUTION_ORDER: RS PHASE 2 | EXPECTED_RESULT: `RESULTS: 55/55 passed`
- PASS_CRITERIA: exact count | ENVIRONMENT_RED: toolchain/pin | BEHAVIORAL_RED: mismatch | EVIDENCE: console, RC | GUARDS: clone clean | DEPENDENCIES: RS01 (runner)
- CLAIM_BOUNDARY: Rust HC2 ≠ Rust Core (do not combine 55/55 with 36/38+2)

# SURFACE RS03 — Rust package qualification (B18; +B6)
- MATRIX_ROWS: B18 (+B6 coverage) | OWNER: P | LANG: Rust MSVC
- ENTRYPOINT: `cargo +stable-x86_64-pc-windows-msvc test --locked` (from `hacp-rs`)
- EXECUTION_ORDER: RS PHASE 3 | EXPECTED_RESULT: `48 passed`, `1 ignored` (`ed25519_v1_red`), `0 failed`; **bind the claim to exact P_HEAD/P_TREE (H2)**
- PASS_CRITERIA: exact accounting | ENVIRONMENT_RED: toolchain/lock | BEHAVIORAL_RED: failure (new regression) — record it | EVIDENCE: cargo output, RC, P identity | GUARDS: P clean | DEPENDENCIES: none
- CLAIM_BOUNDARY: do not retroactively attribute `48 ACTIVE + 1 HISTORICAL IGNORED` to a different identity; B6 coverage is EXACT (RS06 not required)

# SURFACE RS04 — Rust HTTP runtime (B16 + B17)
- MATRIX_ROWS: B16, B17 | OWNER: S (client/procedure) + P (proxy) | LANG: Go client + Rust proxy
- ENTRYPOINT: R8.2 procedure (`docs/verification/rust-http-local-reproduction.md`): cargo build `--locked --offline --bin hacp-rs-proxy` → fixtures 100/5000 → proxy `:18081` (owned PID) → negative smoke 403+DENY → `go run .\benchmarks\rust-http-reproduction -url http://127.0.0.1:18081/api/test -warmup-file … -measured-file …`
- EXECUTION_ORDER: RS PHASE 4 | EXPECTED_RESULT: smoke PASS; `R8_2_WORKLOAD=PASS`; JSON warmup(100)+measured(5000); authorization/local ALLOW-DENY response
- PASS_CRITERIA: all client assertions + counts; exit 0 | ENVIRONMENT_RED: toolchain/port/fixtures/TTL | BEHAVIORAL_RED: smoke/assertion failure | EVIDENCE: complete R8.2 evidence set | CLEANUP: owned PID | GUARDS: S/P clean | DEPENDENCIES: fresh build (NOT stale `target/debug/hacp-rs-proxy.exe`)
- CLAIM_BOUNDARY: **AUTHORIZATION + LOCAL RESPONSE; NO upstream-forwarding claim.** UPSTREAM_REQUIRED=NO; UPSTREAM_TOPOLOGY=NOT_APPLICABLE (topology = Rust proxy + local response + HTTP reproduction client; no upstream service required). B17 ownership is closed (`3018b6b`)

# SURFACE RS05 — Rust timing capture (B19-subset)
- MATRIX_ROWS: B19 (Rust subset) | OWNER: S + P
- ENTRYPOINT: timing metrics from the same RS04 run (no separate runtime required)
- EXECUTION_ORDER: RS PHASE 5 | EXPECTED_RESULT: `TIMING_CAPTURED=YES`; warmup=100, measured=5000
- PASS_CRITERIA: `THRESHOLD=NONE`; `INFORMATIONAL_ONLY=YES`; PASS/FAIL does not depend on latency | ENVIRONMENT_RED: same as RS04 | BEHAVIORAL_RED: not applicable
- EVIDENCE: timing from RS04 (p50/p95/p99/max/RPS; otherwise `NOT REPORTED`) | GUARDS: same as RS04 | DEPENDENCIES: RS04
- CLAIM_BOUNDARY: **RUNTIME VERDICT ≠ TIMING OBSERVATION; NO GO-vs-RUST RANKING; NO GATE-D PASS/FAIL FROM LATENCY**

# SURFACE RS06 — (NOT_APPLICABLE)
- `RS06=NOT_APPLICABLE` — B6 coverage is EXACT (direct Rust tests). NEW_RUST_B6_FILE_REQUIRED=NO.

# SURFACE RS07 — Rust autonomy budget verification (B9)
- MATRIX_ROWS: B9 | OWNER: P | LANG: Rust | REQUIRED_FILES: `C:\Personal\GitHub\Dev\hacp-sidecar\test\matrix\t_rs\RS07_budget_replay_verification.rs` (in t_rs) → target `hacp-spec/hacp-rs/tests/budget_replay_verification.rs`
- ENTRYPOINT: `cargo +stable-x86_64-pc-windows-msvc test --locked --test budget_replay_verification`
- EXECUTION_ORDER: RS PHASE 6 (after Phase 0A locate + Phase 0B validation) | EXPECTED_RESULT: `3 passed, 0 failed` (exhaustion→DENY+BUDGET_EXHAUSTED; boundary→exhausted; under-limit→ALLOW)
- PASS_CRITERIA: all assertions PASS; SKIP=0 | ENVIRONMENT_RED: compile (tracked API mismatch)/toolchain | BEHAVIORAL_RED: exhaustion does not deny / under-limit incorrectly denied on budget grounds
- EVIDENCE: cargo output, RC, file SHA-256, P identity | CLEANUP: `target/` gitignored | GUARDS: P clean (production unchanged) | DEPENDENCIES: RS07 artifact presence (Phase 0A) + identity validation (Phase 0B); admission is a separate human decision
- CLAIM_BOUNDARY: **B9 Rust facet = autonomy budget exhaustion only.** Rust semantics = `autonomy_budget.max_actions`/`current_action_count` → `BUDGET_EXHAUSTED`. Do not claim "Rust token replay verified". RUST_TOKEN_REPLAY_SEMANTICS=NOT_IMPLEMENTED; RUST_DECISION_TOKEN_MAX_USES=NOT_IMPLEMENTED; RUST_DEFAULT_MAX_USES_1=NOT_IMPLEMENTED. Go `max_uses`/replay semantics are NOT transferred. PHYSICAL FILE NAME != SEMANTIC CLAIM NAME (the file name contains "replay"; the authoritative claim is autonomy budget only). Do not conflate this with control-plane journal replay.

---

## Rust readiness summary

| RS | MATRIX_ROWS | ENV | EXEC | FILES | STATIC | UNRESOLVED | PREPARED_FOR_WINDOWS_VERIFICATION |
|---|---|---|---|---|---|---|---|
| RS01 | A4 | READY | READY | YES | PASS | 0 | YES |
| RS02 | A8 | READY | READY | YES | PASS | 0 | YES |
| RS03 | B18 | READY | READY | YES | PASS | 0 | YES |
| RS04 | B16,B17 | READY | READY | YES | PASS | 0 | YES |
| RS05 | B19-subset | READY | READY | YES | PASS | 0 | YES |
| RS06 | (B6) | — | — | — | — | 0 | NOT_APPLICABLE |
| RS07 | B9 | READY | READY | YES (in `t_rs`) | PASS | 0 | YES (after Phase 0A locate + Phase 0B validation) |

`RUST_PACKAGE_READY_FOR_WINDOWS_REPRODUCTION=YES` (condition: RS07 Phase 0A locate + Phase 0B validation per `Readme_map_file_RS.md`).

---

POST-EXECUTION ADMISSION IS A SEPARATE HUMAN DECISION. `ADMISSION = PENDING HUMAN DECISION`, never automatic. No synthetic totals and no Go-vs-Rust ranking.
