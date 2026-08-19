# Distributed Control Plane Verification

## Status

**Gate E / Phase 4b — Complete**

This document maps the distributed-control-plane properties of `hacp-sidecar` to the tests that verify them.

The goal is to answer:

> Where is each Gate E guarantee actually demonstrated?

Run the complete suite with:

```bash
go test ./internal/controlplane -count=1 -v
```

Run the complete repository regression with:

```bash
go test ./... -count=1
```

---

# Evidence Map

| Property | Primary Test |
|---|---|
| Exponential reconnect backoff | `TestNextReconnectBackoff` |
| Backoff cancellation | `TestWaitForReconnectHonorsCancellation` |
| Distributed revoke propagation | `TestDistributedTokenRevocationPropagation` |
| Real evaluator revoke behavior | `TestPipelineDistributedTokenRevocation` |
| Server heartbeat delivery | `TestWatchRevocationsDeliversHeartbeat` |
| Heartbeat refreshes freshness | `TestSubscriberHeartbeatRefreshesControlState` |
| Invalid heartbeat → unsafe | `TestSubscriberInvalidHeartbeatMarksControlStateUnsafe` |
| Two-sidecar convergence | `TestTwoSidecarsConvergeOnDistributedRevocation` |
| Stale state → fail closed | `TestPipelineFailsClosedWhenControlStateStale` |
| Standalone evaluator compatibility | `TestPipelineWithoutControlStateGuardPreservesStandaloneBehavior` |
| Replay + live delivery | `TestControlPlaneReplayAndLiveDelivery` |
| Duplicate revoke idempotency | `TestJournalDuplicateRevocationIsIdempotent` |
| Invalid future cursor rejected | `TestJournalRejectsRevisionAhead` |
| Compacted replay detection | `TestJournalReplayUnavailableAfterCompaction` |
| Server emits `ResetRequired` | `TestWatchRevocationsReturnsResetRequired` |
| Initial control state stale | `TestControlStateStartsStale` |
| Snapshot establishes freshness | `TestControlStateSnapshotEstablishesFreshState` |
| Event advances revision | `TestControlStateEventAdvancesRevision` |
| Disconnect grace semantics | `TestControlStateDisconnectDoesNotImmediatelyMakeStateStale` |
| Max staleness fail-closed | `TestControlStateFailsClosedAfterMaxStaleness` |
| Heartbeat freshness semantics | `TestControlStateHeartbeatRefreshesFreshness` |
| Heartbeat revision mismatch | `TestControlStateHeartbeatRevisionMismatchFailsClosed` |
| Explicit unsafe behavior | `TestControlStateExplicitUnsafeFailsClosedImmediately` |
| Snapshot recovers unsafe state | `TestControlStateValidSnapshotRecoversUnsafeState` |
| Reconnect + missed-event replay | `TestSubscriberReconnectReplaysMissedRevocation` |
| Next revision applied | `TestSubscriberAppliesNextRevision` |
| Duplicate event ignored | `TestSubscriberDuplicateEventIsIgnored` |
| Older event ignored | `TestSubscriberOlderOutOfOrderEventIsIgnored` |
| Revision gap fail-closed | `TestSubscriberRevisionGapFailsClosed` |
| Unknown kind does not advance cursor | `TestSubscriberUnknownRevocationKindFailsWithoutAdvancingRevision` |
| All supported revocation kinds | `TestSubscriberAllSupportedRevocationKinds` |
| Reset + snapshot recovery | `TestSubscriberResetRequiredRecoversFromSnapshot` |
| Event updates `ControlState` | `TestSubscriberEventUpdatesControlState` |
| Gap marks state unsafe | `TestSubscriberRevisionGapMarksControlStateUnsafe` |
| Unknown kind marks unsafe | `TestSubscriberUnknownKindMarksControlStateUnsafe` |
| Snapshot + live subscription | `TestSubscriberSnapshotAndLiveRevocation` |

---

# 1. Reconnect Backoff

## `TestNextReconnectBackoff`

Verifies bounded exponential reconnect behavior.

Expected progression:

