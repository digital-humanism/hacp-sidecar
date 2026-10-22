# README — WINDOWS EXECUTION (RUST reproduction orchestration)

Date: 2026-10-17
Status: `INTERNAL / VERIFICATION TOOLING / NOT TEST EVIDENCE / NOT ACCEPTANCE EVIDENCE / NOT ADMISSION AUTHORIZATION`

Executable representation of the approved **Rust** contracts — not a new execution plan. No new
verification semantics, no matrix-scope expansion, no acceptance/admission automation.

```text
POWERSHELL ORCHESTRATION (RUST)
=
WINDOWS_EXECUTION_MANIFEST_RS.md
+
RUST_VERIFICATION_CONTRACTS.md
+
RUST_MATRIX_RECONCILIATION.md
```

Claim boundaries (preserved): `RUST CORE = 38/38 ACCOUNTED = 36/38 DIRECT + 2 ADJUDICATED` (NOT 38/38
direct); `RUST HC2 = 55/55`; `RUST PACKAGE = 48 ACTIVE + 1 HISTORICAL IGNORED`; `RUST HTTP = AUTHORIZATION
+ LOCAL RESPONSE (no upstream forwarding)`; `RUST TIMING = INFORMATIONAL (no threshold, no Go-vs-Rust
ranking)`; `RS07 = autonomy budget only (no token replay / max_uses)`.

Primary shell: **PowerShell 7+** (`pwsh`); scripts avoid PS7-only constructs and should run on Windows
PowerShell 5.1. Windows-native cmdlets only; no Unix utilities/WSL.

---

## 1. Prerequisites

- Windows x64; PowerShell 7+ (or 5.1).
- Toolchain (inventory by `00`; nothing auto-installed): git, `rustc`/`cargo`/`rustup`
  (`stable-x86_64-pc-windows-msvc`), protoc, python, go.
- Repositories: `SPEC_REPO = C:\Personal\GitHub\Dev\hacp-spec`, `SIDECAR_REPO = C:\Personal\GitHub\Dev\hacp-sidecar`.
- External run/evidence (NOT inside git worktrees): `EvidenceRoot` (default `C:\Temp\hacp-rust-verification\evidence`),
  `RunRoot` (default `C:\Temp\hacp-rust-verification\run`).

## 2. Locate RS07 (PHASE 0A — artifact presence only, no copy/staging/commit)

`RS07_budget_replay_verification.rs` is used directly from `C:\Personal\GitHub\Dev\hacp-sidecar\test\matrix\t_rs\RS07_budget_replay_verification.rs`.
No copy into `hacp-spec` is performed. Phase 0B verifies artifact path / SHA-256 / status / identity.
SHA-256 (`c25fb8fc…`) and records `git status`/`git check-ignore` (expected: **untracked, not staged**).
Untracked RS07 is **sufficient** for verification — admission is a later, separate decision.

## 3. Run order

```powershell
Set-Location C:\Personal\GitHub\Dev\hacp-sidecar\test\matrix\t_rs\Scripts

.\00-Preflight.ps1 -SpecRepo "C:\Personal\GitHub\Dev\hacp-spec" -SidecarRepo "C:\Personal\GitHub\Dev\hacp-sidecar" `
  -EvidenceRoot "C:\Temp\hacp-rust-verification\evidence" -RunRoot "C:\Temp\hacp-rust-verification\run"

.\01-Prepare-Environment.ps1 -SpecRepo "C:\Personal\GitHub\Dev\hacp-spec" -SidecarRepo "C:\Personal\GitHub\Dev\hacp-sidecar" `
  -EvidenceRoot "C:\Temp\hacp-rust-verification\evidence" -RunRoot "C:\Temp\hacp-rust-verification\run"

.\02-Run-Verification.ps1 -SpecRepo "C:\Personal\GitHub\Dev\hacp-spec" -SidecarRepo "C:\Personal\GitHub\Dev\hacp-sidecar" `
  -EvidenceRoot "C:\Temp\hacp-rust-verification\evidence" -RunRoot "C:\Temp\hacp-rust-verification\run"

.\03-Postflight.ps1 -SpecRepo "C:\Personal\GitHub\Dev\hacp-spec" -SidecarRepo "C:\Personal\GitHub\Dev\hacp-sidecar" `
  -EvidenceRoot "C:\Temp\hacp-rust-verification\evidence" -RunRoot "C:\Temp\hacp-rust-verification\run"

.\04-Reconcile-Results.ps1 -EvidenceRoot "C:\Temp\hacp-rust-verification\evidence" -RunRoot "C:\Temp\hacp-rust-verification\run"
```

