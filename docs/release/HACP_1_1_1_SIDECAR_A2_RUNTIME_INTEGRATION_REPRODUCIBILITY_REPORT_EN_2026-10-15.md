# HACP 1.1.1 — SIDECAR-A2 Runtime / Integration Reproducibility Report

Date: 2026-10-15

Classification:

```text
INTERNAL RELEASE-ENGINEERING CLOSURE ARTIFACT
SIDECAR-A2 RUNTIME / INTEGRATION REPRODUCIBILITY REPORT
```

Current stage status:

```text
SIDECAR_A2_MATRIX_A=PASS
SIDECAR_A2_MATRIX_B=PASS
SIDECAR_A2_EXECUTION_RECONCILIATION=PASS

UNRESOLVED_RUNTIME_HOLD=0
UNRESOLVED_PROCEDURE_HOLD=0

PRODUCTION_SEMANTIC_RED=NO

SIDECAR_A2_FULL_ACCEPTANCE=NOT YET DECLARED
```

This report is the required English SIDECAR-A2 runtime/integration closure artifact.

The final `SIDECAR_A2=PASS` declaration remains a separate final cross-check step after this report is materialized and reconciled with the RU internal report, closure table, evidence manifest and R9.2 reconciliation.

---

# 1. Purpose

SIDECAR-A2 was a controlled reproducibility and runtime/integration qualification stage for HACP 1.1.1.

It was not a redesign stage.

It was not an architecture-expansion stage.

It was not authorization to change production semantics in order to obtain a green result.

The stage was required to establish, using explicit execution evidence:

1. SPEC-owned language conformance;
2. SIDECAR-owned runtime and integration behavior;
3. HTTP authorization behavior;
4. real Go upstream forwarding;
5. Rust local-response authorization behavior;
6. trust, revocation, binding, replay and provenance behavior;
7. distributed-control convergence;
8. wire interoperability;
9. external Python-to-Go integration;
10. anti-bypass behavior;
11. package and static qualification surfaces;
12. timing observations within their actual workload boundaries;
13. evidence ownership and reproducibility;
14. absence of unresolved runtime or procedural HOLDs;
15. absence of an established production semantic RED.

The stage followed an evidence-first rule:

```text
EXECUTE THE CONTROLLED PLAN.
DO NOT EXPAND THE CLAIM.
DO NOT SUBSTITUTE HISTORICAL INPUTS FOR CURRENT EXECUTION.
DO NOT CONVERT MISSING EVIDENCE INTO PASS.
```

---

# 2. Controlling execution boundary

Accepted repositories:

```text
S = hacp-sidecar
P = hacp-spec
C = humanist-core
```

These repositories formed the admissible current execution boundary.

Historical G/M trees, mounted historical copies, laboratory repositories and other prior artifacts were not accepted as current release-execution dependencies.

Historical artifacts could be used for context and comparison only.

They could not silently repair a missing current procedure or become current acceptance inputs.

---

# 3. Current hacp-sidecar identity

Current candidate branch:

```text
admit/hacp-1.1.1-pre-release
```

Current local HEAD after Rust HTTP ownership admission:

```text
3018b6bac9753bb7bc4c090d54de71f8bd5cb2f9
```

Commit:

```text
test: admit Rust HTTP reproduction procedure
```

The commit was signed.

Exact admitted paths:

```text
benchmarks/rust-http-reproduction/main.go
docs/verification/rust-http-local-reproduction.md
```

No push was performed as part of the controlled SIDECAR-A2 execution.

---

# 4. Authoritative closure artifacts

## 4.1 R9.1 historical manifest

```text
R9_EVIDENCE_FILE_MANIFEST.csv
ROWS=37
SHA256=AA97925DAB4C82F1482EB1F2B1BEF0E4A2504102565FDBE1C431E85B1EBCD821
```

Disposition:

```text
IMMUTABLE HISTORICAL SNAPSHOT
PRESERVED
NOT OVERWRITTEN
```

---

## 4.2 Current R9.2 evidence manifest

```text
R9_2_EVIDENCE_FILE_MANIFEST.csv
ROWS=91
POST_BASELINE_ROWS=54
SHA256=BABA6E6AA0ACE477BF3290D993A72CA8B47E7B9800AB70CF3111F1CC0C4A9643
```

This is the current curated SIDECAR-A2 acceptance-evidence inventory.

Generated build trees, dependency trees, caches and temporary workspaces were not treated as individual acceptance-evidence records.