```text
0
→ initial

initial
→ 2 × initial

2 × initial
→ 4 × initial

...
→ maximum

maximum
→ maximum
```

The test also verifies that values already above the maximum remain capped.

This proves the reconnect loop does not retry in an uncontrolled tight loop.

---

## `TestWaitForReconnectHonorsCancellation`

Verifies that reconnect waiting is interruptible through `context.Context`.

Scenario:

```text
long reconnect delay configured
        ↓
context cancelled
        ↓
wait exits promptly
```

This prevents shutdown from blocking until a retry timer expires.

---

# 2. Distributed Revocation Propagation

## `TestDistributedTokenRevocationPropagation`

Verifies the data path:

```text
Journal.Revoke
      ↓
gRPC WatchRevocations
      ↓
Subscriber
      ↓
RevocationStoreAdapter
      ↓
local token state revoked
```

This test demonstrates that one authoritative control-plane mutation reaches the sidecar's local enforcement state.

---

## `TestPipelineDistributedTokenRevocation`

Verifies the same path through the real evaluator.

Scenario:

```text
canonical signed request
        ↓
Pipeline
        ↓
ALLOW

control plane revokes token
        ↓
subscriber materializes revocation
        ↓
same Pipeline + same authority
        ↓
DENY
TOKEN_REVOKED
```

This is the primary Gate E vertical-slice proof.

It proves that the distributed control plane affects actual authorization behavior rather than only internal state.

---

# 3. Heartbeats and Freshness

## `TestWatchRevocationsDeliversHeartbeat`

Verifies that the server emits heartbeat messages when there are no new revocation events.

The heartbeat carries the current durable control-plane revision.

---

## `TestSubscriberHeartbeatRefreshesControlState`

Verifies:

```text
matching heartbeat revision
→ freshness refreshed
→ revision unchanged
```

This demonstrates the distinction between:

```text
freshness evidence
```

and:

```text
authority mutation
```

---

## `TestSubscriberInvalidHeartbeatMarksControlStateUnsafe`

Verifies that a heartbeat cannot silently advance local revision state.

Scenario:

```text
sidecar fully materialized revision = R
heartbeat current_revision != R
        ↓
unsafe
```

This prevents heartbeats from masking missing events.

---

# 4. Multi-Sidecar Convergence

## `TestTwoSidecarsConvergeOnDistributedRevocation`

This is the distributed replica convergence proof.

Topology:

```text
             Control Plane
                  │
          revoke token @ N
             ┌────┴────┐
             ▼         ▼
        Sidecar A   Sidecar B
```

The test uses:

- two independent subscribers;
- two independent revocation stores;
- two independent control-state instances;
- two independent evaluator pipelines.

It verifies:

```text
A.last_seen_revision == N
B.last_seen_revision == N

A token revoked
B token revoked

A → DENY / TOKEN_REVOKED
B → DENY / TOKEN_REVOKED
```

This proves convergence of both distributed state and security outcome.

---

# 5. Pipeline Staleness

## `TestPipelineFailsClosedWhenControlStateStale`

Verifies:

```text
fresh ControlState
→ canonical request ALLOW

advance evaluation clock beyond max_staleness
→ same authority
→ DENY
→ CONTROL_STATE_STALE
```

The freshness guard executes before the evaluator reaches token-budget consumption.

This ensures stale distributed authority state fails closed for the correct reason.

---

## `TestPipelineWithoutControlStateGuardPreservesStandaloneBehavior`

Verifies backward compatibility.

When:

```text
Pipeline.ControlState == nil
```

the distributed freshness mechanism is disabled and existing standalone/conformance behavior is preserved.

This prevents Gate E from changing canonical HACP-Core semantics for deployments that do not use distributed control state.

---

# 6. Journal Semantics

## `TestControlPlaneReplayAndLiveDelivery`

Verifies the transition from historical replay into live delivery without a replay/live gap.

The journal exposes retained events newer than the supplied cursor and a notification mechanism under a consistent locking model.

---

## `TestJournalDuplicateRevocationIsIdempotent`

