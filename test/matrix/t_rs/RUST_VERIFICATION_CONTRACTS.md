# RUST VERIFICATION CONTRACTS — RS01–RS07

Date: 2026-10-17
Status: `INTERNAL / PREPARATION / NOT TEST EVIDENCE / NOT ACCEPTANCE EVIDENCE / NOT ADMISSION AUTHORIZATION`
Sequence: PREPARE → LOCATE → STATIC IDENTITY CHECK → HUMAN WINDOWS EXECUTION → HUMAN REVIEW → SEPARATE ADMISSION DECISION. `VERIFY BEFORE ADMIT`; `ARTIFACT PRESENCE != ADMISSION`.
Claim boundaries: `GO != RUST`; `CORE != HC2`; `RUST CORE = 38/38 ACCOUNTED = 36/38 DIRECT + 2 ADJUDICATED` (NOT 38/38 direct); `RUST HC2 = 55/55` (separate verdict); `RUST HTTP = AUTHORIZATION + LOCAL RESPONSE` (NO upstream-forwarding); `RUST TIMING = INFORMATIONAL` (NO threshold, NO Go-vs-Rust ranking).
Hermes — PREPARE ONLY (does not execute cargo/go/pytest/runtime/HTTP/benchmark/fixture).

---

# RS01 — Rust Core reproduction (A4)

- SURFACE_ID: RS01 | MATRIX_ROWS: A4 | REPOSITORY_OWNER: P (SPEC-owned) + S (procedure doc)
- EXACT SOURCE / TOOL PATHS: procedure `hacp-sidecar/docs/verification/rust-core-composition-reproduction.md`; runner `hacp-spec/hacp-rs/src/bin/hacp-rs-runner.rs` (target `hacp-rs/target/release/hacp-rs-runner.exe`); harness `hacp-spec/harness/harness_runner.py`; vectors `hacp-spec/vectors/`; manifest `hacp-spec/harness/conformance_manifest.json`
- TOOLCHAIN: Rust `stable-x86_64-pc-windows-msvc`; protoc REQUIRED (`hacp-rs/build.rs:3`); Python 3.13.x + `harness/requirements.txt`; `Cargo.lock` (locked build)
- ENVIRONMENT CONTRACT: P clone detached at resolved active HACP `1.1.1-rc.1` SPEC source `RC_SPEC_SOURCE`; `CARGO_TARGET_DIR` outside repositories; no services/ports; clean tree
- ENVIRONMENT PREPARATION (MUTATING, human): resolve active `1.1.1-rc.1` SPEC source → `git checkout --detach $RC_SPEC_SOURCE`; `pip install -r harness\requirements.txt`; protoc/VC tools
- ENVIRONMENT PREFLIGHT: `git -C <P> rev-parse HEAD`=resolved `RC_SPEC_SOURCE`; `git -C <P> status --short` empty; `rustc --version`/`cargo --version` stable-msvc; `protoc --version`; `python --version`
- READINESS GATES: IDENTITY/OS/TOOLCHAIN/ABI/CRYPTO(conformance keys)/FIXTURE(vectors+manifest)/ENV_VARS/WORKTREE → `ENVIRONMENT_READY=PASS`
- EXECUTION CONTRACT (per C procedure): `cargo build --locked --release --bin hacp-rs-runner` → `python .\harness\harness_runner.py --runner .\hacp-rs\target\release\hacp-rs-runner.exe --vectors-dir .\vectors --manifest .\harness\conformance_manifest.json --implementation-name hacp-rs --implementation-version 0.1.0-1.1.0 --output console` → raw witness `cargo test --locked --test json_ingress_v1_red -- --nocapture`
- EXACT WINDOWS COMMANDS: see C handoff (`ABC_WINDOWS_VERIFICATION_HANDOFF_RU_2026-10-17.md` §C) — commands reproduced verbatim
- EXPECTED RESULT: `BUILD_RC=0`; **`RESULTS: 13/38 passed` + `CORE_HARNESS_RC=1` — expected strict result**; reconciliation `36/38 direct + 2 adjudicated (CORE-INV1-002, CORE-RUNTIME-004)`; new regressions=0; witness `1 passed`
- PASS CRITERIA: exact strict count + exactly 2 adjudicated + reconciliation PASS
- ENVIRONMENT RED: RC_SPEC_SOURCE/toolchain/protoc/deps | BEHAVIORAL RED: outcome mismatch outside the two adjudicated cases; witness failure
- EVIDENCE OUTPUT: console (strict run + witness), RC, runner SHA-256, toolchain versions, P identity (HEAD/tree)
- CLEANUP: harness terminates runner; `CARGO_TARGET_DIR` gitignored | POST-RUN GUARDS: P clone clean; P_HEAD/P_TREE recorded
- DEPENDENCIES: Phase 0B identity; RS01 does NOT depend on Go surfaces

