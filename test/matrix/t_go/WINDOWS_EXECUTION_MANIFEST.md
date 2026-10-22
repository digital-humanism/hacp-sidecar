# WINDOWS EXECUTION MANIFEST — A/B/C/D + S01–S12

Date: 2026-10-17
Status: `INTERNAL / PREPARATION / NOT TEST EVIDENCE / NOT ACCEPTANCE EVIDENCE / NOT ADMISSION AUTHORIZATION`
Controlling basis: `SIDECAR_MATRIX_FINAL_PREPARATION_AUDIT_RU_2026-10-17.md`. Contract details: `CONTRACTS_S01_S12.md` (S01–S12), `ABC_WINDOWS_VERIFICATION_HANDOFF_RU_2026-10-17.md` (A/B/C), `GO_HTTP_WINDOWS_VERIFICATION_HANDOFF_RU_2026-10-17.md` (D).

```text
THIS MANIFEST DOES NOT AUTHORIZE ADMISSION BEFORE VERIFICATION
PHASE 0 = TRANSFER + IDENTITY CHECK ONLY
```

Fundamental project sequence: PREPARE → TRANSFER → STATIC IDENTITY CHECK → HUMAN WINDOWS EXECUTION → HUMAN RESULT REVIEW → SEPARATE ADMISSION DECISION → STAGE / SIGNED COMMIT. Rules: `VERIFY BEFORE ADMIT`; `TRANSFER != ADMISSION`; an untracked file present in the worktree is SUFFICIENT for human verification.

## Global verdict boundaries (mandatory)

```text
Core != HC2 (separate verdicts even when sharing a toolchain)
Rust Core = 36/38 direct + 2 adjudicated; claiming "Rust 38/38 direct" is FORBIDDEN
Rust HC2 = 55/55 (separate verdict)
Go HTTP B15 = authorization + real upstream forwarding + real upstream response
Rust HTTP B16 = authorization + local response; NO upstream-forwarding claim; NO Go-vs-Rust ranking
Timing = INFORMATIONAL; CURRENT_NORMATIVE_GATE_D_THRESHOLD=NOT ESTABLISHED
HISTORICAL PASS != CURRENT CANDIDATE PASS
```

## Windows execution order (final; sequential; ports conflict)

```text
PHASE 0A  TRANSFER PREPARED VERIFICATION FILES ONLY (explicit paths; NO staging/commit)
PHASE 0B  VERIFY TRANSFERRED FILE IDENTITIES / PATHS / WORKTREE STATE
          (Test-Path, Get-Item, Get-FileHash -Algorithm SHA256, git status --short -- <path>,
           git check-ignore -v -- <path>; gate: STAGED_FILES=0)
PHASE 1   A → S11 → S12                    (Go execution contour S; separate verdicts per matrix row)
PHASE 2   B                                 (H03; 18082 listen / 18083 ABSENT)
PHASE 3   S01 → S02 → S03 → S05 → C → S04 → S09   (P/C conformance contour; Core≠HC2)
PHASE 4   S06 → S07                         (E2E 18080; anti-bypass Docker)
PHASE 5   D → S08                           (HTTP surfaces; D: 18082+18080; S08: 18081)
PHASE 6   S10                               (benchmarks 8080+8000; INFORMATIONAL)
POST-RUN  Aggregate evidence → Human result review
PHASE 7   ADMISSION = PENDING HUMAN DECISION (NOT automatic; requires separate explicit human GO)
```

Environment preparation (builds/installations are MUTATING and human-controlled) takes place within PHASES 1–6 under the surface contracts (`CONTRACTS_S01_S12.md`, handoffs); PHASE 0A/0B are limited to transfer and identity validation.

Ports: D and B use 18082, so they must run sequentially; S06 (sidecar :18080) conflicts with D (upstream :18080), so they must run sequentially; S10 (8080/8000) is a separate execution contour.

## Common environment gates (all surfaces)