Verifies duplicate authoritative revocation behavior.

Scenario:

```text
revoke subject X
→ revision N

revoke subject X again
→ no new state transition
→ no new durable revision
```

This ensures durable revision represents actual authority mutations rather than repeated API calls.

---

## `TestJournalRejectsRevisionAhead`

Verifies that a subscriber cursor cannot claim a revision newer than the authoritative journal.

This protects replay semantics from invalid resume cursors.

---

## `TestJournalReplayUnavailableAfterCompaction`

Verifies retained-history boundaries.

After journal compaction:

```text
requested cursor too old
→ replay unavailable
```

The control plane does not silently skip unavailable authority history.

---

## `TestWatchRevocationsReturnsResetRequired`

Verifies the server-side recovery protocol.

When replay cannot satisfy the requested cursor:

```text
WatchRevocations
→ ResetRequired
```

rather than returning an incomplete event sequence.

---

# 7. ControlState Model

## `TestControlStateStartsStale`

A newly created control state has no authoritative freshness evidence and therefore starts stale.

---

## `TestControlStateSnapshotEstablishesFreshState`

A successful snapshot materialization establishes:

- revision;
- update timestamp;
- fresh state;
- safe state.

---

## `TestControlStateEventAdvancesRevision`

A valid next event advances the fully materialized revision and refreshes state.

---

## `TestControlStateDisconnectDoesNotImmediatelyMakeStateStale`

Verifies disconnect grace semantics.

A transport disconnect:

```text
connected = false
```

does not erase the last valid update timestamp.

The sidecar may continue to evaluate while the last materialized state remains inside the configured freshness window.

---

## `TestControlStateFailsClosedAfterMaxStaleness`

Verifies the freshness boundary directly.

```text
age <= max_staleness
→ fresh

age > max_staleness
→ stale
```

---

## `TestControlStateHeartbeatRefreshesFreshness`

A valid heartbeat refreshes the freshness timestamp without advancing revision.

---

## `TestControlStateHeartbeatRevisionMismatchFailsClosed`

An inconsistent heartbeat marks the control state unsafe.

This is a direct fail-closed invariant.

---

## `TestControlStateExplicitUnsafeFailsClosedImmediately`

Verifies that unsafe state takes precedence over age-based freshness.

Even a recent timestamp cannot make explicitly unsafe distributed state valid.

---

## `TestControlStateValidSnapshotRecoversUnsafeState`

Verifies authoritative recovery.

A valid snapshot can:

- replace local state;
- set a known durable revision;
- clear unsafe state;
- re-establish freshness.

---

# 8. Subscriber Revision Safety

## `TestSubscriberAppliesNextRevision`

Verifies the valid transition:

```text
R
→ event R+1
→ apply
→ cursor R+1
```

---

## `TestSubscriberDuplicateEventIsIgnored`

Verifies:

```text
event.revision == last_seen_revision
→ ignore
```

No state mutation and no cursor regression occur.

---

## `TestSubscriberOlderOutOfOrderEventIsIgnored`

Verifies:

```text
event.revision < last_seen_revision
→ ignore
```

Old delivery cannot overwrite newer materialized state.

---

## `TestSubscriberRevisionGapFailsClosed`

Verifies the central ordering invariant:

```text
last_seen_revision = R
event.revision > R + 1
        ↓
revision gap
        ↓
unsafe
```

The subscriber does not jump over missing authority changes.

---

## `TestSubscriberUnknownRevocationKindFailsWithoutAdvancingRevision`

Verifies that unsupported distributed semantics do not consume durable revision state.

Scenario:

```text
event revision = R + 1
kind unknown
        ↓
error
        ↓
cursor remains R
```

This preserves the meaning of `last_seen_revision` as fully materialized state.

---

## `TestSubscriberAllSupportedRevocationKinds`

Verifies supported kinds:

```text
KEY
TOKEN
ENVELOPE
PARENT_ENVELOPE
```

Each is materialized into the correct local revocation state.

---

# 9. Reconnect and Recovery

## `TestSubscriberReconnectReplaysMissedRevocation`

