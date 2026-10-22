# STATIC EXECUTABILITY AUDIT — RUST WINDOWS ORCHESTRATION

**Date:** 2026-10-17
**Directory:** `Result_Audit/Artefact_Verification_package/Rust/Scripts_RUST`
**Scope:** complete static executability audit of the unified PowerShell orchestration for previously executed Rust procedures. NO EXECUTION / NO RUNTIME REVALIDATION.
**Authority:** THIS TASK + SHA-pinned 7-file set + read-only inspection S/P source + `WINDOWS_EXECUTION_MANIFEST_RS.md` / `RUST_VERIFICATION_CONTRACTS.md` / `RUST_MATRIX_RECONCILIATION.md` (read-only).

---

## 0. Residual fixes applied in this pass

| FIX | DEFECT | ACTION | STATUS |
|---|---|---|---|
| **R1** | completed historical run vs active continuation candidate | explicit `RUN_STATE` lifecycle (`CREATED→PREFLIGHT_COMPLETE→PREPARED→VERIFICATION_COMPLETE→POSTFLIGHT_COMPLETE→RECONCILED`) + phase-compatible auto-resolve (`-RequireState`); `RECONCILED` run never eligible | **PASS** |
| **R2** | owned-process registration could record empty StartTime | `ProcessStartTimeUtc` now MANDATORY at registration (`PROCESS_OWNERSHIP_REGISTRATION=FAIL` otherwise); RS04 wraps Start-Process+Register in try/catch → `ENVIRONMENT_RED`; cleanup verifies PID+StartTime+path before kill | **PASS** |
| **ENV** | §21 proxy env gap (Go workload uses `http.DefaultTransport`/`ProxyFromEnvironment`) | `NO_PROXY=localhost,127.0.0.1` guard for the RS04 localhost probe/workload (appended, does not clobber) | **FIXED** |

---

## A. Script → contract mapping

| SCRIPT | FUNCTION/BLOCK | CONTRACT SOURCE | RS SURFACE | MATRIX ROWS | INPUTS | OUTPUTS | DEPENDENCIES | EXPECTED EXIT |
|---|---|---|---|---|---|---|---|---|
| `00-Preflight.ps1` | preflight gates | MANIFEST_RS §preflight | — (0B) | — | S/P git, toolchain, RS07 source, port 18081 | `PRECHECK.json/txt`, `rs07-transfer-check.json`, `run-context.json` (RUN_STATE=CREATED→PREFLIGHT_COMPLETE) | none (entry) | 0 (PASS) / 2 (BLOCKED) |
| `01-Prepare-Environment.ps1` | RS07 transfer, deps, builds, fixtures | CONTRACTS_RS RS01/RS02/RS04/RS07 | build RS01/RS02 runner + RS04 proxy | — | `PRECHECK.json`=PASS, RS07 source | `build-inventory.json`, `fixture-inventory.json`, RS07 → `hacp-rs/tests/` | 00 PASS | 0 / 2 |
| `02-Run-Verification.ps1` | RS01→RS02→RS03→RS04→RS05→RS07 | CONTRACTS_RS RS01–RS07 | RS01(A4) RS02(A8) RS03(B18,B6) RS04(B16,B17) RS05(B19) RS07(B9) | A4,A8,B6,B9,B16,B17,B18,B19 | `PREPARE_RESULT`=PASS, builds, fixtures | per-surface `result.json`/`identity.json`, `run-summary.json` | 01 PASS; RS05←RS04 | 0 |
| `03-Postflight.ps1` | cleanup + git guards + evidence inventory | MANIFEST_RS §postflight | — | — | `run-summary.json` | `POSTFLIGHT.json` (CLEANUP/GIT/EVIDENCE) | 02 ran | 0 |
| `04-Reconcile-Results.ps1` | 36-row reconciliation | MATRIX_RECONCILIATION | map rows→RS | 36 | `POSTFLIGHT.json`, per-surface results | `RUST_VERIFICATION_SUMMARY.json/md`, `counters.txt` | 03 (POSTFLIGHT_COMPLETE) | 0 |
| `WindowsVerification.Rust.Common.psm1` | shared helpers | — | — | — | — | — | — | — |
| `README_WINDOWS_EXECUTION_RS.md` | human guide | — | — | — | — | — | — | — |

