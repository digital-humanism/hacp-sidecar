# Rust Core Composition Reproduction

## Status

```text
REPRODUCIBLE
DECISION-LEVEL RESULT: 36/38 DIRECT PASS
ADJUDICATED CONFLICTS: 2
NEW REGRESSIONS: 0
```

This note records the public reproduction procedure for the Rust HACP-Core verification surface used by the HACP 1.1.1 release composition.

It is a verification note only.

It does not modify the HACP protocol, normative semantics, release composition, or production runtime behavior.

---

## Immutable Composition Binding

The Rust Core result is reproduced against the exact immutable HACP specification identity bound by the HACP 1.1.1 composition:

```text
HACP_RELEASE=1.1.1
HACP_COMPOSITION_ID=HACP-1.1.1-spec-3ea4adc-sidecar-1bc10ac

HACP_SPEC_COMMIT=
3ea4adcbf6ef2548e41d340709945326026a73b6

SIDECAR_IMPLEMENTATION_COMMIT=
1bc10acbe79620a165cee573c5b260cd8639280f
```

The Rust Core verification described here is SPEC-owned.

The sidecar repository records this reproduction because the result is part of the HACP 1.1.1 runtime/release evidence chain.

A floating branch tip such as `main`, `latest`, or the current tip of an admission branch must not replace the immutable specification commit above.

---

## Source Repository

Public specification repository:

```text
https://github.com/digital-humanism/hacp-spec.git
```

Required immutable source identity:

```text
3ea4adcbf6ef2548e41d340709945326026a73b6
```

All paths in this document are repository-relative.

No machine-local absolute path is part of the reproduction contract.

---

## Toolchain

The recorded Windows reproduction used:

```text
rustc 1.98.1
cargo 1.98.1
Python 3.13.14

Rust toolchain:
stable-x86_64-pc-windows-msvc
```

The Rust source tree is built with the tracked Cargo lock file.

Required build property:

```text
cargo build --locked
```

A different host or compiler installation may produce a different executable hash. The executable hash is evidence for one build instance; it is not a permanent normative identity.

---

## Obtain the Exact Specification Source

Clone the public repository:

```powershell
git clone https://github.com/digital-humanism/hacp-spec.git hacp-spec
```

Enter the clone:

```powershell
Set-Location hacp-spec
```

Materialize the exact composition identity:

```powershell
git checkout --detach 3ea4adcbf6ef2548e41d340709945326026a73b6
```

Verify:

```powershell
git rev-parse HEAD
git status --short
```

Required result:

```text
HEAD:
3ea4adcbf6ef2548e41d340709945326026a73b6

working tree:
clean
```

Do not substitute the current tip of `admit/hacp-rs` for the immutable commit.

---

## Build the Rust Conformance Runner

From the specification repository root:

```powershell
Set-Location hacp-rs
```

Build the runner with the explicit MSVC toolchain:

```powershell
cargo +stable-x86_64-pc-windows-msvc build `
    --locked `
    --release `
    --bin hacp-rs-runner
```

The resulting executable is:

```text
hacp-rs/target/release/hacp-rs-runner.exe
```

Return to the repository root:

```powershell
Set-Location ..
```

The recorded A2 reproduction produced:

```text
BUILD_RC=0

RUNNER_SHA256=
72DC859E20504B8A1E3047548506DA7BBC3B058C1037585DA879032D640B8D29
```

The hash above identifies that specific recorded Windows build. A fresh valid build on another machine is not required to reproduce the same executable hash.

---

## Execute HACP-Core

From the `hacp-spec` repository root:

```powershell
python .\harness\harness_runner.py `
    --runner .\hacp-rs\target\release\hacp-rs-runner.exe `
    --vectors-dir .\vectors `
    --manifest .\harness\conformance_manifest.json `
    --implementation-name hacp-rs `
    --implementation-version 0.1.0-1.1.0 `
    --output console
```

The harness verifies:

```text
HACP Conformance Harness v0.9.2
Protocol version: 1
Spec: 0.9.2 (HACP-Core)

Vector set:
core-0.9.2

Vector-set digest:
sha256:1e167887106463cf89c81f3898e1f3ae4fd905bc807084959c787287f6575d58
```

All 38 Core vectors are executed.

---

## Understanding the Harness Result

The current generic runner harness performs strict comparison of both:

```text
decision outcome
+
reason_codes
```

The recorded execution therefore reports:

```text
RESULTS: 13/38 passed
CORE_HARNESS_RC=1
```

This number must not be interpreted as meaning that only 13 Core decisions are correct.

The 25 reported harness failures divide into two distinct classes.

### Decision-correct cases with reason-code differences

For 23 cases the harness explicitly reports:

```text
outcome_correct: True
```

while the reason-code comparison differs.

A representative form is:

```text
expected decision:
DENY

