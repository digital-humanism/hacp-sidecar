# Readme_map_file — artifact mapping document (NOT an instruction for automatic copying)

Date: 2026-10-17
Package: `/opt/data/workspace/HACP_R1.1.1/Result_Audit/Artefact_Verification_package`
Status: `INTERNAL / PREPARATION / NOT TEST EVIDENCE / NOT ACCEPTANCE EVIDENCE / NOT ADMISSION AUTHORIZATION`

Rule: **CURRENT PACKAGE LOCATION != HISTORICAL / INTENDED LOCAL REPOSITORY LOCATION**. File presence in the package does **NOT** constitute admission into S/P/C. Transfer and admission are separate human-controlled actions (explicit paths, signed commits; no staging or commit was performed at this stage). Admission occurs ONLY after human verification (PHASE 7 = PENDING HUMAN DECISION).

---

## 1. Package artifacts

### 1.1 Readme_map_file.md
- PACKAGE_PATH: `test/matrix/t_go/Readme_map_file.md`
- ARTIFACT_ROLE: mapping document (this file) | SURFACE_ID: ALL | MATRIX_ROWS: —
- TARGET_REPOSITORY: Result_Audit (internal) | TARGET_RELATIVE_PATH: `Result_Audit/Artefact_Verification_package/Readme_map_file.md`
- TRANSFER_REQUIRED=NO | ADMISSION_REQUIRED=NO
- NOTES: remains within the Result_Audit scope; not transferred into S/P/C.

### 1.2 WINDOWS_EXECUTION_MANIFEST.md
- PACKAGE_PATH: `test/matrix/t_go/WINDOWS_EXECUTION_MANIFEST.md`
- ARTIFACT_ROLE: unified Windows execution manifest | SURFACE_ID: A,B,C,D,S01–S12 | MATRIX_ROWS: all 36
- TARGET_REPOSITORY: Result_Audit (internal) | TARGET_RELATIVE_PATH: `Result_Audit/Artefact_Verification_package/WINDOWS_EXECUTION_MANIFEST.md`
- TRANSFER_REQUIRED=NO | ADMISSION_REQUIRED=NO
- NOTES: execution guidance; used directly from the package without modification.

### 1.3 CONTRACTS_S01_S12.md
- PACKAGE_PATH: `test/matrix/t_go/CONTRACTS_S01_S12.md`
- ARTIFACT_ROLE: ENVIRONMENT + EXECUTION contracts S01–S12 | SURFACE_ID: S01–S12 | MATRIX_ROWS: A1–A3,A5–A8,B4,B6,B9,B13,B14,B16,B18,B19
- TARGET_REPOSITORY: Result_Audit (internal) | TARGET_RELATIVE_PATH: `Result_Audit/Artefact_Verification_package/CONTRACTS_S01_S12.md`
- TRANSFER_REQUIRED=NO | ADMISSION_REQUIRED=NO
- NOTES: contracts for surfaces whose executable files already exist in P/C/S (S01–S10), plus the new S11/S12 surfaces.

### 1.4 S11_signature_canonicalization_test.go.overlay
- PACKAGE_PATH: `test/matrix/t_go/S11_signature_canonicalization_test.go.overlay`
- ARTIFACT_ROLE: verification-only test file (B6 gap: JCS/canonicalization + signing failure classes)
- SURFACE_ID: S11 | MATRIX_ROWS: B6
- OVERLAY VIRTUAL PACKAGE TARGET: `internal/wire/s11_matrix_verification_test.go` (generated mapping only; repository-owned source remains under `test/matrix/t_go/`).
- TARGET_REPOSITORY: S (hacp-sidecar) | REPOSITORY_OWNED_SOURCE: **`test/matrix/t_go/S11_signature_canonicalization_test.go.overlay`**
- TRANSFER_REQUIRED=**YES** | ADMISSION_REQUIRED=**YES** | ADMISSION_TIMING=AFTER_HUMAN_VERIFICATION_ONLY | CURRENT_ADMISSION_STATUS=PENDING
- NOTES: `package wire` declaration; uses only tracked APIs (`CanonicalizeJSON`, `VerifyCanonicalization`, `RemoveSignatureField`, `VerifyEd25519`) and the standard library. The target path follows the existing S convention (tests reside alongside their package, as in `internal/wire/key_resolver_test.go`); no new architectural convention is required. Verification-only: production code is not modified.

### 1.5 S12_budget_ledger_test.go.overlay
- PACKAGE_PATH: `test/matrix/t_go/S12_budget_ledger_test.go.overlay`
- ARTIFACT_ROLE: verification-only unit tests (B9 gap: max_uses exhaustion, reuse/replay)
- SURFACE_ID: S12 | MATRIX_ROWS: B9
- OVERLAY VIRTUAL PACKAGE TARGET: `internal/budget/s12_matrix_budget_ledger_test.go` (generated mapping only; repository-owned source remains under `test/matrix/t_go/`).
- TARGET_REPOSITORY: S | REPOSITORY_OWNED_SOURCE: **`test/matrix/t_go/S12_budget_ledger_test.go.overlay`**
- TRANSFER_REQUIRED=**YES** | ADMISSION_REQUIRED=**YES** | ADMISSION_TIMING=AFTER_HUMAN_VERIFICATION_ONLY | CURRENT_ADMISSION_STATUS=PENDING
- NOTES: `package budget`; standard library only, plus the existing `Ledger` API (`ConsumeToken`, `TokenCount`). Convention: `*_test.go` resides alongside the package (for example, `internal/trust/store_test.go`).