**STATIC_SCRIPT_CONTRACT_MAPPING=PASS** (no surfaces outside RS01–RS07; phases 00→01→02→03→04; Phase 7/admission is absent).

## B. Paths (§17)

| PATH (relative) | OWNER | EXPECTED | PATH_EXISTS_STATICALLY |
|---|---|---|---|
| `harness/harness_runner.py` | P | file | **YES** |
| `harness/enforcement_v2_runner.py` | P | file | **YES** |
| `harness/conformance_manifest.json` | P | file | **YES** |
| `harness/requirements.txt` | P | file | **YES** |
| `vectors`, `vectors/enforcement-v2` | P | dir | **YES** |
| `hacp-rs/Cargo.toml`, `Cargo.lock` | P | file | **YES** |
| `hacp-rs/src/evaluate.rs`, `src/jcs.rs` | P | file | **YES** |
| `hacp-rs/src/bin/hacp-rs-runner.rs`, `src/bin/hacp-rs-proxy.rs` | P | file | **YES** (Cargo.toml `[[bin]]` path) |
| `hacp-rs/tests/json_ingress_v1_red.rs` | P | file | **YES** |
| `hacp-rs/tests/budget_replay_verification.rs` | P (transferred) | file | **YES after 01 transfer** (package source `Rust/RS07_budget_replay_verification.rs` present + SHA-verified in 00) |
| `benchmarks/rust-http-reproduction/main.go` | S | file | **YES** |
| `benchmarks/generate-tokens/main.go` | S | file | **YES** |

Build-output paths are aligned (producer==consumer): runner=`cargo-target\release\hacp-rs-runner.exe`, proxy=`cargo-target\debug\hacp-rs-proxy.exe` (01 `Profile` == 02 `$RUNNER`/`$PROXY`).
**STATIC_PATH_AUDIT=PASS.**

## C. Commands / CLI flags (§18/§19)

| COMMAND | FLAGS VALID? | EVIDENCE |
|---|---|---|
| `cargo +stable-x86_64-pc-windows-msvc build --locked --release --bin hacp-rs-runner` | YES | Cargo `[[bin]] name=hacp-rs-runner` |
| `cargo +stable-x86_64-pc-windows-msvc build --locked --offline --bin hacp-rs-proxy` | YES | Cargo `[[bin]] name=hacp-rs-proxy` |
| `cargo +stable-x86_64-pc-windows-msvc fetch --locked --manifest-path …` | YES | standard cargo |
| `cargo +stable-x86_64-pc-windows-msvc test --locked` (RS03) | YES | auto-discovers `tests/*.rs` |
| `cargo +stable-x86_64-pc-windows-msvc test --locked --test json_ingress_v1_red` (RS01 witness) | YES | `tests/json_ingress_v1_red.rs` exists |
| `cargo +stable-x86_64-pc-windows-msvc test --locked --test budget_replay_verification` (RS07) | YES (after 01 transfer) | transferred `tests/budget_replay_verification.rs` |
| `python harness_runner.py --runner --vectors-dir --manifest --implementation-name --implementation-version --output console` | YES | argparse: `--output choices=["console","json"]` (RS01) |
| `python enforcement_v2_runner.py --runner --vectors-dir --implementation-name --output console` | YES | argparse accepts these (RS02) |
| `go run .\benchmarks\rust-http-reproduction -url -warmup-file -measured-file` | YES | main.go `flag.String("url"/"warmup-file"/"measured-file")` |
| `go run .\benchmarks\generate-tokens -count -out -method -path` | YES | main.go `flag.Int/String("count"/"out"/"method"/"path")` |

**STATIC_COMMAND_AUDIT=PASS. STATIC_CLI_FLAG_AUDIT=PASS.**

## D. APIs (§20)

| API | SIGNATURE (source) | USED BY | COMPAT |
|---|---|---|---|
| `hacp_rs::evaluate::evaluate` | `fn(&Value,&Value,&str)->EvalResult` (evaluate.rs:117) | RS07 test (`evaluate::evaluate(&inputs,&context,PUBKEY)`) | MATCH |
| `hacp_rs::jcs::canonicalize` | `fn(&Value)->Result<Vec<u8>,String>` (jcs.rs:14) | RS07 test | MATCH |
| runner entrypoint | `src/bin/hacp-rs-runner.rs` (Cargo `[[bin]]`) | RS01/RS02 `--runner $RUNNER` | MATCH |
| proxy entrypoint | `src/bin/hacp-rs-proxy.rs` (Cargo `[[bin]]`) | RS04 `Start-Process $PROXY` | MATCH |
| budget logic | evaluate.rs:445-450 `max_actions`/`current_action_count`/`BUDGET_EXHAUSTED` | RS07 asserts this | MATCH (autonomy budget only) |

