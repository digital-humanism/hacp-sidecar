# HACP 1.1.1 Artifact Index

This file is a repository-owned evidence-navigation index for the HACP 1.1.1
pre-release candidate in `hacp-sidecar`.

```text
ROLE=EVIDENCE_NAVIGATION_ONLY
NORMATIVE_AUTHORITY=NO
CONTENT_DUPLICATION=NO
HISTORICAL_REWRITE=NO
```

An immutable identity does not imply public reachability.

Controlled evidence is not published merely to make an identity publicly
resolvable.

## Current Sidecar-Owned Composition

```text
PATH=manifests/HACP_1_1_1_COMPOSITION.md
ROLE=SIDECAR_RUNTIME
CLASSIFICATION=CURRENT_PRE_RELEASE_COMPOSITION
CURRENT_AUTHORITY=YES
SHA256=6E1EC7AFA012819A58FA4B3234421B889B32BE63464332CBA778607EDB002C16
RAW_GIT_BLOB=e75a5aa51e676c354650968e0b17f16cb62c2b55
PUBLICATION_STATE=LOCAL_PRE_RELEASE_CANDIDATE
```

Repository-local history mirror:

```text
PATH=docs/history-artifacts/hacp-1.1.1/HACP_1_1_1_COMPOSITION.md
ROLE=REPOSITORY_LOCAL_HISTORY_MIRROR
CLASSIFICATION=CURRENT_PRE_RELEASE_COMPOSITION_MIRROR
CURRENT_AUTHORITY=NO
SHA256=6E1EC7AFA012819A58FA4B3234421B889B32BE63464332CBA778607EDB002C16
RAW_GIT_BLOB=e75a5aa51e676c354650968e0b17f16cb62c2b55
```

## Historical Pre-V2 Composition Draft

```text
PATH=docs/history-artifacts/hacp-1.1.1/HACP_1_1_1_COMPOSITION_PRE_V2.md
RAW_GIT_BLOB=27e9d087e718dddbd2236443fa5c30a9d74e5741
SHA256=23F2CC51A42D21601AE5EFA8C0CDE598C24464C6DACEC1B7FD67F68B868EFEB8
CLASSIFICATION=SUPERSEDED
ROLE=HISTORICAL_PRE_V2_COMPOSITION_DRAFT
CURRENT_AUTHORITY=NO
HISTORICAL_REWRITE=NO
```

The preserved file remains byte-identical to the pre-V2 staged Git blob.
Classification metadata is recorded here instead of modifying that historical
content.

## S10.4 Adoption Consistency Verification

```text
PATH=docs/verification/HACP_1_1_1_ADOPTION_CONSISTENCY_VERIFICATION.md
COMMIT=d34a97abfeb8b93958552616d2d679ac9ce0477e
BLOB=48af1687ca87f30c531dd045b2268ba6c77dc7cd
SHA256=31F2858EEE86C2E9F94980474DB035E7ACB920D45CA1BFE51298AE1E34089656

CURRENT_LOCAL_REACHABLE=NO
REMOTE_REF_REACHABLE=YES
PUBLICLY_RESOLVABLE=YES
CONTROLLED_INTERNAL_ONLY=NO
HISTORICAL_ONLY=YES
CLASSIFICATION=HISTORICAL_PUBLISHED_DOCUMENTARY_EVIDENCE
```

This artifact is historical documentary evidence. Its immutable identity is
retained without treating it as current composition authority.

## S10.7 Cross-Repo Evidence Reconciliation

```text
PATH=docs/verification/HACP_1_1_1_CROSS_REPO_EVIDENCE_RECONCILIATION.md
COMMIT=948d58ab6b385bf6fb34dd697817a73d150ae576
BLOB=e1fdc8a02465478cf3d75f66a1b1a7d27cf47162
SHA256=02107421D077C2CBFF6CD1224B1E5018B11DDD0EB6497DA9C33C47B2EDB4A598

CURRENT_LOCAL_REACHABLE=NO
REMOTE_REF_REACHABLE=NO
PUBLICLY_RESOLVABLE=NO
CONTROLLED_INTERNAL_ONLY=YES
HISTORICAL_ONLY=NO
CLASSIFICATION=CURRENT_EVIDENCE_CONTROLLED_INTERNAL
```

The S10.7 artifact remains controlled evidence. This index records its exact
immutable identity but does not publish or reproduce its content.

## Authorization Boundary

This index does not authorize:

```text
main integration
tag creation
release creation
stable release
production source mutation
historical rewrite
humanist-core mutation
publication of controlled internal evidence
```