All scripts accept explicit paths. `00` creates the run directory `<timestamp>_<run-id>\` (with
`00-preflight\ 01-prepare\ 02-verification\ 03-postflight\ 04-reconciliation\ metadata\`); `01–04`
resolve it automatically (or pass `-RunDir`). `02 -Surface RS0x` **selects one surface for the current 02
execution** (validated to `RS01/RS02/RS03/RS04/RS05/RS07`; `RS06` is never executed). It is **NOT** a re-run of an
already-completed verification run: the run lifecycle still applies (`02` requires a `PREPARED` run; a `VERIFICATION_COMPLETE`
run is not re-opened by `-Surface`), and dependencies are preserved (e.g. `RS05` still requires a PASSing `RS04` in the same run).

## 4–5. STOP rules and hard phase gates

Hard gates are enforced in code (nonzero exit + BLOCKED result, no downstream actions):

```text
00 → 01 GATE (hard): 01 loads 00-preflight/PRECHECK.json and requires PREFLIGHT=PASS.
      missing / malformed / != PASS  →  ENVIRONMENT_READY=BLOCKED, NO prep actions, exit nonzero.
01 → 02 GATE (hard): 02 loads 01-prepare/PREPARE_RESULT.json and requires ENVIRONMENT_READY=PASS.
      != PASS  →  NO RS surface execution, exit nonzero.
00 Preflight BLOCKED  → do NOT run 01   (mandatory toolchains git/rustc/cargo/rustup/protoc/python/go,
                                          port 18081 FREE, staged=0, RS07 package source PRESENT_MATCHING)