`IDENTITY_READY` (HEAD/tree/status baseline) | `OS_READY` (Windows x64) | `TOOLCHAIN_READY` | `ABI_READY` | `CLOCK_READY` (TTL gate where fixtures apply) | `CRYPTO_READY` | `FIXTURE_READY` | `PORTS_READY` | `SERVICES_READY` | `TRUST_READY` | `ENV_VARS_READY` (PROXY variables empty; `HACP_TRUST_KEYS_FILE` unset in test mode) | `WORKTREE_READY` (evidence outside the tree).

---

# SURFACE A — Go provenance tests + full regression
- MATRIX_ROWS: B1,B2,B3,B5,B7,B8,B10(H01/H02),B11,B12 | OWNER: S | LANG: Go ≥ 1.26.5 (`go.mod:3`)
- EXPECTED_LOCAL_REPOSITORY_PATHS: `C:\Personal\GitHub\Dev\hacp-sidecar` | REQUIRED_FILES: `internal/proxy/r4_provenance_verification_test.go` (prepared; CURRENT_ADMISSION_STATUS=PENDING), tracked test suite (159 `func Test`)
- ENVIRONMENT_PREPARATION: `go mod download` (if needed) | PREFLIGHT: `go version`; PROXY variables empty; input SHA-256; status=baseline | READINESS: 12 gates PASS
- ENTRYPOINT: `go test` | EXACT_WINDOWS_COMMANDS:
```powershell
go test .\internal\proxy -run "TestR43" -count=1 -v
go test .\internal\proxy -count=1 -v
go test ./... -count=1
```
- EXECUTION_ORDER: Phase 1, first (fast contract verification contour) | EXPECTED_RESULT: `--- PASS: TestR43H01…`, `--- PASS: TestR43H02…`; `./...` without FAIL; `func Test` count = 159
- PASS_CRITERIA: all three commands succeed (PROCESS_RC=0); SKIP=0 | ENVIRONMENT_RED: compile/toolchain/deps | BEHAVIORAL_RED: executed assertion failure (H01/H02 or other `--- FAIL`)
- EVIDENCE_OUTPUT: `-v` stdout, RC, pre/post guards | CLEANUP: n/a (in-process) | POST_RUN_GUARDS: HEAD/tree/status identical; input SHA values unchanged
- DEPENDENCIES_ON_OTHER_SURFACES: Phase 0A transfer of the r4 file (untracked is sufficient for execution; admission is Phase 7)

# SURFACE B — H03 graceful-shutdown tool
- MATRIX_ROWS: B10(H03) | OWNER: S | LANG: Go; Windows-only (kernel32)
- REQUIRED_FILES: `tools/reproduction/graceful-shutdown/main.go` (prepared; CURRENT_ADMISSION_STATUS=PENDING), `hacp-sidecar.exe` (fresh build), fixture `tokens-h03.jsonl` (1 record; TTL 7200s)
- PREPARATION: build sidecar/tool under `<run>`; generate fixture; record `generated_at`/SHA | PREFLIGHT: port 18082 available / **18083 ABSENT**; `HACP_TRUST_KEYS_FILE` unset; PROXY variables empty; `TOKEN_VALIDITY_AT_START=PASS`
- ENTRYPOINT: `<run>\graceful-shutdown.exe <sidecar.exe> <fixture> <stdout> <stderr> <provenance>`
- EXECUTION_ORDER: Phase 2 | EXPECTED_RESULT: `LISTENER_READY=PASS` → `AUTHORIZED_HTTP_STATUS=502` + `AUTHORIZED_DECISION=ALLOW` → `AUTHORIZED_PROVENANCE_TRIGGER=PASS` → `CTRL_BREAK_RESULT=1` → `PROCESS_EXIT_CODE=0` → `LISTENER_REMOVED=PASS` → `PROVENANCE_PERSISTED_AFTER_SHUTDOWN=PASS` → `R4_3_H03=PASS` (exit 0)
- PASS_CRITERIA: complete marker sequence | ENVIRONMENT_RED: ports/trust/TTL/builds | BEHAVIORAL_RED: ≠502+ALLOW; CTRL_BREAK fails to stop process; provenance not preserved
- EVIDENCE: tool stdout, sidecar logs, provenance.jsonl, SHA-256, guards | CLEANUP: owned child (deferred kill); `Get-Process` check | GUARDS: S clean
- DEPENDENCIES: Phase 0A transfer of the B file (untracked is sufficient; admission is Phase 7)

