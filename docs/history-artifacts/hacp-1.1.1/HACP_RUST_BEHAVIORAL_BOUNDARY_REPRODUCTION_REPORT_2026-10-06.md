# HACP Rust Behavioral Boundary Reproduction Report

**Date:** 2026-10-06  
**Project:** HUMANIST / HACP  
**Release line under verification:** HACP 1.1.1  
**Primary repository:** `hacp-spec`  
**Integration / verification repository:** `hacp-sidecar`  
**Report class:** Engineering verification record  
**Status:** COMPLETE — BEHAVIORAL SPLIT REPRODUCED AND PROVEN  
**Production mutation:** NONE  
**Publication authorization:** NOT ESTABLISHED BY THIS REPORT  
**Release authorization:** NOT ESTABLISHED BY THIS REPORT  

---

## 1. Executive Summary

This report records the controlled reproduction and comparison of two Rust behavioral surfaces associated with the HACP 1.1.1 verification line:

1. the historically verified Rust GitCheck surface, corresponding to `hacp-rs 0.1.0`, and
2. the currently adopted Rust implementation at `hacp-spec` commit `cf309fc5308863f41a9af6999f0a533449431a78`, corresponding to `hacp-rs 0.1.1`.

The purpose of this work was not to redesign the Rust implementation, modify production source, or reinterpret historical evidence. The purpose was to reproduce the documented historical GitCheck result exactly enough to establish a trustworthy baseline, then reproduce the current `0.1.1` surface under the same controlled conditions and compare observable behavior.

The historical GitCheck reproduction completed successfully:

- fresh Rust build: PASS;
- exact documented historical runner identity: reproduced;
- Core conformance: 38/38 PASS;
- HC2 conformance: 55/55 PASS;
- informational Rust benchmark: PASS;
- public source mutation: NONE.

The current `hacp-rs 0.1.1` reproduction also completed successfully at the build and execution level:

- fresh build: PASS;
- current runner identity reproduced;
- HC2 conformance: 55/55 PASS;
- corrected raw-wire `CORE-INV5-006`: PASS;
- historical Core conformance: FAIL;
- Authorization-v2 successor witness tests: PASS;
- infrastructure/evaluator errors during the decisive conformance run: 0;
- public source mutation: NONE.

The decisive result is therefore:

```text
hacp-rs 0.1.0
  Historical Core = 38/38 PASS
  HC2             = 55/55 PASS

hacp-rs 0.1.1 / cf309fc
  Historical Core = FAIL
  HC2             = 55/55 PASS
  Authorization-v2 successor witnesses = PASS
```

This establishes a real observable behavioral boundary between the two Rust versions.

The current `0.1.1` implementation cannot be treated as behaviorally identical to the historically verified `0.1.0` implementation. The evidence shows that `0.1.1` simultaneously:

- preserves the HC2-55 request-binding surface;
- preserves corrected duplicate-member raw-wire rejection behavior;
- changes historical Core decision/reason behavior;
- implements the intended Authorization-v2 successor semantics.

Accordingly:

```text
BEHAVIORAL_SPLIT=PROVEN
HISTORICAL_IDENTITY_PRESERVATION_REQUIRED=YES
RETROACTIVE_REINTERPRETATION_ALLOWED=NO
PRODUCTION_SOURCE_MUTATION=NO
MUTATION_AUTHORIZED=NO
```

This report therefore supports an identity/version-binding decision, not an immediate source modification.

---

## 2. Governing Engineering Rules

The work was performed under the following project rules.

### 2.1 No production change without normative basis and proven RED

No production change may be introduced merely because a local test, CI job, or reproduction wrapper fails. A production change requires a normative basis and a proven behavioral RED.

### 2.2 Tooling and materialization failures are not behavioral RED

The following are explicitly distinct from behavioral non-conformance:

- missing toolchain;
- incorrect shell environment;
- incorrect test target name;
- path-binding error;
- inaccessible junction/symlink/materialization path;
- wrapper invocation error;
- runner identity precondition mismatch caused by harness configuration.

These failures were recorded and corrected without changing production source.

### 2.3 Historical identities are immutable

Historical artifacts and verified identities must not be:

- renamed retroactively;
- overwritten;
- silently repaired;
- reinterpreted as if later semantics had always applied.

### 2.4 Semantic versioning follows observable behavior

A new semantic behavioral version is justified only when observable security/protocol behavior changes.

The current reproduction directly establishes such a change between `hacp-rs 0.1.0` and `hacp-rs 0.1.1`.

### 2.5 Production profiles must pin immutable identities

No production composition may refer to floating identities such as:

```text
latest
current
main
default
```