---

# RS02 — Rust HC2-55 (A8)

- SURFACE_ID: RS02 | MATRIX_ROWS: A8 | REPOSITORY_OWNER: P
- EXACT SOURCE / TOOL PATHS: runner `hacp-rs/target/release/hacp-rs-runner.exe` (built in RS01); harness `hacp-spec/harness/enforcement_v2_runner.py`; vectors `hacp-spec/vectors/enforcement-v2/`
- TOOLCHAIN: identical to RS01
- ENVIRONMENT CONTRACT: same P environment (resolved `RC_SPEC_SOURCE`); runner protocol is stdin/stdout JSON (`harness/runner_protocol.md`)
- ENVIRONMENT PREPARATION: runner already built in RS01 (same `CARGO_TARGET_DIR`)
- ENVIRONMENT PREFLIGHT: runner binary exists; SHA-256 recorded; vectors/enforcement-v2 present
- READINESS GATES: same as RS01 + runner identity
- EXECUTION CONTRACT: `python .\harness\enforcement_v2_runner.py --runner .\hacp-rs\target\release\hacp-rs-runner.exe --vectors-dir .\vectors\enforcement-v2 --implementation-name hacp-rs --output console`
- EXACT WINDOWS COMMANDS: see `CONTRACTS_S01_S12.md` §S04 (verbatim command)
- EXPECTED RESULT: `RESULTS: 55/55 passed`
- PASS CRITERIA: exact `55/55` count; separate verdict family
- ENVIRONMENT RED: toolchain/protoc/RC_SPEC_SOURCE/runner | BEHAVIORAL RED: any HC2 mismatch
- EVIDENCE OUTPUT: console, RC, SHA-256 runner, P identity
- CLEANUP: runner terminated by harness | POST-RUN GUARDS: clean clone
- DEPENDENCIES: RS01 (runner build). Claim boundary: **Rust HC2 ≠ Rust Core** (36/38+2 must not be combined with 55/55)

---

# RS03 — Rust package qualification (B18; also covers B6)

- SURFACE_ID: RS03 | MATRIX_ROWS: B18 (+B6 coverage) | REPOSITORY_OWNER: P (`hacp-spec/hacp-rs`)
- EXACT SOURCE / TOOL PATHS: `hacp-spec/hacp-rs/` (crate `hacp_rs`, 13 tracked test files in `tests/`, `Cargo.lock`)
- TOOLCHAIN: `cargo +stable-x86_64-pc-windows-msvc` (MSVC ABI), protoc (build.rs)
- ENVIRONMENT CONTRACT: resolved active HACP `1.1.1-rc.1` SPEC source `RC_SPEC_SOURCE`; `CARGO_TARGET_DIR` outside repositories; clean tree; network not required (`--locked`)
- ENVIRONMENT PREPARATION: toolchain + protoc
- ENVIRONMENT PREFLIGHT: `cargo --version`; `rustc --version`; `protoc --version`; RC SPEC source identity check P
- READINESS GATES: IDENTITY/TOOLCHAIN/ABI/WORKTREE
- EXECUTION CONTRACT: `cargo +stable-x86_64-pc-windows-msvc test --locked` (from `hacp-rs`)
- EXACT WINDOWS COMMANDS:
  ```powershell
  $SidecarRepo = (git rev-parse --show-toplevel).Trim()
  if ($LASTEXITCODE -ne 0 -or -not $SidecarRepo) { throw 'Unable to resolve hacp-sidecar repository root' }
  $SpecRepo = Join-Path (Split-Path -Parent $SidecarRepo) 'hacp-spec'
  Set-Location (Join-Path $SpecRepo 'hacp-rs')
  cargo +stable-x86_64-pc-windows-msvc test --locked
  ```