actual decision:
DENY

expected reason:
<case-specific semantic reason>

actual reason:
SIGNATURE_FAILURE
```

These cases are decision-level direct passes for the HACP-Core release classification, but they are not strict exact-reason passes under the generic harness comparison.

Therefore:

```text
13 strict outcome+reason passes
+
23 additional decision-correct cases
=
36 decision-level direct passes
```

### Adjudicated outcome conflicts

Exactly two Core cases have an outcome mismatch:

```text
CORE-INV1-002
CORE-RUNTIME-004
```

For both:

```text
expected:
CHECKPOINT

actual:
DENY
```

These are the two preserved adjudicated Rust Core conflicts.

They must not be promoted to direct passes.

---

## Rust Core Classification

The correct HACP 1.1.1 release classification is therefore:

```text
HACP-Core / Rust

36/38 direct decision-level PASS

2 adjudicated conflicts:
CORE-INV1-002
CORE-RUNTIME-004

new regressions:
0
```

Equivalent reconciliation:

```text
38 total Core cases
- 2 adjudicated outcome conflicts
= 36 direct decision-level passes
```

The following claim is prohibited:

```text
Rust 38/38 direct PASS
```

The generic harness result:

```text
13/38 strict outcome+reason matches
```

and the release classification:

```text
36/38 direct decision-level PASS
+ 2 adjudicated conflicts
```

describe different comparison boundaries and must not be collapsed into one number.

---

## CORE-INV5-006 Raw Duplicate-Member Witness

`CORE-INV5-006` requires preservation of duplicate JSON object members.

A normal JSON parse/reserialize cycle may destroy that condition, so the byte-preserving ingress behavior is also exercised directly by the tracked Rust test:

```text
hacp-rs/tests/json_ingress_v1_red.rs
```

Run:

```powershell
Set-Location hacp-rs

cargo +stable-x86_64-pc-windows-msvc test `
    --locked `
    --test json_ingress_v1_red `
    -- `
    --nocapture
```

Expected result:

```text
running 1 test

test json_ingress_v1_must_reject_duplicate_proposed_action_key ... ok

test result:
1 passed
0 failed
0 ignored
0 measured
0 filtered out
```

Recorded A2 result:

```text
RAW_WITNESS_RC=0
```

This witness confirms that the duplicate-member condition reaches the Rust runner without being normalized away by a conventional JSON object model.

---

## Final Reproduction Result

The recorded immutable-composition reproduction established:

```text
SPEC_IDENTITY=
3ea4adcbf6ef2548e41d340709945326026a73b6

RUST_RUNNER_BUILD=PASS

CORE_VECTOR_COUNT=38

STRICT_OUTCOME_AND_REASON_MATCH=13/38

DECISION_CORRECT=36/38

ADJUDICATED_CONFLICT_COUNT=2

ADJUDICATED_CONFLICTS=
CORE-INV1-002
CORE-RUNTIME-004

CORE_INV5_006_RAW_WITNESS=PASS

NEW_REGRESSIONS=0

SOURCE_MUTATION=NO
```

Final classification:

```text
RUST_CORE=
36/38 DIRECT PASS + 2 ADJUDICATED
```

---

## Evidence Boundary

This reproduction establishes the Rust HACP-Core result only.

It does not by itself establish or replace:

```text
Rust HC2-55 qualification
Rust package-test qualification
Rust HTTP proxy runtime qualification
Go sidecar runtime qualification
Go HTTP upstream-forwarding qualification
benchmark or latency evidence
```

Those are separate verification surfaces.

Likewise, this note does not activate Rust as a replacement for the Go enforcement sidecar.

---

## Reproducibility Rules

A valid reproduction must preserve all of the following:

```text
1. exact immutable hacp-spec commit
2. tracked Cargo.lock
3. explicit Rust toolchain identity
4. fresh hacp-rs-runner build
5. canonical Core vector set
6. canonical conformance manifest
7. all 38 Core cases executed
8. CORE-INV1-002 and CORE-RUNTIME-004 retained as adjudicated
9. raw duplicate-member witness executed separately
10. source checkout unchanged after execution
```

Do not:

```text
use a floating branch tip as the source identity
use historical private or mounted workspaces as test inputs
rewrite vectors to force a green result
convert reason-code mismatches into direct strict passes
convert the two adjudicated outcome conflicts into direct passes
claim Rust 38/38 direct
```

---

## Recorded Disposition

```text
RUST_CORE_REPRODUCTION=PASS

RUST_CORE_DIRECT_DECISION_PASS=36/38

RUST_CORE_ADJUDICATED=2

RUST_CORE_NEW_REGRESSIONS=0

REPOSITORY_MUTATION=NO
```