# SURFACE C — Rust Core composition reproduction (A4)
- MATRIX_ROWS: A4 | OWNER: P (SPEC-owned) + S (procedure doc) | LANG: Rust MSVC + Python 3.13 + protoc
- REQUIRED_FILES: `docs/verification/rust-core-composition-reproduction.md` (prepared; see STATE below); P detached pin `3ea4adcbf6ef2548e41d340709945326026a73b6`
- PREPARATION: clone P + `git checkout --detach 3ea4adc…`; `pip install -r harness\requirements.txt`; protoc/VC tools | PREFLIGHT: HEAD=pin; tree clean; toolchains
- ENTRYPOINT: `cargo build --locked --release --bin hacp-rs-runner` → `python .\harness\harness_runner.py --runner .\hacp-rs\target\release\hacp-rs-runner.exe --vectors-dir .\vectors --manifest .\harness\conformance_manifest.json --implementation-name hacp-rs --implementation-version 0.1.0-1.1.0 --output console` → raw witness `cargo test --locked --test json_ingress_v1_red -- --nocapture`
- EXECUTION_ORDER: Phase 3 (after S05; before S04) | EXPECTED_RESULT: `BUILD_RC=0`; **`RESULTS: 13/38 passed` + `CORE_HARNESS_RC=1` — expected strict result**; reconciliation `36/38 direct + 2 adjudicated (CORE-INV1-002, CORE-RUNTIME-004)`; new regressions=0; witness `1 passed`
- PASS_CRITERIA: exact strict count and exactly 2 adjudicated | ENVIRONMENT_RED: pin/toolchain/protoc | BEHAVIORAL_RED: outcome mismatch outside the two adjudicated cases; witness failure
- EVIDENCE: console, RC, runner SHA-256, versions, identity | CLEANUP: harness terminates runner; target/ gitignored | GUARDS: clean clone
- STATE (C3, source-backed, S mirror as of 2026-10-17): MIRROR_FILE_EXISTS=YES; MIRROR_TRACKED_STATE=UNTRACKED; PACKAGE_COPY_EXISTS=NO; WINDOWS_TRANSFER_REQUIRED=TO_BE_CONFIRMED_BY_PHASE_0B; CURRENT_ADMISSION_STATUS=PENDING

# SURFACE D — Go HTTP B15
- MATRIX_ROWS: B15 | OWNER: S | LANG: Go + Python (upstream stdlib)
- REQUIRED_FILES: `benchmarks/go-http-reproduction/main.go`, `docs/verification/go-http-local-reproduction.md` (created in S mirror; Phase 0A transfer; CURRENT_ADMISSION_STATUS=PENDING)
- PREPARATION: warmup-100/measured-5000 fixtures (TTL gate); sidecar build; env `HACP_TRUST_MODE=test` (without `HACP_TRUST_KEYS_FILE`), `HACP_CONTROL_MODE=standalone`, `HACP_SIDECAR_PORT=18082`, `HACP_UPSTREAM=http://127.0.0.1:18080` | PREFLIGHT: ports 18080/18082 available; PROXY variables empty; `TOKEN_VALIDITY_AT_START=PASS`
- ENTRYPOINT: `<run>\go-http-reproduction.exe -url http://127.0.0.1:18082/api/test -warmup-file … -measured-file …`
- EXECUTION_ORDER: Phase 5 (after B; not concurrent with S06) | EXPECTED_RESULT: negative preflight `403+DENY+no forward` (H9; outside B15 verdict) → `WARMUP_FIXTURE_RECORDS=100` → JSON warmup (100/100) → `MEASURED_FIXTURE_RECORDS=5000` → JSON measured (5000/5000) → `GO_HTTP_REPRODUCTION=PASS`
- PASS_CRITERIA: A1–A4 on all requests; exit 0 | ENVIRONMENT_RED: ports/upstream/trust/fixtures/TTL; exit 3 | BEHAVIORAL_RED: A1–A4 violations; exit 1; fail-closed smoke failure (separate verdict)
- EVIDENCE: client/service stdout, JSON summaries, fixture hashes, identity, PID lifecycle | CLEANUP: owned PID sidecar→upstream | GUARDS: S clean; HEAD/tree/status identical
- DEPENDENCIES: Phase 0A transfer of D files (untracked is sufficient; admission is Phase 7)