Exact immutable component/version/commit identities are required.

### 2.6 Core and HC2 evidence remain separate

The project does not synthesize a combined score such as `93/93`.

The evidence families are independent:

```text
Core = 38 cases
HC2  = 55 cases
```

---

## 3. Scope

This stage covered four controlled activities.

### 3.1 Historical GitCheck reproduction

The historically documented Rust GitCheck workflow was reproduced:

1. fresh build;
2. Core conformance;
3. HC2 conformance;
4. corrected raw-wire duplicate-member case;
5. informational benchmark.

### 3.2 Current Rust fresh build

The currently adopted Rust source at `cf309fc...` was built from a clean derived source surface.

### 3.3 Current Rust conformance reproduction

The same GitCheck conformance contract was executed against the current `hacp-rs 0.1.1` runner.

### 3.4 Authorization-v2 successor witness reproduction

The three explicit Authorization-v2 Rust test targets were executed independently.

---

## 4. Out of Scope

This stage did **not**:

- modify public Rust production source;
- modify historical GitCheck source;
- fix Authorization-v2 behavior;
- restore historical Core behavior;
- redesign `evaluate.rs`;
- alter canonical vectors;
- modify `hacp-sidecar` production source;
- modify `humanist-core`;
- create a release;
- create a tag;
- merge a branch;
- publish a new behavioral identity;
- activate a profile;
- authorize production deployment.

---

## 5. Repository Baseline

### 5.1 `hacp-spec`

```text
branch:
admit/hacp-rs

HEAD:
cf309fc5308863f41a9af6999f0a533449431a78

tree:
ebf6bd6817bf1c2698a7d260b7af081bc9002bff

worktree:
clean
```

### 5.2 `hacp-sidecar`

```text
HEAD:
d34a97abfeb8b93958552616d2d679ac9ce0477e

tree:
33265890d5921bb491c7805ec905bab9ea58ae25
```

The sidecar worktree contained one known untracked engineering record:

```text
docs/verification/HACP_RUST_AUTHORIZATION_V2_BEHAVIORAL_BOUNDARY_ENGINEERING_RECORD.md
```

No unexpected sidecar mutation was observed during the reproduction.

### 5.3 Exact sidecar production implementation identity

The production implementation identity relevant to HACP 1.1.1 remained:

```text
1bc10acbe79620a165cee573c5b260cd8639280f
```

with tree:

```text
a47dca5f03034c0697147acb3ca3a37d55ce9f12
```

This Rust reproduction did not alter that implementation.

---

## 6. Historical Rust GitCheck Baseline

The preserved historical GitCheck assembly was located at:

```text
<local-historical-gitcheck-root>
```

The historical verification surface identified Rust as a reference/conformance implementation rather than a production runtime promotion.

### 6.1 Historical package version

The historical fresh build compiled:

```text
hacp-rs v0.1.0
```

### 6.2 Historical runner identity

The exact historical runner identity was:

```text
SHA256:
10535e269db96ff534ff5546e1d35dde1e4e48cb7bcf21cbb0f0df88b0422296
```

### 6.3 Historical conformance script

```text
scripts/51-rust-conformance.sh

SHA256:
35CD427DE7AC16AED99C90A59193EB090DCE29012AE1FDC64080F3BA872C235E
```

### 6.4 Historical benchmark script

```text
scripts/52-rust-benchmark.py

SHA256:
64925A3D764D605350619E3B41706F4D9525EFD14B09EAF6159039E4D79C2F49
```

### 6.5 Historical build script

```text
scripts/50-build-rust.sh

SHA256:
5CD669D9FD630AB98D3E48C3E543B3FB8B49FBF85823C02B79EB3D15B63E4E6A
```

---

## 7. Historical GitCheck Reproduction

### 7.1 Fresh build

The preserved documented build process was reproduced with the expected toolchain.

Observed toolchain:

```text
rustc 1.98.1
cargo 1.98.1
libprotoc 35.1
```

The build completed successfully.

The resulting runner matched the historical documented identity exactly:

```text
EXPECTED_RUNNER_SHA256=
10535e269db96ff534ff5546e1d35dde1e4e48cb7bcf21cbb0f0df88b0422296

ACTUAL_RUNNER_SHA256=
10535e269db96ff534ff5546e1d35dde1e4e48cb7bcf21cbb0f0df88b0422296
```

Result:

```text
DOCUMENTED FRESH BUILD=PASS
RUNNER_IDENTITY=PASS
PUBLIC_SOURCE_MUTATION=NO
```

### 7.2 Historical Core and HC2 reproduction

The historical conformance contract declared:

