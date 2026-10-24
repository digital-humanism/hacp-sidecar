# CONTRACTS S01–S12 — environment + execution contracts (Windows human execution)

Date: 2026-10-17
Status: `INTERNAL / PREPARATION / NOT TEST EVIDENCE / NOT ACCEPTANCE EVIDENCE / NOT ADMISSION AUTHORIZATION`
Basis: `SIDECAR_MATRIX_FINAL_PREPARATION_AUDIT_RU_2026-10-17.md` (S01–S12 are not subject to reopening).
Model: `VERIFICATION SURFACE = ENVIRONMENT CONTRACT + EXECUTION CONTRACT + EXACT FILE/TOOL IDENTITY + EXPECTED EVIDENCE`.

Common rules for all S01–S12:
- Execution: HUMAN / WINDOWS using sibling checkouts under an arbitrary `<workspace>` root: `hacp-sidecar`, `hacp-spec`, and `humanist-core`.
- Identity gates before/after each run: `git rev-parse HEAD`, `git rev-parse HEAD^{tree}`, `git status --short` (baseline; no synthetic cross-environment counters).
- Evidence directory is outside working trees; cleanup is limited to owned processes.
- Verdict boundaries: **Core ≠ HC2** (separate verdicts even when sharing a toolchain); **Rust Core = 36/38 direct + 2 adjudicated (NOT 38/38 direct)**; **Rust HC2 = 55/55, a separate verdict**; **Go HTTP B15 = authorization + real upstream forwarding + real upstream response**; **Rust HTTP B16 = authorization + local response, with NO upstream-forwarding claim**; **Timing = INFORMATIONAL, with no established normative threshold**.
- Execution exit classification: `PROCESS_RC=0` → success; `PROCESS_RC!=0` → classification based on actual output (ENVIRONMENT RED = compile/toolchain/dependency/setup; BEHAVIORAL RED = executed assertion failure). For tools with explicit exit-code contracts (D, S08), use their respective contracts.

---

# S01 — Go conformance (A1 Core-38 + A5 HC2-55)

ENVIRONMENT CONTRACT: REPO=P (`hacp-spec`, detached pin `3ea4adcbf6ef2548e41d340709945326026a73b6` for Core; HC2 uses the same pin), LANG=Go runner (built from `hacp-go/` or using `hacp-conformance-runner.exe` from S — see EXEC), Python 3.13.x + `harness\requirements.txt`; no services/ports/fixtures; `CARGO_TARGET_DIR` n/a; proxy variables do not affect execution (no HTTP traffic); clean tree before/after.
ENVIRONMENT PREPARATION (MUTATING, human): clone P → `git checkout --detach 3ea4adc…`; `pip install -r harness\requirements.txt`; build Go CLI (`go build -o <run>\hacp-go.exe .` in `hacp-go`; interface `hacp-go evaluate --vector <path>` — `hacp-go/main.go`).
ENVIRONMENT PREFLIGHT: `git rev-parse HEAD` = pin; `git status --short` empty; `go version` ≥ 1.26.x; `python --version` ≥ 3.13; `Get-Command go,python`.
READINESS GATES: IDENTITY/OS/TOOLCHAIN/ABI/CLOCK(n/a)/CRYPTO(conformance keys in `harness/keys`)/FIXTURE(vectors+manifest)/PORTS(n/a)/SERVICES(n/a)/TRUST(n/a)/ENV_VARS/WORKTREE — all PASS → `ENVIRONMENT_READY=PASS`.
ENTRYPOINT: `python .\harness\harness.py --mode cli` (legacy CLI engine; `hacp-go` is single-shot `evaluate --vector`; invocation `<binary> evaluate --vector <tmp>` + node branch for `.js` — `harness.py:623-627`; flags `--mode/--binary-path/--vectors-dir/--output` — `harness.py:758-796`).
EXACT WINDOWS COMMANDS (build Go CLI using `go build -o <run>\hacp-go.exe .` in `hacp-spec\hacp-go`; run from `hacp-spec`):
```powershell
# Core-38 (A1) — separate verdict
python .\harness\harness.py --mode cli --binary-path <run>\hacp-go.exe --vectors-dir .\vectors --output console
# HC2-55 (A5) — separate verdict
python .\harness\harness.py --mode cli --binary-path <run>\hacp-go.exe --vectors-dir .\vectors\enforcement-v2 --output console
```
EXECUTION ORDER: Core → HC2 (two blocks, two verdicts).
EXPECTED RESULT: Core `RESULTS: 38/38 passed` (+digest `sha256:1e167887…`); HC2 `RESULTS: 55/55 passed`. PASS CRITERIA: exact counts; RC according to the harness contract.
ENVIRONMENT RED: incorrect pin; missing protoc…(n/a); missing Python dependencies; missing runner build. BEHAVIORAL RED: any vector mismatch (count differs from expected).
EVIDENCE OUTPUT: complete console output from both blocks, RC, runner SHA-256, identity gates, toolchain versions.
CLEANUP: harness processes terminate on their own; build artifacts reside outside the tree or are gitignored. POST-RUN GUARDS: P tree clean; HEAD=pin.
DEPENDENCIES: S05 is not required (this surface uses its own Go runner); sharing the environment with S05 is permitted, but verdicts remain separate.