---

## 4.3 R9.2 reconciliation

```text
R9_2_RESULTS_RECONCILIATION_UPDATED.md
SHA256=B1A8FE5F929360BCB911EFAC6AA27D9EE2D967B7DCD784C22AC9BDC218FAC106
```

Result:

```text
R9_2_CLASSIFICATION=COMPLETE
SIDECAR_A2_MATRIX_A=PASS
SIDECAR_A2_MATRIX_B=PASS
SIDECAR_A2_EXECUTION_RECONCILIATION=PASS
```

---

## 4.4 RU internal closure report

```text
SIDECAR_A2_REPRODUCIBILITY_INTERNAL_REPORT_RU_2026-10-15.md
SHA256=38A76C9874C2ED6DA45C8504EB3EA1EB427400545E90D9D9C946556E0C4DB54F
```

The RU report is the detailed internal execution and boundary record.

This English report preserves the same claim limits and does not extend them.

---

# 5. Matrix A — SPEC-owned language conformance

## 5.1 Go

Observed:

```text
Core = 38/38 PASS
HC2  = 55/55 PASS
```

Classification:

```text
GO_CORE=PASS
GO_HC2=PASS
```

---

## 5.2 TypeScript

The aggregate package regression alone was not used to infer the Core-38 result.

An explicit Core mapping artifact was created:

```text
L2_TYPESCRIPT\04_core38_exact_mapping.txt
SHA256=F3602610EB8DCEA0316F292A3E0C58A604FC2B25DED4371F8838E4CF41C5E0DE
```

It records:

```text
EXPECTED_VECTOR_COUNT=38
EXECUTED_VECTOR_COUNT=38
CORE_VECTOR_RESULT=38/38 PASS

NODE_TEST_ACCOUNTING=1 inventory assertion + 38 Core conformance vectors
NODE_TEST_TOTAL=39
NODE_PASS=39
NODE_FAIL=0
NODE_SKIP=0
NODE_TODO=0
```

HC2:

```text
55/55 PASS
```

Classification:

```text
TYPESCRIPT_CORE=38/38 PASS
TYPESCRIPT_HC2=55/55 PASS
```

---

## 5.3 Python

Observed:

```text
Core = 38/38 PASS
HC2  = 55/55 PASS
```

Classification:

```text
PYTHON_CORE=PASS
PYTHON_HC2=PASS
```

---

## 5.4 Rust Core

Rust Core was reconciled against the exact immutable activated SPEC identity:

```text
HACP_SPEC_COMMIT=3ea4adcbf6ef2548e41d340709945326026a73b6
```

Exact-bound Rust runner:

```text
SHA256=C8294E4C72057B994D9867C90AEF02E1B2039E1F9038D44F2AFA1053F2BF1F61
```

Observed result:

```text
normal corpus:
35/37 direct

raw duplicate-member witness:
1/1 direct

total direct:
36/38 PASS

adjudicated:
CORE-INV1-002
CORE-RUNTIME-004

new regressions:
0
```

Classification:

```text
RUST_CORE=36/38 DIRECT PASS + 2 ADJUDICATED
RUST_CORE_NEW_REGRESSIONS=0
```

The report does not claim:

```text
Rust Core = 38/38 direct
```

because that claim is not supported by the execution evidence.

---

## 5.5 Rust HC2

Observed:

```text
55/55 PASS
```

Classification:

```text
RUST_HC2=PASS
```

Core and HC2 remain separate conformance families.

No artificial combined claim is created.

---

# 6. Matrix B — SIDECAR runtime and integration

The required SIDECAR-A2 runtime and integration surfaces were executed and reconciled.

No unresolved runtime HOLD remains.

No unresolved procedure HOLD remains.

---

# 7. Native Go regression

Fresh native Go execution completed successfully.

Relevant surfaces included:

```text
full package regression
control-plane tests
go vet
runtime package tests
trust
proxy
wire
evaluate
```

No production semantic regression was established.

Classification:

```text
NATIVE_GO_RUNTIME=PASS
PRODUCTION_SEMANTIC_RED=NO
```

---

# 8. Runtime HC2 black-box

Observed:

```text
55/55 PASS
```

Classification:

```text
HC2_RUNTIME_BLACK_BOX=PASS
```

---

# 9. Core runtime diagnostic

The sidecar Core diagnostic reproduced:

```text
15/38 exact reason correspondence
```

