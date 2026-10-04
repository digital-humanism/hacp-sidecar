# HACP Go Verification Status Report

**Date:** 2026-09-29
**Repository:** `hacp-sidecar`
**Verification scope:** isolated pre-publication candidate verification
**Status:** CLOSED — PASS
**Publication:** NOT AUTHORIZED

---

## 1. Purpose

This report records the current status of the controlled Go-sidecar verification performed against the isolated `hacp-sidecar` integration candidate before any protected public-repository update.

The verification is intentionally evidence-first and preserves the following project rules:

- no production change without normative basis and proven RED;
- historical identities are immutable and must remain reproducible;
- behavior-changing security/protocol changes require explicit versioned identity;
- production compositions must use explicit immutable pins and must not depend on floating identities;
- evidence families remain separate and must not be collapsed into a synthetic total;
- no public push is permitted from the isolated integration workspace;
- explicit staging only; no `git add .` or `git add -A`;
- protected public repositories remain unchanged until the candidate has completed its verification chain.

---

## 2. Candidate identity

The Go verification candidate is based on the public `hacp-sidecar` source identity:

```text
branch: main
HEAD: 1bc10acbe79620a165cee573c5b260cd8639280f
```

The isolated candidate additionally contains exactly three staged, additive documentation/composition files:

```text
docs/engineering/gitcheck-composition-integration.md
manifests/HACP_1_1_0_COMPOSITION.md
manifests/HACP_1_1_1_COMPOSITION.md
```

The resulting staged candidate tree is:

```text
f6bb41511cbc96897c0db7cf6ecbaf9001bba73c
```

At every completed verification checkpoint recorded below, the candidate remained:

```text
staged files:   3
unstaged files: 0
source mutation: NONE
commit:          NONE
push:            NONE
```

The protected public working repository has not been modified.

---

## 3. Source reconciliation result

A three-way source comparison was completed between:

1. the public `hacp-sidecar` source at `1bc10ac...`;
2. the GitCheck runtime sidecar tree;
3. the GitCheck frozen assembly sidecar tree.

Raw Windows filesystem comparison initially exposed 36 differing paths. Canonical Git-object comparison subsequently established:

```text
PUBLIC_CANONICAL_MISMATCH_COUNT=0
RUNTIME_CANONICAL_MISMATCH_COUNT=0
FROZEN_CANONICAL_MISMATCH_COUNT=0
```

The raw differences were line-ending / working-tree representation differences, not repository source delta.

Therefore:

```text
SIDECAR_SOURCE_DELTA_FROM_GITCHECK=NONE
```

The repository-owned candidate delta is limited to additive composition metadata and public engineering documentation. GitCheck build/results/verification infrastructure is not imported wholesale.

---

## 4. Verification stages completed

### V1 — Full Go build

**Status:** CLOSED / PASS

Verified binaries:

```text
cmd/sidecar
cmd/hacp-conformance-runner
```

Build command class:

```text
go build -trimpath
```

Observed sidecar binary SHA-256:

```text
2d4ff76ed1cb4cd4b11a3258803ad2d1dbe713cd8438938808ab869b29fcb7a7
```

No candidate mutation occurred.

---

### V2 — Full Go regression

**Status:** CLOSED / PASS

Initial execution of:

```text
go test ./... -count=1
```

encountered an environment/layout dependency because the sidecar control-plane tests expect the sibling `hacp-spec` vector tree.

This was classified as an environment HOLD, not a product RED.

A Windows junction was used to expose the already-preserved local GitCheck `hacp-spec` tree at the expected sibling location. No source copy or source modification was performed.

The full regression was then rerun and passed.

Result:

```text
FULL_GO_REGRESSION=PASS
```

Candidate identity remained unchanged.

---

### V3 — Go vet

**Status:** CLOSED / PASS

Executed:

```text
go vet ./...
```

Result:

```text
exit=0
```

No source or staging mutation occurred.

---

### V4 — Gate E / control-plane verification

**Status:** CLOSED / PASS

Executed the complete `internal/controlplane` test surface with fresh execution.

The passing evidence includes the declared control-plane behavior families, including:

- snapshot and live-event handling;
- reconnect and replay behavior;
- revision-gap fail-closed handling;
- reset-required recovery;
- distributed token revocation propagation;
- control-state freshness / unsafe / recovery behavior;
- subscriber initial-snapshot retry behavior;
- multi-sidecar convergence;
- pipeline fail-closed behavior under stale control state;
- supported revocation-kind handling.

Result:

```text
GATE_E_CONTROLPLANE=PASS
```

This evidence confirms the implemented control-plane runtime surface; it does not transfer normative ownership to the sidecar.

---

### V5 — Conformance / runtime evidence reconciliation

**Status:** CLOSED / PASS

V5 reconciled the Go-sidecar runtime evidence against the preserved E15/E16 evidence chain rather than reinterpreting unrelated language-conformance evidence as sidecar runtime evidence.

The reconciliation established:

```text
SIDECAR_IMPLEMENTATION_COMMIT=1bc10acbe79620a165cee573c5b260cd8639280f
E15_ENFORCEMENT_CONFIRMATION=CLOSED / PASS
E15_BOUNDED_ENFORCEMENT_MATRIX=19/19 PASS
E15_HTTP_PROCESS_E2E=PASS
E16_CONTROL_PLANE_CONFIRMATION=CLOSED / PASS
```

Important evidence-family separation:

```text
HC2_55_LANGUAGE_CONFORMANCE=SEPARATE EVIDENCE FAMILY
SIDECAR_HC2_55_CLAIM=NOT MADE
LEGACY_CORE38_SIDECAR_GATE=HISTORICAL / DISABLED
```

The preserved GitCheck master contract explicitly treats the legacy Core-38 sidecar gate as historical and disabled for the current release gate. A historical partial result must not be reinterpreted as current Core-38 sidecar conformance.

No source mutation occurred.

---

### V6 — Anti-bypass / deployment verification

**Status:** CLOSED / PASS

The original repository anti-bypass deployment test was executed using the available Docker Desktop backend after environment readiness was established.

All five anti-bypass checks passed:

```text
AB-E2E-01 agent can reach sidecar
AB-E2E-02 agent cannot directly reach protected upstream
AB-E2E-03 upstream has no host port bindings
AB-E2E-04 sidecar can reach protected upstream
AB-E2E-05 network membership preserves boundary

Result: 5 passed, 0 failed
```

Cleanup also completed successfully: containers, volume, and test networks were removed.

No candidate mutation occurred.

Observation for later engineering hygiene only: the Docker build context observed during the test was unexpectedly large. This was not a verification failure and did not authorize any repository change.

---

## 5. V7 — External Python ↔ Go E2E preparation

### V7.1 — External E2E contract capture

**Status:** CLOSED / PASS

The current external E2E contract was captured read-only.

The external integration suite is owned by the frozen `humanist-core` checkout at:

```text
HEAD=6d9ae82ed7fefe47b395e4ef68d0a18807866473
```

The exact test target is:

```text
tests/test_hacp_sidecar_integration.py
```

The execution contract requires a running sidecar plus a local test signing identity, including:

```text
HACP_SIDECAR_EXTERNAL=1
HACP_SIDECAR_URL=http://127.0.0.1:8080
HACP_TEST_PRIVATE_KEY=<matching-private-key.pem>
HACP_TEST_SIGNER_KEY_ID=<matching-key-id>
```

The native sidecar startup path uses explicit test trust mode and standalone control-plane defaults for this external E2E path.

No process was started and no pytest execution occurred during contract capture.

---

### V7.2 — External E2E execution preflight

**Status:** PARTIAL PASS

Confirmed:

```text
humanist-core HEAD: 6d9ae82ed7fefe47b395e4ef68d0a18807866473
humanist-core working tree: clean
Python: 3.13.14
pytest: 9.1.1
required Python imports: PASS
port 8000: FREE
port 8080: FREE
sidecar verification binary: present
sidecar verification binary SHA-256: 2d4ff76e...
```

Three local private-key PEM candidates were found. Their contents were not printed. All three were byte-identical locally.

No external E2E was started at this step.

---

## 6. Test-key identity divergence observation

### 6.1 What was observed

Before the private-key comparison step, verification compared multiple local copies of the file named:

```text
test-ed25519-001.pub
```

Two distinct byte-level file identities were observed across repository generations/workspaces.

The check stopped immediately before any private-key identity comparison or external E2E execution.

This was initially treated as a generic canonical-copy divergence. Subsequent inventory established that this interpretation was too strict for the project's versioned-identity architecture.

### 6.2 Correct classification

The condition is classified as:

```text
EXPECTED VERSIONED IDENTITY DIVERGENCE
NOT A GO IMPLEMENTATION DEFECT
NOT A HACP PROTOCOL DEFECT
NOT A SOURCE REGRESSION
```

The project intentionally preserves historical identities instead of silently rewriting them.