- EXPECTED RESULT: `48 passed`, `1 ignored` (`ed25519_v1_red` — historical ignored witness), `0 failed`
- PASS CRITERIA: exact accounting `48 + 1 ignored + 0 failed`; **bind claim to exact P_HEAD/P_TREE (H2)**
- ENVIRONMENT RED: toolchain/lock/protoc | BEHAVIORAL RED: failure (new regression) — record, do not fix
- EVIDENCE OUTPUT: complete cargo output, RC, P_HEAD/P_TREE
- CLEANUP: `target/` gitignored | POST-RUN GUARDS: P clean
- DEPENDENCIES: Phase 0B. Claim boundary: do not retroactively attribute `48 ACTIVE + 1 HISTORICAL IGNORED` to another identity. **B6 coverage is EXACT** (JCS/canonical/signature/canonical-binding/duplicate/ingress — direct tests); RS06 is not required.

---

# RS04 — Rust HTTP runtime reproduction (B16 + B17)

- SURFACE_ID: RS04 | MATRIX_ROWS: B16, B17 | REPOSITORY_OWNER: S (client + procedure) + P (`hacp-rs/src/bin/hacp-rs-proxy.rs`)
- EXACT SOURCE / TOOL PATHS: client `hacp-sidecar/benchmarks/rust-http-reproduction/main.go` (tracked, admitted `3018b6b`); procedure `hacp-sidecar/docs/verification/rust-http-local-reproduction.md`; proxy `hacp-spec/hacp-rs/src/bin/hacp-rs-proxy.rs` (tracked) + `hacp-rs/src/proxy.rs`
- TOOLCHAIN: Go ≥ 1.26.x (client); Rust MSVC + protoc (proxy build). UPSTREAM_REQUIRED=NO; UPSTREAM_TOPOLOGY=NOT_APPLICABLE (topology = Rust proxy + local response + HTTP reproduction client; no upstream service required for readiness)
- ENVIRONMENT CONTRACT: port `18081` (proxy); fixtures `env`/`tok` (warmup 100, measured 5000); `HACP_SIDECAR_PORT=18081`; trust `HACP_TRUST_MODE=test` (without `HACP_TRUST_KEYS_FILE`); `CARGO_TARGET_DIR` outside repositories
- ENVIRONMENT PREPARATION (MUTATING, human): fresh-build proxy `cargo +stable-x86_64-pc-windows-msvc build --locked --offline --manifest-path <P>\hacp-rs\Cargo.toml --bin hacp-rs-proxy`; fresh fixtures 100/5000 (TTL gate)
- ENVIRONMENT PREFLIGHT: proxy binary SHA-256; fixture hashes; `TOKEN_VALIDITY_AT_START=PASS`; port 18081 available; PROXY variables empty
- READINESS GATES: IDENTITY/TOOLCHAIN/ABI/CLOCK(TTL)/FIXTURE/PORTS/TRUST/ENV_VARS
- EXECUTION CONTRACT (per R8.2 procedure): negative fail-closed smoke (`403+DENY`, without headers) → 100 warmup → 5000 measured → cleanup
- EXACT WINDOWS COMMANDS: see `docs/verification/rust-http-local-reproduction.md` (authoritative procedure, verbatim)
- EXPECTED RESULT: smoke PASS; `R8_2_WORKLOAD=PASS`; JSON warmup(100/100) + measured(5000/5000); authorization/local ALLOW-DENY response
- PASS CRITERIA: all client assertions (ALLOW + request-id + counts); exit 0
- ENVIRONMENT RED: toolchain failure; proxy build failure; port conflict; fixture missing/invalid; TTL/clock failure; trust/config failure; proxy startup/readiness failure | BEHAVIORAL RED: smoke failure; assertion failure (after ENVIRONMENT_READY=PASS)
- EVIDENCE OUTPUT: client stdout, JSON summaries, fixture hashes, proxy SHA-256, PID lifecycle, negative-smoke observation
- CLEANUP: owned proxy PID → stop; verify listener shutdown | POST-RUN GUARDS: S/P clean; no remaining processes
- DEPENDENCIES: Phase 0B; fresh build (NOT stale `target/debug/hacp-rs-proxy.exe`). Claim boundary: **AUTHORIZATION + LOCAL RESPONSE; claiming RUST UPSTREAM FORWARDING is FORBIDDEN.** B17: client/procedure S-owned — ownership closed (`3018b6b`).

