# Readme_map_file_RS — Rust artifact mapping document (NOT an automatic copying instruction)

Date: 2026-10-17
Package: repository-owned `test/matrix/t_rs`
Status: `INTERNAL / PREPARATION / NOT TEST EVIDENCE / NOT ACCEPTANCE EVIDENCE / NOT ADMISSION AUTHORIZATION`

Rule: the current verification artifact is used directly from `t_rs`. File presence does **NOT** constitute admission into S/P/C. `ARTIFACT PRESENCE != ADMISSION`. Sequence: PREPARE → LOCATE → STATIC IDENTITY CHECK → HUMAN WINDOWS EXECUTION → HUMAN REVIEW → SEPARATE ADMISSION DECISION.

---

## 1. Rust package artifacts

### 1.1 WINDOWS_EXECUTION_MANIFEST_RS.md
- PACKAGE_PATH: `test/matrix/t_rs/WINDOWS_EXECUTION_MANIFEST_RS.md`
- ARTIFACT_ROLE: authoritative Rust execution manifest (RS01–RS07)
- RS_SURFACE_ID: ALL | MATRIX_ROWS: A4,A8,B6,B9,B16,B17,B18,B19
- TARGET_REPOSITORY: Result_Audit (internal) | TARGET_RELATIVE_PATH: `test/matrix/t_rs/WINDOWS_EXECUTION_MANIFEST_RS.md`
- ARTIFACT_REQUIRED=YES | ARTIFACT_LOCATION=`t_rs` | ADMISSION_REQUIRED=NO | ADMISSION_TIMING=NOT_APPLICABLE | CURRENT_ADMISSION_STATUS=NOT_APPLICABLE
- NOTES: execution guidance; used directly from the package without modification.

### 1.2 RUST_VERIFICATION_CONTRACTS.md
- PACKAGE_PATH: `test/matrix/t_rs/RUST_VERIFICATION_CONTRACTS.md`
- ARTIFACT_ROLE: ENVIRONMENT + EXECUTION contracts RS01–RS07
- RS_SURFACE_ID: RS01–RS07 | MATRIX_ROWS: A4,A8,B18(+B6),B16,B17,B19,B9
- TARGET_REPOSITORY: Result_Audit (internal) | TARGET_RELATIVE_PATH: `test/matrix/t_rs/RUST_VERIFICATION_CONTRACTS.md`
- ARTIFACT_REQUIRED=YES | ARTIFACT_LOCATION=`t_rs` | ADMISSION_REQUIRED=NO | ADMISSION_TIMING=NOT_APPLICABLE | CURRENT_ADMISSION_STATUS=NOT_APPLICABLE
- NOTES: contracts; executable files for RS01–RS05 already exist in P/S; only RS07 is new.

### 1.3 RUST_MATRIX_RECONCILIATION.md
- PACKAGE_PATH: `test/matrix/t_rs/RUST_MATRIX_RECONCILIATION.md`
- ARTIFACT_ROLE: bounded 36-row Rust applicability reconciliation + B6/B9 coverage checks
- RS_SURFACE_ID: ALL | MATRIX_ROWS: A1–A8,B1–B28 (all 36)
- TARGET_REPOSITORY: Result_Audit (internal) | TARGET_RELATIVE_PATH: `test/matrix/t_rs/RUST_MATRIX_RECONCILIATION.md`
- ARTIFACT_REQUIRED=YES | ARTIFACT_LOCATION=`t_rs` | ADMISSION_REQUIRED=NO | ADMISSION_TIMING=NOT_APPLICABLE | CURRENT_ADMISSION_STATUS=NOT_APPLICABLE
- NOTES: does not reopen the overall matrix audit; Rust applicability only.

### 1.4 Readme_map_file_RS.md
- PACKAGE_PATH: `test/matrix/t_rs/Readme_map_file_RS.md`
- ARTIFACT_ROLE: mapping document (this file)
- RS_SURFACE_ID: ALL | MATRIX_ROWS: —
- TARGET_REPOSITORY: Result_Audit (internal) | TARGET_RELATIVE_PATH: `test/matrix/t_rs/Readme_map_file_RS.md`
- ARTIFACT_REQUIRED=YES | ARTIFACT_LOCATION=`t_rs` | ADMISSION_REQUIRED=NO | ADMISSION_TIMING=NOT_APPLICABLE | CURRENT_ADMISSION_STATUS=NOT_APPLICABLE
- NOTES: remains within the Result_Audit scope.

