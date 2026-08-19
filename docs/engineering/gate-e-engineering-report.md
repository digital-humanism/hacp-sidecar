# Gate E Engineering Report

**Project:** Humanist / HACP  
**Component:** `hacp-sidecar`  
**Phase:** 4b  
**Gate:** E — Distributed Control Plane + gRPC Revocation Stream  
**Status:** Complete

## Why this document exists

This report is intentionally different from the protocol reference and the architecture overview.

Those documents describe the final design. This document records how the design was reached: which assumptions failed, which tests exposed hidden contracts, which implementation details were changed, and which invariants became explicit only after a concrete failure.

The goal is to make Gate E reproducible for future contributors rather than merely understandable after the fact.

---

## 1. Starting point

Before Gate E, `hacp-sidecar` already had a working evaluator, deterministic HACP wire semantics, revocation checks, boundary enforcement, and conformance coverage.

Gate E had one central goal: extend that local enforcement model into a distributed control-plane model without weakening fail-closed behavior.

The minimum useful vertical slice was:

```text
Control Plane
    ↓
gRPC revocation stream
    ↓
Subscriber
    ↓
local RevocationStore
    ↓
real evaluator Pipeline
    ↓
ALLOW before revocation
DENY after revocation
```

The final gate additionally required recovery, freshness, and replica convergence.

---

## 2. The core recovery model

Gate E uses snapshot bootstrap followed by a resumable server stream.

```text
startup:
GetRevocationSnapshot → snapshot @ R
                     → materialize local state
                     → WatchRevocations(after_revision=R)
```

Reconnect resumes from the highest fully materialized revision:

```text
disconnect
→ WatchRevocations(after_revision=last_seen_revision)
→ replay missed events
→ continue live delivery
```

If replay history is no longer available:

```text
ResetRequired
→ fetch fresh snapshot
→ atomically replace local state
→ resume from snapshot revision
```

This model became the backbone of every later safety property.

---

## 3. Durable revision vs. stream sequence

One of the first important design decisions was to separate two kinds of ordering.

### Revision

`revision` is the durable global mutation order of the control plane.

It survives reconnect and is the only valid recovery cursor.

### Sequence

`sequence` is local to a single transport stream.

It can restart when a new gRPC stream is created and is never persisted as control-plane state.

This distinction prevents transport lifetime from leaking into authority semantics.

The resulting rule is:

> `last_seen_revision` means the highest revision that has been fully materialized locally, not the highest revision that has merely been observed.

---

## 4. Authoritative journal and replay/live handoff

The control plane maintains:

- a monotonic durable revision;
- authoritative revoked-subject state for snapshots;
- immutable replay events;
- a notification mechanism for live watchers.

A new mutation consumes one revision.

A duplicate revoke is idempotent and consumes no revision.

A subtle race exists at the replay/live boundary. A naive implementation can:

1. read replay history;
2. release synchronization;
3. receive a new event;
4. subscribe for live notifications.

The event between steps 2 and 4 can be missed.

The implemented journal returns replay events and the live notification handle under the same synchronization boundary, closing that gap.

This is a good example of a correctness issue that a happy-path gRPC test would not necessarily expose.

---

## 5. Building the first distributed enforcement proof

The first subscriber implementation performed:

```text
snapshot
→ local replacement
→ set revision cursor
→ open WatchRevocations
→ apply new events
```

A transport-only test proved that state reached the local test store.

The stronger integration test then used a real canonical signed HACP authority and the real evaluator Pipeline:

```text
before distributed revoke:
ALLOW

control plane revokes token
→ journal revision advances
→ gRPC event arrives
→ subscriber materializes revocation

after propagation:
DENY / TOKEN_REVOKED
```

That test became the first proof that Gate E affected actual enforcement behavior rather than only transport state.

---

## 6. Issue: the test revocation store was incomplete

### Symptom

As coverage expanded beyond the first happy path, the test store lacked the complete envelope-related surface required by the new revocation kinds.

