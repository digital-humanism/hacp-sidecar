# HACP Sidecar

Enforcement sidecar for the **Human Agency Continuity Protocol (HACP)**.

Implements the HACP Enforcement profile defined by [`hacp-spec`](https://github.com/digital-humanism/hacp-spec).

HACP Sidecar is a fail-closed enforcement point between an AI agent and protected tools or services. A request is forwarded upstream only after its HACP intent, authorization token, scope, semantic boundary conditions, budget, revocation state, distributed control-state freshness, and provenance requirements have been validated.

## Version domains

| Domain | Version |
|---|---|
| Sidecar release | `v0.5.0` |
| HACP specification release | `0.9.3` |
| HACP-Core conformance baseline | `0.9.2` |
| Wire protocol family | `0.9` |
| Runner Protocol | `1` |

These versions belong to separate release and compatibility domains and MUST NOT be conflated.

---

## Overview

HTTP and MCP traffic explicitly routed through the sidecar carries:

- `X-HACP-Intent-Envelope`
- `X-HACP-Decision-Token`

The sidecar:

1. parses and validates HACP inputs;
2. resolves signer keys;
3. evaluates signer, envelope, and token revocation;
4. verifies Ed25519 signatures;
5. validates expiry and request/action bindings;
6. evaluates semantic scope and boundary transitions;
7. evaluates human/checkpoint semantics;
8. checks budget and replay state;
9. checks distributed control-state freshness when enabled;
10. records provenance;
11. forwards allowed requests to the configured upstream;
12. denies invalid or unsafe requests with deterministic HACP reason codes.

The enforcement pipeline is strictly **fail closed**.

---

# Project Status

**Phase 4 — Gates A–E closed**

| Gate | Focus | Status |
|---|---|---|
| **A** | Protocol correctness — HACP-Core conformance | ✅ Closed |
| **B** | Semantic completeness — boundary matrix | ✅ Closed |
| **C** | Deployability — native/reference topology | ✅ Closed |
| **D** | Operational viability — latency / throughput | ✅ Closed |
| **E** | Distributed management — gRPC control plane | ✅ Closed |

Current canonical conformance result:

```text
RESULTS: 38/38 passed
```

Gate D reference acceptance:

```text
Serial unique-token p99 overhead: 0.59 ms
Target: < 5 ms
Result: PASS
```

Gate E distributed-control-plane validation includes:

```text
revocation propagation        PASS
reconnect + replay            PASS
bounded exponential backoff   PASS
ResetRequired recovery        PASS
heartbeat freshness           PASS
stale fail-closed             PASS
two-sidecar convergence       PASS
full Go regression            PASS
```

---

# Implemented

## Protocol enforcement

- ✅ HACP Intent Envelope parsing and validation
- ✅ HACP Decision Token parsing and validation
- ✅ Ed25519 signature verification
- ✅ JCS-compatible canonicalization
- ✅ signer-key resolution
- ✅ signer-key revocation
- ✅ envelope revocation
- ✅ decision-token revocation
- ✅ envelope/token expiry validation
- ✅ `action_hash` binding
- ✅ HTTP method/path constraint binding
- ✅ principal and policy-context handling
- ✅ deterministic denial reason codes
- ✅ strict fail-closed evaluation

## Semantic enforcement

- ✅ scope containment
- ✅ data-driven boundary matrix
- ✅ fail-closed unknown security-relevant attributes
- ✅ human-required boundary handling
- ✅ semantic boundary-crossing detection

## Runtime state

- ✅ token budget enforcement
- ✅ autonomy budget enforcement
- ✅ replay/budget ledger
- ✅ local revocation state
- ✅ provenance ring buffer
- ✅ asynchronous provenance flush

## Distributed control plane

- ✅ gRPC control-plane contract
- ✅ authoritative monotonic revision journal
- ✅ atomic revocation snapshots
- ✅ resumable server-streaming revocation feed
- ✅ duplicate/old event idempotency
- ✅ revision-gap detection
- ✅ reconnect + replay from `last_seen_revision`
- ✅ bounded exponential reconnect backoff
- ✅ `ResetRequired` + snapshot recovery
- ✅ heartbeat-based freshness
- ✅ unsafe-state tracking
- ✅ stale fail-closed via `CONTROL_STATE_STALE`
- ✅ real evaluator revocation propagation
- ✅ multi-sidecar convergence

## Proxy/runtime

- ✅ real upstream HTTP forwarding
- ✅ dedicated HTTP transport/client
- ✅ HTTP/1.1 persistent connections
- ✅ connection pooling
- ✅ hop-by-hop header filtering
- ✅ environment-driven upstream configuration
- ✅ native/reference deployment topology

## Verification

- ✅ 38/38 HACP-Core conformance vectors
- ✅ Gate E control-plane integration suite
- ✅ full `go test ./...`
- ✅ `go vet`
- ✅ serial Gate D benchmark
- ✅ concurrent rotating-token benchmark
- ✅ high-rate rotating-token benchmark
- ✅ shared-token load characterization
- ✅ GitHub Actions regression workflow

---

# Architecture

```text
┌─────────────┐
│    Agent    │
│ (untrusted) │
└──────┬──────┘
       │
       │ HTTP / MCP
       │ X-HACP-Intent-Envelope
       │ X-HACP-Decision-Token
       ▼
┌─────────────────────────────────────────────┐
│                HACP Sidecar                 │
│                                             │
│  1. Parse / schema validation               │
│  2. Resolve signer keys                     │
│  3. Key revocation                          │
│  4. Ed25519 verification                    │
│  5. Envelope/token revocation               │
│  6. Expiry                                  │
│  7. Action hash binding                     │
│  8. Request constraints                     │
│  9. Scope containment                       │
│ 10. Boundary matrix                         │
│ 11. Human/checkpoint semantics              │
│ 12. Budget / replay                         │
│ 13. Control-state freshness                 │
│ 14. Provenance acceptance                   │
│ 15. Forward or DENY                         │
└──────────────┬──────────────────────────────┘
               │
               │ HTTP
               ▼
         ┌────────────┐
         │  Upstream  │
         │    Tool    │
         └────────────┘

               ▲
               │ gRPC snapshot + revoke stream
               │
        ┌──────┴────────┐
        │ Control Plane │
        └───────────────┘
```

The control plane is asynchronous with respect to request authorization: the sidecar evaluates requests from locally materialized control state rather than synchronously consulting the control plane on every action.

---

# Distributed Control-Plane Model

Gate E extends the sidecar from a local enforcement process into a distributed enforcement replica.

Each sidecar maintains:

- a local revocation store;
- a highest fully materialized revision;
- a control-state freshness timestamp;
- connection state;
- unsafe-state status.

The authoritative control plane provides snapshots, ordered revocation events, heartbeats, replay, and explicit reset recovery.

---

## Revision semantics

For a sidecar whose highest fully materialized revision is `R`:

```text
event.revision == R + 1
→ apply
→ advance to R + 1

event.revision <= R
→ duplicate / old
→ ignore

event.revision > R + 1
→ gap
→ mark unsafe
→ fail closed / recover
```

`last_seen_revision` means the highest revision that has been successfully materialized locally.

It is **not** the highest revision merely observed on the network.

---

## Startup and recovery

Startup:

```text
GetRevocationSnapshot
        ↓
snapshot @ revision R
        ↓
materialize local state
        ↓
WatchRevocations(after_revision=R)
```

Reconnect:

```text
disconnect
    ↓
reconnect(after_revision=last_seen_revision)
    ↓
replay missed events
    ↓
resume live stream
```

If replay history is no longer available:

```text
ResetRequired
    ↓
fresh snapshot
    ↓
atomic local replacement
    ↓
resume stream
```

Reconnect uses bounded exponential backoff and respects `context.Context` cancellation.

---

## Freshness and fail-closed behavior

A temporary control-plane disconnect does not immediately make the sidecar unusable.

The last fully materialized state may remain usable while it is still within the configured freshness interval.

If control state becomes stale or unsafe:

```text
DENY
CONTROL_STATE_STALE
```

Examples:

- maximum staleness exceeded;
- revision gap;
- inconsistent heartbeat;
- unsafe state awaiting snapshot recovery.

Heartbeats refresh freshness but do **not** advance durable revision state and cannot be used to skip missing events.

---

## Multi-sidecar convergence

Gate E validates multiple independent sidecars connected to the same control plane.

After a revocation at revision `N`:

```text
                  Control Plane
                       │
                 revision N
                 revoke token
                   ┌───┴───┐
                   ▼       ▼
              Sidecar A  Sidecar B
              revision N revision N
              revoked    revoked
                 │          │
                 ▼          ▼
                DENY       DENY
             TOKEN_REVOKED TOKEN_REVOKED
```

The system therefore converges on both:

1. the same distributed revocation state;
2. the same security outcome.

---

# Enforcement Model

The sidecar is an explicit enforcement point.

Traffic must be routed through the sidecar by deployment architecture. The sidecar itself does **not** claim to provide OS-level transparent interception.

Prevention of direct upstream bypass is a deployment responsibility and should use mechanisms such as:

- network policy;
- firewall rules;
- service-mesh routing;
- container/network isolation;
- equivalent infrastructure controls.

---

# Repository Structure

```text
hacp-sidecar/
├── .github/
│   └── workflows/
│       └── tests.yml
│
├── benchmarks/
│   ├── benchmark-rotating/
│   ├── generate-tokens/
│   ├── benchmark.ps1
│   ├── benchmark.sh
│   ├── benchmark_rotating.ps1
│   └── benchmark_serial.ps1
│
├── cmd/
│   ├── gen-token/
│   ├── hacp-conformance-runner/
│   └── sidecar/
│
├── deployments/
│   ├── control-plane/
│   └── upstream/
│
├── docs/
│   ├── README.md
│   ├── architecture/
│   │   ├── architecture.md
│   │   └── distributed-control-plane.md
│   ├── gates/
│   │   └── gate-e-distributed-control-plane.md
│   ├── engineering/
│   │   ├── gate-d-benchmark.md
│   │   ├── gate-d-performance-validation.md
│   │   └── gate-e-engineering-report.md
│   └── verification/
│       └── distributed-control-plane-testing.md
│
├── gen/
│   └── controlplane/
│       └── v1/
│           ├── control_plane.pb.go
│           └── control_plane_grpc.pb.go
│
├── internal/
│   ├── budget/
│   ├── controlplane/
│   ├── evaluate/
│   ├── provenance/
│   ├── proxy/
│   ├── scope/
│   └── wire/
│
├── go.mod
├── go.sum
├── LICENSE.md
└── README.md
```

Generated benchmark tokens, benchmark result files, local executables, and machine-local artifacts are not intended to be committed.

---

# Requirements

## Required

- Go version defined by `go.mod`
- Python 3.x for reference/mock services and conformance tooling

## Optional

- [`hey`](https://github.com/rakyll/hey) for HTTP load testing
- Docker for containerized deployment experiments

The native/reference topology does not require Docker.

---

# Build

## Sidecar

Linux/macOS:

```bash
go build -o hacp-sidecar ./cmd/sidecar
```

Windows:

```powershell
go build -o hacp-sidecar.exe ./cmd/sidecar
```

## Conformance runner

Linux/macOS:

```bash
go build -o hacp-conformance-runner ./cmd/hacp-conformance-runner
```

Windows:

```powershell
go build -o hacp-conformance-runner.exe ./cmd/hacp-conformance-runner
```

---

# Test

## Gate E control-plane suite

```bash
go test ./internal/controlplane -count=1 -v
```

This suite covers:

- distributed revocation propagation;
- heartbeat delivery;
- freshness refresh;
- invalid heartbeat handling;
- duplicate/old revision handling;
- revision gaps;
- reconnect and replay;
- bounded exponential backoff;
- `ResetRequired`;
- snapshot recovery;
- stale fail-closed evaluation;
- standalone evaluator compatibility;
- two-sidecar convergence.

## Full regression

```bash
go test ./... -count=1
```

## Static analysis

```bash
go vet ./...
```

Runtime changes should not be committed without passing the relevant tests.

---

# Conformance

## Gate A — Protocol Correctness

**Status: ✅ Closed**

The sidecar passes all canonical HACP-Core v0.9.2 conformance vectors:

```text
RESULTS: 38/38 passed
```

Build the runner:

```powershell
go build -o hacp-conformance-runner.exe ./cmd/hacp-conformance-runner
```

Run the canonical harness from the adjacent specification repository:

```powershell
cd ...\GitHub\hacp-spec\harness

python .\harness.py
```

Expected:

```text
RESULTS: 38/38 passed
```

The harness communicates with implementations through the language-neutral runner protocol.

See:

[`hacp-spec/harness/runner_protocol.md`](https://github.com/digital-humanism/hacp-spec/blob/main/harness/runner_protocol.md)

---

# Boundary Matrix

## Gate B — Semantic Completeness

**Status: ✅ Closed**

Security-relevant transitions are evaluated through a data-driven boundary matrix under:

```text
internal/scope/
```

The matrix evaluates attributes including:

- audience;
- externality;
- reversibility;
- data class;
- resource class;
- principal semantics.

Unknown security-relevant values fail closed.

---

# Reference Deployment

## Gate C — Deployability

**Status: ✅ Closed**

Reference topology:

```text
Client
  ↓
HACP Sidecar
  ↓
Real HTTP Upstream
```

The sidecar performs actual HTTP forwarding rather than local response emulation.

The upstream path uses a pooled HTTP transport with persistent connections and hop-by-hop header filtering.

Docker remains an optional packaging topology rather than a requirement for Gate C.

---

# Performance

## Gate D — Operational Viability

**Status: ✅ Closed**

Acceptance target:

```text
p99 HACP enforcement overhead < 5 ms
```

Reference results:

| Benchmark | Workload | Result |
|---|---|---|
| Serial | 1000 requests, unique token/request | **p99 overhead 0.59 ms** |
| PowerShell rotating | 1000 requests, concurrency 5 | **avg 0.53 ms**, **p95 1.10 ms** |
| Go rotating | 5 × 1000 requests, concurrency 5 | **median p99 1.26 ms** |
| `hey` shared-token | 1000 requests, concurrency 5 | sidecar **p99 3.3 ms**, ~**2761 req/s** |

Serial Gate D acceptance:

```text
Baseline p99: 4.04 ms
Sidecar p99:  4.63 ms
Overhead:     0.59 ms

GATE D PASS
```

Negative deltas in repeated isolated measurements are treated as stochastic baseline/upstream tail behavior, not as evidence that the sidecar makes requests faster.

---

# Distributed Management

## Gate E — Distributed Control Plane

**Status: ✅ Closed**

Gate E establishes the distributed revocation/control-state layer.

Verified capabilities:

- gRPC control-plane protocol;
- durable monotonic revision semantics;
- snapshot bootstrap;
- live revoke stream;
- duplicate/old event idempotency;
- revision-gap detection;
- reconnect and replay;
- bounded exponential reconnect backoff;
- replay-compaction handling;
- `ResetRequired`;
- snapshot recovery;
- heartbeat freshness;
- stale fail-closed behavior;
- real Pipeline revocation propagation;
- two-sidecar convergence.

Run:

```bash
go test ./internal/controlplane -count=1 -v
go test ./... -count=1
```

Both passed at Gate E completion.

See:

- [`docs/gates/gate-e-distributed-control-plane.md`](docs/gates/gate-e-distributed-control-plane.md)
- [`docs/engineering/gate-e-engineering-report.md`](docs/engineering/gate-e-engineering-report.md)

The normative gRPC contract lives in:

[`hacp-spec/proto/hacp/control/v1/control_plane.proto`](https://github.com/digital-humanism/hacp-spec/blob/main/proto/hacp/control/v1/control_plane.proto)

---

# Reason Codes

The sidecar uses deterministic HACP reason codes.

| Code | Condition |
|---|---|
| `INVALID_ENVELOPE` | missing or malformed intent envelope |
| `INVALID_ACTION` | missing or malformed proposed action |
| `KEY_REVOKED` | signer key revoked |
| `SIGNATURE_FAILURE` | signature verification failed |
| `ENVELOPE_REVOKED` | intent envelope revoked |
| `TOKEN_REVOKED` | decision token revoked |
| `CONTROL_STATE_STALE` | distributed control state is stale or unsafe |
| `ENVELOPE_EXPIRED` | envelope exceeded `expires_at` |
| `TOKEN_EXPIRED` | token exceeded `expires_at` |
| `SCOPE_EXCEEDED` | proposed action outside authorized scope/binding |
| `BOUNDARY_CROSSING` | semantic boundary transition not permitted |
| `UNKNOWN_ATTRIBUTE` | unknown security-relevant attribute |
| `BUDGET_EXHAUSTED` | token/autonomy budget exhausted |
| `HUMAN_REQUIRED` | human authorization/checkpoint required |
| `CHECKPOINT_TIMEOUT` | required checkpoint timed out |
| `TRACEABILITY_FAILURE` | provenance/audit acceptance failed |
| `POLICY_DENIED` | policy explicitly denied execution |

Normative definitions are maintained in `hacp-spec`.

---

# Security Model

The agent runtime is treated as potentially hostile.

Core protections include:

- fail-closed validation;
- cryptographic authorization;
- request/action binding;
- semantic scope enforcement;
- boundary-matrix evaluation;
- signer/envelope/token revocation;
- bounded autonomy;
- replay/budget state;
- distributed control-state freshness;
- provenance-before-forwarding;
- deterministic denial reasons.

## Network bypass

Routing through the sidecar alone is not an operating-system security boundary.

Deployments requiring bypass resistance must independently prevent unauthorized direct connections to protected upstream services.

Typical mechanisms:

- Kubernetes NetworkPolicy;
- firewall rules;
- service-mesh policy;
- namespace/network isolation;
- equivalent infrastructure controls.

---

# Out of Scope

The current implementation does not claim to provide:

- eBPF enforcement;
- transparent iptables interception;
- dynamic-library injection;
- agent bytecode modification;
- full OS-level process isolation;
- generic non-HTTP data-plane transports;
- production external persistence for the control-plane journal;
- globally distributed budget/replay databases.

These may be added without changing HACP-Core authorization semantics.

---

# Development Workflow

Before committing runtime changes:

```bash
gofmt -w .
go test ./...
go vet ./...
```

For changes affecting HACP semantics:

1. run the full Go regression;
2. rebuild the conformance runner;
3. run HACP-Core conformance;
4. verify `38/38`;
5. run Gate E tests if distributed-control behavior changed;
6. run Gate D benchmarks if the hot path changed.

Runtime changes must not regress canonical conformance.

---

# Gate Discipline

```text
Phase 4.1 — Design
✅ Complete

Phase 4.2 — MVP Implementation
✅ Complete

Gate A — Protocol Correctness
✅ Closed

Gate B — Semantic Boundary Matrix
✅ Closed

Gate C — Reference Deployment
✅ Closed

Gate D — Operational Performance
✅ Closed

Gate E — Distributed Control Plane
✅ Closed
```

Typical ownership:

```text
Gate B / semantic enforcement
→ internal/scope/

Gate C / deployment and proxy behavior
→ deployments/
→ internal/proxy/

Gate D / performance regression
→ benchmarks/

Gate E / distributed control plane
→ internal/controlplane/
→ gen/controlplane/
→ hacp-spec/proto/
```

---

# CI

GitHub Actions runs the Gate E integration suite and the full Go regression.

Workflow:

```text
.github/workflows/tests.yml
```

The workflow checks out both:

```text
hacp-sidecar/
hacp-spec/
```

so integration tests can use the canonical HACP vectors as an external source of truth.

---

# Documentation

Architecture:

```text
docs/architecture/architecture.md
```

Gate D:

```text
docs/engineering/gate-d-benchmark.md
docs/engineering/gate-d-performance-validation.md
```

Gate E:

```text
docs/gates/gate-e-distributed-control-plane.md
docs/engineering/gate-e-engineering-report.md
```

Normative protocol:

[`digital-humanism/hacp-spec`](https://github.com/digital-humanism/hacp-spec)

---

# License

This repository contains the HACP Enforcement Sidecar implementation and is licensed under the **GNU Affero General Public License v3.0 (AGPLv3)**, with commercial dual licensing available for enterprise deployments, closed-source embedding, and OEM integration.

For commercial licensing inquiries:

`digital.humanism.collective@protonmail.com`

## Relationship to other HACP repositories

| Repository | Purpose | License |
|---|---|---|
| [`hacp-spec`](https://github.com/digital-humanism/hacp-spec) | open protocol specification, schemas, vectors, control-plane contract | CC BY 4.0 |
| [`humanist-core`](https://github.com/digital-humanism/humanist-core) | reference SDK | AGPLv3 + commercial dual |
| [`hacp-sidecar`](https://github.com/digital-humanism/hacp-sidecar) | enforcement sidecar | AGPLv3 + commercial dual |

This separation keeps the protocol itself open and vendor-neutral while allowing implementation repositories to maintain their own sustainability model.

See [`LICENSE.md`](LICENSE.md) for repository licensing terms.

---

# Contributing

Contributions must preserve:

- normative HACP conformance;
- fail-closed semantics;
- deterministic denial behavior;
- security-relevant evaluation ordering;
- boundary-matrix behavior;
- distributed revision invariants;
- freshness/fail-closed semantics;
- provenance-before-forwarding;
- benchmark reproducibility.

Before submitting a pull request:

```bash
go test ./...
go vet ./...
```

For protocol/runtime changes, also verify:

```text
HACP-Core conformance: 38/38 PASS
```

Distributed-control changes should run the Gate E suite.

Performance-sensitive changes should run the Gate D benchmark suite.

---

# Contact

**Digital Humanism Collective**

`digital.humanism.collective@protonmail.com`
