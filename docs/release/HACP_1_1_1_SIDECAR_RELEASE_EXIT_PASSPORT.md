# HACP 1.1.1 — SIDECAR Release Exit Passport

**Repository:** `digital-humanism/hacp-sidecar`
**Release target:** HACP 1.1.1
**Status:** PRE-RELEASE CANDIDATE — RELEASE READY
**Passport stage:** SIDECAR-A4
**Activation performed:** No

> The passport does not create release readiness. It records proven release readiness.

## Candidate identity

```text
branch:
admit/hacp-1.1.1-pre-release

local signed release candidate:
e59ac78b106f62e2fecfe03190876768e6216c75

candidate parent / previous public admission identity:
9e88a444b9c4a33b0767f326b391d2fba06dadfc

candidate tree:
ce7ca13f442797be3dc1a756e0b9fa2293fb317d
```

The release candidate has a verified GOOD SSH ED25519 signature.

The signed successor candidate has now been published to the public admission branch. Its parent remains the immediately preceding public admission identity.

## Cross-repository bindings

Earlier verified execution/admission bindings:

`	ext
HACP SPEC verified candidate:
03885b2b062905072040dc245c3d25d283c249c4

humanist-core verified candidate:
91360a1f35585c078981880f95ba566a822e7dde
`

These identities were verified during the SIDECAR release-readiness sequence.

Current public release-exit identities:

`	ext
HACP SPEC public admission:
b1fe606d8bd9972adc70949702fc16f406cd9d79

humanist-core public admission:
29611af2c43ae44ff3b262610e0834d66496bcb7

HACP SIDECAR public admission:
e59ac78b106f62e2fecfe03190876768e6216c75
`

The later SPEC and humanist-core successors are documentary/release-exit descendants and do not redefine the already verified lower behavioral composition.

## Candidate scope

The successor candidate changes only the tracked Rust verification-contract/tooling surface required to make text identity checkout-independent.

```text
test/matrix/t_rs/RUST_VERIFICATION_CONTRACTS.md
test/matrix/t_rs/Readme_map_file_RS.md
test/matrix/t_rs/Scripts/00-Preflight.ps1
test/matrix/t_rs/Scripts/01-Prepare-Environment.ps1
test/matrix/t_rs/Scripts/02-Run-Verification.ps1
test/matrix/t_rs/Scripts/WindowsVerification.Rust.Common.psm1
test/matrix/t_rs/WINDOWS_EXECUTION_MANIFEST_RS.md
```

No Rust protocol, security, or production-runtime semantic change is claimed by this successor commit.

## Release checks

```text
public admission identity        PASS
candidate ancestry               PASS
candidate signature              PASS
candidate source boundary        PASS
public document audit            PASS
runtime/integration verification PASS
clean candidate reproduction     PASS
cross-repository bindings        PASS
claim hygiene                    PASS
Family B release-gate use        NO
publication performed            PASS
activation performed             NO
```

## HACP-Core evidence

```text
Go          38/38 PASS
TypeScript  38/38 PASS
Python      38/38 PASS
Rust        36/38 direct PASS + 2 adjudicated
```

The Rust Core result is intentionally not represented as `38/38 direct`.

Core evidence is not aggregated with HC2 into a synthetic combined score.

## HACP-Enforcement revision 2 / HC2-55

```text
Go          55/55 PASS
TypeScript  55/55 PASS
Python      55/55 PASS
Rust        55/55 PASS
```

Core and HC2-55 remain separate evidence families.

## Runtime boundary

The verified runtime evidence preserves the established implementation boundary:

```text
Go HTTP:
authorization + real upstream forwarding + upstream response

Rust HTTP:
authorization + local ALLOW/DENY response
no upstream-forwarding claim
```

No latency ranking between these different execution paths is asserted.

## Verification identity correction

Clean reproduction established that raw worktree byte identity for selected text verification inputs could depend on checkout EOL representation.

The accepted correction uses canonical text identity for the affected verification inputs:

```text
optional UTF-8 BOM removal
CRLF -> LF
lone CR -> LF
SHA-256 over canonical bytes
```

The correction is limited to verification tooling/contracts and does not redefine HACP protocol behavior.

Result:

```text
t_go                                      PASS
t_rs                                      PASS
checkout-independent text identity        PASS
candidate reproducibility                  PASS
```

## Public evidence

Repository-relative public evidence available in this repository includes:

```text
docs/release/HACP_1_1_1_SIDECAR_A2_RUNTIME_INTEGRATION_REPRODUCIBILITY_REPORT_EN_2026-10-15.md
docs/release/SIDECAR_A2_FINAL_CLOSURE_DECISION_2026-10-15.md
```

No private or local filesystem path is required by this passport.

## Publication state

```text
public admission branch:
admit/hacp-1.1.1-pre-release

public admission identity:
e59ac78b106f62e2fecfe03190876768e6216c75

signed release candidate:
e59ac78b106f62e2fecfe03190876768e6216c75

candidate parent:
9e88a444b9c4a33b0767f326b391d2fba06dadfc

candidate published:
YES

main integration:
NO

tag:
NO

release:
NO

activation:
NO
```

This passport does not authorize or perform push, merge, tagging, release publication, or activation.

## Release eligibility

```text
SIDECAR_A2=PASS
SIDECAR_A3=PASS
CANDIDATE_REPRODUCIBLE=PASS

SIDECAR_RELEASE_READY=YES
```

Next stage:

```text
NEXT_STAGE=SIDECAR-A5

PUSH_AUTHORIZED=NO
MERGE_AUTHORIZED=NO
TAG_AUTHORIZED=NO
RELEASE_AUTHORIZED=NO
ACTIVATION_AUTHORIZED=NO
```