### Root cause

The test double had been written for the initial token/key path and had silently become narrower than the runtime contract.

### Resolution

The test store was expanded to cover the complete mutation and inspection surface, including envelope state and atomic snapshot replacement semantics.

### Lesson

Security test doubles should model the full state contract. Otherwise missing runtime branches can be hidden by an artificially narrow test implementation.

---

## 7. Issue: the distributed adapter did not fully satisfy the evaluator contract

### Symptom

The adapter connecting distributed state to the evaluator failed interface compatibility.

### Root cause

The evaluator `RevocationStore` contract also required `LastUpdated() time.Time`.

### Resolution

The adapter was completed and compile-time interface assertions were added:

```go
var _ MutableRevocationStore = (*RevocationStoreAdapter)(nil)
var _ evaluate.RevocationStore = (*RevocationStoreAdapter)(nil)
```

Snapshot replacement creates a fresh underlying store and swaps it under synchronization, avoiding partially visible snapshot application.

### Lesson

Compile-time assertions are valuable at subsystem boundaries. They turn an integration assumption into an explicit build-time contract.

---

## 8. Revision state machine

After the first vertical slice, revision handling was made explicit:

```text
event.revision == last + 1
→ apply
→ advance cursor

event.revision <= last
→ duplicate or old
→ ignore

event.revision > last + 1
→ gap
→ unsafe
→ recovery required
```

Dedicated tests cover:

- next revision;
- duplicate event;
- older out-of-order event;
- forward gap;
- unknown revocation kind;
- all supported revocation kinds.

The most important rule is that the cursor advances only after successful materialization.

A failed event must never make the sidecar claim a revision it does not actually contain.

---

## 9. Issue: the first reconnect fault-injection test was wrong

### Initial approach

The first attempt used a stream interceptor that returned `Unavailable` from a `SendMsg` path.

### Problem

This coupled the test to gRPC transport internals and made it difficult to prove exactly which revision had become observable before the artificial failure.

The test was therefore capable of failing for the wrong reason or proving less than intended.

### Resolution

It was replaced with a deterministic reconnect test server:

1. first watch delivers revision 1;
2. first watch returns `Unavailable`;
3. second watch records the subscriber's `after_revision`;
4. revision 2 is committed while reconnect is controlled;
5. the second watch is released;
6. revision 2 is replayed;
7. the next revision is delivered live.

### What this proved

Reconnect uses the sidecar's materialized local truth:

```text
Watch(after_revision=last_seen_revision)
```

### Lesson

Fault injection should model observable protocol boundaries, not incidental transport implementation points.

---

## 10. Replay retention and `ResetRequired`

Replay cannot be retained forever.

The journal therefore separates authoritative current state from replay history.

When a subscriber requests a revision older than the retained replay window:

```text
replay unavailable
→ ResetRequired
→ subscriber marks distributed state unsafe
→ fresh snapshot
→ atomic local replacement
→ resume stream
```

History compaction does not remove authoritative snapshot state.

This makes retention bounded without making recovery ambiguous.

---

## 11. Freshness is not the same as connectivity

A major Gate E design decision was to separate transport connection state from authorization freshness.

`ControlState` tracks:

- last fully materialized revision;
- last valid update time;
- connection status;
- unsafe status;
- maximum staleness.

A short network disconnect does not immediately invalidate already materialized authority state.

```text
disconnected but still fresh
→ evaluation may continue
```

Once the configured freshness window expires:

```text
DENY / CONTROL_STATE_STALE
```

Certain consistency failures become unsafe immediately, including revision gaps and inconsistent heartbeats.

This distinction avoids both extremes:

- unsafe indefinite operation while disconnected;
- unnecessary total denial on every short transport flap.

---

## 12. Heartbeats: freshness without authority advancement

Heartbeats exist to prove that a control plane is alive even when there are no new revocations.

A valid heartbeat can refresh time freshness only if:

