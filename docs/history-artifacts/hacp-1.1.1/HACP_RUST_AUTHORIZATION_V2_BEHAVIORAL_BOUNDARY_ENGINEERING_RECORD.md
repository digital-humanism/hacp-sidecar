# HACP Rust Authorization-v2 Behavioral Boundary Engineering Record

**Status:** DRAFT — ENGINEERING EVIDENCE\
**Publication status:** UNCOMMITTED / UNPUBLISHED\
**Decision status:** PENDING\
**Release authorization:** NONE\
**Production mutation authorization:** NONE

## 1. Purpose

This record captures a bounded engineering finding discovered during HACP 1.1.1 conformance identity revalidation.

It documents three distinct steps:

1. identification of an observable Rust conformance regression;
2. isolation of the behavioral change responsible for that regression;
3. experimental confirmation that the historical Core-compatible behavior and the active Authorization-v2 successor behavior represent a real behavioral boundary.

This document does **not** decide the final versioning, composition, branch, release, or publication model for that boundary.

In particular, it does not authorize:

- production source mutation;
- modification of canonical vectors;
- reinterpretation of historical conformance evidence;
- replacement of the active Enforcement Revision 2 semantics;
- creation of a new semantic version;
- activation of a successor composition;
- commit, push, merge, tag, or release.

The governing engineering rule remains:

```text
NO PRODUCTION CHANGE WITHOUT NORMATIVE BASIS AND PROVEN RED.
```

---

## 2. Relevant identities

### Current adopted `hacp-spec`

```text
commit:
cf309fc5308863f41a9af6999f0a533449431a78

tree:
ebf6bd6817bf1c2698a7d260b7af081bc9002bff

commit subject:
fix: align hacp-rs authorization-v2 enforcement
```

The commit changed:

```text
hacp-rs/Cargo.lock
hacp-rs/Cargo.toml
hacp-rs/src/evaluate.rs
```

### Immediate predecessor

```text
d796559fe5a53e8419eaf211da6102a68ce54b4a
```

Relevant `evaluate.rs` Git blob:

```text
b3f54257169f9442e47c283be8819a44b9ff71ee
```

### Current `evaluate.rs`

Relevant Git blob:

```text
5a7ed545098394bcd76074757ab9d94e7f6f9423
```

### Preserved GitCheck evaluator

Preserved source:

```text
hacp-spec/hacp-rs/src/evaluate.rs
```

SHA-256:

```text
d1a1e009e398ecc459b4f6d1a90170cea77b2769dfa3aa0bacd8206dbc4379d0
```

Read-only provenance verification established:

```text
GITCHECK_EQUALS_PARENT_TEXT=True
GITCHECK_EQUALS_CURRENT_TEXT=False
PARENT_EQUALS_CURRENT_TEXT=False
```

Therefore, the preserved GitCheck evaluator corresponds to the predecessor behavior, not to the current `cf309fc` evaluator behavior.

---

## 3. Historical verified behavior

The preserved HACP-Rust GitCheck surface had already established an independently reproducible Rust conformance result.

The preserved fresh runner identity was:

```text
SHA256:
10535e269db96ff534ff5546e1d35dde1e4e48cb7bcf21cbb0f0df88b0422296
```

Its verified conformance result was:

```text
Core Family A = 38/38 PASS
Core Family B = 38/38 PASS

HC2 Family A = 55/55 PASS
HC2 Family B = 55/55 PASS
```

Core provenance was explicitly:

```text
37 normal Core corpus cases
+
corrected raw-wire CORE-INV5-006
=
Core-38
```

The corresponding verification did not modify canonical vectors and did not redefine normative HACP semantics.

At the same engineering point, Authorization-v2 remained a separate successor boundary:

```text
authorization-v2
ACTIVATED
NOT VERIFIED
NOT PROD-ELIGIBLE
```

Known Authorization-v2 RED witnesses included:

```text
action-hash mismatch reason
invalid intent-envelope signature precedence
authenticated applicable signed DENY token behavior
```

This establishes the historical baseline:

```text
Core-compatible Rust behavior = VERIFIED

Authorization-v2 successor behavior = SEPARATE RED / HOLD BOUNDARY
```

---

# 4. Moment One — The Problem

During HACP 1.1.1 conformance identity revalidation, the current adopted Rust evaluator at:

```text
cf309fc5308863f41a9af6999f0a533449431a78
```

did not preserve the previously verified Core behavior.

The current evaluator produced decision-level Core failures including:

```text
CORE-INV1-002
expected: CHECKPOINT
actual:   DENY / SIGNATURE_FAILURE

CORE-RUNTIME-004
expected: CHECKPOINT
actual:   DENY / SIGNATURE_FAILURE
```

The same current surface continued to satisfy HC2-55.

This created a concrete engineering RED:

```text
previously verified Rust Core-38 behavior
was not preserved by the current adopted evaluator.
```

The failure was initially classified conservatively.

Possible causes considered included:

```text
vector defect
harness defect
fixture defect
environment/tooling issue
historical provenance mismatch
implementation behavioral regression
```

No production mutation was authorized during that investigation.

Further evidence eliminated the vector, harness, environment, and historical-provenance explanations.

The problem therefore became:

```text
CURRENT ADOPTED RUST EVALUATOR
DOES NOT PRESERVE
PREVIOUSLY VERIFIED CORE-38 BEHAVIOR.
```

---

# 5. Moment Two — Isolation of the Behavioral Cause

A read-only provenance bridge compared:

```text
A. preserved GitCheck evaluate.rs

B. predecessor:
   d796559fe5a53e8419eaf211da6102a68ce54b4a

C. current:
   cf309fc5308863f41a9af6999f0a533449431a78
```

The result was exact:

```text
GitCheck == predecessor behavior
GitCheck != current behavior
predecessor != current behavior
```

The critical behavioral ordering in the GitCheck/predecessor evaluator was:

```text
Step 1: Checkpoint pre-evaluation
...
Step 1b: Checkpoint state
...
Step 2: Clock and envelope expiry
...
later authorization / token verification paths
```

The current `cf309fc` evaluator introduced a new block before Core checkpoint processing:

```text
Authorization-v2 envelope trust and authentication

- envelope key revocation
- unsupported HMAC signer rejection
- trusted-key boundary
- envelope canonicalization
- envelope signature verification
- envelope revocation

then:

Step 1: Checkpoint pre-evaluation
```

This changed observable precedence.

A request which previously reached the historical checkpoint decision could now terminate earlier with an Authorization-v2 authentication failure.

The same commit also included other Authorization-v2 behavioral corrections, including:

```text
token action_hash mismatch:
HASH_MISMATCH
→
SIGNATURE_FAILURE

authenticated applicable DecisionToken DENY:
→
POLICY_DENIED
```

The causal implementation delta was therefore isolated to the Authorization-v2 alignment performed by `cf309fc`.

At this point the engineering finding became:

```text
cf309fc introduced observable Authorization-v2 behavior
across a previously verified Core evaluation boundary.
```

However, provenance alone did not yet establish the correct remediation.

A counterfactual execution test was therefore required.

---

# 6. Candidate Remedy

The candidate remedy was intentionally narrower than a source fix.

It did **not** propose:

```text
reverting Authorization-v2 normative semantics
changing canonical vectors
weakening active Enforcement Revision 2
editing historical evidence
silently reverting cf309fc in production
```

Instead, the hypothesis was:

```text
The historical Core-compatible evaluator behavior
and the Authorization-v2 successor behavior
must not be represented as one unconditional behavioral identity.
```

To test that hypothesis without modifying public source, an ephemeral derived candidate was created.

Its composition was:

```text
base source:
cf309fc5308863f41a9af6999f0a533449431a78

only derived override:
hacp-rs/src/evaluate.rs

override identity:
exact preserved GitCheck / predecessor evaluator

SHA256:
d1a1e009e398ecc459b4f6d1a90170cea77b2769dfa3aa0bacd8206dbc4379d0
```

No public source file was modified.

No historical source file was modified.

No commit or publication action occurred.