The remaining known cases preserve the relevant decision-level behavior while exposing historical reason-code divergence.

This diagnostic is not used to replace the SPEC-owned Core conformance claims.

---

# 10. Trust, revocation, expiration, binding, budget and replay

Required current runtime surfaces were exercised for:

```text
trust lifecycle
signature verification
canonicalization
revocation
expiration
action binding
request constraints
budget
replay
```

Classification:

```text
PASS
```

No new semantic RED was established.

---

# 11. Provenance

The package:

```text
internal/provenance
```

contains no test files.

Therefore:

```text
ZERO PACKAGE TESTS
```

was not classified as a provenance test PASS.

Required provenance behavior was instead exercised through explicit runtime/process scenarios.

## H01

```text
accepted-before-forward
PASS
```

## H02

Provenance-capacity failure:

```text
HTTP 403
DENY
TRACEABILITY_FAILURE
NO UPSTREAM
PASS
```

## H03

Graceful-shutdown persistence:

```text
HACP authorization = ALLOW
upstream transport unavailable
HTTP result = 502
graceful shutdown succeeds
listener removed
provenance persisted
PASS
```

The authorization/transport boundary is:

```text
DENY + 403
=
HACP denied / no forward

ALLOW + 2xx
=
HACP allowed / upstream succeeded

ALLOW + 502
=
HACP allowed / upstream transport failed
```

A transport failure after an ALLOW decision is not reclassified as a HACP DENY.

Classification:

```text
PROVENANCE_REQUIRED_BEHAVIOR=PASS
R4_PROVENANCE_MAPPING=CLOSED
```

---

# 12. Distributed control

Distributed-control semantics were reproduced using:

```text
one in-memory control plane

two independent stores
two independent states
two independent subscribers
two independent evaluators
```

The two independent replica pipelines converged to the same revision and revocation state, producing:

```text
DENY
TOKEN_REVOKED
```

Classification:

```text
DISTRIBUTED_CONTROL=PASS
```

Claim boundary:

```text
independent in-process replica pipelines
```

This report does not claim two independent operating-system sidecar processes.

---

# 13. Wire interoperability

Observed:

```text
5/5 PASS
```

Classification:

```text
WIRE_INTEROPERABILITY=PASS
```

---

# 14. External Python-to-Go E2E

Fresh external execution:

```text
fail-closed case = PASS
remaining four cases = PASS

TOTAL = 5/5 PASS
```

Classification:

```text
EXTERNAL_PYTHON_GO_E2E=PASS
```

The previous HOLD was superseded by current execution evidence.

---

# 15. Anti-bypass

Observed:

```text
AB-E2E-01
AB-E2E-02
AB-E2E-03
AB-E2E-04
AB-E2E-05

5/5 PASS
```

Classification:

```text
ANTI_BYPASS=PASS
```

---

# 16. Go HTTP reproduction

The Go HTTP reproduction boundary is:

```text
authorization
+
real upstream forwarding
+
real upstream response
```

The required reproduction completed successfully.

Classification:

```text
GO_HTTP_REPRODUCTION=PASS
```

---

# 17. Rust package qualification

The required Rust static qualification surface contains:

```text
49 exact static items
```

Observed:

```text
48 active tests PASS
1 historical ed25519-v1 witness ignored
```

The ignored witness is historical and contractually understood.

It is not treated as an unexplained skip.

Classification:

```text
RUST_PACKAGE_STATIC_ITEMS=49
RUST_PACKAGE_ACTIVE_PASS=48
RUST_PACKAGE_HISTORICAL_IGNORED=1
RUST_PACKAGE_QUALIFICATION=PASS
```

---

# 18. Rust HTTP reproduction

Rust HTTP is a distinct surface from:

```text
Rust Core
Rust HC2
Rust package qualification
```

Accepted A2 architecture:

```text
AUTHORIZATION_PLUS_LOCAL_RESPONSE
```

The Rust HTTP reproduction does not claim real upstream forwarding.

---

## 18.1 Fresh build

Rust proxy build:

```text
PASS
```

---

## 18.2 Negative fail-closed smoke

Request without required HACP authorization material produced:

```text
HTTP 403
X-HACP-Decision: DENY
```

Classification:

```text
PASS
```

---

## 18.3 Warm-up

```text
100 requests
serial
fresh signed tokens
PASS
```

---

## 18.4 Measured workload

```text
5000 requests
serial
5000/5000 success
```

