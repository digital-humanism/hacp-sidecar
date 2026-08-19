# Distributed Control Plane Architecture

## Status

**Gate E / Phase 4b — Complete**

This document describes the stable distributed-control-plane architecture of `hacp-sidecar`.

It intentionally focuses on the resulting system model rather than the development history. For implementation lessons and debugging history, see the engineering report.

---

## Purpose

The distributed control plane extends HACP Sidecar from a local enforcement process into a distributed enforcement replica.

The control plane is responsible for distributing revocation state and freshness evidence to one or more sidecars without becoming a synchronous dependency on every protected request.

The resulting model is:

```text
authority changes
      ↓
control plane
      ↓
snapshot / ordered event stream
      ↓
sidecar-local materialized state
      ↓
deterministic local enforcement
```

The sidecar continues to make authorization decisions locally from fully materialized state.

---

## Components

```text
                         ┌────────────────────┐
                         │   Control Plane    │
                         │                    │
                         │  RevocationJournal │
                         │  Snapshot service  │
                         │  Watch stream      │
                         │  Heartbeats        │
                         └─────────┬──────────┘
                                   │
                                   │ gRPC
                                   │
                    ┌──────────────┴──────────────┐
                    │                             │
                    ▼                             ▼
          ┌──────────────────┐          ┌──────────────────┐
          │   HACP Sidecar A │          │   HACP Sidecar B │
          │                  │          │                  │
          │ Subscriber       │          │ Subscriber       │
          │ RevocationStore  │          │ RevocationStore  │
          │ ControlState     │          │ ControlState     │
          │ Evaluation       │          │ Evaluation       │
          └─────────┬────────┘          └─────────┬────────┘
                    │                             │
                    ▼                             ▼
             protected upstream            protected upstream
```

Each sidecar owns its own local materialized state.

The control plane provides ordering and recovery, but it is not queried synchronously for every request.

---

## Protocol Contract

The normative protocol lives in:

```text
hacp-spec/proto/hacp/control/v1/control_plane.proto
```

The gRPC service defines two primary operations:

```text
GetRevocationSnapshot
WatchRevocations
```

The stream can carry:

```text
RevocationEvent
Heartbeat
ResetRequired
```

---

## Durable Revision Model

### `revision`

`revision` is the durable global ordering of control-plane mutations.

It represents authoritative mutation order and survives stream reconnects.

For a sidecar whose highest fully materialized revision is `R`:

```text
event.revision == R + 1
→ apply
→ advance to R + 1

event.revision <= R
→ duplicate / old event
→ ignore

event.revision > R + 1
→ revision gap
→ unsafe
→ fail closed / recover
```

The sidecar MUST NOT advance its revision cursor before the corresponding state mutation has been fully materialized locally.

---

### `sequence`

`sequence` is transport-local ordering inside one server stream.

It:

- starts again for a new stream;
- is not durable;
- is not a recovery cursor;
- is not a substitute for `revision`.

The distinction is:

```text
revision
→ durable authority mutation order

sequence
→ per-stream delivery order
```

---

## Meaning of `last_seen_revision`

`last_seen_revision` means:

> the highest control-plane revision whose state has been fully materialized into the local sidecar state.

It does not mean:

- highest revision received;
- highest revision parsed;
- highest revision announced by heartbeat;
- highest revision observed before an apply failure.

This distinction prevents a sidecar from accidentally skipping state after a partial failure.

---

## Revocation Journal

The control plane maintains an authoritative revocation journal.

The journal provides:

- monotonic revisions;
- deterministic replay;
- an atomic snapshot view;
- idempotent duplicate revocation handling;
- retained event history for resume/replay;
- replay-unavailable detection after compaction.

A duplicate revoke that does not change authoritative state does not consume a new durable revision.

---

## Snapshot Model

A snapshot is a complete materialized revocation view at a specific durable revision.

Conceptually:

```text
RevocationSnapshot {
    revision
    entries[]
    generated_at
}
```

The sidecar applies a snapshot as an atomic local replacement.

Only after successful replacement may it set:

```text
last_seen_revision = snapshot.revision
```

---

## Startup

A subscriber starts from a snapshot, then enters the ordered stream.

```text
startup
   ↓
GetRevocationSnapshot
   ↓
snapshot @ revision R
   ↓
replace local revocation state
   ↓
last_seen_revision = R
   ↓
WatchRevocations(after_revision=R)
   ↓
live events
```

This prevents a startup race between loading current state and receiving new mutations.

---

## Reconnect and Replay

A temporary transport failure does not require rebuilding state from scratch.

Reconnect resumes from the highest fully materialized revision:

```text
stream disconnect
       ↓
bounded reconnect backoff
       ↓
WatchRevocations(
    after_revision = last_seen_revision
)
       ↓
replay missed events
       ↓
resume live delivery
```

The control plane replays only revisions newer than the supplied cursor.

---

## Replay History Unavailable

The control plane may compact retained event history.

If a sidecar reconnects from a revision older than retained replay history, the server cannot safely synthesize the missing delta.

The server then sends:

```text
ResetRequired
```

Recovery becomes:

```text
ResetRequired
      ↓
mark distributed state unsafe
      ↓
GetRevocationSnapshot
      ↓
atomically replace local state
      ↓
last_seen_revision = snapshot.revision
      ↓
resume WatchRevocations
```

This is preferable to silently skipping unavailable history.

---

## Reconnect Backoff

Reconnect uses bounded exponential backoff.

Conceptually:

```text
initial
  ↓
2 × initial
  ↓
4 × initial
  ↓
...
  ↓
maximum
  ↓
maximum
```

The retry wait respects `context.Context` cancellation so shutdown is not blocked by the backoff timer.

A successfully established stream resets the reconnect sequence.

---

## Control-State Freshness

Distributed revocation state has both:

1. a durable revision;
2. a freshness property.

The sidecar tracks a `ControlState` containing:

- last fully materialized revision;
- last valid update time;
- connected/disconnected state;
- unsafe state;
- maximum allowed staleness.

---

## Disconnect Semantics

A short control-plane disconnect does not immediately invalidate the last materialized state.

```text
disconnect
   ↓
state remains locally available
   ↓
freshness window still valid
   ↓
evaluation may continue
```

This avoids turning every transient network interruption into an immediate application outage.

---

## Fail-Closed Staleness

If the materialized control state becomes too old, the sidecar fails closed.

```text
now - last_valid_update > max_staleness
→ DENY
→ CONTROL_STATE_STALE
```

Unsafe state may also fail closed immediately.

Typical unsafe conditions include:

- revision gap;
- unknown or invalid distributed event;
- inconsistent heartbeat;
- `ResetRequired` before recovery completes.

---

## Heartbeats

Heartbeats prove that a sidecar's currently materialized revision is still current.

A heartbeat contains the control plane's current durable revision.

A heartbeat is valid only when:

```text
heartbeat.current_revision
==
sidecar.last_seen_revision
```

A valid heartbeat:

- refreshes freshness;
- does not mutate revocation state;
- does not advance revision.

An inconsistent heartbeat:

```text
server revision != fully materialized local revision
```

marks distributed state unsafe.

Heartbeats MUST NOT be used to skip missing revocation events.

---

## Evaluation Pipeline Integration

The evaluator depends only on a minimal freshness contract.

Conceptually:

```go
type ControlStateGuard interface {
    IsFresh(time.Time) bool
}
```

This avoids coupling `internal/evaluate` directly to the control-plane package.

When no control-state guard is configured:

```text
standalone / conformance mode
→ existing evaluation semantics preserved
```

When a control-state guard is configured:

```text
fresh
→ evaluation continues

stale / unsafe
→ DENY
→ CONTROL_STATE_STALE
```

---

## Revocation Store Adapter

The subscriber writes distributed revocation state through an adapter that also satisfies the evaluator's revocation interface.

This allows the same materialized store to be used by:

```text
gRPC subscriber
        ↓
RevocationStoreAdapter
        ↓
evaluation pipeline
```

Snapshot replacement constructs a fresh local revocation state and swaps it atomically.

---

## Multi-Sidecar Convergence

Multiple sidecars connected to one control plane are expected to converge on the same durable state.

Example:

```text
Control Plane
revision 42
revoke token T
      │
 ┌────┴────┐
 ▼         ▼
A          B
rev=42     rev=42
T revoked  T revoked
 │          │
 ▼          ▼
DENY       DENY
TOKEN_     TOKEN_
REVOKED    REVOKED
```

Convergence is defined over both:

- materialized revision state;
- resulting enforcement decision.

---

## Failure Semantics Summary

| Condition | Behavior |
|---|---|
| Duplicate event | Ignore |
| Older event | Ignore |
| Next expected revision | Apply |
| Revision gap | Unsafe / fail closed |
| Stream disconnect | Reconnect with backoff |
| Replay available | Replay missed events |
| Replay unavailable | `ResetRequired` → snapshot |
| Valid heartbeat | Refresh freshness |
| Heartbeat revision mismatch | Unsafe |
| Fresh local state | Evaluate normally |
| Stale local state | `DENY / CONTROL_STATE_STALE` |

---

## Security Properties

The architecture is designed to preserve the following properties:

### No silent revision skipping

A sidecar cannot advance its durable cursor beyond state it has actually materialized.

### No replay ambiguity

Reconnect is explicitly anchored to durable revision state.

### No unsafe recovery guess

If required replay is unavailable, recovery requires a complete authoritative snapshot.

### No heartbeat-based state skipping

Heartbeats prove freshness only; they do not transport authority changes.

### Fail-closed stale authority

A sidecar refuses authorization when distributed authority state can no longer be considered fresh enough.

### Deterministic replica convergence

Independent sidecars consuming the same authoritative history converge on the same security state and outcome.

---

## Implementation Locations

```text
internal/controlplane/
    adapter.go
    journal.go
    server.go
    state.go
    subscriber.go
```

Generated gRPC bindings:

```text
gen/controlplane/v1/
```

Evaluator integration:

```text
internal/evaluate/control_state.go
internal/evaluate/pipeline.go
```

Normative protocol:

```text
hacp-spec/proto/hacp/control/v1/control_plane.proto
```

---

## Verification

The architecture is exercised through the Gate E test suite:

```bash
go test ./internal/controlplane -count=1 -v
```

See:

```text
docs/verification/distributed-control-plane-testing.md
```

for the property-to-test evidence map.

---

## Result

Gate E establishes a distributed control-state model where sidecars can:

- bootstrap from complete authority state;
- consume ordered changes;
- survive transient disconnects;
- replay missed history;
- recover after history compaction;
- detect unsafe revision divergence;
- validate freshness independently of mutation traffic;
- fail closed when authority state becomes stale;
- converge deterministically across replicas.


---

**Contact:** [digital.humanism.collective@protonmail.com](mailto:digital.humanism.collective@protonmail.com)