# S02 — TypeScript conformance (A2 Core-38 + A6 HC2-55)

ENVIRONMENT: REPO=P (`hacp-ts/`), Node.js + npm (record versions using `node --version; npm --version`); `npm install` if necessary (`node_modules` may already exist); clean tree.
PREFLIGHT: `node --version`, `npm --version`, `Get-Command node,npm`; pin-check P.
ENTRYPOINT: `python .\harness\harness.py --mode cli --binary-path .\hacp-ts\dist\cli.js` (legacy CLI engine; `CLITarget` automatically invokes `node dist\cli.js evaluate --vector <tmp>` for `.js` targets — `harness.py:623-627`; CLI interface `hacp-ts evaluate --vector <path>` — `hacp-ts/src/cli.ts:51-56`).
EXACT WINDOWS COMMANDS (after `npm run build` in `hacp-ts`; run from `hacp-spec`):
```powershell
# Core-38 (A2) — separate verdict
python .\harness\harness.py --mode cli --binary-path .\hacp-ts\dist\cli.js --vectors-dir .\vectors --output console
# HC2-55 (A6) — separate verdict
python .\harness\harness.py --mode cli --binary-path .\hacp-ts\dist\cli.js --vectors-dir .\vectors\enforcement-v2 --output console
# package regression (associated A2 contract; do NOT combine with Core claim)
npm run test:conformance
```
EXECUTION ORDER: Core → HC2, separate verdicts.
EXPECTED: Core-38 mapping (historically: 1 inventory assertion + 38 vectors = 39/39); HC2 `55/55`.
ENV RED: missing node/npm; incorrect pin. BEHAVIORAL RED: vector mismatch.
EVIDENCE: npm/harness output, RC, versions, identity gates. CLEANUP: processes terminate; do not commit `node_modules`. GUARDS: clean tree.
DEPENDENCIES: none. CLAIM: do not derive the TS Core claim from the aggregate npm count (historical A2 qualification).

# S03 — Python conformance (A3 Core-38 + A7 HC2-55)

ENVIRONMENT: REPO=C (`humanist-core`), Python 3.13.x, `pip install -e ".[dev]"` (per C CI); vectors from adjacent P (pin `3ea4adc…`; record P identity).
PREFLIGHT: `python --version`; `pytest --version`; P pin-check.
COMMANDS:
```powershell
pytest tests\conformance\test_core_vectors.py -v -rs      # Core-38 (A3) — separate verdict
pytest tests\conformance\test_enforcement_v2_vectors.py -v -rs   # HC2-55 (A7) — separate verdict
```
EXPECTED: Core 38/38; HC2 55/55; SKIP=0 (`-rs` mandatory; SKIP≠PASS).
ENV RED: missing python/pytest/deps; P pin mismatch. BEHAVIORAL RED: vector mismatch; SKIP when the environment is present (classify as ENV RED).
EVIDENCE: complete pytest output, RC, identity gates. CLEANUP: n/a. GUARDS: C/P clean.
DEPENDENCIES: none.