Observed timing:

```text
mean = 19.9190 ms
p50  = 20.0561 ms
p95  = 24.4273 ms
p99  = 32.182 ms
max  = 88.6014 ms
```

These measurements are workload-specific observations.

---

## 18.5 Rust HTTP repository ownership

The Rust HTTP reproduction procedure and client were admitted into S through signed commit:

```text
3018b6bac9753bb7bc4c090d54de71f8bd5cb2f9
```

Exact admitted paths:

```text
benchmarks/rust-http-reproduction/main.go
docs/verification/rust-http-local-reproduction.md
```

Classification:

```text
RUST_HTTP_RUNTIME=PASS
RUST_HTTP_REPRODUCIBILITY=PASS
RUST_HTTP_DOCUMENTARY_REMEDIATION=PASS
RUST_HTTP_REPOSITORY_OWNERSHIP=PASS
RUST_HTTP=PASS
```

No push was required for the local controlled A2 closure execution.

---

# 19. Timing and Gate D disposition

Recorded observations:

```text
Go serial p99 overhead        = 39.97 ms
PowerShell rotating p99       = 47.26 ms
Go rotating p99 overhead      = 0.96 ms
Rust local-response p99       = 32.182 ms
```

These are not equivalent workloads.

Go HTTP includes real upstream forwarding.

Rust HTTP returns a local authorization result.

Therefore this report does not rank Go versus Rust performance from these measurements.

No current tracked normative Gate D threshold was established for the applicable current workload and measurement boundary.

Classification:

```text
TIMING_EXECUTION=COMPLETE
CURRENT_NORMATIVE_GATE_D_THRESHOLD=NOT ESTABLISHED
GATE_D_APPLICABILITY=PROVEN N/A
PERFORMANCE_SEMANTIC_RED=NO
```

Historical or informational measurements do not create a new normative threshold.

---

# 20. Evidence-manifest correction history

An initial mechanical R9.2 manifest attempt captured a large generated filesystem surface, including build/dependency artifacts.

That result was rejected as an acceptance-evidence inventory.

The final curated R9.2 manifest contains:

```text
91 rows
54 post-baseline rows
```

and explicitly avoids treating generated dependency/build trees as separate acceptance evidence.

This correction changed metadata organization only.

It did not mutate production repositories or test semantics.

---

# 21. Prohibited interpretations and actions

The following were explicitly not permitted during SIDECAR-A2.

## 21.1 Historical execution substitution

Not permitted:

```text
build from G/M
execute historical G/M binaries as current acceptance inputs
use stale historical procedures to bypass a missing current owned procedure
```

## 21.2 Missing evidence converted to PASS

Not permitted:

```text
NOT RUN -> PASS
missing procedure -> PASS
zero tests -> PASS
unknown skip -> PASS
historical observation -> current PASS
```

## 21.3 Production semantic changes without RED

Rule:

```text
NO PRODUCTION CHANGE WITHOUT NORMATIVE BASIS AND PROVEN RED
```

No production semantic RED was established during A2.

## 21.4 Repository redesign

A2 did not authorize structural redesign, API redesign or architecture reshaping.

## 21.5 Rust claim inflation

Not permitted:

```text
Rust Core = 38/38 direct
```

Accepted result:

```text
36/38 direct + 2 adjudicated
```

## 21.6 Core/HC2 aggregation

No artificial universal `93/93` claim was created.

Core and HC2 remain separate normative surfaces.

## 21.7 TypeScript aggregate-count inference

The TypeScript Core result was not inferred from package totals.

It was separately mapped to the exact 38-vector Core surface.

## 21.8 Zero-test provenance PASS

`internal/provenance` having no tests was not treated as runtime provenance PASS.

## 21.9 Authorization/transport conflation

Not permitted:

```text
ALLOW + HTTP 502 -> DENY
```

Authorization and transport results remain distinct.

## 21.10 Rust upstream-forwarding overclaim

Not permitted:

```text
Rust HTTP performs real upstream forwarding
```

Current accepted Rust surface is local-response authorization.

## 21.11 Direct Go-versus-Rust latency ranking

Not permitted because the measured paths are different.

## 21.12 Automatic Gate D failure

A historical number alone cannot establish a current release failure without a current normative workload/measurement binding.

## 21.13 Synthetic current E15 claim

Historical E15 evidence remains historical.

No synthetic current `E15=19/19` claim is created.

## 21.14 Two-process distributed overclaim

