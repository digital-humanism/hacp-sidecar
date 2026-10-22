# README — WINDOWS EXECUTION (HACP 1.1.1 verification orchestration)

Date: 2026-10-17
Status: `INTERNAL / VERIFICATION TOOLING / NOT TEST EVIDENCE / NOT ACCEPTANCE EVIDENCE / NOT ADMISSION AUTHORIZATION`

These scripts are the **executable representation of the already-approved contracts** — not a new
execution plan. They add no verification semantics, no matrix-scope expansion, no acceptance
criteria, and **no admission automation**.

```text
POWERSHELL ORCHESTRATION
=
WINDOWS_EXECUTION_MANIFEST.md
+
CONTRACTS_S01_S12.md
+
Readme_map_file.md
```

Claim boundaries preserved: `Core != HC2`, `GO != RUST`, `GO HTTP B15 = authorization + real upstream
forwarding + real upstream response`, `RUST HTTP B16 = authorization + local response`, `TIMING =
INFORMATIONAL (no threshold)`. `VERIFY BEFORE ADMIT`; `TRANSFER != ADMISSION`.

Primary shell: **PowerShell 7+** (`pwsh`). Where a surface needs Windows PowerShell 5.1 it is noted.
Do not assume Unix utilities or WSL. Windows-native cmdlets only.

---

## 1. Prerequisites

- Windows x64; PowerShell 7+ (`pwsh`), Windows PowerShell 5.1 available for `S07`/`S10` scripts.
- Toolchain (inventory by `00`; nothing is auto-installed): git, Go ≥ 1.26.x, Python 3.13.x + pip,
  pytest, Node + npm, Rust `stable-x86_64-pc-windows-msvc` + cargo/rustup, protoc, Docker + docker compose,
  VS/VC Build Tools (MSVC ABI).
- Local repositories at:
  - `SIDECAR_ROOT = C:\Personal\GitHub\Dev\hacp-sidecar`
  - `SPEC_ROOT    = C:\Personal\GitHub\Dev\hacp-spec`
  - `CORE_ROOT    = C:\Personal\GitHub\Dev\humanist-core`
- External runtime/evidence dirs (NOT inside tracked repos), e.g. `C:\Temp\hacp-1.1.1-verification\{run,evidence}`.

## 2. Transfer prepared verification files (PHASE 0A — transfer only)

Transfer these prepared artifacts to their intended repository paths (explicit paths; **no staging/commit**):

| Surface | Target (relative to `SIDECAR_ROOT`, S-owned) |
|---|---|
| A | `internal/proxy/r4_provenance_verification_test.go` |
| B | `tools/reproduction/graceful-shutdown/main.go` |
| C | `docs/verification/rust-core-composition-reproduction.md` |
| D | `benchmarks/go-http-reproduction/main.go`, `docs/verification/go-http-local-reproduction.md` |
| S11 | `internal/wire/canonicalization_signing_test.go` |
| S12 | `internal/budget/ledger_test.go`, `internal/proxy/budget_replay_verification_test.go` |

Package sources: `<PACKAGE_ROOT>\S11_*.go`, `<PACKAGE_ROOT>\S12_*.go` (S11/S12); A/B/C/D from their
prepared locations (see `Readme_map_file.md`). Untracked presence in the worktree is **sufficient** for
human verification — admission is a later, separate decision.

## 3. Verify package/file SHA (PHASE 0B)

`00-Preflight.ps1` records `Get-FileHash -Algorithm SHA256` for each transferred file and compares the
S11/S12 hashes to the package record. Review `preflight-summary.json` + `transferred-file-check.json`.

## 4–10. Run the orchestration

All scripts accept explicit paths (no hardcoded user profile). From the transferred package:

```powershell
Set-Location <PACKAGE_ROOT>\Scripts

# 00 — READ-ONLY preflight (identity, files, toolchain, ports, env). Does NOT run tests.
.\00-Preflight.ps1 `
  -SidecarRoot "C:\Personal\GitHub\Dev\hacp-sidecar" `
  -SpecRoot    "C:\Personal\GitHub\Dev\hacp-spec" `
  -CoreRoot    "C:\Personal\GitHub\Dev\humanist-core" `
  -RunRoot     "C:\Temp\hacp-1.1.1-verification\run" `
  -EvidenceRoot "C:\Temp\hacp-1.1.1-verification\evidence"