# S04 — Rust HC2-55 (A8; extension of C)

ENVIRONMENT: identical to C (P detached pin `3ea4adc…`; Rust `stable-x86_64-pc-windows-msvc`; protoc REQUIRED (`hacp-rs/build.rs:3`); Python+harness; `Cargo.lock`). Runner `hacp-rs-runner` is already built by the C procedure.
COMMANDS:
```powershell
python .\harness\enforcement_v2_runner.py --runner .\hacp-rs\target\release\hacp-rs-runner.exe --vectors-dir .\vectors\enforcement-v2 --implementation-name hacp-rs --output console
```
EXECUTION ORDER: after A4 block (C) — same environment, separate verdict.
EXPECTED: `RESULTS: 55/55 passed` (Rust HC2 is a separate family; do NOT combine with Core 36/38+2).
ENV RED: toolchain/protoc/pin. BEHAVIORAL RED: any HC2 mismatch.
EVIDENCE: console, RC, identity. GUARDS: clone clean. DEPENDENCIES: C (runner build).

# S05 — HC2-55 runtime black-box (B4)

ENVIRONMENT: REPO=S (runner `hacp-conformance-runner.exe`, built using `go build -o <run>\hacp-conformance-runner.exe .\cmd\hacp-conformance-runner`) + P (harness, vectors/enforcement-v2, pin). Python 3.13.
COMMANDS (from `hacp-spec\harness`):
```powershell
python .\enforcement_v2_runner.py --runner ..\..\hacp-sidecar\hacp-conformance-runner.exe --vectors-dir ..\vectors\enforcement-v2 --implementation-name hacp-sidecar --implementation-version 0.5.0 --output console --verbose
```
EXPECTED: `RESULTS: 55/55 passed` (sidecar runtime black-box). Distinguish from S01 A5 (SPEC-owned Go runner); verdicts are separate.
ENV RED: runner build; pin; dependencies. BEHAVIORAL RED: mismatch (including reason-code classes — see historical qualification concerning strict-reason diagnostics).
EVIDENCE: console+verbose, RC, runner SHA-256. GUARDS: S/P clean. DEPENDENCIES: none (sharing the environment with S01 is permitted).

# S06 — External Python↔Go E2E (B13)

