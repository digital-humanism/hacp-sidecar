# GitCheck Composition Integration

Status: Candidate engineering record

## Purpose

This document records how the HACP R1.1.1 GitCheck verification assembly was
reconciled with the public `hacp-sidecar` repository and how its verified
composition knowledge is being integrated without importing the assembly
itself.

The objective is reproducibility without repository restructuring,
historical rewriting, or silent semantic drift.

## 1. Reconciliation Boundary

The public repository candidate baseline is:

```text
hacp-sidecar
commit:
1bc10acbe79620a165cee573c5b260cd8639280f
```

The GitCheck assembly contained both:

```text
sidecar/
.assembly/hacp-sidecar-1bc10acbe79620a165cee573c5b260cd8639280f/
```

The first tree was the GitCheck runtime/source copy.

The second tree was the frozen source assembly anchor.

## 2. Canonical Source Identity

A three-way comparison was performed between:

```text
public hacp-sidecar checkout
GitCheck runtime sidecar tree
GitCheck frozen .assembly sidecar tree
```

The public checkout contained 121 tracked files.

No tracked file was missing from either GitCheck sidecar tree.

The GitCheck runtime and frozen trees were byte-for-byte identical across the
tracked surface.

The Windows public working tree initially produced raw SHA-256 differences
for a subset of files because `core.autocrlf=true` caused working-tree line
ending representation to differ from the Linux-origin GitCheck trees.

The comparison was repeated using canonical Git blob normalization.

Final result:

```text
PUBLIC_CANONICAL_MISMATCH_COUNT=0
RUNTIME_CANONICAL_MISMATCH_COUNT=0
FROZEN_CANONICAL_MISMATCH_COUNT=0
```

Therefore:

```text
SIDECAR_SOURCE_DELTA=NONE
```

No production source copy from GitCheck is required.

## 3. GitCheck Assembly Classification

GitCheck is a self-contained release-verification workspace.

Its major surfaces include:

```text
.assembly/
build/
hacp-spec/
manifests/
results/
scripts/
sidecar/
source-archives/
verification/
```

These surfaces have different roles and MUST NOT be treated as a single
repository payload.

### `.assembly/`

Frozen source-composition anchor.

Classification:

```text
VERIFICATION / PROVENANCE
DO_NOT_IMPORT_WHOLESALE
```

### `build/`

Fresh GitCheck-generated build outputs.

Classification:

```text
GENERATED
DO_NOT_IMPORT
```

### `results/`

Fresh GitCheck verification outputs.

Classification:

```text
GENERATED EVIDENCE
DO_NOT_IMPORT
```

### `verification/`

Preserved verification machinery and evidence inputs.

Classification:

```text
EVIDENCE / VERIFICATION INFRASTRUCTURE
DO_NOT_IMPORT_AS_A_TREE
```

Individual immutable evidence identities may be referenced by repository
composition records where required.

### `source-archives/`

Immutable source provenance material.

Classification:

```text
PROVENANCE EVIDENCE
DO_NOT_IMPORT
```

### `scripts/`

GitCheck-only orchestration.

The scripts assume GitCheck-local paths and execution topology such as:

```text
sidecar/
hacp-spec/
verification/
build/
results/
source-archives/
```

Classification:

```text
VALIDATED VERIFICATION RECIPES
NOT REPOSITORY SCRIPTS AS-IS
DO_NOT_COPY_WHOLESALE
```

### `manifests/`

GitCheck composition and orchestration identity records.

The GitCheck manifests are retained as verification-assembly records.

Their release-composition semantics may be adapted into repository-owned
immutable composition metadata.

Classification:

```text
SOURCE OF COMPOSITION KNOWLEDGE
ADAPT SEMANTICS
DO_NOT_COPY_AS-IS
```

## 4. Transfer Decision

The integration boundary is:

```text
TRANSFER_AS_IS=NONE
```

Specifically, the following are not copied into the repository:

```text
GitCheck sidecar source
GitCheck .assembly tree
GitCheck build outputs
GitCheck result outputs
GitCheck verification tree
GitCheck source archives
GitCheck orchestration scripts wholesale
GitCheck manifests wholesale
README_GITCHECK.md
```

The repository change is intentionally additive.

## 5. Repository-Owned Composition Surface

The repository-owned composition surface introduced by this candidate is:

```text
manifests/HACP_1_1_0_COMPOSITION.md
manifests/HACP_1_1_1_COMPOSITION.md
docs/engineering/gitcheck-composition-integration.md
```

The composition manifests are separate artifacts so that historical
composition identities can remain immutable.

A successor composition MUST NOT require rewriting an older published
composition.

## 6. Version-Domain Separation

The following version domains remain independent:

```text
hacp-sidecar repository release
hacp-sidecar repository commit
HACP release composition
behavioral component generation
wire/object version
runner protocol version
evidence identity
verification-assembly identity
```

A value from one domain MUST NOT be silently substituted for a value in
another domain.

## 7. Historical Preservation

The historical `hacp-sidecar` `v0.5.0` release remains unchanged.

The post-`v0.5.0` history contains observable security/runtime behavioral
changes and therefore must not be retroactively described as the same
behavioral state merely because the repository release tag was not advanced.

Historical release artifacts are preserved in their original lifecycle
context.

## 8. Evidence Boundaries

HACP-Core and Enforcement revision-2 HC2 evidence are independent families.

Report separately:

```text
HACP-Core: 38/38
HC2-55:    55/55
```

Never synthesize them into a combined conformance score.

Generated performance measurements likewise do not redefine conformance
identity.

## 9. Candidate Verification Boundary

Materializing these files does not itself establish release readiness.

Before any transfer to the protected public working repository, the Mount
candidate must complete the declared verification cycle applicable to the
resulting composition.

Verification evidence must remain separated by family, including as
applicable:

```text
build
unit/regression tests
sidecar tests
conformance
control-plane verification
external E2E
clean-tree and reproducibility checks
exact candidate diff review
```

## 10. Publication Boundary

The current work is performed first in the isolated integration candidate:

```text
isolated pre-publication integration workspace
```

The protected public development checkout is not modified at this stage.

Only the exact verified candidate delta may later be transferred to the
protected public repository.

Public repository hygiene remains mandatory:

```text
explicit file staging only
no git add .
no git add -A
signed commits
English public commit messages
remote identity verification
clean final worktree
```

## 11. Non-Claims

This integration does not:

```text
change HACP protocol semantics
change wire/object identity
change existing production source
import the GitCheck assembly
rewrite historical release evidence
assign a new hacp-sidecar repository SemVer
declare the HACP 1.1.1 release complete
```

Those decisions remain owned by their respective controlled release stages.

## 12. Current Candidate Result

```text
PUBLIC_BASELINE=1bc10acbe79620a165cee573c5b260cd8639280f
SIDECAR_SOURCE_DELTA=NONE
GITCHECK_IMPORTED_WHOLESALE=NO
REPOSITORY_CHANGE=ADDITIVE_COMPOSITION_METADATA_AND_DOCUMENTATION
HISTORICAL_REWRITE=NO
PROTECTED_PUBLIC_REPOSITORY_MODIFIED=NO
STATUS=CANDIDATE
```