# 01 — controlled MUTATING env prep (deps, builds, fixtures, S06 key readiness). No tests.
.\01-Prepare-Environment.ps1 `
  -SidecarRoot "C:\Personal\GitHub\Dev\hacp-sidecar" `
  -SpecRoot    "C:\Personal\GitHub\Dev\hacp-spec" `
  -CoreRoot    "C:\Personal\GitHub\Dev\humanist-core" `
  -RunRoot     "C:\Temp\hacp-1.1.1-verification\run" `
  -EvidenceRoot "C:\Temp\hacp-1.1.1-verification\evidence"

# 02 — MAIN orchestrator: 16 surfaces in approved order. Per-surface verdicts.
.\02-Run-Verification.ps1 `
  -SidecarRoot "C:\Personal\GitHub\Dev\hacp-sidecar" `
  -SpecRoot    "C:\Personal\GitHub\Dev\hacp-spec" `
  -CoreRoot    "C:\Personal\GitHub\Dev\humanist-core" `
  -RunRoot     "C:\Temp\hacp-1.1.1-verification\run" `
  -EvidenceRoot "C:\Temp\hacp-1.1.1-verification\evidence"

# 03 — cleanup (owned PIDs only) + final identity guards + evidence inventory.
.\03-Postflight.ps1 `
  -SidecarRoot "C:\Personal\GitHub\Dev\hacp-sidecar" `
  -SpecRoot    "C:\Personal\GitHub\Dev\hacp-spec" `
  -CoreRoot    "C:\Personal\GitHub\Dev\humanist-core" `
  -RunRoot     "C:\Temp\hacp-1.1.1-verification\run" `
  -EvidenceRoot "C:\Temp\hacp-1.1.1-verification\evidence"

# 04 — reconcile actual results to the authoritative 36-row matrix (NO tests, NO acceptance).
.\04-Reconcile-Results.ps1 `
  -RunRoot     "C:\Temp\hacp-1.1.1-verification\run" `
  -EvidenceRoot "C:\Temp\hacp-1.1.1-verification\evidence"
```

## 5. Interpret 00 (preflight)

- `preflight-summary.json/.txt` → `PREFLIGHT_READY=YES/NO`. If `NO`, resolve identity/toolchain/file
  issues before running `01`. `00` never auto-runs the next script.
- Check `STAGED_FILES_TOTAL=0`, `transferred-file-check.json` (no `IDENTITY_RED`), toolchain present,
  ports `FREE` (unexpected occupied ports are flagged, never killed).

## 7. Interpret 01 (environment prep)

- `environment-preparation.json` → `ENVIRONMENT_PREPARATION=READY` (or `READY_WITH_REDS`).
- `build-inventory.json` / `fixture-inventory.json` record path/size/SHA-256/build-command/RC/HEAD.
- Missing required toolchain version ⇒ `ENVIRONMENT_RED` for that surface (no silent substitution).
- `01` leaves **no services running**; services start in `02` for unambiguous PID ownership.

## 8. Interpret 02 (execution)

