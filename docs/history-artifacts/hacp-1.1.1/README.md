# HACP 1.1.1 History Artifacts

This directory is a repository-owned navigation surface for HACP 1.1.1
composition and evidence history in `hacp-sidecar`.

```text
ROLE=EVIDENCE_NAVIGATION_ONLY
NORMATIVE_AUTHORITY=NO
CONTENT_DUPLICATION=NO
HISTORICAL_REWRITE=NO
```

The files in this directory do not replace normative specification authority,
runtime implementation identity, or release authorization.

## Current Repository-Owned Composition

The current sidecar-owned composition manifest is:

```text
../../../../manifests/HACP_1_1_1_COMPOSITION.md
```

Its repository role is:

```text
COMPOSITION_ROLE=SIDECAR_RUNTIME
```

The repository-local history mirror is:

```text
HACP_1_1_1_COMPOSITION.md
```

The canonical manifest and its repository-local history mirror must remain
byte-identical within `hacp-sidecar`.

## Historical Pre-V2 Composition Draft

The preserved pre-V2 draft is:

```text
HACP_1_1_1_COMPOSITION_PRE_V2.md
```

Its classification is:

```text
CLASSIFICATION=SUPERSEDED
ROLE=HISTORICAL_PRE_V2_COMPOSITION_DRAFT
CURRENT_AUTHORITY=NO
HISTORICAL_REWRITE=NO
```

The preserved historical file is not edited to add this metadata. Its original
content remains byte-identical to the pre-V2 staged Git blob.

## Evidence Navigation Boundary

`ARTIFACT_INDEX.md` provides navigation metadata for relevant immutable
evidence identities.

An immutable commit or blob identity does not by itself imply public remote
reachability.

Every evidence navigation entry must distinguish, where applicable:

```text
CURRENT_LOCAL_REACHABLE
REMOTE_REF_REACHABLE
PUBLICLY_RESOLVABLE
CONTROLLED_INTERNAL_ONLY
HISTORICAL_ONLY
CLASSIFICATION
```

Controlled evidence is not published merely to make an immutable identity
publicly resolvable.

## Release Boundary

This history-navigation surface does not authorize:

```text
main integration
tag creation
release creation
stable release
production source mutation
historical rewrite
humanist-core mutation
```

The current pre-release composition remains subject to the complete E21
candidate verification and activation boundary.