Verifies resume semantics after a deterministic stream failure.

Scenario:

```text
receive revision 1
        ↓
stream disconnect
        ↓
control plane commits revision 2
        ↓
reconnect(after_revision=1)
        ↓
revision 2 replayed
        ↓
revision 3 delivered live
```

This proves missed revocations are recovered rather than lost across reconnect.

---

## `TestSubscriberResetRequiredRecoversFromSnapshot`

Verifies the complete recovery path after history compaction.

Scenario:

```text
sidecar at revision 1
        ↓
disconnect
        ↓
control plane advances
        ↓
required replay history compacted
        ↓
ResetRequired
        ↓
fresh snapshot at revision 3
        ↓
local state replaced
        ↓
Watch(after_revision=3)
        ↓
revision 4 delivered live
```

This is the strongest recovery-path test in Gate E.

---

# 10. Subscriber ↔ ControlState Wiring

## `TestSubscriberEventUpdatesControlState`

Verifies that successful event materialization updates both:

- subscriber cursor;
- `ControlState`.

---

## `TestSubscriberRevisionGapMarksControlStateUnsafe`

Verifies that revision-gap detection is propagated into the evaluator-facing safety state.

---

## `TestSubscriberUnknownKindMarksControlStateUnsafe`

Verifies that unsupported distributed event semantics mark the state unsafe.

---

## `TestSubscriberSnapshotAndLiveRevocation`

Verifies normal startup behavior:

```text
snapshot
→ materialize
→ stream
→ live revocation
```

This is the baseline subscriber lifecycle.

---

# Verification Layers

Gate E intentionally verifies behavior at multiple layers.

## Unit / state-machine level

Examples:

```text
ControlState tests
revision tests
backoff tests
journal tests
```

These prove local invariants directly.

---

## Protocol / streaming level

Examples:

```text
WatchRevocations heartbeat
ResetRequired
reconnect replay
snapshot + live delivery
```

These prove gRPC/recovery behavior.

---

## Enforcement integration level

Examples:

```text
TestPipelineDistributedTokenRevocation
TestPipelineFailsClosedWhenControlStateStale
```

These prove distributed state reaches actual authorization decisions.

---

## Distributed replica level

Example:

```text
TestTwoSidecarsConvergeOnDistributedRevocation
```

This proves independent enforcement replicas converge on the same state and decision.

---

# Expected Full Gate E Result

A successful run contains all control-plane tests as `PASS` and ends with:

```text
PASS
ok  hacp-sidecar/internal/controlplane
```

The repository-level regression must also pass:

```bash
go test ./... -count=1
```

At Gate E completion both suites passed.

---

# CI

GitHub Actions runs:

```bash
go test ./internal/controlplane -count=1 -v
go test ./... -count=1
```

The CI workspace checks out both:

```text
hacp-sidecar/
hacp-spec/
```

because integration tests use canonical HACP vectors from the specification repository.

---

# What This Evidence Does and Does Not Prove

The Gate E suite demonstrates the intended distributed-control-plane semantics implemented by the sidecar.

It provides strong regression evidence for:

- ordering;
- replay;
- recovery;
- freshness;
- fail-closed behavior;
- distributed convergence.

It is not by itself a formal proof of:

- Byzantine fault tolerance;
- secure production key distribution;
- durable multi-region persistence;
- transport authentication deployment correctness;
- absence of implementation vulnerabilities;
- absence of infrastructure-level bypass.

Those remain deployment and security-engineering concerns beyond this test suite.

---

# Related Documentation

Architecture:

```text
docs/architecture/distributed-control-plane.md
```

Gate completion:

```text
docs/gates/gate-e-distributed-control-plane.md
```

Engineering history:

```text
docs/engineering/gate-e-engineering-report.md
```

Normative protocol:

```text
hacp-spec/proto/hacp/control/v1/control_plane.proto
```

---

## Result

Gate E is considered complete because its required behavior is not only implemented but mapped to explicit executable evidence across:

```text
state machine
→ protocol stream
→ recovery
→ evaluator
→ multiple sidecars
```
