# HACP 1.1.0 Composition

Status: DRAFT — HISTORICAL COMPOSITION RECONSTRUCTION

## Purpose

This document is the repository-owned composition record for the HACP 1.1.0
release line as represented by `hacp-sidecar`.

It is intentionally separate from the `hacp-sidecar` repository release
version, package versions, wire/object versions, behavioral component
versions, and evidence identities.

This file does not rewrite historical release artifacts.

## Historical Preservation Rule

Historical identities are immutable.

A later composition record MUST NOT retroactively reinterpret, rename,
overwrite, or silently repair the HACP 1.1.0 composition.

Any successor composition is recorded in a separate immutable artifact.

## Repository Version Domain

The historical `hacp-sidecar` repository release `v0.5.0` remains a separate
repository-release identity.

HACP release identities and `hacp-sidecar` repository-release identities
belong to different version domains and MUST NOT be conflated.

## Exact Composition Binding

The exact immutable `hacp-sidecar` commit binding for this HACP 1.1.0
composition has not yet been established by the current E18-E Slice 6
repository reconciliation evidence.

Therefore:

```text
HACP_RELEASE=1.1.0
SIDECAR_IMPLEMENTATION_IDENTITY=UNRESOLVED
COMPOSITION_BINDING=HOLD
PUBLICATION_ELIGIBLE=NO
```

No floating reference such as `main`, `latest`, or `current` may replace the
missing immutable identity.

## Behavioral Component Binding

The exact behavioral-component generation mapping for HACP 1.1.0 remains to
be bound by the authoritative Build / Version Matrix.

No component generation is inferred by this document.

## Evidence Binding

Historical evidence remains valid only in the lifecycle context in which it
was produced.

Independent evidence families MUST remain separate.

In particular, HACP-Core and Enforcement revision-2 evidence MUST NOT be
combined into a synthetic conformance score.

## Publication Condition

This draft may become an immutable historical composition record only after
all required immutable identities are established and verified.

Until then:

```text
STATUS=DRAFT
HISTORICAL_REWRITE=FORBIDDEN
FLOATING_REFERENCES=FORBIDDEN
COMPOSITION_BINDING=HOLD
```