**STATIC_API_AUDIT=PASS** (no compilation performed; signatures and usage checked statically).

## E. Env vars / trust (§21/§22)

| VAR | SOURCE | SET/UNSET | SCOPE | RESTORE | LEAKAGE |
|---|---|---|---|---|---|
| `CARGO_TARGET_DIR` | 01/02 `cargoEnv` (run work-root) | SET (run-scoped) | per-cmd `ExtraEnv` | run-scoped; removed with work-root | none |
| `HACP_TRUST_MODE` | 02 RS04 | SET=`test` | process | §22 contract | none |
| `HACP_TRUST_KEYS_FILE` | 02 RS04 | UNSET (`Remove-Item Env:`) | process | §22: must be UNSET | none |
| `HACP_SIDECAR_PORT` | 02 RS04 | SET=`18081` | process | — | none |
| `HTTP_PROXY/HTTPS_PROXY/ALL_PROXY` | inherit | inherit | process | — | **guarded**: `NO_PROXY=localhost,127.0.0.1` for localhost workload (§21) |
| `NO_PROXY` | 02 RS04 | SET (append localhost) | process | — | closes proxy misroute |

Trust (§22): **`HACP_TRUST_MODE=test`, `HACP_TRUST_KEYS_FILE=UNSET`** — confirmed in 02 RS04; no new trust semantics introduced.
**STATIC_ENV_AUDIT=PASS** (the proxy gap was closed by a guard).

## F. Dependency graph (§23)

```
00 (RUN_STATE=PREFLIGHT_COMPLETE)  ──00→01 gate (PRECHECK.json=PASS)──►
01 (RUN_STATE=PREPARED)            ──01→02 gate (PREPARE_RESULT=PASS)──►
02 (RUN_STATE=VERIFICATION_COMPLETE): RS01 → RS02 → RS03 → RS04 → RS05(←RS04 PASS) → RS07  ──►
03 (RUN_STATE=POSTFLIGHT_COMPLETE)  ──►
04 (RUN_STATE=RECONCILED, ←03 integrity)
```
The code enforces the graph in two ways: (1) explicit gates (`00->01`, `01->02`, `RS05 ← RS04`), and (2) **R1 lifecycle** (`-RequireState`: 01 requires `PREFLIGHT_COMPLETE`, 02 requires `PREPARED`, 03 requires `VERIFICATION_COMPLETE`, 04 requires `POSTFLIGHT_COMPLETE`). An uncompleted phase means the run is not phase-compatible and is BLOCKED.
**STATIC_DEPENDENCY_GRAPH=PASS.**

## G. Ports (§24)

RS04 proxy = **:18081**. Lifecycle: preflight availability (`PORT_18081_OCCUPIED`) → start ownership (`Test-RvPortOwned`) → **no upstream** (`UPSTREAM_REQUIRED=NO`; no upstream is started) → probe `127.0.0.1:18081` → workload `127.0.0.1:18081` → cleanup (`Stop-RvOwnedProcesses`) → postflight listener check. The Go B15 topology is not carried over (Rust = local response).
**STATIC_PORT_MODEL=PASS.**

## H. Process ownership (§25)

`Start-Process` → PID → **StartTime (mandatory, R2)** → ExecutablePath → registry (`pid-registry.jsonl` with RunId) → cleanup verifies **PID+StartTime+path** before kill → PID-reuse protection (`OWNERSHIP_MISMATCH` → DO NOT KILL) → port residue check → postflight integrity (`CLEANUP_COMPLETE=NO` on mismatch/leak). **Unknown processes are never killed** (kill only after an ownership match). Registration without StartTime = `PROCESS_OWNERSHIP_REGISTRATION=FAIL` → `ENVIRONMENT_RED`.
**STATIC_PROCESS_OWNERSHIP=PASS.**