# SURFACE S01 — Go conformance (A1 Core-38 + A5 HC2-55)
- MATRIX_ROWS: A1,A5 | OWNER: P | LANG: Go runner + Python harness | PATHS: `hacp-go/`, `harness/`, `vectors/`
- REQUIRED_FILES: existing files (NEW_FILE=NO) | PREPARATION/PREFLIGHT/READINESS: see `CONTRACTS_S01_S12.md` §S01
- ENTRYPOINT: `python .\harness\harness.py --mode cli --binary-path <run>\hacp-go.exe` (Core: `--vectors-dir .\vectors`; HC2: `--vectors-dir .\vectors\enforcement-v2` — CLI engine, `harness.py:623-627,758-796`)
- EXECUTION_ORDER: Phase 3 | EXPECTED_RESULT: Core `RESULTS: 38/38` (digest `sha256:1e167887…`); HC2 `RESULTS: 55/55` — **two separate verdicts**
- PASS_CRITERIA: exact counts | ENVIRONMENT_RED: pin/deps/runner | BEHAVIORAL_RED: vector mismatch
- EVIDENCE: console from both commands, RC, runner SHA, identity | CLEANUP: n/a | GUARDS: P clean | DEPENDENCIES: none (environment may be shared with S05)

# SURFACE S02 — TS conformance (A2 + A6)
- MATRIX_ROWS: A2,A6 | OWNER: P (hacp-ts) | LANG: TS/Node | PATHS: `hacp-ts/`
- ENTRYPOINT: `python .\harness\harness.py --mode cli --binary-path .\hacp-ts\dist\cli.js` (Core: `--vectors-dir .\vectors`; HC2: `--vectors-dir .\vectors\enforcement-v2`; CLITarget invokes `node dist\cli.js` for `.js` — `harness.py:623-627`); package regression: `npm run test:conformance`
- EXECUTION_ORDER: Phase 3 | EXPECTED_RESULT: Core 38/38 (do not combine with mapping claim 39/39); HC2 55/55 | PASS: exact counts, SKIP=0
- ENVIRONMENT_RED: node/npm/`npm install`/pin | BEHAVIORAL_RED: mismatch | EVIDENCE: output, RC, versions | GUARDS: P clean | DEPENDENCIES: none

# SURFACE S03 — Python conformance (A3 + A7)
- MATRIX_ROWS: A3,A7 | OWNER: C (+vectors P pin) | LANG: Python 3.13 | PATHS: `humanist-core/tests/conformance/`
- ENTRYPOINT: `pytest tests\conformance\test_core_vectors.py -v -rs` (Core) / `pytest tests\conformance\test_enforcement_v2_vectors.py -v -rs` (HC2)
- EXECUTION_ORDER: Phase 3 | EXPECTED_RESULT: Core 38/38; HC2 55/55; **0 skipped** | PASS: exact counts; SKIP→ENV RED
- ENVIRONMENT_RED: deps/pin | BEHAVIORAL_RED: mismatch | EVIDENCE: pytest output, RC | GUARDS: C/P clean | DEPENDENCIES: none

