# Gate E — Distributed Control Plane

**Status:** Complete  
**Phase:** 4b  
**Component:** `hacp-sidecar`  
**Protocol:** HACP distributed revocation control plane over gRPC

## Purpose

Gate E extends the HACP sidecar from a locally evaluated enforcement point into a distributed enforcement system capable of receiving, recovering, and safely applying revocation state from a shared control plane.

The design preserves the HACP fail-closed property while avoiding an unnecessary immediate outage on short-lived control-plane disconnects.

## Architecture

```text
                    Control Plane
                         │
                  durable revision
                         │
                  gRPC server stream
                         │
              ┌──────────┴──────────┐
              ▼                     ▼
         HACP Sidecar A        HACP Sidecar B
              │                     │
       local revocation        local revocation
            state                   state
              │                     │
       ControlState            ControlState
              │                     │
          Pipeline                Pipeline
```

Each sidecar maintains its own fully materialized local revocation state.

The control plane provides ordering and recovery; it is not consulted synchronously for every protected action.

## Protocol contract

The distributed control-plane contract is defined in:

```text
hacp-spec/proto/hacp/control/v1/control_plane.proto
```

The service exposes:

- `GetRevocationSnapshot`
- `WatchRevocations`

Supported revocation kinds:

- key
- token
- envelope
- parent envelope

The watch stream can carry:

- revocation events;
- heartbeats;
- `ResetRequired` recovery instructions.

## Revision semantics

`revision` is the durable global ordering of control-plane mutations.

`sequence` is transport-local ordering within one stream and is not persisted.

For a sidecar whose fully materialized revision is `R`:

```text
event.revision == R + 1
→ apply
→ advance to R + 1

event.revision <= R
→ duplicate or old event
→ ignore

event.revision > R + 1
→ revision gap
→ unsafe / fail closed
→ recover
```

A sidecar MUST NOT advance `last_seen_revision` until the corresponding revocation state has been successfully materialized.

`last_seen_revision` therefore always means the highest fully materialized durable revision, not merely the highest observed revision.

## Snapshot and stream

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
WatchRevocations(after_revision=last_seen_revision)
    ↓
replay missed events
    ↓
live stream
```

If required replay history is no longer retained:

```text
ResetRequired
    ↓
GetRevocationSnapshot
    ↓
replace local state atomically
    ↓
resume stream
```

## Authoritative journal

The control plane maintains an authoritative monotonic revision journal.

A new revocation mutation consumes exactly one new durable revision.

Duplicate revocation of an already-revoked subject is idempotent and does not consume a new revision.

Snapshot state and replay history are separated: replay history may be compacted while the authoritative snapshot remains complete.

## Freshness and fail-closed behavior

Each sidecar tracks:

- last fully materialized revision;
- last valid control-state update time;
- connection state;
- unsafe state;
- maximum permitted staleness.

A network disconnect alone does not immediately revoke local authority.

The last fully materialized state may continue to be used while it remains fresh.

When the configured maximum staleness interval is exceeded:

```text
DENY
CONTROL_STATE_STALE
```

Revision gaps, inconsistent heartbeats, and invalid distributed state may mark the control state unsafe immediately.

Standalone/conformance evaluation remains compatible: when no distributed `ControlState` guard is configured, the existing evaluation behavior is preserved.

## Heartbeats

Heartbeats prove freshness without advancing durable revision state.

A heartbeat is valid only when its control-plane revision matches the sidecar's fully materialized revision.

Heartbeats MUST NOT be used to skip missing revocation events.

If a heartbeat claims a different revision than the sidecar has fully materialized, the control state becomes unsafe and authorization fails closed until recovery succeeds.

## Reconnect behavior

Reconnect uses bounded exponential backoff.

The implementation verifies:

- initial retry delay;
- exponential growth;
- maximum delay cap;
- prompt cancellation through `context.Context`.

## Distributed revocation propagation

Gate E verifies the complete runtime path:

```text
valid signed authority
      ↓
Pipeline
      ↓
ALLOW

Control Plane
      ↓
revoke token
      ↓
gRPC stream
      ↓
Subscriber
      ↓
local RevocationStore
      ↓
same Pipeline / same signed authority
      ↓
DENY / TOKEN_REVOKED
```

This proves that distributed control-plane state is materialized into the real evaluator path rather than only into an isolated test store.

## Multi-sidecar convergence

Gate E verifies two independent sidecars connected to the same control plane.

After one token revocation:

```text
control-plane revision = N

sidecar A:
last_seen_revision = N
token revoked
DENY / TOKEN_REVOKED

sidecar B:
last_seen_revision = N
token revoked
DENY / TOKEN_REVOKED
```

This demonstrates deterministic convergence on both distributed state and security outcome.

## Recovery guarantees

The recovery model is:

```text
normal:
snapshot → stream

disconnect:
last_seen_revision → reconnect → replay → stream

replay history unavailable:
ResetRequired → snapshot → stream
```

A revision gap, invalid revocation kind, inconsistent heartbeat, or stale state is treated as unsafe rather than silently tolerated.

## Validation

Run the Gate E suite:

```bash
go test ./internal/controlplane -count=1 -v
```

Run the complete sidecar regression suite:

```bash
go test ./... -count=1
```

Both were passing at Gate E completion.

CI executes both commands through GitHub Actions.

## Gate E exit criteria

| Criterion | Status |
|---|---|
| gRPC control-plane protocol | ✅ |
| Durable revision semantics | ✅ |
| Authoritative revision journal | ✅ |
| Snapshot bootstrap | ✅ |
| Live revocation stream | ✅ |
| Duplicate / old event handling | ✅ |
| Revision-gap detection | ✅ |
| Reconnect + replay | ✅ |
| Bounded reconnect backoff | ✅ |
| Missed-event recovery | ✅ |
| `ResetRequired` snapshot recovery | ✅ |
| Heartbeat freshness | ✅ |
| Stale fail-closed | ✅ |
| Real evaluator revocation propagation | ✅ |
| Multi-sidecar convergence | ✅ |
| Full Go regression | ✅ |
| GitHub Actions integration | ✅ |

## Result

Gate E establishes a distributed revocation control plane in which HACP sidecars can continue operating through short control-plane interruptions, recover deterministically from missed history, converge across replicas, and fail closed when distributed authority state can no longer be considered fresh.

**Gate E / Phase 4b is complete.**


---

**Contact:** [digital.humanism.collective@protonmail.com](mailto:digital.humanism.collective@protonmail.com)