---

# RS05 — Rust HTTP benchmark/timing (B19 subset)

- SURFACE_ID: RS05 | MATRIX_ROWS: B19 (Rust subset) | REPOSITORY_OWNER: S + P
- EXACT SOURCE / TOOL PATHS: same measured workload as RS04 (rust-http tool emits timing metrics)
- TOOLCHAIN: identical to RS04
- ENVIRONMENT CONTRACT: identical to RS04; timing captured in the same run
- ENVIRONMENT PREPARATION / PREFLIGHT / READINESS: same as RS04
- EXECUTION CONTRACT: RS05 uses RS04 evidence; no separate runtime required (tool already prints timing)
- EXACT WINDOWS COMMANDS: no separate command (timing from RS04 run)
- EXPECTED RESULT: `TIMING_CAPTURED=YES`; warmup=100, measured=5000
- PASS CRITERIA: `THRESHOLD=NONE`; `INFORMATIONAL_ONLY=YES`; PASS/FAIL does NOT depend on latency
- ENVIRONMENT RED: same as RS04 | BEHAVIORAL RED: not applicable (timing is observation)
- EVIDENCE OUTPUT: timing metrics from RS04 output (p50/p95/p99/max/RPS if reported; otherwise `NOT REPORTED`)
- CLEANUP: same as RS04 | POST-RUN GUARDS: same as RS04
- DEPENDENCIES: RS04. Claim boundary: **RUNTIME VERDICT ≠ TIMING OBSERVATION; NO GO-vs-RUST LATENCY RANKING; NO GATE-D PASS/FAIL FROM LATENCY.**

---

# RS06 — Rust signature/canonicalization additional verification

```text
RS06=NOT_APPLICABLE
```
Basis: `B6_RUST_COVERAGE=EXACT` (see `RUST_MATRIX_RECONCILIATION.md` §B6). Direct tracked Rust tests cover JCS, canonicalization, signature verification, canonical form binding, duplicate member handling, and invalid JSON/ingress. `NEW_RUST_B6_FILE_REQUIRED=NO`. No mechanical mirroring of Go S11 (§11 of the task).

---

# RS07 — Rust autonomy budget verification (B9)