ENVIRONMENT: C + S; Python 3.13 + C dependencies (`pip install -e .` in `hacp-sidecar-integration`, `pip install -r requirements-test.txt`); freshly built sidecar `<run>\hacp-sidecar.exe` (`go build -o <run>\hacp-sidecar.exe .\cmd\sidecar`); S mock upstream `deployments\upstream\server.py --port 18084`; test key material (see KEY MATERIAL); ports: sidecar 18080, upstream 18084.
S06 EXACT MODE — one mode, no alternatives (source-backed: fixture attach branch `tests/test_hacp_sidecar_integration.py:60-72`; historical evidence name `R6_2_EXTERNAL_E2E`; spawn mode excluded: fixture requires nonempty `HACP_SIDECAR_ARGS` for spawn (`:74-77`), but the sidecar binary accepts no CLI arguments — configuration is environment-only):
```text
SIDECAR_START_MODE     = EXTERNAL
SIDECAR_BINARY_SOURCE  = fresh build `go build -o <run>\hacp-sidecar.exe .\cmd\sidecar` (S worktree)
SIDECAR_COMMAND        = <run>\hacp-sidecar.exe (env: HACP_SIDECAR_PORT=18080, HACP_UPSTREAM=http://127.0.0.1:18084, HACP_TRUST_MODE/HACP_TRUST_KEYS_FILE per Integration doc)
SIDECAR_ARGS           = NOT_APPLICABLE (binary has no CLI arguments; environment-only configuration)
SIDECAR_PORT           = 18080
HACP_SIDECAR_BIN       = SET (mandatory: `sidecar_bin` fixture skips the suite if the binary cannot be resolved — conftest.py:35-43)
HACP_SIDECAR_REPO      = UNSET (avoid ambiguous search behavior in conftest.py:10-12)
HACP_SIDECAR_EXTERNAL  = SET (=1)
HACP_SIDECAR_URL       = SET (=http://127.0.0.1:18080)
HACP_TEST_PRIVATE_KEY  = SET (path to hacp-test-key.pem; missing variable → pytest.skip (:97); missing file → pytest.fail (:100-101))
HACP_TEST_SIGNER_KEY_ID= SET (=key-ed25519-test-001; source: Integration doc:255)
PROCESS_OWNER          = HUMAN
STARTUP_READINESS      = `_wait_for_tcp` (HTTPX TCP 1s × 30 — test file:44-58) + human preflight of upstream `/health`
CLEANUP_OWNER          = HUMAN (stop owned sidecar+upstream PIDs; confirm using `Get-Process`)
```
COMMANDS:
```powershell
pytest tests\test_hacp_sidecar_integration.py -v -rs
```
EXPECTED: 5 passed, **0 skipped** (otherwise BLOCKED/ENV RED — SKIP≠PASS), exit 0. Actual expectations of the five tests: missing headers → `X-HACP-Decision != ALLOW` (`:145-147`); action hash matches the sidecar form (`:149-171`); signatures are self-consistent (`:173-184`); valid signing → `ALLOW`+`200` (`:186-206`); SidecarClient → `ALLOW`+`200` (`:208-224`).
KEY MATERIAL: deterministic test-only Ed25519; source = seed procedure in `humanist-core\docs\Integration with HACP Sidecar.md` §8 (seed value is NOT transferred into package documents); key id `key-ed25519-test-001`; export PKCS8 PEM → `hacp-test-key.pem` (expected path `humanist-core\tests\hacp-test-key.pem` — `HACP_SDK_VERIFICATION_AND_TEST_HARDENING.md:542`); validity: request time inside the signed pair TTL (`issued_at..expires_at = now..now+300` — `:115-133`) and `HACP_TIME_TOLERANCE` → TTL_GATE=APPLIES (pair TTL 300s; `X-HACP-Expires-In`/fixture-TTL=7200s NOT_APPLICABLE — generate-tokens not used).
ENV RED: missing binary/environment/keys; occupied port; missing upstream; TTL/clock. BEHAVIORAL RED: any of the five tests fails after ENVIRONMENT_READY=PASS.
EVIDENCE: pytest output `-rs`, RC, environment fixtures (without disclosing key values), identity gates. CLEANUP: stop sidecar+upstream PIDs; verify via `Get-Process`. GUARDS: S/C clean.
DEPENDENCIES: Phase 0A transfer; upstream mock (shared preparation with B/D contour).

# S07 — Anti-bypass AB-E2E-01..05 (B14)

ENVIRONMENT: S + Docker Desktop + compose; `deployments/docker-compose.yml` (upstream/control-plane/sidecar/agent services; untrusted/protected/control networks; provenance-data volume); network access for `alpine:latest` pull; compose-file ports available.
PREFLIGHT: `docker compose version`; `docker compose -f deployments\docker-compose.yml ps` (no unrelated containers permitted); `Get-NetTCPConnection` for compose ports.
COMMAND: `powershell -File .\scripts\test-anti-bypass.ps1` (expected `Result: 5 passed, 0 failed`; `exit 1` on any failure).
ENV RED: missing Docker; occupied ports; unavailable pull; startup health-check window exceeded (NOT PASS and NOT AB failure — retry).
BEHAVIORAL RED: any AB-E2E-01..05 failure (anti-bypass boundary violation).
EVIDENCE: script stdout, `docker compose ps`, logs on failure, RC. CLEANUP: `docker compose -f deployments\docker-compose.yml down -v` (same file; no prune). GUARDS: no remaining containers.
DEPENDENCIES: none.

# S08 — Rust HTTP runtime + B19-Rust-subset (B16)