```text
PROVENANCE=37_CORE_CORPUS_PLUS_CORRECTED_RAW_WIRE_CORE_INV5_006
CORE_TARGET=38/38
HC2_TARGET=55/55
NO_SYNTHETIC_COMBINED_SCORE
```

Observed result:

```text
Core normal Family A = 37/37 PASS
Core normal Family B = 37/37 PASS

HC2 Family A = 55/55 PASS
HC2 Family B = 55/55 PASS

Evaluator/infrastructure errors = 0
```

The corrected raw-wire `CORE-INV5-006` case also passed.

Its canonical request identity was:

```text
SHA256:
8d95b2d935f24b74e26ad9e81200d87b2cbbc092713ba78ac1ead3c011888fe4
```

Observed result:

```json
{"protocol_version":"1","decision":"DENY","reason_codes":["INVALID_ACTION"]}
```

Final historical conformance result:

```text
CORE_RESULT=38/38_PASS
HC2_RESULT=55/55_PASS
RESULT=RUST_CONFORMANCE_PASS
```

### 7.3 Historical informational benchmark

The benchmark was executed with:

```text
WARMUP_N=100
REQUESTED_SAMPLE_N=5000
```

Observed current reproduction snapshot:

```text
CORE_EVALUATE_P50_MS=8.591186
CORE_EVALUATE_P99_MS=13.724525

HC2_EVALUATE_P50_MS=8.430076
HC2_EVALUATE_P99_MS=11.516339
```

The benchmark explicitly remained informational:

```text
PERFORMANCE_GATE=INFORMATIONAL
CONFORMANCE_GATE=UNCHANGED
RESULT=HACP_RUST_BENCHMARK_PASS
```

Historical reproduction therefore closed as:

```text
HISTORICAL_RUST_GITCHECK_REPRODUCTION=PASS
DOCUMENTED_BUILD=REPRODUCED
DOCUMENTED_CONFORMANCE=REPRODUCED
DOCUMENTED_BENCHMARK=REPRODUCED
DOCUMENTED_RUNNER_IDENTITY=REPRODUCED
PUBLIC_SOURCE_MUTATION=NO
```

---

## 8. Current Rust Identity — `hacp-rs 0.1.1`

The current source surface at `cf309fc...` reported:

```text
name = "hacp-rs"
version = "0.1.1"
```

Exact identities:

```text
Cargo.toml SHA256:
379398a445ed3abbb0e0bce456e27e75e0854430f66ef78e07cabb96315e2b0d

evaluate.rs SHA256:
74580de3a9e345b64e2548edd07ddb4a4d7058c2f847cf0ebb9d8468c23fe041

evaluate.rs Git blob:
5a7ed545098394bcd76074757ab9d94e7f6f9423
```

The exact source commit was:

```text
cf309fc5308863f41a9af6999f0a533449431a78
```

---

## 9. Current `0.1.1` Fresh Build

A separate derived workspace was created:

```text
<local-current-rust-reproduction-root>
```

The public repository remained read-only.

### 9.1 Source identity check

Before build:

```text
EXPECTED_CARGO_SHA256=
379398a445ed3abbb0e0bce456e27e75e0854430f66ef78e07cabb96315e2b0d

ACTUAL_CARGO_SHA256=
379398a445ed3abbb0e0bce456e27e75e0854430f66ef78e07cabb96315e2b0d

EXPECTED_EVALUATE_SHA256=
74580de3a9e345b64e2548edd07ddb4a4d7058c2f847cf0ebb9d8468c23fe041

ACTUAL_EVALUATE_SHA256=
74580de3a9e345b64e2548edd07ddb4a4d7058c2f847cf0ebb9d8468c23fe041
```

Result:

```text
DERIVED_SOURCE_IDENTITY=PASS
```

### 9.2 Build environment

A pinned Rust builder image was used:

```text
rust@sha256:c49256cbe5ea0188bc658a689500d70c41eb51f009a7a7be209caf60a944f3ec
```

Toolchain:

```text
rustc 1.98.1
cargo 1.98.1
libprotoc 35.1
```

### 9.3 Build result

The current package compiled as:

```text
hacp-rs v0.1.1
```

Result:

```text
CARGO_BUILD_RC=0
BUILD_RC=0
FRESH_BUILD=PASS
```

Current fresh runner:

```text
SIZE:
1089576 bytes

SHA256:
64dcf8a2c54e9c59ccb8ffddc39642df6a669cf9b4a1786d4395b84bf941fd05
```

Platform:

```text
ELF 64-bit LSB pie executable, x86-64
```

Source identity remained unchanged after build.

Final build classification:

```text
CURRENT_0_1_1_FRESH_BUILD=PASS
DERIVED_SOURCE_IDENTITY_PRESERVED=YES
PUBLIC_SOURCE_MUTATION=NO
```

---

## 10. Tooling / Harness Failures Encountered During Current Reproduction

Several failures occurred before the decisive behavioral test. They are recorded here specifically to prevent accidental misclassification as Rust behavioral RED.

### 10.1 PowerShell wildcard + `-LiteralPath`

Initial derived source copy used a wildcard with `Copy-Item -LiteralPath`.

Observed error:

```text
cannot find path ...\hacp-rs\*
```

Classification:

```text
BUILD_STARTED=NO
FAILURE_CLASS=WRAPPER_INVOCATION
BEHAVIORAL_RESULT=NOT_ESTABLISHED
```

Correction:

- replaced wildcard use with explicit `Get-ChildItem | Copy-Item`.

### 10.2 WSL Rust toolchain missing from PATH

An early WSL build attempt returned:

```text
rustc: command not found
BUILD_RC=127
```

Classification:

```text
FAILURE_CLASS=ENVIRONMENT_TOOLING
BUILD_STARTED=NO
BEHAVIORAL_RESULT=NOT_ESTABLISHED
```

Correction:

- switched to the pinned Rust builder image.

### 10.3 Container login shell PATH reset

Using `bash -lc` caused the Rust toolchain to disappear from PATH inside the pinned container.

Observed:

```text
rustc: command not found
TOOLCHAIN_PREFLIGHT_RC=127
```

Classification:

```text
FAILURE_CLASS=CONTAINER_PATH_ENVIRONMENT
BUILD_STARTED=NO
```

Correction:

- removed login shell behavior;
- used explicit PATH;
- invoked `/usr/local/cargo/bin/rustc` and `/usr/local/cargo/bin/cargo`.

### 10.4 `robocopy` failure on preserved GitCheck filesystem object

Attempting to replicate the entire GitCheck root produced:

```text
ERROR 1920
Access to this file from the system is unavailable
```

Classification:

```text
CONFORMANCE_STARTED=NO
FAILURE_CLASS=MATERIALIZATION_TOOLING
BEHAVIORAL_RESULT=NOT_ESTABLISHED
```

Correction:

- abandoned whole-root copy;
- created a minimal derived verification surface.

### 10.5 Historical GitCheck root remained hard-bound

The first derived conformance script changed only the runner SHA but retained:

```text
GC='<historical-gitcheck-root>'
```

As a result, the script still evaluated the historical runner.

Observed:

```text
EXPECTED_RUNNER_SHA256=64dcf8...
ACTUAL_RUNNER_SHA256=10535e...
STOP=RUNNER_IDENTITY_MISMATCH
PRECONDITIONS_READY=0
```

Classification:

```text
FAILURE_CLASS=HARNESS_ROOT_BINDING
CONFORMANCE_STARTED=NO
BEHAVIORAL_RESULT=NOT_ESTABLISHED
```

Correction:

The derived script was changed in exactly two harness-only dimensions:

1. GitCheck verification root;
2. expected runner SHA.

A normalized comparison established:

```text
ONLY_ROOT_AND_RUNNER_SHA_CHANGED=True
```

No conformance rule, vector, expected decision, reason code, or corpus logic was changed.

### 10.6 Nonexistent aggregate Authorization-v2 test target

The first Authorization-v2 test command used:

```text
cargo test --test authorization_v2
```

Cargo reported that no such target existed.

Available targets included:

```text
authorization_v2_action_hash_reason_red
authorization_v2_envelope_signature_red
authorization_v2_signed_deny_red
```

Classification:

```text
AUTHORIZATION_V2_WITNESSES_EXECUTED=NO
FAILURE_CLASS=INVALID_TEST_TARGET
BEHAVIORAL_RESULT=NOT_ESTABLISHED
```

Correction:

- all three explicit test targets were executed independently.

---

## 11. Current `0.1.1` Historical Core / HC2 Conformance

A minimal derived verification surface was created at:

```text
<local-current-rust-conformance-root>
```

The current `hacp-spec` source was materialized from the exact Git object.

The derived conformance script was bound to:

```text
GC='/mnt/c/Personal/GitHub/Mount/hacp-current-rust-conformance'
SPEC="$GC/hacp-spec"
RUNNER="$GC/build/rust/hacp-rs-runner"
RESULT_DIR="$GC/results/rust-conformance"
```

The current runner precondition passed:

```text
EXPECTED_RUNNER_SHA256=
64dcf8a2c54e9c59ccb8ffddc39642df6a669cf9b4a1786d4395b84bf941fd05

ACTUAL_RUNNER_SHA256=
64dcf8a2c54e9c59ccb8ffddc39642df6a669cf9b4a1786d4395b84bf941fd05
```