## I. TTL / fixtures (§26/§27)

TTL contract = **7200s** (2h). Fixtures: generator `benchmarks/generate-tokens`; **warmup=100** (`tokens-warmup-100.jsonl`), **measured=5000** (`tokens-5000.jsonl`) — names and counts are aligned between 01 and 02; consumer `rust-http-reproduction` loads `loadTokens(warmup,100)` / `loadTokens(measured,5000)`. Env/token fields: `-method GET -path /api/test`. No references to historical/deleted fixtures. (Fixtures were not generated; runtime was not measured.)
**STATIC_TTL_FIXTURE_AUDIT=PASS.**

## J. Exit codes (§28)

| SURFACE | EXPECTED RC | SPECIAL | PARSER |
|---|---|---|---|
| RS01 | `CORE_HARNESS_RC=1` | **EXPECTED_NONZERO** (strict 13/38) | `13/38` + witness `1 passed` → EXPECTED_NONZERO; ≠ → RED |
| RS02 | 0 | — | `55/55` |
| RS03 | 0 | `48 passed / 0 failed / 1 ignored` | cargo summary sum |
| RS04 | 0 (workload) + smoke `403+DENY` | — | `R8_2_WORKLOAD=PASS` |
| RS07 | 0 | `3 passed / 0 failed` | cargo summary sum |

RS01 `13/38 strict + RC=1` is **not** classified as a behavioral failure (`EXPECTED_NONZERO`); reconciliation is `36/38 direct + 2 adjudicated = 38/38 accounted`, **never** `38/38 DIRECT`.
**STATIC_EXIT_CODE_AUDIT=PASS.**

## K. Output parsers (§29)

- Cargo multi-binary summary: aggregates all `test result:` lines (passed/failed/ignored), independent of order (DEFECT-5 fix); correct for RS03/RS07.
- Harness: `RESULTS: {passed}/{total} passed` → matches `13/38` / `55/55`; consistent with the line format (harness_runner:351 / enforcement_v2_runner:345).
- HTTP headers: `StatusCode==403 && X-HACP-Decision==DENY` (negative smoke).
- Workload: `R8_2_WORKLOAD=PASS` (main.go:166).
- Timing: regex with default `NOT_REPORTED`; `INFORMATIONAL_ONLY`, `THRESHOLD=NONE`.
**STATIC_OUTPUT_PARSER_AUDIT=PASS.**

## L. Evidence graph (§40)

Creation: `Invoke-RvNative` → per-step `command.txt / stdout.log / stderr.log / exit_code.txt`; `Write-SurfaceResult` → per-surface `result.json / identity.json`. Run-level: `run-context.json`, `PRECHECK/PREPARE/POSTFLIGHT/reconciliation`, `run-summary.json`.
Consumption: 01←`00-preflight/PRECHECK.json`; 02←`01-prepare/build-inventory.json` (RS01 build RC); 04←`03-postflight/POSTFLIGHT.json` + per-surface `result.json`; RS05←`RS04/RS05-timing.json` + `RS04/result.json`. All expected evidence files are created by a preceding phase; therefore no `STATIC_EXECUTABILITY_RED` (no dangling dependency in which a script expects a file that is never produced).
**STATIC_EVIDENCE_GRAPH=PASS.**

## M. 36-row reconciliation (§36/§37)

`04` mapping: `A4=RS01, A8=RS02, B6=RS03, B9=RS07, B16=RS04, B17=RS04, B18=RS03, B19=RS05` → **RUST_APPLICABLE_ROWS=8**, remaining 28 = `RUST_NOT_APPLICABLE`, **TOTAL=36** (counters sum to 36). Exact evidence map: A4→RS01, A8→RS02, B6→RS03, B9→RS07, B16→RS04, B17→RS04/static ownership, B18→RS03, B19→RS05. No synthetic totals. RS01 `EXPECTED_NONZERO` → `RUST_VERIFIED` (claim 36/38+2 pending human acceptance).
**STATIC_36_ROW_MAPPING=PASS.**

## N. Secret handling (§41)

Logs/evidence contain only SET/UNSET, key IDs, SHA-256, and public values. The environment snapshot labels proxy/trust variables as public. RS07 uses `TEST_PUBKEY_HEX` (public key). Private keys, seeds, and secret bytes are **not logged**.
**STATIC_SECRET_HANDLING=PASS.**