ENVIRONMENT: S (client `benchmarks/go-http-reproduction`… no — Rust HTTP client: `benchmarks/rust-http-reproduction/main.go`) + P (`hacp-rs` proxy, `src/bin/hacp-rs-proxy.rs`); Rust MSVC (build `--locked --offline --bin hacp-rs-proxy`, `CARGO_TARGET_DIR` outside repository); Go ≥ 1.26.5; fresh fixtures (`generate-tokens -count 100/5000`); TTL 7200s (`generate-tokens/main.go:40`) → `TOKEN_VALIDITY_AT_START=PASS`; port 18081; **no upstream** (local response); trust: `HACP_TRUST_MODE` per R8.2 procedure (conformance key).
PROCEDURE: `docs/verification/rust-http-local-reproduction.md` §1-9 (prepared; tracked).
COMMANDS (condensed): cargo build proxy → fixtures → start proxy `HACP_SIDECAR_PORT=18081` (owned PID) → negative smoke (403 + `X-HACP-Decision: DENY` without headers) → `go run .\benchmarks\rust-http-reproduction -url http://127.0.0.1:18081/api/test -warmup-file <run>\tokens-warmup-100.jsonl -measured-file <run>\tokens-5000.jsonl`.
EXPECTED: `R8_2_WORKLOAD=PASS`; JSON summaries warmup(100)+measured(5000); fail-closed smoke PASS. **Timing is INFORMATIONAL (B19 subset); no thresholds; NO upstream-forwarding claims; NO Go-vs-Rust ranking.**
ENV RED: toolchain/port/fixtures/TTL. BEHAVIORAL RED: client class-A violations (decision/request-id), fail-closed smoke failure.
EVIDENCE: complete R8.2 evidence set (build log, proxy SHA-256, fixture hashes, smoke JSON, summaries, RC). CLEANUP: stop owned PID, check listener. GUARDS: S/P clean.
DEPENDENCIES: none.

# S09 — Rust package qualification (B18)

ENVIRONMENT: P (`hacp-rs`), Rust MSVC, `Cargo.lock`; clean tree; **claim identity binding (H2)**: record `P_HEAD`/`P_TREE` — the "48 active + 1 ignored" claim applies only to that identity.
COMMAND: `cargo +stable-x86_64-pc-windows-msvc test --locked` (from `hacp-rs`).
EXPECTED: 48 passed, 1 ignored (`ed25519_v1_red` historical witness), 0 failed.
ENV RED: toolchain/lock. BEHAVIORAL RED: failure (new regression) — record, do not fix.
EVIDENCE: complete cargo output, RC, P identity. CLEANUP: target/ gitignored. GUARDS: P clean.
DEPENDENCIES: none.

# S10 — Benchmark workloads Go/PS (B19: Go serial, PS rotating, Go rotating)

ENVIRONMENT: S; Go ≥ 1.26.5; PowerShell 5.1+ (rotating requires 7+ with `ForEach-Object -Parallel`); running sidecar :8080 + upstream :8000 (block-owned, launch: env `HACP_SIDECAR_PORT=8080`, `HACP_UPSTREAM=http://127.0.0.1:8000`, `HACP_CONTROL_MODE=standalone`, `HACP_TRUST_MODE=test`; `python deployments\upstream\server.py 8000`); fresh `benchmarks/tokens.jsonl` (`go run .\benchmarks\generate-tokens -count 1000 -out benchmarks\tokens.jsonl -method GET -path /api/test`); TTL gate; empty proxy variables; ports available before startup.
COMMANDS: `.\benchmarks\benchmark_serial.ps1`; `.\benchmarks\benchmark_rotating.ps1`; `go run .\benchmarks\benchmark-rotating`.
EXPECTED: console statistics (p50/p95/p99/max; RPS where reported). **TIMING=INFORMATIONAL; current normative Gate D threshold = NOT ESTABLISHED; PASS/FAIL does not depend on latency; "GATE D …" verdict lines in script output are historical markers, not a normative gate.** Known measurement defects (D-08 exit contract, D-09 warmup) must be considered during interpretation.
ENV RED: ports/configuration/fixtures/TTL. BEHAVIORAL RED: not applicable (informational); record request failures as observed facts.
EVIDENCE: stdout from all three runs, RC, fixture hashes/generated_at, identity gates. CLEANUP: stop owned PIDs (sidecar→upstream); removing `benchmarks/tokens.jsonl` is not required (gitignored). GUARDS: S clean.
DEPENDENCIES: none.

