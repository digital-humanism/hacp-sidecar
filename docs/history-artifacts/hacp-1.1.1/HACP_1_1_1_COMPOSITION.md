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

## Exact Immutable Binding

```text
HACP_RELEASE=1.1.1
HACP_COMPOSITION_ID=HACP-1.1.1-spec-3ea4adc-sidecar-1bc10ac
HACP_SPEC_COMMIT=3ea4adcbf6ef2548e41d340709945326026a73b6
SIDECAR_IMPLEMENTATION_COMMIT=1bc10acbe79620a165cee573c5b260cd8639280f
```

The peer identities above are immutable release-composition bindings.

Floating references such as `main`, `latest`, or an unpinned branch name are
not composition identities.

## Repository Ownership Boundary

This manifest is owned by `hacp-sidecar`.

The corresponding `hacp-spec` repository owns its own
`HACP_1_1_1_COMPOSITION.md` with:

```text
COMPOSITION_ROLE=SPEC_AUTHORITY
```

The two repository-owned manifests are not required to be byte-identical.

They must converge exactly on:

```text
HACP_RELEASE
HACP_COMPOSITION_ID
HACP_SPEC_COMMIT
SIDECAR_IMPLEMENTATION_COMMIT
```

## Runtime Identity Boundary

The runtime implementation bound by this composition is the exact
`SIDECAR_IMPLEMENTATION_COMMIT` declared in the immutable binding above.

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