A logical signer identifier alone must not be treated as sufficient authority to collapse distinct cryptographic identities across historical/successor compositions.

### 6.3 Historical key identity

The public `hacp-spec` baseline records the historical deterministic conformance identity:

```text
key id:
key-ed25519-test-001

public key:
9d17f1bbcc0845865e670f526413fb7a510380798fe300b6c98e28f3a3b0fdb3

derivation:
SHA-256(b"hacp-conformance-v0.9-key-001")
```

That identity is referenced by the historical Core vectors, HC2 vectors, conformance harness, vector-baking tooling, and historical release/conformance evidence.

Accordingly, the historical `001` identity is immutable and must remain reproducible.

### 6.4 Security meaning

The observed divergence is an identity observation, not an authorization result.

No HACP evaluation was executed against a deliberately mismatched key during this capture.

Therefore the report records:

```text
OBSERVED_RESULT=IDENTITY_DIVERGENCE_DETECTED
EXPECTED_SECURITY_BEHAVIOR_IF_EVALUATED=FAIL_CLOSED / DENY
ACTUAL_DENY_EVIDENCE=NOT YET EXECUTED IN THIS V7 SUBSTAGE
```

The fail-closed mismatch behavior must be verified explicitly as a separate successor trust-boundary test and must not be claimed solely from this observation.

---

## 7. V7A — Test-key successor decision

### 7.1 Read-only inventory

**Status:** CLOSED / PASS

A cross-repository inventory was performed across:

- public `hacp-spec` baseline;
- isolated `hacp-sidecar` candidate;
- frozen `humanist-core` baseline;
- Rust laboratory `hacp-spec` workspace.

The inventory confirmed that `key-ed25519-test-001` currently appears in several semantically different categories:

1. immutable historical conformance vectors and evidence;
2. compatibility/test trust paths in the Go sidecar;
3. environment-driven external E2E configuration;
4. Rust correspondence / historical-fixture tests;
5. documentation and release records.

A mechanical global replacement would therefore destroy historical reproducibility and is forbidden.

### 7.2 Adopted direction

The intended trust transition is additive:

```text
key-ed25519-test-001
    historical / immutable / compatibility identity

key-ed25519-test-002
    successor candidate identity
```

The following rules apply:

```text
001 IS NOT RENAMED
001 IS NOT OVERWRITTEN
001 IS NOT REBAKED OUT OF HISTORICAL VECTORS
001 REMAINS REPRODUCIBLE

002 IS ADDITIVE
002 IS A NEW SUCCESSOR TRUST IDENTITY
002 REQUIRES EXPLICIT VERSIONED BINDING
```

This is security-visible behavior and therefore requires an explicit accepted implementation plan before any production/repository implementation change.

### 7.3 Required successor negative boundary

The successor verification plan must prove both positive and negative trust boundaries:

```text
001-signed artifact against 001 historical profile
    -> historical expected result

002-signed artifact against 002 successor profile
    -> successor expected result

001-signed artifact against 002-only trust
    -> DENY / fail closed

002-signed artifact against 001-only trust
    -> DENY / fail closed
```

These results must be executed and observed; they must not be inferred from key-file divergence alone.

---

## 8. What remains

The Go-sidecar verification is not yet complete.

### V7A implementation planning

Required next step:

```text
V7A — Test-Key Successor Implementation Plan
```

The plan must define at minimum:

- exact successor key identity (`key-ed25519-test-002` candidate);
- exact derivation/generation procedure;
- exact public-key representation;
- exact allowed files;
- exact repositories/workspaces permitted for initial implementation;
- historical surfaces explicitly forbidden from modification;
- trust-profile/component-version impact;
- positive and negative verification matrix;
- STOP conditions;
- rollback / non-publication boundary;
- evidence artifact names.

No implementation is authorized until that plan is accepted.

### V7 external E2E execution

After the successor trust identity is implemented in the isolated/laboratory surfaces and the intended E2E binding is explicit, execute a fresh external Python ↔ Go E2E run.

Expected evidence family remains separate:

```text
Python ↔ Go external E2E: expected 5/5
```

The actual result must be recorded only after execution.

### V8 — Documentary/composition assertions

Still required.

V8 must audit all candidate composition claims against established evidence and catch unresolved publication blockers.

Known blocker to resolve before publication:

```text
HACP_1_1_0_COMPOSITION.md
SIDECAR_IMPLEMENTATION_IDENTITY=UNRESOLVED
COMPOSITION_BINDING=HOLD
PUBLICATION_ELIGIBLE=NO
```