The experiment asked two separate questions:

```text
1. Does restoring the verified evaluator boundary restore Core-38
   while preserving HC2-55?

2. Does the unresolved Authorization-v2 successor behavior then
   reappear as a separate RED boundary?
```

Both conditions were necessary for the hypothesis to be supported.

---

# 7. Moment Three — Experimental Confirmation

## 7.1 Reproduction environment

The candidate was freshly built using the pinned builder identity:

```text
image:
rust@sha256:c49256cbe5ea0188bc658a689500d70c41eb51f009a7a7be209caf60a944f3ec

rustc:
1.98.1 (48a229cea 2026-09-01)

cargo:
1.98.1 (797e8a9bc 2026-08-05)

protoc:
libprotoc 35.1
```

Preserved `protoc` SHA-256:

```text
d2c65c3ea5eeb59427f684ed3e0a0cd458386122fc9f1b98de946a7242b53d31
```

Fresh build result:

```text
BUILD_RC=0
```

Derived candidate runner SHA-256:

```text
8430e911af6514cecd28a943bbdc54ff709b9e4623583b72755667b8dc8ce54f
```

The candidate runner identity is evidence for this experiment only and is not a release identity.

---

## 7.2 Core and HC2 result

The normal Core corpus produced:

```text
CORE_NORMAL_COUNT=37

Family A:
37/37 PASS

Family B:
37/37 PASS

Infrastructure errors:
0
```

HC2-55 produced:

```text
HC2_COUNT=55

Family A:
55/55 PASS

Family B:
55/55 PASS

Infrastructure errors:
0
```

The raw-wire duplicate-member witness retained its exact request identity:

```text
CORE-INV5-006 raw request SHA256:

8d95b2d935f24b74e26ad9e81200d87b2cbbc092713ba78ac1ead3c011888fe4
```

Observed result:

```json
{
  "decision": "DENY",
  "reason_codes": ["INVALID_ACTION"]
}
```

Therefore:

```text
Core Family A = 38/38 PASS
Core Family B = 38/38 PASS

HC2 Family A = 55/55 PASS
HC2 Family B = 55/55 PASS
```

The historical Core-compatible behavior was fully restored.

---

## 7.3 Authorization-v2 result

The same derived candidate was then tested against the Authorization-v2 witness surface.

The witness:

```text
authorization_v2_action_hash_mismatch_uses_current_enforcement_reason
```

failed as expected.

Observed result:

```text
actual:
HASH_MISMATCH

active Enforcement Revision 2 expectation:
SIGNATURE_FAILURE
```

Test result:

```text
AUTHORIZATION_V2_TEST_RC=101
AUTHORIZATION_V2_CANDIDATE_STATUS=RED
```

Therefore the experiment produced the exact boundary condition under investigation:

```text
CORE_GREEN
AUTHORIZATION_V2_RED
```

---

# 8. Engineering Finding

The experiment establishes all of the following simultaneously.

### Finding A — historical behavior remains reproducible

The previously verified Core-compatible Rust behavior is not lost or ambiguous.

It is reproducible from preserved source and continues to satisfy:

```text
Core-38 = 38/38
HC2-55  = 55/55
```

### Finding B — current Authorization-v2 semantics are real behavior

The Authorization-v2 changes are not merely refactoring, naming, documentation, or implementation cleanup.

They alter observable security/protocol outcomes.

For example:

```text
HASH_MISMATCH
vs
SIGNATURE_FAILURE
```

and:

```text
CHECKPOINT
vs
early DENY / SIGNATURE_FAILURE
```

are externally observable behavioral differences.

### Finding C — the two requirements cannot be represented as the same unconditional historical behavior

Restoring historical Core behavior causes the known Authorization-v2 RED to reappear.

Applying the current Authorization-v2 ordering unconditionally causes historical Core conformance regression.

Therefore:

```text
HISTORICAL CORE-COMPATIBLE BEHAVIOR
and
AUTHORIZATION-V2 SUCCESSOR BEHAVIOR

form a real behavioral boundary.
```

This is not a vector-repair problem.

