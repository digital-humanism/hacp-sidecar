# HACP Sidecar

Enforcement sidecar for HACP (Human Agency Continuity Protocol).

Implements the HACP-Enforcement profile per [`hacp-spec`](https://github.com/digital-humanism/hacp-spec).

## Overview

HACP Sidecar is a fail-closed enforcement point that sits between an AI agent and external tools. It ensures that no action is executed without a valid, cryptographically signed HACP decision token.

The sidecar intercepts MCP tool calls and HTTP requests, verifies `X-HACP-Intent-Envelope` and `X-HACP-Decision-Token` headers, and either forwards the request upstream or denies it with a deterministic reason code.

## Scope (Phase 4 MVP)

- MCP tool calls routed through the sidecar
- HTTP requests sent through an explicit `HTTP_PROXY` to the sidecar
- Verification of `X-HACP-Intent-Envelope` and `X-HACP-Decision-Token`
- Scope guard checks and boundary matrix evaluation
- Budget and replay protection
- Ed25519 signature verification (RFC 8032, pure mode)
- Revocation via authenticated control channel (gRPC streaming)
- Local provenance ring buffer with asynchronous flush
- Docker Compose demonstration with agent, sidecar, mock control plane, and mock upstream
- Allow-path latency benchmark (p99 overhead recorded)

## Out of Scope

- eBPF enforcement (planned post-Gate D)
- Transparent iptables or network-layer interception
- Dynamic library injection
- Agent bytecode modification
- Full OS-level process isolation
- Generic non-HTTP tool transports

## Architecture

```
┌─────────────┐
│   Agent     │
│ (untrusted) │
└──────┬──────┘
       │ HTTP/MCP
       │ (X-HACP-Intent-Envelope + X-HACP-Decision-Token)
       ▼
┌─────────────────────────────────────────┐
│          HACP Sidecar                   │
│                                         │
│  1. Parse headers (base64url → JSON)    │
│  2. Schema validation                   │
│  3. Key revocation check                │
│  4. Ed25519 signature verification      │
│  5. Envelope/token revocation check     │
│  6. Expiry validation                   │
│  7. Action hash binding                 │
│  8. Request binding (constraints)       │
│  9. Scope guard + boundary matrix       │
│ 10. Budget + replay protection          │
│ 11. Provenance ring buffer accept       │
│ 12. Forward or DENY                     │
└──────────┬──────────────┬───────────────┘
           │              │
           │ HTTP/MCP     │ gRPC stream
           ▼              ▼
    ┌────────────┐  ┌─────────────┐
    │  Upstream  │  │   Control   │
    │   Tool     │  │    Plane    │
    └────────────┘  └─────────────┘
```

## Verification Order (Normative)

Per `HACP-SPEC-0.9-draft.md` §5.1 and `wire/crypto-profile.md`:

1. Schema / required fields
2. Key resolution and revocation check (`KEY_REVOKED` if revoked)
3. Signature verification (only if key is not revoked)
4. Envelope/token revocation checks
5. Expiry validation
6. Token binding (`action_hash` + `constraints`)
7. Scope containment and boundary matrix
8. Budget and replay state
9. Provenance ring buffer acceptance
10. Forward request

The pipeline is strictly fail-closed: any validation failure results in `DENY` with a deterministic reason code.

## Requirements

- Go 1.22+
- Docker (for `docker-compose` testing)

## Build

```bash
go build -o hacp-sidecar ./cmd/sidecar
```

## Run

```bash
export HACP_SIDECAR_PORT=8080
./hacp-sidecar
```

## Test

```bash
go test ./...
```

## Docker Compose

```bash
docker-compose -f deployments/docker-compose.yml up
```

Spins up:
- `agent` — mock agent sending requests through sidecar
- `sidecar` — HACP enforcement sidecar
- `control-plane` — mock gRPC control plane for revocation
- `upstream` — mock upstream tool receiving forwarded requests

## Configuration

| Environment Variable | Default | Description |
|---|---|---|
| `HACP_SIDECAR_PORT` | `8080` | HTTP listen port |
| `HACP_SIDECAR_MODE` | `enforce` | `enforce`, `shadow`, or `disabled` |
| `HACP_CONTROL_ENDPOINT` | `localhost:50051` | gRPC control plane endpoint |
| `HACP_REVOCATION_STALENESS_MS` | `5000` | Max revocation staleness before fail-closed |
| `HACP_CLOCK_SKEW_SECONDS` | `5` | Max clock skew tolerance |
| `HACP_PROVENANCE_BUFFER_SIZE` | `10000` | Ring buffer capacity |
| `HACP_PROVENANCE_FLUSH_PATH` | `/var/log/hacp/provenance.jsonl` | Async flush target |

## Response Headers

On denied requests, the sidecar returns:

```http
HTTP/1.1 403 Forbidden
X-HACP-Decision: DENY
X-HACP-Reason: SIGNATURE_FAILURE
X-HACP-Request-Id: req-abc123
```

On allowed requests:

```http
HTTP/1.1 200 OK
X-HACP-Decision: ALLOW
X-HACP-Request-Id: req-def456
```

## Reason Codes

The sidecar uses deterministic reason codes from `hacp-spec/error-model.md`:

| Code | Condition |
|---|---|
| `INVALID_ENVELOPE` | Missing/malformed envelope header |
| `INVALID_ACTION` | Missing/malformed action data |
| `KEY_REVOKED` | Signer key revoked |
| `SIGNATURE_FAILURE` | Invalid Ed25519 signature |
| `ENVELOPE_REVOKED` | Envelope in revocation denylist |
| `TOKEN_REVOKED` | Token in revocation denylist or replayed |
| `ENVELOPE_EXPIRED` | Envelope past `expires_at` |
| `TOKEN_EXPIRED` | Token past `expires_at` |
| `SCOPE_EXCEEDED` | Request outside token scope/binding |
| `BOUNDARY_CROSSING` | Boundary matrix violation |
| `UNKNOWN_ATTRIBUTE` | Unrecognized security-relevant attribute |
| `BUDGET_EXHAUSTED` | Autonomy budget consumed |
| `HUMAN_REQUIRED` | Human-required action by system principal |
| `CHECKPOINT_TIMEOUT` | Checkpoint unresolved past deadline |
| `TRACEABILITY_FAILURE` | Provenance or audit chain failure |
| `POLICY_DENIED` | Explicit policy denial |

## Conformance

The sidecar is designed to pass all `hacp-spec` conformance vectors, including:

- 38 Core conformance vectors (INV-1 through INV-5, INV-7)
- Runtime profile vectors (checkpoint state machine)
- Enforcement-specific binding and replay tests

## Security Model

The sidecar assumes the agent runtime is **potentially hostile** and attempts to bypass controls. Mitigations include:

- Explicit `HTTP_PROXY` configuration (agent cannot bypass sidecar)
- Network policy blocking direct egress
- Fail-closed on all validation failures
- Provenance acceptance before forwarding (no forward without audit)
- Revocation staleness check with bounded threshold

See `hacp-spec/threat-model.md` sections 3.6–3.14 for full enforcement threat model.

## License

This repository contains the HACP Enforcement Sidecar implementation, licensed under the **GNU Affero General Public License v3.0 (AGPLv3)**, with Commercial Dual Licensing available for enterprise deployments, closed-source embedding, and OEM integration.

For commercial licensing inquiries, contact: `digital.humanism.collective@protonmail.com` (placeholder).

### Relationship to other HACP repositories

| Repository | Purpose | License |
|---|---|---|
| [`hacp-spec`](https://github.com/digital-humanism/hacp-spec) | Open standard (specification, schemas, conformance suite) | CC BY 4.0 |
| [`humanist-core`](https://github.com/digital-humanism/humanist-core) | Reference SDK (Python) | AGPLv3 + commercial dual |
| [`hacp-sidecar`](https://github.com/digital-humanism/hacp-sidecar) | Enforcement sidecar (Go) | AGPLv3 + commercial dual |

This deliberate separation ensures the protocol remains a vendor-neutral open standard, while reference tooling, enforcement implementations, and enterprise integrations maintain a sustainable commercial model.

### AGPLv3 notice for network deployment

Under AGPLv3 §13, any network-accessible deployment of this sidecar that serves external users requires the source code of the deployed version (including any modifications) to be made available to those users. Commercial licenses waive this requirement.

## Contributing

This repository follows the HACP gate discipline:

- Phase 4.1 (Design): ✅ Complete
- Phase 4.2 (MVP Implementation): In progress
- Phase 4.3 (Gate D Exit): Pending

All changes must maintain conformance with the normative `hacp-spec` documents.