# SURFACE S04 — Rust HC2-55 (A8; ext. C)
- MATRIX_ROWS: A8 | OWNER: P | LANG: Rust MSVC | PATHS: `hacp-rs/`, `harness/`
- ENTRYPOINT: `python .\harness\enforcement_v2_runner.py --runner .\hacp-rs\target\release\hacp-rs-runner.exe --vectors-dir .\vectors\enforcement-v2 --implementation-name hacp-rs --output console`
- EXECUTION_ORDER: Phase 3, immediately after C | EXPECTED_RESULT: `RESULTS: 55/55` (Rust HC2 is a separate verdict; do NOT combine with Core 36/38+2)
- PASS: exact count | ENVIRONMENT_RED: toolchain/protoc | BEHAVIORAL_RED: mismatch | EVIDENCE: console, RC | GUARDS: clean clone | DEPENDENCIES: C (runner)

# SURFACE S05 — HC2-55 runtime black-box (B4)
- MATRIX_ROWS: B4 | OWNER: S runner + P harness | LANG: Go runner + Python
- ENTRYPOINT: `python .\enforcement_v2_runner.py --runner C:\Personal\GitHub\Dev\hacp-sidecar\hacp-conformance-runner.exe --vectors-dir ..\vectors\enforcement-v2 --implementation-name hacp-sidecar --implementation-version 0.5.0 --output console --verbose`
- EXECUTION_ORDER: Phase 3 | EXPECTED_RESULT: `RESULTS: 55/55` (sidecar black-box; ≠ S01 A5)
- PASS: exact count | ENVIRONMENT_RED: runner build/pin | BEHAVIORAL_RED: mismatch | EVIDENCE: verbose console, RC, runner SHA | GUARDS: S/P clean | DEPENDENCIES: runner build

# SURFACE S06 — External Python↔Go E2E (B13)
- MATRIX_ROWS: B13 | OWNER: C↔S | LANG: Python + Go
- EXACT MODE (single mode, no alternatives): SIDECAR_START_MODE=EXTERNAL; `HACP_SIDECAR_EXTERNAL=1`, `HACP_SIDECAR_URL=http://127.0.0.1:18080`, `HACP_SIDECAR_BIN=SET`, `HACP_SIDECAR_REPO=UNSET`, `HACP_SIDECAR_ARGS=NOT_APPLICABLE`; `HACP_TEST_PRIVATE_KEY`/`HACP_TEST_SIGNER_KEY_ID`=SET (key id `key-ed25519-test-001`, do not disclose values); PROCESS_OWNER=HUMAN; STARTUP_READINESS=TCP `_wait_for_tcp`; CLEANUP_OWNER=HUMAN; mock upstream :18084 is mandatory (ALLOW tests require `200`)
- ENTRYPOINT: `pytest tests\test_hacp_sidecar_integration.py -v -rs` (complete contract: `CONTRACTS_S01_S12.md` §S06)
- EXECUTION_ORDER: Phase 4 (not concurrent with D) | EXPECTED_RESULT: 5 passed, 0 skipped | PASS: 5/5; SKIP≠PASS
- ENVIRONMENT_RED: binary/env/port | BEHAVIORAL_RED: test failure | EVIDENCE: pytest `-rs`, RC | CLEANUP: pytest fixture; `Get-Process` | GUARDS: S/C clean | DEPENDENCIES: sidecar build

# SURFACE S07 — Anti-bypass (B14)
- MATRIX_ROWS: B14 | OWNER: S | LANG: PowerShell + Docker
- ENTRYPOINT: `powershell -File .\scripts\test-anti-bypass.ps1` (compose: `deployments\docker-compose.yml`)
- EXECUTION_ORDER: Phase 4 | EXPECTED_RESULT: `Result: 5 passed, 0 failed`, exit 0
- PASS: 5/5 AB-E2E | ENVIRONMENT_RED: Docker/ports/pull/startup-window | BEHAVIORAL_RED: AB assertion failure | EVIDENCE: stdout, `docker compose ps`, logs | CLEANUP: `docker compose -f deployments\docker-compose.yml down -v` | GUARDS: no remaining containers | DEPENDENCIES: none