Either the exact historical HACP 1.1.0 sidecar binding must be established, or the document must remain explicitly non-publication payload. This cannot be silently ignored.

### V9 — Final candidate identity / exact diff

Still required after all mandatory verification surfaces close.

Must capture:

- final candidate tree;
- exact staged file list;
- exact diff;
- `git diff --check`;
- candidate artifact hashes;
- no unstaged changes;
- no unintended source changes.

### Protected public repository transfer

Not yet authorized.

Only after the isolated candidate is fully verified may the exact verified delta be applied to the protected public `hacp-sidecar` repository, followed by:

- exact diff review;
- explicit staging;
- signed commit;
- push;
- remote identity verification;
- clean working-tree verification.

---

## 9. Current verification matrix

| Verification stage | Status | Result |
|---|---|---|
| Source reconciliation | CLOSED | PASS |
| Candidate diff review | CLOSED | PASS |
| Verification preflight | CLOSED | PASS |
| V1 full Go build | CLOSED | PASS |
| V2 full Go regression | CLOSED | PASS |
| V3 `go vet` | CLOSED | PASS |
| V4 Gate E / control-plane | CLOSED | PASS |
| V5 runtime/conformance evidence reconciliation | CLOSED | PASS |
| V6 anti-bypass deployment verification | CLOSED | PASS |
| V7 external E2E contract capture | CLOSED | PASS |
| V7 execution preflight | CLOSED | PASS; successor identity resolved through V7A |
| V7A successor-key read-only inventory | CLOSED | PASS |
| V7A successor-key implementation plan | CLOSED | PASS |
| V7 successor trust implementation | CLOSED | PASS — additive runtime trust binding |
| V7 successor positive/negative trust tests | CLOSED | PASS |
| V7 fresh Python ↔ Go external E2E | CLOSED | 5/5 PASS |
| V8 documentary/composition assertions | CLOSED | PASS; HACP 1.1.0 HOLD preserved |
| V9 final identity / exact diff | CLOSED | PASS |
| protected public-repository transfer | NOT AUTHORIZED | — |

No synthetic total is reported across these evidence families.

---

## 10. Current decision

```text
GO_VERIFICATION=CLOSED / PASS
CURRENT_BLOCK=V9 FINAL EVIDENCE RECONCILIATION
GO_SOURCE_REGRESSION=NOT ESTABLISHED
GO_IMPLEMENTATION_DEFECT_FROM_KEY_DIVERGENCE=NOT ESTABLISHED
HISTORICAL_REWRITE=FORBIDDEN
SUCCESSOR_KEY_IDENTITY=PROVEN
SUCCESSOR_RUNTIME_BINDING=PROVEN
PYTHON_GO_EXTERNAL_E2E=5/5 PASS
HACP_1_1_0_COMPOSITION_BINDING=HOLD
HACP_1_1_1_IMPLEMENTATION_BINDING=RESOLVED
PUBLIC_TRANSFER=NOT YET AUTHORIZED
PUSH=NOT AUTHORIZED
```

The isolated Go-sidecar verification cycle is complete. No Go production defect was established. The successor trust identity and cross-version boundary were proven, fresh Python ↔ Go external E2E completed 5/5 PASS, HACP 1.1.0 remains explicitly on historical composition HOLD, and the HACP 1.1.1 implementation binding is resolved. The remaining action is the controlled transfer decision for the exact verified documentation/composition payload.

---

## 11. Required carry-forward note

The key-divergence observation discovered during V7 must remain part of the final Go verification record.

It demonstrates why versioned trust identity must be explicit and why historical cryptographic artifacts must not be silently normalized across generations.

The final report must preserve the distinction between:

```text
identity divergence observed
```

and:

```text
mismatched-key DENY behavior actually executed and proven
```

The latter remains future evidence until the successor trust-boundary test is run.

---

**END OF REPORT**



---

## V7A Successor Identity Runtime Verification

```text
successor identity:
key-ed25519-test-002

runtime binding:
PASS

fresh Python <-> Go external E2E:
5/5 PASS

cross-version fail-closed:
001 -> 002-only trust = DENY / HTTP 403 / SIGNATURE_FAILURE
002 -> 001-only trust = DENY / HTTP 403 / SIGNATURE_FAILURE

production Go source change:
NONE
```

Classification:

```text
EXPECTED VERSIONED IDENTITY SEPARATION
ADDITIVE SUCCESSOR IDENTITY
HISTORICAL 001 PRESERVED
FAIL-CLOSED CROSS-VERSION BOUNDARY PROVEN
```