```text
heartbeat.current_revision == last_seen_revision
```

It must never advance the materialized revision cursor.

If the heartbeat advertises a different revision, the sidecar becomes unsafe.

This prevents heartbeats from hiding missed events.

The server also prioritizes pending events over heartbeats. If the journal revision has advanced, the subscriber must first receive the corresponding revocation event.

---

## 13. Issue: stale-state test returned `BUDGET_EXHAUSTED`

This was one of the most informative failures in Gate E.

### Expected result

The test established fresh state, successfully evaluated a single-use canonical token, then advanced time beyond `maxStaleness`.

Expected:

```text
DENY / CONTROL_STATE_STALE
```

### Actual result

```text
BUDGET_EXHAUSTED
```

### Root cause

The stale guard had not fired. Evaluation continued far enough to consume the same single-use token again.

The test updated one clock field but the Pipeline uses `RequestContext.EffectiveClock()`, which still observed the old `Policy.Clock` value.

The test fixture therefore contained inconsistent time sources.

### Resolution

All clock inputs used by `EffectiveClock()` were moved to the same stale timestamp.

The final proof became:

```text
fresh distributed state → ALLOW
stale distributed state → DENY / CONTROL_STATE_STALE
```

### Lesson

Virtual time is security state in expiry/freshness tests. All runtime clock sources must be controlled consistently.

Also, an unexpected downstream reason code is often evidence that an earlier guard was never reached.

---

## 14. Integration hazard: a nil token is not globally invalid

While placing the new stale-state guard, it was tempting to simplify Pipeline validation to:

```text
nil envelope OR nil token → invalid
```

That would have been a regression.

The existing Pipeline supports:

1. Envelope + DecisionToken;
2. Envelope without DecisionToken under an autonomy budget;
3. Envelope without DecisionToken when human/checkpoint flow is required.

The distributed guard therefore had to integrate without collapsing those existing execution modes.

### Lesson

Cross-cutting safety checks must preserve the full pre-existing state machine, not only the most common path.

---

## 15. Issue: incomplete evaluator wiring during ControlState integration

Several compile-time integration errors appeared while adding the Pipeline guard:

- `ControlStateGuard` existed but `Pipeline.ControlState` was missing in the saved source;
- `ReasonControlStateStale` had not yet been added;
- evaluator reason constants were actually located in `interfaces.go`, not where initially assumed.

The practical fix was to inspect the actual source tree instead of continuing from memory:

```powershell
Select-String -Path .\internal\evaluate\*.go -Pattern "ControlState|ReasonTokenRevoked"
```

### Lesson

When a large local change set is in flight, source search plus compiler output is more reliable than architectural memory.

---

## 16. Backoff existed before its exit criterion was proven

Reconnect already implemented bounded exponential backoff, but the complete test output did not contain dedicated evidence for that property.

Gate E was intentionally not considered complete on that point until explicit tests were added:

- `TestNextReconnectBackoff`
- `TestWaitForReconnectHonorsCancellation`

The tests verify:

- initial delay;
- doubling;
- maximum cap;
- values above the cap remain capped;
- context cancellation exits promptly.

### Small test bug

The first test version treated `waitForReconnect` as a `Subscriber` method. The real implementation is a package-level helper.

The test was corrected after checking the actual function signature.

### Lesson

An implemented recovery mechanism is not an exit criterion until its semantics have dedicated evidence.

---

## 17. Final runtime proof: two-sidecar convergence

The final distributed test intentionally used two independent sidecar-local stacks connected to one control plane:

- independent Subscriber instances;
- independent RevocationStores;
- independent ControlStates;
- independent budget ledgers;
- independent key resolvers;
- independent evaluator Pipelines.

Both initially evaluated the same canonical authority as ALLOW.

One control-plane token revocation then produced:

```text
Control Plane revision N
          │
      ┌───┴───┐
      ▼       ▼
 Sidecar A  Sidecar B
 revision N revision N
 revoked    revoked
      │       │
      ▼       ▼
TOKEN_REVOKED TOKEN_REVOKED
```

The test asserts both state convergence and security-outcome convergence.

That is stronger than merely comparing revision counters.

---

## 18. CI topology

The Gate E integration tests consume canonical HACP material from `hacp-spec`, so CI reproduces a sibling-repository layout:

```text
workspace/
├── hacp-sidecar/
└── hacp-spec/
```

The workflow runs:

```text
go test ./internal/controlplane -count=1 -v
go test ./... -count=1
```

A GitHub Actions Node.js runtime deprecation warning was also observed. It was an infrastructure warning, not a Gate E runtime or test failure. The practical maintenance action is to keep `actions/checkout` and `actions/setup-go` on current Node 24-based major versions.

### Lesson

Separate product failures, test failures, and CI platform warnings. They require different responses.

---

## 19. Gate E invariants that should not be weakened

### Fully materialized cursor

```text
last_seen_revision = highest fully materialized durable revision
```

### No silent gaps

A forward revision gap is unsafe and requires recovery.

### Heartbeats do not skip authority mutations

Heartbeat freshness can never advance revision.

### Reconnect resumes from local materialized truth

```text
Watch(after_revision=last_seen_revision)
```

### Missing replay history is explicit

```text
ResetRequired → snapshot recovery
```

### Connectivity and freshness are separate

A disconnect is not immediately stale, but stale state must fail closed.

### Distributed convergence includes enforcement

Replica convergence is not complete until independent Pipelines produce the same security result.

---

## 20. Troubleshooting guide

### The stream is connected but a revocation is not visible locally

Check:

1. snapshot revision;
2. subscriber `last_seen_revision`;
3. event revision is exactly `last + 1`;
4. revocation kind is supported;
5. adapter write path succeeds;
6. cursor advances only after successful materialization.

### Events disappear after reconnect

Check:

1. reconnect request uses `after_revision=last_seen_revision`;
2. replay/live handoff is synchronized;
3. required replay history is still retained;
4. unavailable replay produces `ResetRequired` rather than silent continuation.

### Heartbeats unexpectedly make the sidecar unsafe

Verify:

```text
heartbeat.current_revision == fully materialized revision
```

A heartbeat advertising a future revision is intentionally rejected.

### A stale-state test returns another denial reason

Inspect `EffectiveClock()` and every clock source in the request fixture.

If the result is `BUDGET_EXHAUSTED`, the stale guard probably did not fire and evaluation continued to token consumption.

### Reconnect tests are flaky

Prefer a deterministic test server with controlled stream phases over interceptor-level fault injection.

---

## 21. What we would do first next time

If Gate E were rebuilt from scratch, the most efficient sequence would be:

1. specify revision, sequence, and materialized-cursor semantics before server code;
2. define snapshot + resumable stream recovery from day one;
3. write revision state-machine tests before reconnect tests;
4. separate ControlState freshness from connection state immediately;
5. add compile-time adapter assertions immediately;
6. use deterministic protocol-level fault injection;
7. centralize virtual-clock construction for evaluator tests;
8. add backoff tests together with backoff code;
9. require two-sidecar convergence as an explicit exit criterion;
10. define the sibling-repository CI topology early.

---

## 22. Final result

Gate E is not primarily a gRPC milestone.

Its real result is a distributed authority model with explicit answers to the questions that matter for enforcement:

```text
What revision do I actually have?
Did I fully materialize it?
Did I miss anything?
Can I recover incrementally?
When must I take a snapshot?
How long may I trust disconnected local state?
When must I fail closed?
Will another sidecar reach the same security result?
```

The completed implementation answers those questions with protocol rules, runtime behavior, and test evidence.

**Gate E / Phase 4b — Complete.**

---

**Contact:** [digital.humanism.collective@protonmail.com](mailto:digital.humanism.collective@protonmail.com)