# S11 — Signing / canonicalization (B6; NEW verification file)

ENVIRONMENT: S; Go ≥ 1.26.5; no services/fixtures/ports; in-process test.
REQUIRED FILE: `test/matrix/t_go/S11_signature_canonicalization_test.go.overlay` (repository-owned verification file; see Readme_map_file.md).
ENTRYPOINT: `Scripts/02-Run-Verification.ps1 -Surface S11`; repository-owned source `test/matrix/t_go/S11_signature_canonicalization_test.go.overlay` is bound by Go overlay into package `internal/wire`.
EXPECTED: all file tests PASS: canonicalization (key sort, whitespace, array order, big-int, solidus, line-feed escape, invalid input, VerifyCanonicalization, RemoveSignatureField) + signing failure classes (valid/tampered/corrupted/wrong-key/canonical-binding).
EXIT CLASSIFICATION: PROCESS_RC model (C1 rule).
ENV RED: compile/missing toolchain. BEHAVIORAL RED: assertion failure (for example, signature verified over non-canonical bytes).
EVIDENCE: `-v` output, RC, file SHA-256, identity gates. CLEANUP: n/a. GUARDS: S clean; repository-owned verification source resides at `test/matrix/t_go/S11_signature_canonicalization_test.go.overlay`.
DEPENDENCIES: repository-owned source `test/matrix/t_go/S11_signature_canonicalization_test.go.overlay`; execution binding — BOUND / VERIFIED via `Scripts/02-Run-Verification.ps1` Go overlay; admission is Phase 7, a separate human decision.

# S12 — Budget / replay (B9; NEW verification files)

ENVIRONMENT: S; Go ≥ 1.26.5; in-process (httptest); no external services.
REQUIRED FILES: `test/matrix/t_go/S12_budget_ledger_test.go.overlay`; `test/matrix/t_go/S12_budget_replay_verification_test.go.overlay` (repository-owned verification files).
ENTRYPOINT:
```powershell
# Scripts/02-Run-Verification.ps1 -Surface S12 binds this source by Go overlay into internal/budget.
# The same S12 invocation binds this source by Go overlay into internal/proxy.
```
EXPECTED: unit — exactly-maxUses, reuse-deny, invalid-inputs, independence; integration — default maxUses=1 denies the second use, max_uses exhaustion (2 uses → deny), replay remains denied; denial = 403 + `X-HACP-Decision: DENY` + `X-HACP-Reason: BUDGET_EXHAUSTED`; denied request is NOT forwarded (upstream count is checked).
BOUNDARY: token budget/replay semantics; NOT control-plane event replay.
EXIT CLASSIFICATION: PROCESS_RC model.
ENV RED: compile. BEHAVIORAL RED: assertion failure (for example, replay accepted).
EVIDENCE: `-v` output, RC, file SHA-256 hashes, identity gates. CLEANUP: n/a. GUARDS: S clean.
DEPENDENCIES: repository-owned sources `test/matrix/t_go/S12_budget_ledger_test.go.overlay` and `test/matrix/t_go/S12_budget_replay_verification_test.go.overlay`; execution binding — BOUND / VERIFIED via `Scripts/02-Run-Verification.ps1` Go overlays; admission is Phase 7; S12b uses tracked helpers `signedBindingEnvelope`/`signBindingObject` from `internal/proxy/request_binding_test.go`.

---

# Contract status summary

| SURFACE | ENVIRONMENT_CONTRACT_COMPLETE | EXECUTION_CONTRACT_COMPLETE | REQUIRED_FILES_PRESENT | STATIC_VALIDATION | UNRESOLVED | PREPARED_FOR_WINDOWS_VERIFICATION |
|---|---|---|---|---|---|---|
| S01–S10 | YES | YES | YES (existing tests/tools) | PASS (reference check) | 0 | YES |
| S11 | YES | BOUND / VERIFIED | YES (repository-owned) | PASS | 0 | YES |
| S12 | YES | BOUND / VERIFIED | YES (repository-owned) | PASS | 0 | YES |