It is not a harness-reinterpretation problem.

It is not an environment problem.

It is not resolved by silently rewriting historical conformance expectations.

---

# 9. What This Record Does Not Decide

This record intentionally stops before architectural adjudication.

It does not yet decide:

```text
whether hacp-rs 0.1.1 is the correct successor identity;

whether a new hacp-rs behavioral version is required;

whether the behavioral distinction belongs to
component versioning,
profile versioning,
composition versioning,
or an explicit evaluator dispatch boundary;

whether cf309fc should remain on the current public line;

whether the engineering work should continue on
a separate public branch,
a private engineering branch,
or another controlled integration surface;

what exact production source mutation is permitted;

what exact successor composition becomes release-eligible.
```

Those are subsequent versioning and architecture decisions.

They require separate explicit authorization.

---

# 10. Required Preservation Properties

Any future solution must preserve the following evidence.

## Historical identity preservation

The verified historical/Core-compatible behavior must remain reproducible.

It must not be retroactively reinterpreted as Authorization-v2 behavior.

## Explicit successor identity

If Authorization-v2 behavior remains observably different, its activation must be explicit and tied to an identifiable immutable behavioral composition.

It must not silently replace a historical behavior under an unchanged identity.

## Immutable composition

Production composition must pin exact versions or identities.

It must not depend on:

```text
latest
current
default
main
implicit behavior selection
```

## Separate conformance families

Core-38 and HC2-55 remain separate evidence families.

They must not be combined into a synthetic score.

## No vector repair to conceal implementation divergence

Canonical vectors must not be changed merely to make the current implementation appear conformant.

---

# 11. Current Classification

## FACT

```text
Historical GitCheck Rust Core:
38/38 PASS

Historical GitCheck Rust HC2:
55/55 PASS

GitCheck evaluator:
matches d796559 predecessor behavior

Current cf309fc evaluator:
different from GitCheck/predecessor behavior

cf309fc:
moves Authorization-v2 envelope authentication
before Core checkpoint pre-evaluation

Current observable Core regression:
established

Ephemeral candidate:
cf309fc base + exact GitCheck/predecessor evaluator

Candidate Core:
38/38 PASS

Candidate HC2:
55/55 PASS

Candidate Authorization-v2:
RED

Public source mutation:
NO

Historical source mutation:
NO

Commit:
NO

Push:
NO

Release:
NO
```

## INFERENCE

```text
The current Rust evaluator combines behavioral requirements
that belong to distinct behavioral generations or compositions.

Historical/Core-compatible semantics and Authorization-v2 successor
semantics require an explicit version/composition boundary rather
than an implicit unconditional replacement.
```

## OPEN QUESTIONS

```text
What exact immutable identity owns historical Core-compatible behavior?

What exact immutable identity owns Authorization-v2 behavior?

Is hacp-rs 0.1.1 already the intended successor behavioral identity,
or were two behavioral identities unintentionally combined within it?

Where should behavior selection live:
component,
profile,
composition,
or another explicit versioned boundary?

What normative artifact authorizes the successor activation?

What is the exact allowed-files set for implementation?

What branch/publication model should carry the engineering work?
```

## DECISION

```text
PROBLEM:
PROVEN

CAUSAL IMPLEMENTATION BOUNDARY:
PROVEN

CANDIDATE REMEDY:
EXPERIMENTALLY CONFIRMED

BEHAVIORAL BOUNDARY:
PROVEN

FINAL VERSIONING / COMPOSITION DESIGN:
NOT YET DECIDED

PRODUCTION SOURCE MUTATION:
NOT AUTHORIZED

COMMIT:
NOT AUTHORIZED

PUSH:
NOT AUTHORIZED

PUBLICATION STATUS:
PENDING
```

---

# 12. Next Step

The next controlled engineering step is documentary and read-only:

```text
Authorization-v2 Behavioral Identity
and Composition Reconciliation
```

Its purpose is to determine the correct immutable identities and composition model before any production source mutation.

This record should remain evidence, not become an implementation authorization by implication.

---

**End of engineering record.**