### 1.5 RS07_budget_replay_verification.rs  ← verification-only SOURCE
- PACKAGE_PATH: `test/matrix/t_rs/RS07_budget_replay_verification.rs`
- ARTIFACT_ROLE: verification-only Rust test — **RS07 — Rust autonomy budget verification** (B9 gap: `autonomy_budget.max_actions` exhaustion fail-closed). PHYSICAL FILE NAME != SEMANTIC CLAIM NAME: the file name contains "replay" (historical/physical naming); the authoritative claim covers autonomy-budget exhaustion ONLY; Rust token-replay/max_uses semantics are NOT claimed (RUST_TOKEN_REPLAY_SEMANTICS=NOT_IMPLEMENTED).
- RS_SURFACE_ID: RS07 | MATRIX_ROWS: B9
- TARGET_REPOSITORY: P (hacp-spec/hacp-rs) | TARGET_RELATIVE_PATH: **`hacp-rs/tests/budget_replay_verification.rs`**
- INTENDED PORTABLE WINDOWS LOCATION: `<workspace>\hacp-spec\hacp-rs\tests\budget_replay_verification.rs`
- ARTIFACT_REQUIRED=**YES** | ARTIFACT_LOCATION=**`t_rs`** | ADMISSION_REQUIRED=**YES** | ADMISSION_TIMING=AFTER_HUMAN_VERIFICATION_ONLY | CURRENT_ADMISSION_STATUS=PENDING
- NOTES: `include_str!("../../vectors/core_inv1_001_golden.json")` is resolved relative to `hacp-rs/tests/` (tracked-test convention). Uses only tracked APIs: `hacp_rs::evaluate::evaluate`, `hacp_rs::jcs::canonicalize` + dev-deps (base64/ed25519_dalek/serde_json/sha2/hex), as in the tracked `tests/authorization_v2_signed_deny_red.rs`. The target path follows the existing `tests/*.rs` convention. Verification-only: does not modify production `evaluate.rs` or introduce new production semantics.

---

## 2. Related existing Rust surfaces (outside the package; executable files already exist)

| Surface | Source | RS | Location | Artifact/Admission |
|---|---|---|---|---|
| RS01 Rust Core procedure | `docs/verification/rust-core-composition-reproduction.md` | RS01 | `hacp-sidecar/docs/verification/` (untracked in S) | TO_BE_CONFIRMED_BY_PHASE_0B / PENDING |
| RS01/RS02 runner | `hacp-rs/src/bin/hacp-rs-runner.rs` | RS01,RS02 | `hacp-spec/hacp-rs/src/bin/` (tracked) | NO / NO (tracked) |
| RS03 crate + tests | `hacp-rs/` (13 test files) | RS03 | `hacp-spec/hacp-rs/` (tracked) | NO / NO (tracked) |
| RS04 client | `benchmarks/rust-http-reproduction/main.go` | RS04 | `hacp-sidecar/benchmarks/rust-http-reproduction/` (admitted `3018b6b`) | NO / ADMITTED |
| RS04 procedure | `docs/verification/rust-http-local-reproduction.md` | RS04 | `hacp-sidecar/docs/verification/` (admitted `3018b6b`) | NO / ADMITTED |
| RS04 proxy source | `hacp-rs/src/bin/hacp-rs-proxy.rs` + `src/proxy.rs` | RS04 | `hacp-spec/hacp-rs/src/` (tracked) | NO / NO (tracked) |

Note: the RS01 procedure `rust-core-composition-reproduction.md` exists as an untracked file in the S snapshot mirror; `WINDOWS_STATE=TO_BE_CONFIRMED_BY_PHASE_0B`; admission is Phase 7. The RS04 files are already ADMITTED (signed commit `3018b6b`); their admission status is source-backed.

---

## 3. Applying the mapping (human-controlled)

1. Locate exact file (RS PHASE 0A): repository-owned `test/matrix/t_rs/RS07_budget_replay_verification.rs`. Do not copy anything into `hacp-spec`; verify the RS01 procedure separately under Phase 0B.
2. Verify canonical text SHA-256/path/status/identity (RS PHASE 0B): remove an optional UTF-8 BOM, normalize CRLF/CR to LF, hash the resulting bytes with SHA-256, then verify path existence and repository status; classification `PRESENT_MATCHING` / `MISSING` / `CONFLICT`; gate `STAGED_FILES=0`.
3. Execute `WINDOWS_EXECUTION_MANIFEST_RS.md` (RS PHASE 1–6).
4. Human review (POST-RUN).
5. Separate admission decision (PHASE 7 = PENDING HUMAN DECISION; never automatic).
6. Only after explicit GO: explicit staging + signed commit (explicit paths; no `git add .`/`git add -A`; verify signature/HEAD/tree/status; no unrelated files).

`VERIFY BEFORE ADMIT`; `ARTIFACT PRESENCE != ADMISSION`. The verification artifact remains in `t_rs`; no automatic copying into `hacp-spec` is performed.