The duplicate vector precondition also passed:

```text
EXPECTED_DUP_VECTOR_SHA256=
c686eee312e6f380ebd553eef82dda532b5833c28a3d623d0f351f729ba8cbe6

ACTUAL_DUP_VECTOR_SHA256=
c686eee312e6f380ebd553eef82dda532b5833c28a3d623d0f351f729ba8cbe6
```

Final precondition marker:

```text
PRECONDITIONS_READY=1
```

This was the first decisive current conformance execution without tooling or harness contamination.

---

## 12. Current Core Normal Corpus Result

Observed inventory:

```text
37
```

### 12.1 Family A — decision-only

Result:

```text
35/37 PASS
```

Failures:

```text
CORE-INV1-002
expected=CHECKPOINT
actual=DENY

CORE-RUNTIME-004
expected=CHECKPOINT
actual=DENY
```

This establishes an observable decision-level change.

### 12.2 Family B — strict decision + reason

Result:

```text
12/37 PASS
```

Failures included:

```text
CORE-INV1-002
DECISION expected=CHECKPOINT actual=DENY

CORE-INV1-005
expected=['ENVELOPE_EXPIRED']
actual=['SIGNATURE_FAILURE']

CORE-INV1-006
expected=['CHECKPOINT_TIMEOUT']
actual=['SIGNATURE_FAILURE']

CORE-INV2-002
expected=['BOUNDARY_CROSSING']
actual=['SIGNATURE_FAILURE']

CORE-INV2-003
expected=['BOUNDARY_CROSSING']
actual=['SIGNATURE_FAILURE']

CORE-INV2-004
expected=['BOUNDARY_CROSSING']
actual=['SIGNATURE_FAILURE']

CORE-INV2-005
expected=['SCOPE_EXCEEDED']
actual=['SIGNATURE_FAILURE']

CORE-INV2-006
expected=['BOUNDARY_CROSSING']
actual=['SIGNATURE_FAILURE']

CORE-INV2-007
expected=['BOUNDARY_CROSSING']
actual=['SIGNATURE_FAILURE']

CORE-INV2-008
expected=['UNKNOWN_ATTRIBUTE']
actual=['SIGNATURE_FAILURE']

CORE-INV3-002
expected=['HASH_MISMATCH']
actual=['SIGNATURE_FAILURE']

CORE-INV3-003
expected=['TOKEN_ENVELOPE_MISMATCH']
actual=['SIGNATURE_FAILURE']

CORE-INV3-004
expected=['TOKEN_EXPIRED']
actual=['SIGNATURE_FAILURE']

CORE-INV4-002
expected=['TOKEN_REVOKED']
actual=['SIGNATURE_FAILURE']

CORE-INV4-003
expected=['TRACEABILITY_FAILURE']
actual=['SIGNATURE_FAILURE']

CORE-INV4-004
expected=['TRACEABILITY_MISSING']
actual=['SIGNATURE_FAILURE']

CORE-INV4-005
expected=['TRACEABILITY_FAILURE']
actual=['SIGNATURE_FAILURE']

CORE-INV5-007
expected=['HASH_MISMATCH']
actual=['SIGNATURE_FAILURE']

CORE-INV7-002
expected=['BUDGET_EXHAUSTED']
actual=['SIGNATURE_FAILURE']

CORE-INV7-005
expected=['ENVELOPE_REVOKED']
actual=['SIGNATURE_FAILURE']

CORE-INV7-006
expected=['ENVELOPE_REVOKED']
actual=['SIGNATURE_FAILURE']

CORE-RUNTIME-002
expected=['CHECKPOINT_TIMEOUT']
actual=['SIGNATURE_FAILURE']

CORE-RUNTIME-003
expected=['HASH_MISMATCH']
actual=['SIGNATURE_FAILURE']

CORE-RUNTIME-004
DECISION expected=CHECKPOINT actual=DENY

CORE-RUNTIME-005
expected=['HUMAN_RESOLUTION_REQUIRED']
actual=['SIGNATURE_FAILURE']
```

The dominant change is therefore not random test instability. It is a systematic change in precedence and reason selection, with `SIGNATURE_FAILURE` occurring before many historical Core conditions that previously determined the outcome.

---

## 13. Current HC2-55 Result

Observed inventory:

```text
55
```

Result:

```text
Family A decision-only = 55/55 PASS
Family B strict        = 55/55 PASS
```

Therefore:

```text
CURRENT_0_1_1_HC2=PASS
```