Actual evidence:

```text
two independent in-process replica pipelines
```

Not claimed:

```text
two independent OS sidecar processes
```

## 21.15 Historical manifest overwrite

The R9.1 SHA remains the identity of the historical snapshot.

It was not overwritten to represent later evidence.

## 21.16 Broad Git staging

Not permitted:

```text
git add .
git add -A
```

Rust HTTP admission used explicit path staging only.

## 21.17 Automatic push

Commit authorization did not imply push authorization.

Observed:

```text
COMMIT=YES
PUSH=NO
```

## 21.18 Merge/tag/release activation

SIDECAR-A2 did not authorize:

```text
merge to main
tag
release
publish
activation
```

## 21.19 Floating production versions

Production composition must use immutable explicit identities, not `latest`.

## 21.20 A3 used as an A2 blocker

Clean-clone candidate reproduction belongs to SIDECAR-A3.

It is not an unresolved SIDECAR-A2 execution HOLD.

---

# 22. Production semantic assessment

Across the required current execution surfaces, no new production semantic failure was established.

Classification:

```text
PRODUCTION_SEMANTIC_RED=NO
```

No evidence established a new:

```text
protocol regression
security regression
wire incompatibility
required conformance regression
mandatory semantic release blocker
```

---

# 23. HOLD assessment

After Rust Core reconciliation and Rust HTTP ownership admission:

```text
UNRESOLVED_RUNTIME_HOLD=0
UNRESOLVED_PROCEDURE_HOLD=0
```

This is a required SIDECAR-A2 execution/reconciliation exit condition.

---

# 24. A2 / A3 boundary

The next stage is:

```text
SIDECAR-A3 — Clean-clone candidate reproduction
```

A3 is separate from A2.

An unexecuted A3 clean-clone procedure does not keep A2 execution/reconciliation open.

---

# 25. SIDECAR-A2 closure state

At the time this English report is produced:

```text
Matrix A                         = PASS
Matrix B                         = PASS
R9.2 evidence manifest           = PASS
R9.2 results reconciliation      = PASS
RU internal closure report       = PASS / MATERIALIZED
EN runtime/integration report    = THIS DOCUMENT
```

Therefore all required documentary closure artifacts are now present.

The only remaining action is the final A2 cross-check and explicit declaration.

---

# 26. Final classification before explicit declaration

```text
SIDECAR_A2_MATRIX_A=PASS
SIDECAR_A2_MATRIX_B=PASS

SIDECAR_A2_EXECUTION=PASS
SIDECAR_A2_RESULTS_RECONCILIATION=PASS

R9_2_EVIDENCE_MANIFEST=PASS
R9_2_RESULTS_RECONCILIATION=PASS

RU_A2_CLOSURE_REPORT=PASS
EN_A2_CLOSURE_REPORT=PASS

UNRESOLVED_RUNTIME_HOLD=0
UNRESOLVED_PROCEDURE_HOLD=0

PRODUCTION_SEMANTIC_RED=NO

RUST_CORE=
36/38 DIRECT PASS + 2 ADJUDICATED

GO_HTTP=
PASS
AUTHORIZATION_PLUS_REAL_UPSTREAM_FORWARDING

RUST_HTTP=
PASS
AUTHORIZATION_PLUS_LOCAL_RESPONSE
NO UPSTREAM FORWARDING CLAIM

CURRENT_NORMATIVE_GATE_D_THRESHOLD=
NOT ESTABLISHED

GATE_D_APPLICABILITY=
PROVEN N/A

SIDECAR_A2_FULL_ACCEPTANCE=
PENDING FINAL CROSS-CHECK
```

---

# 27. Next action

Perform one final documentary cross-check across:

```text
SIDECAR-A2_CLOSURE_TABLE.md

R9_2_EVIDENCE_FILE_MANIFEST.csv

R9_2_RESULTS_RECONCILIATION_UPDATED.md

SIDECAR_A2_REPRODUCIBILITY_INTERNAL_REPORT_RU_2026-10-15.md

HACP_1_1_1_SIDECAR_A2_RUNTIME_INTEGRATION_REPRODUCIBILITY_REPORT_EN_2026-10-15.md
```

If identities and closure claims remain consistent and no new HOLD or RED is present, the next authorized declaration is:

```text
SIDECAR_A2=PASS
```

No merge, tag, push, release publication or SIDECAR-A3 execution is implied by that declaration.