# SURFACE S08 — Rust HTTP runtime + B19-Rust-subset (B16)
- MATRIX_ROWS: B16, B19(Rust subset) | OWNER: S+P | LANG: Go client + Rust MSVC
- ENTRYPOINT: R8.2 procedure (`docs/verification/rust-http-local-reproduction.md`): cargo build `--locked --offline --bin hacp-rs-proxy` → fixtures 100/5000 → proxy `:18081` (owned PID) → negative smoke 403+DENY → `go run .\benchmarks\rust-http-reproduction -url http://127.0.0.1:18081/api/test -warmup-file … -measured-file …`
- EXECUTION_ORDER: Phase 5 | EXPECTED_RESULT: smoke PASS; `R8_2_WORKLOAD=PASS`; JSON warmup(100)+measured(5000); timing INFORMATIONAL
- PASS: all client assertions (ALLOW + request-id), counts | ENVIRONMENT_RED: toolchain/port/fixtures/TTL | BEHAVIORAL_RED: assertion/smoke failure | EVIDENCE: complete R8.2 evidence set | CLEANUP: owned PID | GUARDS: S/P clean | DEPENDENCIES: none (B19 subset consolidated)

# SURFACE S09 — Rust package qualification (B18)
- MATRIX_ROWS: B18 | OWNER: P | LANG: Rust MSVC
- ENTRYPOINT: `cargo +stable-x86_64-pc-windows-msvc test --locked` (from `hacp-rs`)
- EXECUTION_ORDER: Phase 3 | EXPECTED_RESULT: 48 passed + 1 ignored (`ed25519_v1_red` witness) + 0 failed; **bind claim to P_HEAD/P_TREE (H2)**
- PASS: exact accounting | ENVIRONMENT_RED: toolchain/lock | BEHAVIORAL_RED: failure | EVIDENCE: cargo output, RC, P identity | GUARDS: P clean | DEPENDENCIES: none

# SURFACE S10 — Benchmark workloads (B19: Go serial / PS rotating / Go rotating)
- MATRIX_ROWS: B19 (remaining portion) | OWNER: S | LANG: PowerShell + Go
- ENTRYPOINT: `.\benchmarks\benchmark_serial.ps1`; `.\benchmarks\benchmark_rotating.ps1`; `go run .\benchmarks\benchmark-rotating`; execution contour: sidecar :8080 + upstream :8000 (owned), `benchmarks\tokens.jsonl` (1000, fresh; TTL gate)
- EXECUTION_ORDER: Phase 6 (last; INFORMATIONAL) | EXPECTED_RESULT: console statistics p50/p95/p99/max (+RPS); verdict-like "GATE D …" lines are historical markers
- PASS_CRITERIA: informational (PASS/FAIL does not depend on latency); request failures are recorded | ENVIRONMENT_RED: ports/fixtures/TTL | BEHAVIORAL_RED: not applicable
- EVIDENCE: stdout, RC, fixture hashes | CLEANUP: owned PID sidecar→upstream | GUARDS: S clean | DEPENDENCIES: none; consider D-08/D-09 when interpreting

# SURFACE S11 — Signing/canonicalization (B6; new file)
- MATRIX_ROWS: B6 | OWNER: S | LANG: Go | REQUIRED_FILES: `S11_signature_canonicalization_test.go.overlay` (in package) → location `test/matrix/t_go/S11_signature_canonicalization_test.go.overlay`
- ENTRYPOINT: `Scripts\02-Run-Verification.ps1 -Surface S11` (Go overlay → `internal\wire`) | EXECUTION_ORDER: Phase 1; repository-owned source may remain untracked until separate Phase 7 admission
- EXPECTED_RESULT: PASS: canonicalization (sort/whitespace/array/big-int/solidus/line-feed/invalid/VerifyCanonicalization/RemoveSignatureField) + signing (valid/tampered/corrupted/wrong-key/canonical-binding)
- PASS: all assertions PASS | ENVIRONMENT_RED: compile | BEHAVIORAL_RED: assertion failure (for example, signature verified over non-canonical bytes)
- EVIDENCE: `-v` output, RC, SHA-256 repository-owned source + generated overlay mapping | CLEANUP: n/a | GUARDS: S clean | DEPENDENCIES: repository-owned S11 source present; admission is Phase 7