This is important because the current Rust implementation is not generally nonfunctional. It preserves the bounded HC2 request-binding surface while diverging from historical Core semantics.

---

## 14. Current Corrected `CORE-INV5-006` Raw-Wire Result

The corrected raw-wire duplicate-member case remained valid.

Request identity:

```text
SHA256:
8d95b2d935f24b74e26ad9e81200d87b2cbbc092713ba78ac1ead3c011888fe4
```

Request properties:

```text
physical_lines: 1
verb_member_count: 2
contains "verb":"read": True
contains "verb":"delete": True
```

Runner result:

```json
{"protocol_version":"1","decision":"DENY","reason_codes":["INVALID_ACTION"]}
```

Validation:

```text
RAW_REQUEST_SHA_MATCH=YES
RAW_REQUEST_SINGLE_LINE=YES
RAW_STDERR_EMPTY=YES
RAW_DECISION_MATCH=YES
RAW_REASON_CODES_MATCH=YES
RAW_VALIDATION_RC=0
```

Therefore the duplicate-member raw-wire behavior did not regress.

---

## 15. Current `0.1.1` Final Historical Conformance Result

The current conformance execution produced:

```text
Core normal Family A = 35/37
Core normal Family B = 12/37

HC2 Family A = 55/55
HC2 Family B = 55/55

Evaluator/infrastructure errors = 0
```

Final script marker:

```text
RESULT=RUST_CONFORMANCE_FAIL
```

This is a true behavioral conformance RED relative to the historical Core contract.

Classification:

```text
CURRENT_0_1_1_HISTORICAL_CORE=FAIL
CURRENT_0_1_1_HC2=PASS
INFRASTRUCTURE_ERRORS=0
BEHAVIORAL_RED=PROVEN
```

---

## 16. Authorization-v2 Successor Witness Inventory

Three explicit current Rust test targets were present.

### 16.1 Action-hash reason witness

```text
tests/authorization_v2_action_hash_reason_red.rs
SHA256:
ed0888e7f47403ad37706a8e60989d1b22a8a04f6950ef21cb978de91eea0519
```

The asserted successor behavior is:

```text
token action_hash mismatch
→ decision DENY
→ reason SIGNATURE_FAILURE
```

### 16.2 Intent-envelope signature precedence witness

```text
tests/authorization_v2_envelope_signature_red.rs
SHA256:
c25d16e6c022f9c373568108d91e1f5d6d834bd28d98073af1c2645e70f1b785
```

The asserted successor behavior is:

```text
invalid IntentEnvelope signature
→ decision DENY
→ reason SIGNATURE_FAILURE
```

and the invalid signature is rejected before trusting envelope claims.

### 16.3 Authenticated signed-DENY witness

```text
tests/authorization_v2_signed_deny_red.rs
SHA256:
dacdfe1eadc7f6fe9658afd7b8deb6947293818c82b1c9aa1bfe4cbc5687ce07
```

The asserted successor behavior is:

```text
authenticated applicable signed DENY DecisionToken
→ authoritative DENY
→ POLICY_DENIED when no explicit token reason is supplied
```

---

## 17. Authorization-v2 Successor Witness Result

All three witness targets were run independently.

Observed result:

```text
ACTION_HASH_WITNESS_RC=0
ENVELOPE_SIGNATURE_WITNESS_RC=0
SIGNED_DENY_WITNESS_RC=0

AUTHORIZATION_V2_SUCCESSOR_WITNESSES=PASS
```

Source and runner identities remained preserved:

```text
SOURCE_AND_RUNNER_IDENTITY_PRESERVED=YES
PUBLIC_SOURCE_MUTATION=NO
```

Therefore:

```text
CURRENT_0_1_1_AUTHORIZATION_V2=PASS
```

---

## 18. Reproduced Behavioral Comparison

The controlled comparison is now:

| Surface | Historical `hacp-rs 0.1.0` | Current `hacp-rs 0.1.1` |
|---|---:|---:|
| Fresh build | PASS | PASS |
| Core historical contract | 38/38 PASS | FAIL |
| Core decision-only normal corpus | 37/37 PASS | 35/37 |
| Core strict normal corpus | 37/37 PASS | 12/37 |
| Corrected `CORE-INV5-006` | PASS | PASS |
| HC2 decision-only | 55/55 PASS | 55/55 PASS |
| HC2 strict | 55/55 PASS | 55/55 PASS |
| Authorization-v2 successor witnesses | not the historical target | PASS |
| Infrastructure errors in decisive conformance run | 0 | 0 |
| Public source mutation | NONE | NONE |

This is a reproducible behavioral split.

---