- SURFACE_ID: RS07 | MATRIX_ROWS: B9 | REPOSITORY_OWNER: P (`hacp-spec/hacp-rs`)
- EXACT SOURCE / TOOL PATHS: new verification-only test `test/matrix/t_rs/RS07_budget_replay_verification.rs` in the `hacp-sidecar` repository → target `hacp-spec/hacp-rs/tests/budget_replay_verification.rs`; production `hacp-rs/src/evaluate.rs` (Step 9, NOT modified); golden vector `hacp-spec/vectors/core_inv1_001_golden.json`
- TOOLCHAIN: `cargo +stable-x86_64-pc-windows-msvc`
- ENVIRONMENT CONTRACT: resolved active HACP `1.1.1-rc.1` SPEC source `RC_SPEC_SOURCE`; `CARGO_TARGET_DIR` outside repositories; clean tree; dev-deps already in crate (base64, ed25519_dalek, serde_json, sha2, hex — as in tracked tests)
- ENVIRONMENT PREPARATION: Phase 0A locate existing RS07 artifact in `t_rs`; Phase 0B SHA/path/status/identity validation; toolchain
- ENVIRONMENT PREFLIGHT: file present at target; canonical text SHA-256 matches the package identity after optional UTF-8 BOM removal and CRLF/CR -> LF normalization; `git check-ignore` confirms not ignored; `STAGED_FILES=0`
- READINESS GATES: IDENTITY/TOOLCHAIN/ABI/WORKTREE
- EXECUTION CONTRACT: `cargo +stable-x86_64-pc-windows-msvc test --locked --test budget_replay_verification`
- EXACT WINDOWS COMMANDS:
  ```powershell
  $SidecarRepo = (git rev-parse --show-toplevel).Trim()
  if ($LASTEXITCODE -ne 0 -or -not $SidecarRepo) { throw 'Unable to resolve hacp-sidecar repository root' }
  $SpecRepo = Join-Path (Split-Path -Parent $SidecarRepo) 'hacp-spec'
  Set-Location (Join-Path $SpecRepo 'hacp-rs')
  cargo +stable-x86_64-pc-windows-msvc test --locked --test budget_replay_verification
  ```
- EXPECTED RESULT: `3 passed, 0 failed` — `rs07_b9_budget_exhaustion_is_fail_closed`, `rs07_b9_replay_counter_at_limit_is_exhausted`, `rs07_b9_budget_not_exhausted_allows`
- PASS CRITERIA: exhaustion → `DENY` + `BUDGET_EXHAUSTED`; under-limit → not budget-denied (`ALLOW`); SKIP=0
- ENVIRONMENT RED: compilation (tracked API mismatch) / toolchain / pin | BEHAVIORAL RED: exhaustion not denied / under-limit incorrectly budget-denied
- EVIDENCE OUTPUT: cargo output, RC, canonical text SHA-256 for RS07 text inputs, P identity
- CLEANUP: `target/` gitignored | POST-RUN GUARDS: P clean (production unchanged)
- DEPENDENCIES: Phase 0A artifact presence + Phase 0B identity validation. Claim boundary: **B9 Rust facet = autonomy budget exhaustion only.** Source-backed Rust behavior: `envelope.autonomy_budget.max_actions` vs `context.current_action_count`; `current_action_count >= max_actions` → `DENY` + `BUDGET_EXHAUSTED`. Do not claim "Rust token replay verified".
- RUST_TOKEN_REPLAY_SEMANTICS=NOT_IMPLEMENTED
- RUST_DECISION_TOKEN_MAX_USES=NOT_IMPLEMENTED
- RUST_DEFAULT_MAX_USES_1=NOT_IMPLEMENTED
- Go semantics (DecisionToken `max_uses`, token reuse/replay, default maxUses=1) are NOT transferred into Rust. Do not conflate this with control-plane journal replay.
- PHYSICAL FILE NAME != SEMANTIC CLAIM NAME: package filename `RS07_budget_replay_verification.rs` contains "replay" (historical/physical filename), but the authoritative semantic surface name is **RS07 — Rust autonomy budget verification**; token-replay semantics are NOT claimed for Rust.

---

# Rust package readiness table

| RS | MATRIX_ROWS | ENV | EXEC | FILES | STATIC | UNRESOLVED | PREPARED_FOR_WINDOWS_VERIFICATION |
|---|---|---|---|---|---|---|---|
| RS01 | A4 | READY | READY | YES (C procedure) | PASS | 0 | YES |
| RS02 | A8 | READY | READY | YES | PASS | 0 | YES |
| RS03 | B18 (+B6) | READY | READY | YES | PASS | 0 | YES |
| RS04 | B16,B17 | READY | READY | YES | PASS | 0 | YES |
| RS05 | B19-subset | READY | READY | YES | PASS | 0 | YES |
| RS06 | (B6) | — | — | — | — | 0 | NOT_APPLICABLE |
| RS07 | B9 | READY | READY | YES (in `t_rs`) | PASS | 0 | YES (after Phase 0A locate + Phase 0B validation) |
