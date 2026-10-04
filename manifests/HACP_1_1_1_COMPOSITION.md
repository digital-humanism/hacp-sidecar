# HACP 1.1.1 Composition

Status: DRAFT — RELEASE-CANDIDATE COMPOSITION

## Purpose

This document records the repository-owned HACP 1.1.1 composition binding
relevant to `hacp-sidecar`.

It is derived from the verified GitCheck assembly while preserving a strict
boundary between repository source and release-verification infrastructure.

The GitCheck assembly itself is not imported into this repository.

## Version-Domain Separation

The following identities are independent:

```text
HACP release composition
hacp-sidecar repository release
hacp-sidecar repository commit
behavioral component identity
wire/object identity
runner protocol identity
evidence identity
verification-assembly identity
```

They MUST NOT be conflated.

## Sidecar Implementation Identity

```text
HACP_RELEASE=1.1.1
SIDECAR_IMPLEMENTATION_COMMIT=1bc10acbe79620a165cee573c5b260cd8639280f
```

The canonical source identity at this commit was reconciled across:

```text
public hacp-sidecar checkout
GitCheck runtime sidecar tree
GitCheck frozen .assembly sidecar tree
```

All tracked files matched the expected Git blobs after Git normalization.

Raw filesystem differences observed on Windows were attributable to working
tree line-ending representation and did not constitute repository source
delta.

## GitCheck Source Provenance

GitCheck preserved an immutable source archive for the exact sidecar commit:

```text
FILE=hacp-sidecar-1bc10acbe79620a165cee573c5b260cd8639280f.tar.gz
SHA256=3418cbff1e8fefcfa423f6318d213906126002821f3f4d0414d0a91bc88c7a1f
```

The archive is verification/provenance evidence and is not repository payload.

## GitCheck Composition Boundary

The GitCheck workspace contains release-verification infrastructure including:

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

Those directories belong to the GitCheck assembly topology.

They are not imported wholesale into `hacp-sidecar`.

## Repository-Owned Delta

The repository-owned integration is intentionally additive and limited to
composition metadata and public engineering documentation.

No production source file is copied from the GitCheck sidecar tree because
the canonical repository source identity is already identical.

## Behavioral Component Binding

The current reconciliation established multiple post-`v0.5.0` behavioral
boundaries in the sidecar history, including trust lifecycle, distributed
control runtime, request binding, and scope/reason semantics.

Exact component-family identifiers and final behavioral-version bindings are
owned by the authoritative Build / Version Matrix.

This draft therefore does not invent missing final component identities.

## Evidence Families

Evidence families remain separate.

```text
HACP-Core:
  38 vectors

HACP Enforcement revision 2:
  55 HC2 vectors
```

These counts MUST NOT be summed into a synthetic conformance score.

## GitCheck Verification Role

GitCheck is a verification assembly, not a source repository overlay.

Its orchestration scripts, generated build outputs, generated results,
preserved verification inputs, and source archives remain evidence or
verification infrastructure unless separately adopted through an explicit
repository-owned contract.

## Floating Reference Rule

No production composition may use:

```text
latest
current
main
default branch
other floating references
```

Every production composition binding must resolve to an explicit immutable
identity.

## Current Candidate State

```text
HACP_RELEASE=1.1.1
SIDECAR_IMPLEMENTATION_COMMIT=1bc10acbe79620a165cee573c5b260cd8639280f
SOURCE_DELTA_FROM_GITCHECK=NONE
GITCHECK_IMPORTED_WHOLESALE=NO
PRODUCTION_SOURCE_CHANGE=NO
COMPOSITION_METADATA_CHANGE=ADDITIVE
STATUS=DRAFT
```

Final publication eligibility remains subject to the complete candidate
verification cycle and final release decision.
