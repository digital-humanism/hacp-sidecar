# HACP Sidecar Architecture

## Overview

HACP Sidecar is a fail-closed enforcement proxy that intercepts AI agent requests
and verifies HACP cryptographic tokens before forwarding to upstream tools.

The implementation follows the HACP-Enforcement profile defined in
[`hacp-spec`](https://github.com/digital-humanism/hacp-spec).

## Package Layout

```
hacp-sidecar/
├── cmd/
│   ├── sidecar/                   # HTTP enforcement proxy (main entry point)
│   ├── gen-token/                 # Test token generator utility
│   └── hacp-conformance-runner/   # Conformance test adapter (stdin/stdout JSON)
│
└── internal/
    ├── wire/                      # HACP wire protocol parsing
    │   ├── envelope.go            # IntentEnvelope parsing and validation
    │   ├── token.go               # DecisionToken parsing
    │   ├── jcs.go                 # RFC 8785 JSON Canonicalization (JCS)
    │   ├── crypto.go              # Ed25519 signature verification
    │   ├── headers.go             # HTTP header extraction
    │   └── key_resolver.go        # Static key resolver for test keys
    │
    ├── evaluate/                  # Normative verification pipeline
    │   ├── interfaces.go          # Core types (Decision, RequestContext, etc.)
    │   ├── pipeline.go            # 10-step evaluation pipeline
    │   ├── revocation.go          # In-memory revocation store
    │   └── scope.go               # Scope guard (basic validation)
    │
    ├── budget/                    # Autonomy budget tracking
    │   └── ledger.go              # Atomic token consumption ledger
    │
    ├── provenance/                # Audit trail
    │   ├── ring.go                # Ring buffer with async flush
    │   └── noop.go                # No-op writer for conformance tests
    │
    └── proxy/                     # HTTP reverse proxy
        └── handler.go             # Request interception and forwarding
```

## Components

### `cmd/sidecar` — HTTP Enforcement Proxy

Main entry point for production use. Runs as a long-lived process that:
- Listens on configurable HTTP port (default 8080)
- Intercepts incoming requests with HACP headers
- Invokes evaluation pipeline
- Forwards ALLOW decisions to upstream
- Returns DENY with deterministic reason codes
- Flushes provenance buffer periodically

**Graceful shutdown:** SIGTERM/SIGINT triggers provenance flush and clean exit.

### `cmd/gen-token` — Test Token Generator

Utility for generating valid HACP headers for manual testing:
- Generates Ed25519 keypair (or uses fixed test key)
- Creates IntentEnvelope with configurable scope
- Creates DecisionToken bound to proposed action
- Outputs curl-ready headers

Used for integration testing and development.

### `cmd/hacp-conformance-runner` — Conformance Adapter

Adapts the sidecar to the language-neutral runner protocol defined in
[`hacp-spec/harness/runner_protocol.md`](https://github.com/digital-humanism/hacp-spec/blob/main/harness/runner_protocol.md).

**Responsibilities:**
- Reads JSON requests from stdin (one per line)
- Parses `intent_envelope`, `proposed_action`, `decision_token`
- Invokes `evaluate.Pipeline`
- Returns JSON responses on stdout
- Isolates state per `vector_id` (resets budget ledger)
- Writes diagnostics to stderr (never stdout)

**Key implementation details:**
- Uses `json.RawMessage` to preserve original bytes (prevents canonicalization issues)
- Extracts `policy_context.clock` for reproducible timestamp validation
- Passes `checkpoint` context for runtime vectors

### `internal/wire` — Wire Protocol

Handles HACP header parsing and cryptographic operations.

**`envelope.go`:**
- Parses `X-HACP-Intent-Envelope` header (base64url → JSON)
- Validates schema (required fields, types)
- Computes canonical payload (JCS)
- Provides `RemoveSignatureField` for verification

**`token.go`:**
- Parses `X-HACP-Decision-Token` header
- Validates schema
- Extracts `action_hash`, `constraints`, `decision`

**`jcs.go`:**
- Implements RFC 8785 JSON Canonicalization Scheme
- Uses `json.Decoder.UseNumber()` to preserve integer precision
- Handles `canonicalizeNumber` without scientific notation
- Provides `CanonicalizeJSON` and `VerifyCanonicalization`

**Critical fix:** `canonicalizeNumber` uses `strconv.FormatInt` for integers
instead of `strconv.FormatFloat` with `'g'` format, which produces scientific
notation for large numbers (e.g., `1.786e+09` instead of `1786000000`).

**`crypto.go`:**
- `VerifyEd25519` — RFC 8032 pure Ed25519 signature verification
- `SHA256`, `SHA256Hex` — hash utilities
- `Base64URLDecode`, `Base64URLEncode` — base64url without padding
- `ParseEd25519PublicKey` — hex to public key

**`key_resolver.go`:**
- `StaticKeyResolver` — in-memory key registry
- `AddKeyFromHex` — load test keys for conformance

### `internal/evaluate` — Verification Pipeline

Implements the normative 10-step verification pipeline per
`HACP-SPEC-0.9-draft.md` §5.1 and `wire/crypto-profile.md`.

**`pipeline.go` — Evaluation Steps:**

```
1. Schema validation (required fields, types)
   ↓
2. Key resolution + revocation check
   ↓ KEY_REVOKED if revoked
3. Signature verification (envelope + token)
   ↓ SIGNATURE_FAILURE if invalid
4. Envelope/token revocation checks
   ↓ ENVELOPE_REVOKED or TOKEN_REVOKED
5. Expiry validation (envelope.expires_at, token.expires_at)
   ↓ ENVELOPE_EXPIRED or TOKEN_EXPIRED
6. Action hash binding (token.action_hash == SHA-256(JCS(proposed_action)))
   ↓ SIGNATURE_FAILURE if mismatch
7. Request binding (token.constraints match request)
   ↓ SCOPE_EXCEEDED if mismatch
8. Scope containment (action within envelope.scope)
   ↓ SCOPE_EXCEEDED if outside
9. Budget/replay protection (consume token from ledger)
   ↓ BUDGET_EXHAUSTED if consumed
10. Provenance acceptance (record decision before forward)
    ↓ TRACEABILITY_FAILURE if buffer full
    ↓
    FORWARD or DENY
```

**Fail-closed mandate:** Any validation failure results in `DENY` with a
deterministic reason code from `hacp-spec/error-model.md`.

**`interfaces.go`:**
- `Decision` — evaluation result (`Allow`, `ReasonCode`, `Error`)
- `RequestContext` — request metadata (`Method`, `Path`, `RequestID`, `Clock`, `ProposedAction`)
- `RevocationStore` — interface for revocation checks
- `ScopeGuard` — interface for scope validation
- `ProvenanceWriter` — interface for provenance recording

**`revocation.go`:**
- `InMemoryRevocationStore` — simple in-memory store for MVP
- `RevokeToken`, `RevokeEnvelope`, `RevokeKey` — revocation API
- `IsTokenRevoked`, `IsEnvelopeRevoked`, `IsKeyRevoked` — check API

**`scope.go`:**
- `DefaultScopeGuard` — basic scope validation
- Checks action attributes against envelope scope
- Returns `SCOPE_EXCEEDED` if outside boundaries
- **Pending (Gate B):** full boundary matrix with data-driven transitions

### `internal/budget` — Autonomy Budget

Implements token consumption ledger for replay protection.

**`ledger.go`:**
- `Ledger` — atomic token consumption tracker
- `ConsumeToken(tokenID)` — mark token as consumed
- `IsTokenConsumed(tokenID)` — check if already used
- `Reset()` — clear state (used by conformance runner per vector_id)

**Concurrency:** Uses `sync.Mutex` for thread-safe access.

### `internal/provenance` — Audit Trail

Records all decisions (ALLOW and DENY) for audit and forensics.

**`ring.go`:**
- `RingBuffer` — fixed-capacity circular buffer
- `Accept(event)` — record provenance event
- `Flush()` — write buffer to disk asynchronously
- **Fail-closed:** if buffer cannot accept, request is denied

**`Event` structure:**
```go
type Event struct {
    Timestamp   int64  `json:"timestamp"`
    RequestID   string `json:"request_id"`
    EnvelopeID  string `json:"envelope_id"`
    TokenID     string `json:"token_id"`
    ActionHash  string `json:"action_hash"`
    Decision    string `json:"decision"`
    ReasonCode  string `json:"reason_code,omitempty"`
    LatencyNs   int64  `json:"latency_ns"`
}
```

**`noop.go`:**
- `NoopWriter` — discards all events (used by conformance runner)
- Always returns success (never fails)

### `internal/proxy` — HTTP Reverse Proxy

Intercepts HTTP requests and enforces HACP policy.

**`handler.go`:**
- Extracts `X-HACP-Intent-Envelope` and `X-HACP-Decision-Token` headers
- Parses base64url → JSON → structs
- Invokes `evaluate.Pipeline`
- On ALLOW: forwards request to upstream, adds `X-HACP-Request-Id`
- On DENY: returns 403 Forbidden with `X-HACP-Decision` and `X-HACP-Reason`
- Records provenance event before forwarding

**Upstream forwarding:**
- Preserves original request method, path, headers, body
- Adds `X-HACP-Request-Id` for correlation
- Strips `X-HACP-*` headers before forwarding (upstream doesn't need them)

## Key Architectural Decisions

### 1. Fail-Closed by Default

Any validation failure results in `DENY`. There is no "open on error" mode.

**Rationale:** Security failures are silent. A sidecar that allows requests on
error creates a bypass that attackers can exploit.

### 2. Deterministic Reason Codes

Every DENY includes a reason code from `hacp-spec/error-model.md`:
- `INVALID_ENVELOPE` — schema violation
- `SIGNATURE_FAILURE` — crypto failure
- `TOKEN_REVOKED` — revocation
- `SCOPE_EXCEEDED` — boundary violation
- etc.

**Rationale:** Enables debugging and policy enforcement without exposing
internal state.

### 3. Provenance Before Forward

The provenance event is recorded **before** the request is forwarded upstream.

**Rationale:** If the sidecar crashes after forwarding but before recording,
there is no audit trail. Recording first ensures all decisions are logged.

### 4. No External Dependencies

The sidecar uses only Go stdlib (no external crypto libraries, no HTTP frameworks).

**Rationale:** Minimizes attack surface, simplifies deployment, ensures
reproducibility across platforms.

### 5. Async Provenance Flush

Provenance events are written to a ring buffer in memory and flushed to disk
asynchronously by a background goroutine.

**Rationale:** Synchronous disk I/O on every request would add unacceptable
latency. The ring buffer provides backpressure (fail-closed if full).

### 6. Policy Preflight

For human-required actions, the sidecar checks `principal_kind` and `verb`
**before** full envelope parsing.

**Rationale:** Prevents false `INVALID_ENVELOPE` errors when the envelope
contains dummy signatures for testing. The policy decision (CHECKPOINT) is
made before crypto verification.

### 7. Clock from Policy Context

The evaluation pipeline uses `RequestContext.Clock` (from `policy_context.clock`)
instead of `time.Now()` for expiry checks.

**Rationale:** Enables reproducible conformance tests with fixed timestamps.
In production, `Clock` is set to `time.Now().Unix()`.

### 8. State Isolation in Conformance Runner

The conformance runner resets the budget ledger on each new `vector_id`.

**Rationale:** Prevents cross-test contamination. Each vector should be
evaluated in isolation, not affected by previous vectors.

## Conformance Architecture

The sidecar passes 38/38 HACP-Core conformance vectors through 8 architectural
alignments:

1. **JCS canonicalization** — `json.Decoder.UseNumber()` preserves integer precision
2. **Clock handling** — `policy_context.clock` for reproducible tests
3. **Action hash binding** — SHA-256(JCS(proposed_action)), not envelope
4. **State isolation** — budget ledger reset per vector_id
5. **Principal bindings** — human checkpoint resolution enabled
6. **Checkpoint propagation** — full checkpoint context passed
7. **Policy preflight** — human-required check before crypto
8. **Provenance binding** — response includes provenance event ID

These fixes ensure the sidecar correctly implements the HACP-Core specification
as verified by the canonical conformance suite.

## Threat Model

The sidecar assumes the agent runtime is **potentially hostile** and attempts
to bypass controls. Mitigations include:

- **Explicit HTTP_PROXY** — agent cannot bypass sidecar without network policy
- **Fail-closed** — all validation failures result in DENY
- **Provenance first** — no forward without audit
- **Revocation staleness** — bounded threshold for control plane sync
- **No dynamic loading** — all policy is compiled in (no plugin system)

Full threat model: [`hacp-spec/threat-model.md`](https://github.com/digital-humanism/hacp-spec/blob/main/threat-model.md) sections 3.6–3.14.

## Current Status

**Gates A–D closed.** 

- Gate A: 38/38 conformance vectors pass
- Gate B: Data-driven boundary matrix implemented in `internal/scope/matrix.go`
- Gate C: Docker Compose reference deployment with demo scenarios
- Gate D: p99 overhead validated (see [postmortem](../engineering/gate-d-benchmark.md) and [technical report](../engineering/gate-d-performance-validation.md))

## Future Work

### Gate E: Distributed Control Plane

Replace the current HTTP-based control plane with authenticated gRPC streaming for real-time policy management.

**Planned features:**
- Bidirectional streaming for real-time revocations (envelopes, tokens, keys)
- Policy snapshot sync on sidecar startup
- Delta updates for incremental policy changes
- Health checks and operational telemetry
- Distributed budget ledger with eventual consistency

**Implementation plan:**
- `proto/controlplane.proto` — gRPC service definitions
- `internal/controlplane/client.go` — streaming gRPC client
- `internal/evaluate/revocation.go` — integrate streaming revocation updates
- `cmd/control-plane/` — reference control plane server
- Hybrid architecture: periodic snapshot pull + push notifications for instant revocation

## References

- HACP Specification: [`hacp-spec/HACP-SPEC-0.9-draft.md`](https://github.com/digital-humanism/hacp-spec/blob/main/HACP-SPEC-0.9-draft.md)
- Wire Protocol: [`hacp-spec/wire/encoding.md`](https://github.com/digital-humanism/hacp-spec/blob/main/wire/encoding.md)
- Crypto Profile: [`hacp-spec/wire/crypto-profile.md`](https://github.com/digital-humanism/hacp-spec/blob/main/wire/crypto-profile.md)
- Error Model: [`hacp-spec/error-model.md`](https://github.com/digital-humanism/hacp-spec/blob/main/error-model.md)
- Conformance Suite: [`hacp-spec/CONFORMANCE-SUITE.md`](https://github.com/digital-humanism/hacp-spec/blob/main/CONFORMANCE-SUITE.md)
- Runner Protocol: [`hacp-spec/harness/runner_protocol.md`](https://github.com/digital-humanism/hacp-spec/blob/main/harness/runner_protocol.md)

**Contact:** [digital.humanism.collective@protonmail.com](mailto:digital.humanism.collective@protonmail.com)
