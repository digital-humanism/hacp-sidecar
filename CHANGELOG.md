# Changelog

All notable changes to `hacp-sidecar` are documented in this file.

The format is inspired by Keep a Changelog, and this project uses semantic versioning for repository releases.

## [0.5.0] - 2026-08-21

Stable promotion of the `v0.5.0-rc.1` release candidate.

No runtime, protocol, or enforcement semantics changed after the release candidate.

### Verified

- Clean-clone Go dependency resolution passes.
- Full Go regression suite passes.
- Gate E distributed control-plane suite passes.
- Distributed revocation and two-sidecar convergence remain verified.
- Python ↔ Go sidecar interoperability remains verified.
- Gate D p99 sidecar overhead remains within the release target.

### Release compatibility

- Sidecar release: `v0.5.0`
- HACP specification release: `0.9.3`
- HACP-Core conformance baseline: `0.9.2`
- HACP wire version: `0.9`
- Runner Protocol: `1`

## [0.5.0-rc.1] - 2026-08-20

### Added

- Distributed HACP control plane over gRPC.
- Revocation snapshot synchronization.
- Resumable `WatchRevocations` stream.
- Revision tracking and replay after reconnect.
- Duplicate and out-of-order event handling.
- Fail-closed behavior on revision gaps.
- Snapshot recovery through reset-required semantics.
- Control-plane heartbeat and freshness tracking.
- `CONTROL_STATE_STALE` enforcement behavior.
- Distributed key, token, envelope, and parent-envelope revocation.
- Multi-sidecar revocation convergence validation.
- Gate E engineering and verification documentation.
- Gate D performance validation and benchmark documentation.

### Changed

- Release metadata and documentation aligned for the first public release candidate.
- Public documentation paths normalized to portable forms.
- Repository hygiene improved for build, runtime, benchmark, and local environment artifacts.

### Verified

- Full Go regression suite passes.
- Gate E distributed control-plane tests pass.
- Python ↔ Go sidecar interoperability verified.
- Distributed revocation propagation validated.
- Gate D p99 sidecar overhead remains within the release target.

### Release compatibility

- Sidecar release: `v0.5.0-rc.1`
- HACP specification release target: `0.9.3`
- HACP-Core conformance baseline: `0.9.2`
- HACP wire version: `0.9`
- Runner Protocol: `1`

## Pre-release history

Development before `v0.5.0-rc.1` was conducted as an unreleased implementation and validation phase covering Gates A through E.

---

**Contact:** [digital.humanism.collective@protonmail.com](mailto:digital.humanism.collective@protonmail.com)