### 1.6 S12_budget_replay_verification_test.go.overlay
- PACKAGE_PATH: `test/matrix/t_go/S12_budget_replay_verification_test.go.overlay`
- ARTIFACT_ROLE: verification-only integration tests (B9 gap: default maxUses=1, exhaustion, replay → 403/DENY/BUDGET_EXHAUSTED without forwarding)
- SURFACE_ID: S12 | MATRIX_ROWS: B9
- OVERLAY VIRTUAL PACKAGE TARGET: `internal/proxy/s12_matrix_budget_replay_verification_test.go` (generated mapping only; repository-owned source remains under `test/matrix/t_go/`).
- TARGET_REPOSITORY: S | REPOSITORY_OWNED_SOURCE: **`test/matrix/t_go/S12_budget_replay_verification_test.go.overlay`**
- TRANSFER_REQUIRED=**YES** | ADMISSION_REQUIRED=**YES** | ADMISSION_TIMING=AFTER_HUMAN_VERIFICATION_ONLY | CURRENT_ADMISSION_STATUS=PENDING
- NOTES: `package proxy`; uses TRACKED helpers `signedBindingEnvelope` / `signBindingObject` from `internal/proxy/request_binding_test.go` and tracked APIs `SynthesizeProposedAction`, `NewHandler`, `evaluate.NewPipeline`, `budget.NewLedger`, `provenance.NewRingBuffer/NewNoopWriter`, `wire.*`. The pattern follows the precedent of `r4_provenance_verification_test.go` (in-process `httptest`). Does NOT conflate this with control-plane event replay (B9 boundary).

---

## 2. Related previously prepared artifacts (outside the package; NOT created by this task)

| Artifact | Role | Surfaces | Location | Transfer/Admission |
|---|---|---|---|---|
| `ABC_WINDOWS_VERIFICATION_HANDOFF_RU_2026-10-17.md` | handoff + contracts | A,B,C | `Result_Audit/` | NO/NO |
| `GO_HTTP_WINDOWS_VERIFICATION_HANDOFF_RU_2026-10-17.md` | handoff + contracts | D | `Result_Audit/` | NO/NO |
| `WINDOWS_REPRODUCTION_ENVIRONMENT_MATRIX_RU_2026-10-17.md` | env matrix | A,B,C,D | `Result_Audit/` | NO/NO |
| `GO_HTTP_IMPLEMENTATION_SPEC_RU_2026-10-17.md` | spec | D | `Result_Audit/` | NO/NO |
| `SIDECAR_MATRIX_FINAL_PREPARATION_AUDIT_RU_2026-10-17.md` | authoritative control basis | ALL | `Result_Audit/` | NO/NO |
| `benchmarks/go-http-reproduction/main.go` | D client (previously materialized in S mirror) | D | `hacp-sidecar/benchmarks/go-http-reproduction/` (untracked in S) | CURRENT_ADMISSION_STATUS=PENDING (admission — PHASE 7) |
| `docs/verification/go-http-local-reproduction.md` | D procedure | D | `hacp-sidecar/docs/verification/` (untracked in S) | CURRENT_ADMISSION_STATUS=PENDING (admission — PHASE 7) |
| `internal/proxy/r4_provenance_verification_test.go` | A (H01/H02) | A | `hacp-sidecar/internal/proxy/` (untracked in S) | CURRENT_ADMISSION_STATUS=PENDING (admission — PHASE 7) |
| `tools/reproduction/graceful-shutdown/main.go` | B (H03) | B | `hacp-sidecar/tools/reproduction/graceful-shutdown/` (untracked in S) | CURRENT_ADMISSION_STATUS=PENDING (admission — PHASE 7) |
| `docs/verification/rust-core-composition-reproduction.md` | C | C | `hacp-sidecar/docs/verification/` (untracked in S) | CURRENT_ADMISSION_STATUS=PENDING (admission — PHASE 7) |

Note: the listed S files exist as untracked files in the S snapshot mirror (MIRROR_TRACKED_STATE=UNTRACKED); WINDOWS_STATE=TO_BE_CONFIRMED_BY_PHASE_0B. Their admission requires a separate human decision (PHASE 7; historical precedent: signed commit `3018b6b`, which is NOT a current authorization).

---

## 3. Applying the mapping (human-controlled)

1. Transfer exact files (PHASE 0A — transfer only: `S11_*`/`S12_*` into TARGET_RELATIVE_PATH; files in §2 depend on Phase 0B verification; explicit paths; no staging or commit).
2. Verify SHA/path/status (PHASE 0B — `Test-Path`, `Get-Item`, `Get-FileHash -Algorithm SHA256`, `git status --short -- <path>`, `git check-ignore -v -- <path>`; classification `ALREADY_PRESENT_MATCHING` / `TRANSFER_REQUIRED` / `CONFLICT`; gate `STAGED_FILES=0`).
3. Execute Windows verification manifest (PHASE 1–6).
4. Human review (POST-RUN).
5. Separate admission decision (PHASE 7 = `ADMISSION = PENDING HUMAN DECISION`; never automatic).
6. Only after explicit GO: explicit staging + signed commit (explicit paths; signed commits; no `git add .`/`git add -A`; verify signature/HEAD/tree/status; no unrelated files).

Fundamental sequence: PREPARE → TRANSFER → STATIC IDENTITY CHECK → HUMAN WINDOWS EXECUTION → HUMAN RESULT REVIEW → SEPARATE ADMISSION DECISION → STAGE / SIGNED COMMIT. `VERIFY BEFORE ADMIT`; `TRANSFER != ADMISSION`; an untracked file in the worktree is sufficient for human verification. No automatic copying without a human decision; this document records intent only.
