# HACP 1.1.1 Composition

Status: PRE-RELEASE CANDIDATE

Repository:

```text
hacp-sidecar
```

Composition role:

```text
COMPOSITION_ROLE=SIDECAR_RUNTIME
```

This repository owns the runtime and enforcement view of the HACP 1.1.1
release composition.

## Release Train Identity

```text
HACP_RELEASE=1.1.1
HACP_RELEASE_CANDIDATE=1.1.1-rc.1
FINAL_RELEASE_TAG=v1.1.1
```

The HACP 1.1.1 release candidate is a mutable working candidate.

```text
RC=MUTABLE
FINAL_RELEASE_TAG_STATE=IMMUTABLE
```

Corrective commits, documentation corrections, CI or reproduction corrections,
and other bounded release-readiness changes may be applied while the active
release candidate is being verified.

The active release candidate is not an immutable final Git tag.

The exact Git representation of the mutable release candidate is defined
separately at the release-candidate representation checkpoint.

Raw Git commit SHA values remain factual Git identities and evidence.
They are not manually maintained cross-repository release-composition bindings.

Floating references such as `main` or `latest` are not final release identities.

## Repository Ownership Boundary

This manifest is owned by `hacp-sidecar`.

The corresponding `hacp-spec` repository owns its own
`HACP_1_1_1_COMPOSITION.md` with:

```text
COMPOSITION_ROLE=SPEC_AUTHORITY
```

The two repository-owned manifests are not required to be byte-identical.

They must converge on the HACP 1.1.1 release-train semantics:

```text
HACP_RELEASE
HACP_RELEASE_CANDIDATE
FINAL_RELEASE_TAG
RC=MUTABLE
FINAL_RELEASE_TAG_STATE=IMMUTABLE
```

Repository commit SHA values may be recorded as factual evidence but are not
manually maintained peer bindings for the active release candidate.

## Runtime Identity Boundary

The runtime and enforcement implementation remains repository-owned by
`hacp-sidecar`.

For a concrete verification run, the actual runtime commit SHA may be recorded
as factual evidence. It is not a manually maintained peer binding in the
release-candidate composition.

This composition record does not modify or reinterpret the frozen runtime
implementation.

## Release Boundary

This pre-release composition record does not itself authorize:

```text
main integration
tag creation
stable release
production source mutation
humanist-core mutation
```

Historical identities and evidence remain immutable.