# SURFACE S12 — Budget/replay (B9; new files)
- MATRIX_ROWS: B9 | OWNER: S | LANG: Go | REQUIRED_FILES: `S12_budget_ledger_test.go.overlay` → `test/matrix/t_go/S12_budget_ledger_test.go.overlay`; `S12_budget_replay_verification_test.go.overlay` → `test/matrix/t_go/S12_budget_replay_verification_test.go.overlay`
- ENTRYPOINT: `Scripts\02-Run-Verification.ps1 -Surface S12` (Go overlays → `internal\budget` + `internal\proxy`)
- EXECUTION_ORDER: Phase 1; repository-owned sources may remain untracked until separate Phase 7 admission | EXPECTED_RESULT: unit (exactly-maxUses/reuse-deny/invalid/independence) + integration (default maxUses=1 deny; max_uses=2 exhaustion; replay stays denied) with `403 + DENY + BUDGET_EXHAUSTED` and no forwarding
- PASS: all assertions PASS | ENVIRONMENT_RED: compile | BEHAVIORAL_RED: replay accepted / exhaustion not enforced
- EVIDENCE: `-v` output, RC, SHA-256 repository-owned sources + generated overlay mappings | CLEANUP: n/a | GUARDS: S clean | DEPENDENCIES: repository-owned S12 sources present; S12b uses tracked helpers from `internal/proxy/request_binding_test.go`; admission is Phase 7

---

## Final readiness summary

| SURFACE | MATRIX_ROWS | ENV | EXEC | FILES | STATIC | UNRESOLVED | PREPARED_FOR_WINDOWS_VERIFICATION |
|---|---|---|---|---|---|---|---|
| A | B1,B2,B3,B5,B7,B8,B10,B11,B12 | READY | READY | YES | PASS | 0 | YES |
| B | B10(H03) | READY | READY | YES | PASS | 0 | YES |
| C | A4 | READY | READY | YES | PASS | 0 | YES |
| D | B15 | READY | READY | YES | PASS | 0 | YES |
| S01 | A1,A5 | READY | READY | YES | PASS | 0 | YES |
| S02 | A2,A6 | READY | READY | YES | PASS | 0 | YES |
| S03 | A3,A7 | READY | READY | YES | PASS | 0 | YES |
| S04 | A8 | READY | READY | YES | PASS | 0 | YES |
| S05 | B4 | READY | READY | YES | PASS | 0 | YES |
| S06 | B13 | READY | READY | YES | PASS | 0 | YES |
| S07 | B14 | READY | READY | YES | PASS | 0 | YES |
| S08 | B16,B19-subset | READY | READY | YES | PASS | 0 | YES |
| S09 | B18 | READY | READY | YES | PASS | 0 | YES |
| S10 | B19 remaining portion | READY | READY | YES | PASS | 0 | YES |
| S11 | B6 | READY | READY | YES (in package) | PASS | 0 | YES (after Phase 0A transfer) |
| S12 | B9 | READY | READY | YES (in package) | PASS | 0 | YES (after Phase 0A transfer) |

`FULL_VERIFICATION_PACKAGE=PREPARED_FOR_TRANSFER_AND_WINDOWS_EXECUTION` (condition: Phase 0A transfer per `Readme_map_file.md`; admission is NOT required for execution).

---

POST-EXECUTION ADMISSION IS A SEPARATE HUMAN DECISION. PHASE 7 = `ADMISSION = PENDING HUMAN DECISION`; it is NOT automatically authorized. After separate explicit human GO: explicit paths only, no `git add .`, no `git add -A`, signed commits, verify signature, verify HEAD/tree/status, no unrelated files. Admission-state terminology: PREPARED_IN_PACKAGE → TRANSFERRED_UNTRACKED → HUMAN_VERIFIED → ADMISSION_PENDING → ADMITTED.
