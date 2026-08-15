# HACP Sidecar

Enforcement sidecar for the **Human Agency Continuity Protocol (HACP)**.

Implements the HACP Enforcement profile defined by [`hacp-spec`](https://github.com/digital-humanism/hacp-spec).

HACP Sidecar acts as a fail-closed enforcement point between an AI agent and external tools or services. A request is forwarded upstream only after its HACP intent, authorization token, scope, boundary conditions, budget, revocation state, and provenance requirements have been successfully validated.

---

## Overview

HACP Sidecar provides a runtime enforcement boundary for agent actions.

HTTP and MCP traffic explicitly routed through the sidecar carries:

- `X-HACP-Intent-Envelope`
- `X-HACP-Decision-Token`

The sidecar:

1. parses and validates the HACP headers;
2. resolves signer keys;
3. evaluates key, envelope, and token revocation;
4. verifies Ed25519 signatures;
5. validates expiry and request bindings;
6. evaluates semantic scope and boundary transitions;
7. checks budget/replay state;
8. records provenance;
9. forwards an allowed request to the configured upstream service;
10. denies invalid requests with deterministic HACP reason codes.

The enforcement pipeline is strictly **fail-closed**.

---

## Project Status

**Phase 4 MVP — Gates A–D closed**

| Gate | Focus | Status |
|---|---|---|
| A | Protocol correctness — HACP-Core conformance | ✅ Closed |
| B | Semantic completeness — boundary matrix | ✅ Closed |
| C | Deployability — native/reference sidecar topology | ✅ Closed |
| D | Operational viability — latency and throughput | ✅ Closed |
| E | Distributed management — authenticated control plane | ⏸ Pending |

Current conformance result:

```text
RESULTS: 38/38 passed
```

Current Gate D acceptance result:

```text
Serial unique-token p99 overhead: 0.59 ms
Target: < 5 ms
Result: PASS
```

---

## Implemented

### Protocol enforcement

- ✅ HACP Intent Envelope parsing and validation
- ✅ HACP Decision Token parsing and validation
- ✅ Ed25519 signature verification
- ✅ JCS-compatible canonicalization
- ✅ Signer-key resolution
- ✅ Signer-key revocation checks
- ✅ Envelope revocation checks
- ✅ Decision-token revocation checks
- ✅ Envelope and token expiry validation
- ✅ `action_hash` binding
- ✅ HTTP method/path constraint binding
- ✅ Principal and policy-context handling
- ✅ Deterministic HACP denial reason codes
- ✅ Strict fail-closed evaluation

### Semantic enforcement

- ✅ Scope containment
- ✅ Data-driven boundary matrix
- ✅ Fail-closed handling of unknown security-relevant attributes
- ✅ Human-required boundary handling
- ✅ Semantic boundary-crossing detection

### Runtime state

- ✅ Token budget enforcement
- ✅ Replay/budget ledger
- ✅ Local revocation state
- ✅ Provenance ring buffer
- ✅ Asynchronous provenance flush

### Proxy/runtime

- ✅ Real upstream HTTP forwarding
- ✅ Dedicated HTTP transport/client
- ✅ HTTP/1.1 persistent connections
- ✅ Connection pooling
- ✅ Hop-by-hop header filtering
- ✅ Upstream configuration through environment variables
- ✅ Native/reference deployment topology

### Verification

- ✅ 38/38 HACP-Core conformance vectors
- ✅ Unit test suite
- ✅ `go vet`
- ✅ Serial Gate D benchmark
- ✅ Concurrent rotating-token benchmark
- ✅ High-rate rotating-token benchmark
- ✅ Shared-token load characterization

---

## Pending

### Gate E — Distributed Management

The next major implementation gate covers:

- ⏸ authenticated gRPC control plane;
- ⏸ revocation propagation/streaming;
- ⏸ bounded revocation freshness;
- ⏸ distributed revocation state;
- ⏸ distributed budget/replay state;
- ⏸ control-plane reconnect/recovery behavior.

The current HTTP control-plane/revocation paths are reference/MVP mechanisms and are not the final Gate E transport.

---

## Architecture

```text
┌─────────────┐
│   Agent     │
│ (untrusted) │
└──────┬──────┘
       │
       │ HTTP / MCP
       │ X-HACP-Intent-Envelope
       │ X-HACP-Decision-Token
       ▼
┌─────────────────────────────────────────────┐
│                 HACP Sidecar                │
│                                             │
│  1. Parse / schema validation               │
│  2. Token decision                          │
│  3. Resolve signer keys                     │
│  4. Key revocation                          │
│  5. Ed25519 signature verification          │
│  6. Envelope/token revocation               │
│  7. Expiry                                  │
│  8. Action hash binding                     │
│  9. Request constraints                     │
│ 10. Scope containment                       │
│ 11. Boundary matrix                         │
│ 12. Human/checkpoint semantics              │
│ 13. Budget / replay state                   │
│ 14. Provenance acceptance                   │
│ 15. Forward or DENY                         │
└──────────────────┬──────────────────────────┘
                   │
                   │ HTTP
                   ▼
             ┌────────────┐
             │  Upstream  │
             │   Tool     │
             └────────────┘

                   ▲
                   │
                   │ Control / revocation
                   │
             ┌─────────────┐
             │   Control   │
             │    Plane    │
             └─────────────┘
```

The control-plane connection shown above is currently represented by the MVP/reference control mechanism. Authenticated gRPC streaming is planned for Gate E.

---

## Enforcement Model

The sidecar is an explicit enforcement point.

Traffic must be routed through the sidecar by the deployment architecture. The sidecar itself does **not** claim to provide an OS-level transparent interception boundary.

Prevention of direct upstream bypass is a deployment responsibility and should be enforced using mechanisms such as:

- network policy;
- firewall rules;
- service-mesh routing;
- container/network isolation;
- equivalent infrastructure controls.

---

## Verification Order

The implementation follows the normative HACP evaluation pipeline defined by the specification and crypto profile.

Conceptually, evaluation proceeds through:

1. schema and required-field validation;
2. token decision validation;
3. signer-key resolution;
4. signer-key revocation;
5. cryptographic signature verification;
6. envelope/token revocation;
7. temporal validity;
8. action binding;
9. request constraint binding;
10. scope containment;
11. semantic boundary evaluation;
12. human/checkpoint rules;
13. budget/replay state;
14. provenance acceptance;
15. upstream forwarding.

Any validation failure results in a deterministic `DENY`.

No request is forwarded after an enforcement failure.

---

## Repository Structure

```text
hacp-sidecar/
├── benchmarks/
│   ├── benchmark-rotating/
│   │   └── main.go
│   ├── generate-tokens/
│   │   └── main.go
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
│   ├── ARCHITECTURE.md
│   └── postmortems/
│       └── gate-d-benchmark.md
│
├── internal/
│   ├── budget/
│   ├── evaluate/
│   ├── provenance/
│   ├── proxy/
│   ├── scope/
│   └── wire/
│
├── go.mod
├── LICENSE.md
└── README.md
```

Generated benchmark tokens, benchmark result files, and local executables are development artifacts and are not intended to be committed.

---

## Requirements

### Required

- Go 1.22+
- Python 3.x for the reference/mock services and conformance harness

### Optional

- [`hey`](https://github.com/rakyll/hey) for HTTP load testing
- Docker for containerized deployment experiments

The native/reference topology does not require Docker.

---

## Build

### Sidecar

Linux/macOS:

```bash
go build -o hacp-sidecar ./cmd/sidecar
```

Windows:

```powershell
go build -o hacp-sidecar.exe ./cmd/sidecar
```

### Conformance runner

Linux/macOS:

```bash
go build -o hacp-conformance-runner ./cmd/hacp-conformance-runner
```

Windows:

```powershell
go build -o hacp-conformance-runner.exe ./cmd/hacp-conformance-runner
```

### Benchmark utilities

Linux/macOS:

```bash
go build -o benchmarks/generate-tokens ./benchmarks/generate-tokens
go build -o benchmarks/benchmark-rotating ./benchmarks/benchmark-rotating
```

Windows:

```powershell
go build -o .\benchmarks\generate-tokens.exe .\benchmarks\generate-tokens
go build -o .\benchmarks\benchmark_rotating.exe .\benchmarks\benchmark-rotating
```

---

## Native Reference Topology

The current reference setup uses four processes:

```text
┌────────────────────┐
│ Benchmark / Client │
└─────────┬──────────┘
          │
          ▼
┌────────────────────┐
│ HACP Sidecar :8080 │
└─────────┬──────────┘
          │
          ▼
┌────────────────────┐
│ Upstream      :8000│
└────────────────────┘

┌────────────────────┐
│ Control Plane :5000│
└────────────────────┘
```

### 1. Start mock upstream

Windows PowerShell:

```powershell
cd deployments
python upstream\server.py 8000
```

The reference upstream uses HTTP/1.1 and explicit `Content-Length` so persistent connections can be exercised correctly during performance tests.

### 2. Start reference control plane

```powershell
cd deployments
python control-plane\server.py 5000
```

### 3. Start HACP Sidecar

From the repository root:

```powershell
$env:HACP_SIDECAR_PORT="8080"
$env:HACP_UPSTREAM="http://127.0.0.1:8000"
$env:HACP_PROVENANCE_FLUSH_PATH="provenance.jsonl"

.\hacp-sidecar.exe
```

Linux/macOS:

```bash
export HACP_SIDECAR_PORT=8080
export HACP_UPSTREAM=http://127.0.0.1:8000
export HACP_PROVENANCE_FLUSH_PATH=provenance.jsonl

./hacp-sidecar
```

---

## Configuration

Current runtime configuration includes:

| Environment Variable | Default / Example | Description |
|---|---|---|
| `HACP_SIDECAR_PORT` | `8080` | HTTP listen port |
| `HACP_UPSTREAM` | `http://127.0.0.1:8000` | Upstream service |
| `HACP_PROVENANCE_FLUSH_PATH` | `provenance.jsonl` | Provenance flush target |

Additional evaluation defaults such as clock skew and revocation staleness are defined by the evaluation pipeline and will become part of the distributed Gate E configuration model.

---

## Test

### Unit Tests

```bash
go test ./...
```

Current expected result:

```text
ok      hacp-sidecar/internal/scope
```

Packages without dedicated unit-test files are still compiled as part of `go test ./...`.

### Static Analysis

```bash
go vet ./...
```

Both commands should complete without errors before committing changes.

---

## Manual Integration Test

Generate a valid HACP request:

```bash
go run ./cmd/gen-token > headers.txt
```

Then send it through the sidecar:

```bash
curl -i http://127.0.0.1:8080/api/test \
  -H "X-HACP-Intent-Envelope: $(grep 'X-HACP-Intent-Envelope' headers.txt | cut -d: -f2-)" \
  -H "X-HACP-Decision-Token: $(grep 'X-HACP-Decision-Token' headers.txt | cut -d: -f2-)"
```

Expected result:

```http
HTTP/1.1 200 OK
X-HACP-Decision: ALLOW
```

---

## Conformance

### Gate A — Protocol Correctness

**Status: ✅ Closed**

The sidecar passes all current HACP-Core conformance vectors.

```text
RESULTS: 38/38 passed
```

### Build runner

```powershell
go build -o hacp-conformance-runner.exe ./cmd/hacp-conformance-runner
```

### Run harness

From the `hacp-spec` repository:

```powershell
cd C:\Personal\GitHub\Dev\hacp-spec\harness
python .\harness.py
```

Expected:

```text
RESULTS: 38/38 passed
```

The harness communicates with implementations through the language-neutral HACP conformance runner protocol.

---

## Conformance Alignment

To satisfy the HACP-Core vectors, the implementation includes:

1. **Canonical JSON processing**  
   JSON number handling preserves the representation required by signature verification and canonicalization.

2. **Clock handling**  
   Evaluation can use conformance-provided policy clock context for deterministic temporal tests.

3. **Action hash binding**  
   Decision tokens bind to the proposed action through the HACP-defined action hash.

4. **State isolation**  
   Conformance state is isolated so budget/replay state does not leak between independent test vectors.

5. **Principal/checkpoint semantics**  
   Evaluation preserves the HACP distinction between the acting principal and human checkpoint/authorization semantics.

6. **Checkpoint propagation**  
   Checkpoint information is propagated through the evaluation pipeline.

7. **Human-required evaluation**  
   Human-required policy semantics are evaluated according to the normative ordering.

8. **Provenance binding**  
   Conformance responses preserve the required provenance/event bindings.

For the normative runner contract, see:

[`hacp-spec/harness/runner_protocol.md`](https://github.com/digital-humanism/hacp-spec/blob/main/harness/runner_protocol.md)

---

## Boundary Matrix

### Gate B — Semantic Completeness

**Status: ✅ Closed**

Security-relevant semantic transitions are evaluated through a data-driven boundary matrix.

The matrix operates across attributes such as:

- audience;
- externality;
- reversibility;
- data class;
- resource class;
- principal semantics.

Unknown security-relevant values fail closed rather than silently defaulting to a permissive interpretation.

Boundary evaluation lives under:

```text
internal/scope/
```

---

## Reference Deployment

### Gate C — Deployability

**Status: ✅ Closed**

Gate C is validated using the native/reference topology:

```text
Client
  ↓
HACP Sidecar
  ↓
Real HTTP Upstream
```

with a separate reference control-plane process.

The sidecar performs actual HTTP forwarding rather than local response emulation.

The upstream connection uses a dedicated pooled HTTP client with:

- persistent HTTP/1.1 connections;
- reusable idle connections;
- bounded connection management;
- hop-by-hop header filtering.

Docker/container deployment remains useful as a packaging option, but Gate C does not depend on Docker.

---

## Performance

### Gate D — Operational Viability

**Status: ✅ Closed**

Acceptance target:

```text
p99 HACP enforcement overhead < 5 ms
```

Performance is evaluated using several complementary workloads.

### Reference Results

| Benchmark | Workload | Result |
|---|---|---|
| Serial | 1000 requests, unique token/request | **p99 overhead 0.59 ms** |
| PowerShell rotating | 1000 requests, concurrency 5, unique token/request | **avg overhead 0.53 ms**, **p95 overhead 1.10 ms** |
| Go high-rate rotating | 5 × 1000 requests, concurrency 5, unique token/request | **median p99 overhead 1.26 ms** |
| `hey` shared-token | 1000 requests, concurrency 5 | sidecar **p99 3.3 ms**, ~**2761 req/s** |

All reference benchmark requests completed successfully.

### Serial Gate D Acceptance

Reference result:

```text
Baseline:
  p99: 4.04 ms

Sidecar:
  p99: 4.63 ms

p99 overhead:
  0.59 ms

GATE D PASS
```

### High-Rate Rotating-Token Characterization

The Go high-rate benchmark performs:

```text
1000 requests
concurrency = 5
unique DecisionToken per request
```

Repeated execution is used because isolated p99 measurements are sensitive to operating-system, scheduler, transport, and upstream tail latency.

Five-run reference p99 deltas:

```text
Run 1: -12.99 ms
Run 2:  +1.26 ms
Run 3:  +3.51 ms
Run 4: +10.85 ms
Run 5: -19.38 ms
```

Median:

```text
median p99 overhead = 1.26 ms
```

This is below the Gate D target of `5 ms`.

Negative values are not interpreted as the sidecar making requests faster. They indicate that the independent baseline sample experienced a larger stochastic tail in that run.

### Shared-Token Load Characterization

Reference `hey` result:

```text
Requests:    1000
Concurrency: 5

Baseline:
  Average:      1.7 ms
  p99:         33.0 ms
  Requests/sec: 2885.5

Sidecar:
  Average:      1.8 ms
  p99:          3.3 ms
  Requests/sec: 2760.6

Sidecar success:
  1000 / 1000
```

The baseline and sidecar workloads use the same HTTP keep-alive policy.

The isolated baseline p99 spike is treated as stochastic system/upstream tail behavior rather than sidecar acceleration.

---

## Running Benchmarks

### Serial Gate D benchmark

Windows:

```powershell
.\benchmarks\benchmark_serial.ps1
```

Purpose:

```text
Primary serial Gate D acceptance
1000 requests
unique token per request
```

### Concurrent rotating-token benchmark

```powershell
.\benchmarks\benchmark_rotating.ps1
```

Purpose:

```text
PowerShell concurrent characterization
1000 requests
concurrency 5
unique DecisionToken per request
```

### High-rate rotating-token benchmark

Build:

```powershell
go build `
  -o .\benchmarks\benchmark_rotating.exe `
  .\benchmarks\benchmark-rotating
```

Run:

```powershell
.\benchmarks\benchmark_rotating.exe
```

For regression characterization, run multiple iterations:

```powershell
1..5 | ForEach-Object {
    Write-Host ""
    Write-Host "================ RUN $_ ================"
    .\benchmarks\benchmark_rotating.exe
    Start-Sleep -Seconds 2
}
```

### Shared-token `hey` benchmark

```powershell
.\benchmarks\benchmark.ps1
```

This workload:

- generates fresh pre-signed HACP tokens;
- uses one shared DecisionToken for the `hey` sidecar workload;
- runs 1000 requests;
- uses concurrency 5;
- compares direct upstream traffic with sidecar traffic using the same keep-alive policy.

### Token Generator

Build:

```powershell
go build `
  -o .\benchmarks\generate-tokens.exe `
  .\benchmarks\generate-tokens
```

Example:

```powershell
.\benchmarks\generate-tokens.exe `
  -count 1000 `
  -out .\benchmarks\tokens.jsonl `
  -method GET `
  -path "/api/test"
```

The generated DecisionToken constraints include:

```json
{
  "method": "GET",
  "path": "/api/test",
  "max_uses": 99999
}
```

---

## Response Headers

### Allowed request

```http
HTTP/1.1 200 OK
X-HACP-Decision: ALLOW
```

### Denied request

```http
HTTP/1.1 403 Forbidden
X-HACP-Decision: DENY
X-HACP-Reason: SIGNATURE_FAILURE
```

Additional request/provenance identifiers may be returned depending on the execution path.

---

## Reason Codes

The sidecar uses deterministic HACP reason codes.

| Code | Condition |
|---|---|
| `INVALID_ENVELOPE` | Missing or malformed intent envelope |
| `INVALID_ACTION` | Missing or malformed proposed action |
| `KEY_REVOKED` | Signer key is revoked |
| `SIGNATURE_FAILURE` | Signature verification failed |
| `ENVELOPE_REVOKED` | Intent envelope is revoked |
| `TOKEN_REVOKED` | Decision token is revoked |
| `ENVELOPE_EXPIRED` | Envelope exceeded `expires_at` |
| `TOKEN_EXPIRED` | Token exceeded `expires_at` |
| `SCOPE_EXCEEDED` | Proposed action is outside authorized scope/binding |
| `BOUNDARY_CROSSING` | Semantic boundary transition is not permitted |
| `UNKNOWN_ATTRIBUTE` | Unknown security-relevant attribute |
| `BUDGET_EXHAUSTED` | Token/autonomy budget exhausted |
| `HUMAN_REQUIRED` | Human authorization/checkpoint required |
| `CHECKPOINT_TIMEOUT` | Required checkpoint timed out |
| `TRACEABILITY_FAILURE` | Provenance/audit acceptance failed |
| `POLICY_DENIED` | Policy explicitly denied execution |

The normative definitions are maintained by `hacp-spec`.

---

## Security Model

The agent runtime is treated as potentially hostile.

The sidecar is responsible for validating the HACP authorization attached to traffic that reaches the enforcement point.

Core protections include:

- fail-closed validation;
- cryptographic authorization;
- request/action binding;
- semantic scope enforcement;
- boundary-matrix evaluation;
- signer/envelope/token revocation;
- budget and replay state;
- provenance-before-forwarding;
- deterministic denial reasons.

### Network bypass

Routing through the sidecar alone is not an operating-system security boundary.

A deployment that requires bypass resistance must additionally prevent the agent from establishing unauthorized direct connections to protected upstream services.

Possible enforcement mechanisms include:

- Kubernetes NetworkPolicy;
- firewall rules;
- service-mesh policy;
- namespace/network isolation;
- equivalent infrastructure controls.

---

## Out of Scope for Current MVP

- eBPF enforcement
- transparent iptables interception
- dynamic-library injection
- agent bytecode modification
- full OS-level process isolation
- generic non-HTTP transports
- production distributed control plane
- production distributed replay/budget database

These may be implemented in later phases without changing the core HACP authorization semantics.

---

## Docker

Containerized deployment is optional.

If the repository contains a Docker Compose topology, it may be started with:

```bash
docker-compose -f deployments/docker-compose.yml up
```

The authoritative Gate C reference topology is currently the native multi-process setup documented above.

---

## Development Workflow

Before committing runtime changes:

```bash
go test ./...
go vet ./...
```

Then rebuild the affected executables.

For changes that affect HACP semantics:

1. run `go test ./...`;
2. run `go vet ./...`;
3. rebuild the conformance runner;
4. run the HACP conformance suite;
5. verify `38/38`;
6. run relevant performance regression tests if the runtime path changed.

Example:

```powershell
gofmt -w .\internal\proxy\handler.go
gofmt -w .\internal\evaluate\pipeline.go

go test ./...
go vet ./...

go build -o hacp-sidecar.exe ./cmd/sidecar
go build -o hacp-conformance-runner.exe ./cmd/hacp-conformance-runner
```

Runtime changes must not regress normative conformance.

---

## Gate Discipline

Current development status:

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
⏸ Pending
```

New features should preserve this separation.

Typical ownership:

```text
Gate B / semantic enforcement
→ internal/scope/

Gate C / deployment and proxy behavior
→ deployments/
→ internal/proxy/

Gate D / performance regression
→ benchmarks/

Gate E / distributed control-plane integration
→ control-plane/runtime management components
```

---

## Documentation

Architecture:

```text
docs/ARCHITECTURE.md
```

Gate D benchmark analysis:

```text
docs/postmortems/gate-d-benchmark.md
docs/postmortems/gate-d-performance-validation.md
```

Normative HACP specification:

[`digital-humanism/hacp-spec`](https://github.com/digital-humanism/hacp-spec)

---

## License

This repository contains the HACP Enforcement Sidecar implementation and is licensed under the **GNU Affero General Public License v3.0 (AGPLv3)**, with Commercial Dual Licensing available for enterprise deployments, closed-source embedding, and OEM integration.

For commercial licensing inquiries:

`digital.humanism.collective@protonmail.com`

### Relationship to other HACP repositories

| Repository | Purpose | License |
|---|---|---|
| [`hacp-spec`](https://github.com/digital-humanism/hacp-spec) | Open protocol specification, schemas, and conformance suite | CC BY 4.0 |
| [`humanist-core`](https://github.com/digital-humanism/humanist-core) | Reference SDK | AGPLv3 + commercial dual |
| [`hacp-sidecar`](https://github.com/digital-humanism/hacp-sidecar) | Enforcement sidecar | AGPLv3 + commercial dual |

The separation allows the HACP protocol itself to remain an open, vendor-neutral specification while reference tooling and enforcement implementations can maintain an independent sustainability model.

### AGPLv3 network deployment

AGPLv3 contains source-disclosure obligations for modified software made available for interaction over a network.

Organizations requiring different licensing terms for proprietary deployments, embedding, redistribution, or OEM integration may use a commercial license.

See [`LICENSE.md`](LICENSE.md) for repository licensing terms.

---

## Contributing

Contributions must preserve:

- normative HACP conformance;
- fail-closed semantics;
- deterministic denial behavior;
- security-relevant evaluation ordering;
- boundary-matrix behavior;
- provenance-before-forwarding guarantees;
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

Performance-sensitive changes should also run the Gate D benchmark suite.

---

## Contact

**Digital Humanism Collective**

`digital.humanism.collective@protonmail.com`