## 19. Nature of the Behavioral Split

The current `0.1.1` behavior is not merely a broken version of `0.1.0`.

The evidence demonstrates two simultaneous properties.

### 19.1 Historical Core behavior changed

Examples include:

```text
CHECKPOINT → DENY
```

and systematic replacement of many historical reason outcomes by:

```text
SIGNATURE_FAILURE
```

This reflects altered verification precedence and trust/signature evaluation order.

### 19.2 Authorization-v2 successor semantics are implemented

The same current implementation passes explicit tests requiring:

```text
action_hash mismatch
→ SIGNATURE_FAILURE

invalid intent-envelope signature
→ SIGNATURE_FAILURE

authenticated applicable signed DENY token
→ POLICY_DENIED
```

Therefore the behavioral divergence is intentional in shape, even though its compatibility with the historical Core conformance surface is not preserved.

---

## 20. Behavioral Identity Conclusion

The reproduction proves:

```text
hacp-rs 0.1.0 != hacp-rs 0.1.1
```

in observable protocol/security behavior.

The two versions therefore cannot be treated as one immutable behavioral identity.

This directly satisfies the project rule for semantic behavioral version separation:

> A new component semantic version is justified when observable security/protocol behavior changes.

The current evidence establishes such a change.

---

## 21. Historical Identity Preservation Consequence

The historical GitCheck result remains valid for the historical surface:

```text
hacp-rs 0.1.0
Core 38/38 PASS
HC2 55/55 PASS
```

Nothing in the current reproduction invalidates or retroactively changes that result.

The current `0.1.1` result must therefore not be used to reinterpret the historical `0.1.0` identity.

Likewise, the historical Core result must not be silently claimed for `0.1.1`.

Correct treatment requires two distinct immutable behavioral identities.

---

## 22. Version / Build Matrix Consequence

The authoritative Build / Version Matrix should eventually record the two surfaces separately.

At minimum, the matrix must be capable of expressing:

```text
Rust behavioral generation A
  package = hacp-rs 0.1.0
  historical Core = 38/38 PASS
  HC2 = 55/55 PASS
  historical GitCheck runner SHA =
    10535e269db96ff534ff5546e1d35dde1e4e48cb7bcf21cbb0f0df88b0422296

Rust behavioral generation B
  package = hacp-rs 0.1.1
  source commit =
    cf309fc5308863f41a9af6999f0a533449431a78
  historical Core = FAIL
  HC2 = 55/55 PASS
  Authorization-v2 successor witnesses = PASS
  current runner SHA =
    64dcf8a2c54e9c59ccb8ffddc39642df6a669cf9b4a1786d4395b84bf941fd05
```

The exact final behavioral generation identifiers remain a version/build-matrix ownership decision and should not be invented by this report.

---

## 23. Composition Consequence

The HACP 1.1.1 composition must not imply that:

```text
hacp-rs 0.1.0
```

and:

```text
hacp-rs 0.1.1
```

are interchangeable.

Any composition that references Rust behavior must pin an exact immutable identity.

This report does not itself determine which Rust behavioral generation belongs in a production composition.

It only proves that the choice is semantically meaningful and cannot be left implicit.

---

## 24. What This Report Does Not Prove

This report does **not** prove that:

- `hacp-rs 0.1.1` should be reverted;
- historical Core semantics are necessarily the desired future semantics;
- Authorization-v2 semantics should be removed;
- the sidecar implementation is defective;
- HACP 1.1.1 is ready for release;
- HACP 1.1.1 is blocked solely because of Rust;
- `hacp-rs 0.1.1` is production-authorized;
- a new release should be published;
- a production profile should be activated.

Those decisions require separate normative and release-stage analysis.

---

## 25. What This Report Does Prove

This report does prove that:

1. the historical Rust GitCheck evidence is reproducible;
2. the historical runner identity is reproducible;
3. historical `hacp-rs 0.1.0` satisfies Core 38/38 and HC2 55/55;
4. current `hacp-rs 0.1.1` builds successfully from exact adopted source;
5. current `0.1.1` preserves HC2 55/55;
6. current `0.1.1` preserves corrected raw-wire duplicate-member rejection;
7. current `0.1.1` does not satisfy the historical Core contract;
8. the current historical-Core failure occurs with zero evaluator/infrastructure errors;
9. current `0.1.1` passes the three explicit Authorization-v2 successor witnesses;
10. the behavioral difference between `0.1.0` and `0.1.1` is observable and security/protocol relevant;
11. therefore, the two surfaces require distinct behavioral identity treatment;
12. no public production source change was required to establish this conclusion.

---

## 26. Final Evidence Matrix