01 branch gate (hard): active repositories used by t_rs must remain on `admit/hacp-1.1.1-pre-release`; no checkout/detach is performed by verification.
01 Prepare BLOCKED    → do NOT run 02
BEHAVIORAL_RED        → do NOT auto-repair; record only; never mutate production
03 guard failure      → 04 marks EVIDENCE_INTEGRITY=FAIL and RUST_LOCAL_REPRODUCTION_VERIFIED=NO
```

Run identity + run-specific work root (R1): each run owns `<RunRoot>\<RunId>\` (`bin\`, `cargo-target\`,
`fixtures\`, `tmp\`, `pid-registry.jsonl`) tied to the exact evidence run.
- `00` WITHOUT `-RunDir` **ALWAYS creates a NEW run identity** (new RunId/RunDir/WorkRoot) — never reuses or selects the latest existing run.
- `00` WITH explicit `-RunDir` uses exactly that path (no auto-substitution / no latest-run replacement); an invalid/conflicting `RunDir` => `PREFLIGHT=BLOCKED`.
- `01–04` WITHOUT `-RunDir` auto-resolve **ONLY when exactly ONE phase-compatible active run** exists (valid `metadata\run-context.json` **and** a `RUN_STATE` appropriate for the requested phase). A completed (`RECONCILED`) run is never phase-compatible, so it never blocks a new run. `0` eligible or `>1` eligible => `BLOCKED`; the human must pass `-RunDir` explicitly. No latest/first/newest guessing.
- `run-context.json` records RunId, CreatedAtUtc, RunDir, WorkRoot, EvidenceRoot, CreatedByScript, `RUN_SELECTION_MODE` (NEW_EXPLICIT_CREATION / EXPLICIT_RUNDIR / UNIQUE_ACTIVE_AUTO_RESOLVE) and an explicit `RUN_STATE` lifecycle (`CREATED → PREFLIGHT_COMPLETE → PREPARED → VERIFICATION_COMPLETE → POSTFLIGHT_COMPLETE → RECONCILED`). Each phase advances the state; execution is never silently rebound to another run. An explicit `-RunDir` is authoritative but must be phase-compatible (`RUN_STATE_INCOMPATIBLE` otherwise).

PID ownership (R2): `03` stops only PIDs registered in the current run's `pid-registry.jsonl`, and verifies ownership before each kill (PID + actual process StartTime, plus ExecutablePath when available). If a PID was reused (StartTime/path mismatch) it is **NOT** killed; this is recorded as `OWNERSHIP_MISMATCH` and forces `CLEANUP_COMPLETE=NO` (=> `EVIDENCE_INTEGRITY=FAIL`). A registered PID that already exited is `PROCESS_ALREADY_EXITED` (not a failure). Only `Stop-Process -Id <verified-owned-pid>` is used — never broad kills. Registration itself **requires a reliable `ProcessStartTimeUtc`** (never `$null`): if the process StartTime cannot be obtained, registration fails (`PROCESS_OWNERSHIP_REGISTRATION=FAIL` => `ENVIRONMENT_RED`) rather than recording an owned process with an empty StartTime.

Dependency chain in `02`: **RS05 requires RS04 PASS** (result.json exists + classification=PASS). If RS04
is missing/not PASS, RS05 is `BLOCKED` and reconciliation marks `B19=RUST_EVIDENCE_INCOMPLETE` (not
`RUST_VERIFIED`). `RUST_LOCAL_REPRODUCTION_VERIFIED=YES` additionally requires `03` postflight integrity
(`CLEANUP_COMPLETE`, `GIT_GUARDS_PASS`, `EVIDENCE_INVENTORY_COMPLETE` all `YES`); otherwise
`EVIDENCE_INTEGRITY=FAIL`.

PowerShell 5.1 compatibility: the HTTP negative-smoke probe (`Invoke-RvLocalProbe`) does NOT use the
PS7-only `-SkipHttpErrorCheck`; it captures `403 + X-HACP-Decision: DENY` from the error response in both
PS5.1 and PS7.

Execution failure policy (§39): `ENVIRONMENT_RED` stops dependent surfaces (continue independent ones
only with `-ContinueIndependentSurfaces`); `BEHAVIORAL_RED` is recorded, never repaired; expected
non-zero (RS01 strict `CORE_HARNESS_RC=1`) is `EXPECTED_NONZERO`, not a failure.

## 6. VERIFY BEFORE ADMIT (§56)

```text
ARTIFACT PRESENCE != ADMISSION
UNTRACKED RS07 IS ALLOWED FOR VERIFICATION
NO STAGING
NO COMMIT
BEFORE HUMAN RESULT REVIEW
```

---

## Surface run order (§28) and expected results

| Surface | Matrix rows | Command core | Expected |
|---|---|---|---|
| RS01 | A4 | `hacp-rs-runner` + Core renderer + raw-wire witness | **36/38 direct PASS + 2 adjudicated** (`CORE-INV1-002`, `CORE-RUNTIME-004`); `CORE-INV5-006` raw-wire PASS; final accounting `36 direct + 2 adjudicated` |
| RS02 | A8 | `enforcement_v2_runner.py` | `55/55` |
| RS03 | B18 (+B6) | `cargo +stable-x86_64-pc-windows-msvc test --locked` | `48 passed`, `1 ignored`, `0 failed` |
| RS04 | B16,B17 | owned `hacp-rs-proxy` :18081 + negative smoke + rust-http workload | smoke `403 + X-HACP-Decision: DENY`; warmup `100/100`; measured `5000/5000`; `R8_2_WORKLOAD=PASS` |
| RS05 | B19 | timing from RS04 (**requires RS04 PASS**; else BLOCKED) | p50/p95/p99/max/RPS or `NOT_REPORTED`; `INFORMATIONAL_ONLY=YES`, `THRESHOLD=NONE` |
| RS06 | — | **NOT_APPLICABLE** (B6 EXACT coverage) | no run |
| RS07 | B9 | `cargo … test --locked --test budget_replay_verification` | `3 passed`, `0 failed` (autonomy budget only) |

RS04 topology = `client → Rust proxy :18081 → local response`; `UPSTREAM_REQUIRED=NO` (no upstream is
started; forwarding is never checked). Service processes are **owned** (recorded PID) and stopped only
by PID in `03`.

## Interpret outputs

- `00-preflight\PRECHECK.json` → `PREFLIGHT=PASS|BLOCKED`.
- `01-prepare\PREPARE_RESULT.json` → `ENVIRONMENT_READY=PASS|BLOCKED`.
- `02-verification\RS0x\result.json` (§38 schema) → per-surface classification
  (`PASS` / `ENVIRONMENT_RED` / `BEHAVIORAL_RED` / `EXPECTED_NONZERO` / `BLOCKED`).
- `03-postflight\POSTFLIGHT.json` → `CLEANUP_COMPLETE`, `GIT_GUARDS_PASS`, `EVIDENCE_INVENTORY_COMPLETE`.
- `04-reconciliation\RUST_MATRIX_RESULT.{csv,json}` + `RUST_VERIFICATION_SUMMARY.md` → 36-row status +
  counters that sum to 36.

## Admission is NOT automatic (§52)

The maximum automatic marker is:

```text
RUST_LOCAL_REPRODUCTION_VERIFIED=YES
HUMAN_ACCEPTANCE_DECISION_REQUIRED=YES
```

`04` **never** writes `SIDECAR_ACCEPTED`, `RELEASED`, `ACCEPTED`, or `FROZEN`. Final acceptance is a
human decision after evidence review. Phase 7 (admission) is outside these scripts and requires a
separate explicit human GO (explicit paths, no `git add .`, signed commit, verify signature/HEAD/tree).

## Secret handling (§58)

Logs never contain private keys, secret env values, or tokens — only `SET`/`UNSET`, key IDs, SHA-256,
and redacted metadata.