## O. PowerShell compatibility (§44)

Only syntax/API usage was checked (not installation). No PS7-only constructs (`??`, `?.`, ternary, `-Parallel`, `$PSNativeCommandUseErrorActionPreference`, `$PSStyle`) were found; compatible with PS5.1 + PS7. The HTTP probe is PS5.1-safe (`Get-RvHeaderValue`/`Invoke-RvLocalProbe`).
**STATIC_PS_COMPATIBILITY=PASS.**

## P. Hidden dependency / reference scan (§34/§35)

All file/directory/executable/script/fixture/log/manifest references either exist now **or** are created by a preceding phase (`CREATED_BY_PREVIOUS_PHASE`; see L). No `IMPLICITLY_ASSUMED / UNKNOWN / HISTORICAL_ONLY_BUT_REQUIRED` references. No prohibited or laboratory dependencies (`Mount`, `G:\`/`M:\`, VPS-only, obsolete `HACP_RUST` lab, historical logs). The only `HACP_RUST*` reference is the declared `EvidenceRoot` (`C:\Temp\hacp-rust-verification\evidence`), which is valid.
**STATIC_HIDDEN_REFERENCE_AUDIT=PASS.**

---

## Required final static verdicts (§49)

```text
STATIC_SCRIPT_CONTRACT_MAPPING=PASS
STATIC_PATH_AUDIT=PASS
STATIC_COMMAND_AUDIT=PASS
STATIC_CLI_FLAG_AUDIT=PASS
STATIC_API_AUDIT=PASS
STATIC_ENV_AUDIT=PASS
STATIC_DEPENDENCY_GRAPH=PASS
STATIC_PORT_MODEL=PASS
STATIC_PROCESS_OWNERSHIP=PASS
STATIC_TTL_FIXTURE_AUDIT=PASS
STATIC_EXIT_CODE_AUDIT=PASS
STATIC_OUTPUT_PARSER_AUDIT=PASS
STATIC_EVIDENCE_GRAPH=PASS
STATIC_HIDDEN_REFERENCE_AUDIT=PASS
STATIC_36_ROW_MAPPING=PASS
STATIC_SECRET_HANDLING=PASS
STATIC_PS_COMPATIBILITY=PASS

STATIC_EXECUTABILITY_AUDIT=PASS
NON_RUNTIME_ORCHESTRATION_RISKS=CLOSED
```

**Claim boundaries (§51) preserved:** Core = `38/38 ACCOUNTED = 36/38 direct + 2 adjudicated` (not `38/38 direct`); HC2 = `55/55`; Package = `48 ACTIVE + 1 HISTORICAL IGNORED`; HTTP = `AUTHORIZATION + LOCAL RESPONSE`, NO upstream; Timing = `INFORMATIONAL`, NO threshold; RS07 = `AUTONOMY BUDGET ONLY` (no token-replay/max_uses/maxUses=1 claim).

**Counters (§52):** `TESTS=0 BUILDS=0 BENCHMARKS=0 RUNTIME=0 HTTP_REQUESTS=0 FIXTURES_GENERATED=0`; `PRODUCTION_FILES_CHANGED=0 STAGING=0 COMMITS=0 PUSH=0` (no bytes changed outside `Scripts_RUST`).

This does **NOT** mean runtime-verified / tests passed / acceptance / admission / release.


---

# TARGETED FINAL CORRECTION PROOFS (D1–D6)

Bounded corrective pass over the same SHA-gated `Scripts_RUST` set (input gate 7/7). Only the six confirmed static
defects were touched; no new architectural audit, no Rust/verification/matrix rework.

## Proof table (§31)

| DEFECT_ID | AFFECTED_FILES | OLD_BEHAVIOR | FIX | STATIC_EVIDENCE | DEPENDENCIES_CHECKED | STATUS |
|---|---|---|---|---|---|---|
| **D1** MSVC toolchain pin | `01`, `02` | `cargo fetch/build/test` ran unpinned (default toolchain) for fetch, runner/proxy builds, RS01 witness | every Rust command now runs `cargo +stable-x86_64-pc-windows-msvc …` (explicit toolchain) | all 6 cargo Arg-lists begin `'+stable-x86_64-pc-windows-msvc'`; `Args=@('fetch'/'build'/'test')` unpinned count = 0 | build profiles unchanged (runner=release, proxy=debug) | **PASS** |
| **D2** hidden pip dependency | `01` | `Exe='pip'` standalone (undeclared dep vs `python`-only preflight) | `Exe='python'; Args=@('-m','pip','install','-r',…)` via the exact Python executable | `Exe='pip'` absent; `Args=@('-m','pip','install'` present; requirements path quoted correctly | preflight `python` gate covers the module | **PASS** |
| **D3** RS04 env leak | `02`, `Common` | `HACP_TRUST_MODE/HACP_TRUST_KEYS_FILE/HACP_SIDECAR_PORT/NO_PROXY` mutated, never restored | `Save-RvEnvironment` → apply temp env → `try { RS04 } finally { Restore-RvEnvironment }`; SET restored as-is, NOT_EXISTED removed | `$rs04Env = Save-RvEnvironment …`; `finally { Restore-RvEnvironment -Saved $rs04Env }`; `NO_PROXY=localhost,127.0.0.1` guard kept | NO_PROXY guard still bypasses proxy for localhost probe/workload | **PASS** |
| **D4** Git identity guard | `03` | only `staged=0` + `tracked=0`; head drift recorded but not enforced; P not checked against pin | enforce `P_HEAD == 3ea4adcb…` (frozen pin) + `S_HEAD == PRECHECK_S_HEAD` + `S_TREE == PRECHECK_S_TREE`; `gitPass` includes all | `EXPECTED_P_HEAD=3ea4adcb…`, `P_HEAD_PIN_ENFORCED`, `S_HEAD_BASELINE_ENFORCED`, `gitPass … -and $pHeadOk -and $sHeadOk -and $sTreeOk` | staged/tracked/RS07-allowed guards kept (§15) | **PASS** |
| **D5** reg-fail raw kill | `02` | `Register` failure branch did raw `Stop-Process -Id $proxy.Id` (no ownership revalidation) | capture `$proxy.StartTime` at start; on failure verify **PID+StartTime** before any `Stop-Process`; if identity unverifiable → **DO NOT KILL** (fail-safe) + record integrity-fail | raw PID-only kill removed; `$obsStart -eq $proxyStart` guard; `PROCESS_NOT_KILLED … fail-safe` path | normal cleanup still ownership-verified (`Stop-RvOwnedProcesses`) | **PASS** |
| **D6** `-Surface`/lifecycle | `02`, `README` | README claimed `-Surface` re-runs a surface of a completed run (contradicts `PREPARED`-required lifecycle) | `-Surface` = select one surface for the *current* 02 run (not a completed-run re-run); `[ValidateSet('RS01','RS02','RS03','RS04','RS05','RS07')]`; README corrected | `ValidateSet` present (RS06 excluded); README overclaim removed; lifecycle (`02` requires `PREPARED`) unchanged | `RS05 ← RS04 PASS` dependency preserved even under `-Surface` | **PASS** |

## Targeted static verdicts (§37)

```text
D1_MSVC_TOOLCHAIN_PINNING=PASS
D2_PIP_DEPENDENCY_REMOVED=PASS
D3_RS04_ENV_RESTORE=PASS
D4_GIT_IDENTITY_GUARD=PASS
D5_PID_CLEANUP_OWNERSHIP=PASS
D6_SURFACE_LIFECYCLE_ALIGNMENT=PASS
```

## Derived assertions (§38)

```text
ALL_RUST_COMMANDS_TOOLCHAIN_PINNED=YES
STANDALONE_PIP_EXECUTABLE_DEPENDENCY=NO
RS04_ENV_RESTORED=YES
P_HEAD_PIN_ENFORCED=YES
S_HEAD_BASELINE_ENFORCED=YES
RAW_PID_ONLY_KILL_PATHS=0
SURFACE_PARAMETER_VALIDATED=YES
README_RERUN_OVERCLAIM_REMOVED=YES
```

## Regression check (§39)

`PREVIOUS_FIXES_REGRESSION_CHECK=PASS` — RS07 path/SHA gates, 00→01 gate, 01→02 gate, run lifecycle (`RequireState`),
postflight integrity consumption, run-specific work root, PID StartTime ownership, PS5.1 HTTP probe, RS05←RS04,
cargo result parsing, RS01 EXPECTED_NONZERO — all still present after the six edits.

## Contract preservation (§28/§29)

No RS01–RS07 claim semantics changed; RS04 local-response/no-upstream topology, RS05 informational timing,
RS06 NOT_APPLICABLE, RS07 autonomy-budget-only all intact. 36-row mapping exact
(`A4→RS01, A8→RS02, B6→RS03, B9→RS07, B16→RS04, B17→RS04, B18→RS03, B19→RS05`; **8 applicable / 28 not / 36 total**).


## Residual static fixes (F1–F3)

| ID | FILE | FIX | PROOF | STATUS |
|---|---|---|---|---|
| **F1** | `03` | fail-closed preflight Git identity: `PRECHECK_GIT_IDENTITY_AVAILABLE` (requires S_HEAD+S_TREE); default `S_HEAD/S_TREE ok = false`; true only after proven comparison; missing/malformed => `GIT_GUARDS_PASS=NO` | `PRECHECK_GIT_IDENTITY_REQUIRED=YES`, `UNKNOWN_S_BASELINE_CAN_NEVER_PASS=YES`, missing/malformed fails closed | **PASS** |
| **F2** | `02` | `$proxy = $null; $proxyStart = $null` initialized before the RS04 `try` (StrictMode-safe); D5 ownership-safe cleanup preserved | `PROXY_INITIALIZED_BEFORE_TRY=YES`, `START_PROCESS_FAILURE_STRICTMODE_SAFE=YES`, `RAW_PID_ONLY_KILL_PATHS=0` | **PASS** |
| **F3** | audit §C | command table now shows actual pinned `cargo +stable-x86_64-pc-windows-msvc …` forms | `STATIC_COMMAND_TABLE_MATCHES_CURRENT_CODE=YES`, `STALE_UNPINNED_CARGO_ROWS=0` | **PASS** |

Chain checks: (F1) missing/malformed preflight identity -> `S_*_BASELINE_ENFORCED=NO` -> `GIT_GUARDS_PASS=NO` -> `EVIDENCE_INTEGRITY=FAIL` -> `RUST_LOCAL_REPRODUCTION_VERIFIED=NO`; (F2) no error branch dereferences an undefined variable; (F3) 01/02 cargo commands == section C == D1 proof (all pinned). Regression D1–D6 intact.

---

# CURRENT-STATE SUPERSESSION AMENDMENT

Status:

```text
CURRENT_STATE_AMENDMENT=ACTIVE
HISTORICAL_AUDIT_RETAINED=YES
HISTORICAL_TEXT_REWRITTEN=NO
```

This amendment records the current executable state after the bounded `t_rs\Scripts`
corrective pass.

The earlier sections of this document remain retained as historical static-audit
evidence. Where an earlier statement conflicts with this amendment, this amendment
controls the current orchestration model only.

## Superseded orchestration assumptions

The following historical assumptions are no longer part of the current execution
model:

```text
PowerShell 5.1 orchestration                         -> SUPERSEDED
EXPECTED_NONZERO classification                      -> SUPERSEDED
Rust Core "38/38 ACCOUNTED" synthetic wording       -> SUPERSEDED
frozen hacp-spec P_HEAD pin 3ea4adcb...              -> SUPERSEDED
detached-pin / checkout --detach model               -> SUPERSEDED
RS07 transfer into hacp-spec                         -> SUPERSEDED
hacp-rs/tests/budget_replay_verification.rs target   -> SUPERSEDED
RS07_UNTRACKED_ALLOWED postflight exception          -> SUPERSEDED
clean-worktree-as-postflight-identity model          -> SUPERSEDED
```

These historical statements are not deleted because they describe the state audited
at the time of the original static review.

## Current authoritative execution model

### PowerShell

```text
00-Preflight.ps1                    PowerShell >= 7.0
01-Prepare-Environment.ps1         PowerShell >= 7.0
02-Run-Verification.ps1            PowerShell >= 7.0
03-Postflight.ps1                  PowerShell >= 7.0
04-Reconcile-Results.ps1           PowerShell >= 7.0
WindowsVerification.Rust.Common.psm1 parser-clean
```

All six PowerShell artifacts parse with:

```text
TOTAL_PARSE_ERROR_COUNT=0
```

### Repository identity

Both public repositories are required to remain on:

```text
admit/hacp-1.1.1-pre-release
```

`00-Preflight.ps1` records, for both `hacp-spec` and `hacp-sidecar`:

```text
HEAD
TREE
STATUS_SHORT
BRANCH
EXPECTED_BRANCH
```

`03-Postflight.ps1` compares the final repository state against that exact preflight
baseline.

The current postflight invariant is therefore:

```text
POSTFLIGHT_IDENTITY =
    SAME HEAD
    AND SAME TREE
    AND SAME STATUS_SHORT
    AND SAME BRANCH
    AND branch == admit/hacp-1.1.1-pre-release
```

A worktree that was already controlled-dirty at preflight is not required to become
artificially clean. It is required not to drift during execution.

No frozen historical `P_HEAD` pin is enforced.

### RS01 — Rust Core

Current evidence model:

```text
36/38 direct + 2 adjudicated
```

Adjudicated cases:

```text
CORE-INV1-002
CORE-RUNTIME-004
```

Additional direct raw-wire witness:

```text
CORE-INV5-006 = PASS
DENY / INVALID_ACTION
```

Forbidden reporting:

```text
38/38 direct
38/38 ACCOUNTED
```

The reconciliation layer does not synthesize those formulations.

### RS07 — autonomy-budget verification

Authoritative source artifact:

```text
C:\Personal\GitHub\Dev\hacp-sidecar\test\matrix\t_rs\RS07_budget_replay_verification.rs
```

The artifact is not transferred into `hacp-spec`.

Execution uses a run-owned temporary Cargo overlay under the run work root.

The production Rust implementation remains a read-only path dependency:

```text
C:\Personal\GitHub\Dev\hacp-spec\hacp-rs
```

The RS07 execution model requires:

```text
cargo generate-lockfile --offline
cargo test --locked --offline
expected = 3 passed, 0 failed, 0 ignored
```

RS07 claim boundary remains:

```text
AUTONOMY BUDGET ONLY
NO token replay claim
NO max_uses claim
NO default maxUses=1 claim
```

### Classification model

Current shared classification vocabulary excludes `EXPECTED_NONZERO`.

Current relevant classes are:

```text
PASS
ENVIRONMENT_RED
BEHAVIORAL_RED
BLOCKED
NOT_RUN
INFORMATIONAL
```

RS01 therefore produces ordinary current-state classifications rather than treating
a non-zero harness result as a special success class.

### Reconciliation and admission boundary

Current reconciliation preserves:

```text
RUST_LOCAL_REPRODUCTION_VERIFIED
HUMAN_ACCEPTANCE_DECISION_REQUIRED=YES
```

It never automatically emits acceptance, admission, freeze, or release.

The Rust Core statement used by reconciliation is:

```text
36/38 direct + 2 adjudicated
CORE-INV1-002
CORE-RUNTIME-004
CORE-INV5-006 raw-wire PASS
never 38/38 direct
```

### Mutation boundary

Current orchestration correction preserves:

```text
hacp-spec mutation      = NO
humanist-core mutation  = NO
```

RS07 temporary execution material is created only inside the run-owned work area.

The final static mutation scan reports:

```text
HACP_SPEC_WRITE_PATTERN_HITS=0
```

## Final current-state audit

A bounded read-only audit of all six `t_rs\Scripts` PowerShell artifacts produced:

```text
TOTAL_PARSE_ERROR_COUNT=0
TOTAL_STALE_HIT_COUNT=0
HACP_SPEC_WRITE_PATTERN_HITS=0
T_RS_SCRIPTS_FINAL_AUDIT=PASS
```

The following evidence root remains valid and is explicitly not the historical
`HACP_RUST\HACP v1.1.0` laboratory workspace:

```text
C:\Temp\hacp-rust-verification\evidence
```

Final current-state conclusion:

```text
T_RS_SCRIPT_CORRECTIVE_PASS=CLOSED
STATIC_EXECUTABILITY_CURRENT_STATE=PASS
RUNTIME_VERIFICATION_NOT_ASSERTED_BY_THIS_AMENDMENT
HUMAN_ACCEPTANCE_DECISION_REQUIRED=YES
NO_AUTOMATIC_ADMISSION
NO_AUTOMATIC_RELEASE
```