```text
HISTORICAL HACP-RS 0.1.0

Fresh build:
PASS

Runner identity:
10535e269db96ff534ff5546e1d35dde1e4e48cb7bcf21cbb0f0df88b0422296

Core:
38/38 PASS

HC2:
55/55 PASS

Corrected CORE-INV5-006:
PASS

Benchmark:
INFORMATIONAL PASS

Historical GitCheck:
REPRODUCED


CURRENT HACP-RS 0.1.1 / cf309fc

Fresh build:
PASS

Runner identity:
64dcf8a2c54e9c59ccb8ffddc39642df6a669cf9b4a1786d4395b84bf941fd05

Historical Core normal Family A:
35/37

Historical Core normal Family B:
12/37

Historical Core final:
FAIL

HC2 Family A:
55/55 PASS

HC2 Family B:
55/55 PASS

Corrected CORE-INV5-006:
PASS

Authorization-v2 action_hash witness:
PASS

Authorization-v2 envelope-signature witness:
PASS

Authorization-v2 signed-DENY witness:
PASS

Authorization-v2 successor witnesses:
PASS

Evaluator/infrastructure errors:
0

Public source mutation:
NO
```

---

## 27. FACT / INFERENCE / OPEN QUESTION / DECISION

### FACT

```text
Historical hacp-rs 0.1.0 GitCheck reproduction = PASS.

Historical hacp-rs 0.1.0:
  Core = 38/38 PASS
  HC2 = 55/55 PASS.

Current hacp-rs 0.1.1 at cf309fc:
  fresh build = PASS,
  historical Core = FAIL,
  HC2 = 55/55 PASS,
  corrected CORE-INV5-006 = PASS,
  Authorization-v2 successor witnesses = PASS.

Current decisive conformance evaluator/infrastructure errors = 0.

Public source mutation = NO.
```

### INFERENCE

```text
hacp-rs 0.1.0 and hacp-rs 0.1.1 represent distinct observable
security/protocol behavioral identities.

The current 0.1.1 surface is not equivalent to the historical
0.1.0 surface.

The difference is not explained by tooling, wrapper, materialization,
runner identity, vector identity, or evaluator infrastructure.

The current 0.1.1 behavior is consistent with the explicit
Authorization-v2 successor witnesses.
```

### OPEN QUESTION

```text
What exact immutable behavioral generation identifiers should be assigned
to the historical 0.1.0 surface and the current 0.1.1 successor surface
in the authoritative Build / Version Matrix?

Which behavioral generation should be bound to each future composition
or production profile?

Does S10.5 require only identity revalidation documentation, or does it
require a separate release-blocking decision for the current 0.1.1
historical-Core incompatibility?
```

### DECISION

```text
HISTORICAL_GITCHECK_REPRODUCTION=PASS

CURRENT_0_1_1_BUILD=PASS

CURRENT_0_1_1_HC2=PASS

CURRENT_0_1_1_HISTORICAL_CORE=FAIL

CURRENT_0_1_1_AUTHORIZATION_V2=PASS

BEHAVIORAL_SPLIT=PROVEN

HISTORICAL_IDENTITY_PRESERVATION_REQUIRED=YES

RETROACTIVE_REINTERPRETATION=PROHIBITED

PUBLIC_SOURCE_MUTATION=NO

MUTATION_AUTHORIZED=NO
```

---

## 28. Recommended Next Stage

The next stage should be documentary and identity-oriented rather than an immediate implementation change.

Recommended sequence:

1. record this behavioral boundary as formal S10.5 evidence;
2. bind historical `hacp-rs 0.1.0` and current `hacp-rs 0.1.1` to distinct immutable behavioral identities in the authoritative Version / Build Matrix;
3. reconcile the HACP 1.1.1 composition with that identity distinction;
4. determine whether current `0.1.1` historical-Core incompatibility is:
   - an accepted successor-generation property,
   - a release-blocking incompatibility,
   - or a reason to introduce an explicitly separated successor profile/composition;
5. do not modify production source until that normative/identity decision is complete.

---

## 29. Closure

This stage is complete.

The historical Rust GitCheck implementation has been independently reproduced.

The currently adopted Rust `0.1.1` implementation has been independently rebuilt and exercised.

The observed result is not ambiguous:

```text
historical 0.1.0:
Core PASS / HC2 PASS

current 0.1.1:
historical Core FAIL / HC2 PASS / Authorization-v2 PASS
```

The behavioral boundary is therefore established by direct execution evidence rather than inference.

No production mutation was required.

The engineering task now transitions from reproduction to explicit immutable identity binding and release/composition reconciliation.

---

**END OF REPORT**