- Order: `A → S11 → S12 → B → S01 → S02 → S03 → S05 → C → S04 → S09 → S06 → S07 → D → S08 → S10`.
- Per-surface evidence dir under `EVIDENCE_ROOT\<SURFACE>\` (stdout/stderr/command/exit-code/result.json).
- `verification-results.json/.csv` → one verdict per surface. **No synthetic combined score.**
- Default: **STOP on material RED** (IDENTITY/BEHAVIORAL/ENVIRONMENT). Independent surfaces continue
  only with `-ContinueIndependentSurfaces`. No automatic retry of `BEHAVIORAL_RED`.
- Resume one surface after an environment fix: `.\02-Run-Verification.ps1 -Surface S06 ...`
- Boundaries kept: Core≠HC2 separate verdicts; C/S04 Rust `36/38 direct + 2 adjudicated` (NOT 38/38)
  and `55/55`; S05 sidecar black-box `55/55`; S06 `5 passed, 0 skipped` (SKIP≠PASS, EXTERNAL mode);
  S10 timing `INFORMATIONAL` (no latency PASS/FAIL; `GATE D` lines = `HISTORICAL_MARKER_ONLY`).

## 10. Run 03 (postflight)

Stops only owned PIDs from the registry (no wildcard `Stop-Process`/`taskkill /IM`, no `docker prune`),
checks owned listeners removed, re-checks S/P/C HEAD/tree/status vs preflight (`IDENTITY_RED` on drift or
new unexpected tracked modifications), builds `evidence-inventory.csv`. Does not delete evidence.

## 11. Human review

`04-Reconcile-Results.ps1` maps actual surface results to the authoritative 36-row matrix and prints
counters that **sum to 36**. It reconciles evidence only. **No automatic acceptance.**

## 12. Admission is explicitly NOT automatic (PHASE 7)

`04` does **not** emit `SIDECAR_A2=PASS`, `RELEASE_READY=YES`, `ACCEPTANCE=PASS`, or `GO`. Phase 7
(admission) is outside these scripts and requires a **separate explicit human GO** after review
(explicit paths only, no `git add .`/`git add -A`, signed commits, verify signature/HEAD/tree/status).

---

## Safety boundary (all scripts)

```text
VERIFY BEFORE ADMIT      TRANSFER != ADMISSION
NO git add . / git add -A / commit / push / merge / tag / release
NO staging, NO admission automation, NO acceptance automation
NO automatic RED remediation
NO external network targets (localhost only within 02 runtime contracts)
NO docker system/container prune
Only owned PIDs are ever stopped
Sensitive values (private key/seed) never logged
```

## Orchestration corrections (static — D-1 / D-2 / D-3 / O-1)

- **D-1 (build output identity)**: the TypeScript CLI output identity is `SpecRoot\hacp-ts\dist\cli.js`
  (npm `run build` emits into `hacp-ts\dist`), consumed by S02/S06 as `Join-Path $SpecRoot 'hacp-ts\dist\cli.js'`.
  `01` records/verifies this exact producer path in `build-inventory.json` (no invented `BUILD_ROOT\dist` claim).
- **D-2 (S10 fixture)**: `S10_FIXTURE_MODEL = ORCHESTRATION_OWNED`. `01-Prepare-Environment.ps1`
  generates a fresh repo-local `benchmarks\tokens.jsonl` using the contract-defined command
  (`count=1000`, `GET`, `/api/test`), records path/size/SHA-256/generated-at/TTL source in
  `fixture-inventory.json`, and `02` runs all three S10 benchmark entrypoints from the sidecar root,
  where they consume that exact path. The fixture is gitignored/untracked and does not require cleanup.
- **D-3 (env isolation)**: each surface runs under a scoped environment (`Save-WvEnvironment` -> surface sets
  exact `HACP_*` env -> `Restore-WvEnvironment` in `finally`). Previously SET vs UNSET is preserved
  (UNSET restored as UNSET, not `''`); restoration is guaranteed on exceptions; values are never logged.
- **O-1 (C expected-nonzero)**: the C surface's strict witness (`13/38`) is the approved `36/38 direct + 2
  adjudicated` state (NOT `38/38`). `Resolve-WvStepClassification -ExpectedNonZero` (scoped to C only) classifies
  a sanctioned expected-nonzero RC with the witness present as the approved state; unexpected nonzero
  (witness absent) and every other surface's `RC != 0` remain RED.

## Script readiness / cross-reference

| SCRIPT | ROLE | MATERIALIZES |
|---|---|---|
| `00-Preflight.ps1` | read-only pre-execution verification | PHASE 0B + global gates |
| `01-Prepare-Environment.ps1` | controlled env prep | PHASE 1–6 prerequisites |
| `02-Run-Verification.ps1` | main orchestrator (16 surfaces) | PHASE 1–6 execution |
| `03-Postflight.ps1` | cleanup + identity guards + evidence inventory | POST-RUN guards |
| `04-Reconcile-Results.ps1` | 36-row reconciliation (no acceptance) | matrix reconciliation |
| `README_WINDOWS_EXECUTION.md` | human execution guide | this document |

`WindowsVerification.Common.psm1` is the shared helper module (documented supporting artifact).
