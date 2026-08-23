# PH-1B.3 Completion Report
## Liveness / Authorization Readiness Separation

**Project:** Humanist / HACP  
**Component:** hacp-sidecar  
**Stage:** PH-1B.3  
**Status:** COMPLETE  
**Verification date:** 2026-08-23

## Objective

Separate process liveness from authorization readiness so that a running sidecar is not automatically considered safe to act as an enforcement point.

The stage is governed by the invariant:

> Loss of certainty may only make authorization more restrictive.

## Result

PH-1B.3 is technically complete.

The sidecar now exposes two independent operational signals:

- `/healthz` reports process liveness.
- `/readyz` reports whether the replica can currently act as a safe authorization enforcement point.

A replica may therefore be alive but not authorization-ready.

## Verified Runtime Semantics

### Standalone mode

- `/healthz` returns `200`.
- `/readyz` returns `200`.
- Local revocation mutation remains available.

### Distributed mode

- The sidecar starts alive but not ready until valid control state is established.
- Temporary control-plane loss may retain readiness while the last synchronized state remains fresh.
- Once the configured maximum staleness is exceeded, readiness fails closed.
- Valid recovery restores readiness automatically without restarting the sidecar.
- Legacy local revocation mutation endpoints are fenced locally and do not fall through to the protected upstream.
- Graceful sidecar shutdown releases the listener cleanly.

## Integration Verification

The following lifecycle was verified through a real TCP control channel:

1. Control plane absent:
   - `/healthz` → `200`
   - `/readyz` → `503`

2. Control plane becomes available:
   - valid snapshot acquired
   - revocation watch established
   - `/readyz` → `200`
   - no sidecar restart required

3. Control plane disconnects:
   - readiness remains `200` while synchronized state is still fresh

4. Maximum staleness exceeded:
   - `/readyz` → `503`

5. Control plane recovers:
   - watch/snapshot state is re-established
   - `/readyz` → `200`
   - no sidecar restart required

6. Distributed local revocation fencing:
   - `/revoke/token` → `404`
   - `/revoke/envelope` → `404`
   - `/revoke/key` → `404`

7. Graceful shutdown:
   - sidecar exits cleanly
   - HTTP listener is released

## Regression Verification

Final verification:

```text
git diff --check
→ clean

go test ./... -count=1
→ PASS

git status --short
→ clean
```

Passing tested packages included:

- `cmd/sidecar`
- `internal/controlplane`
- `internal/scope`
- `internal/trust`
- `internal/wire`

## Relevant Signed Implementation Checkpoints

- `80583b6` — PH-1B.3A: separate authorization readiness from liveness
- `feaf570` — PH-1B.3B: bind readiness to control-state freshness
- `c403678` — PH-1B.3C.1: validate control-plane runtime configuration
- `be92733` — PH-1B.3C.2A: build shared control runtime graph
- `755aeda` — PH-1B.3C.2B: retry initial control-state snapshot
- `17be5df` — PH-1B.3C.3A: require secure control-plane transport
- `d5b146d` — PH-1B.3C.3B: build control-plane transport credentials
- `de85bcc` — PH-1B.3C.4A: fence local revocation mutations by runtime mode
- `f536c67` — PH-1B.3C.4B: wire distributed control runtime

## Architectural Conclusion

PH-1B.3 establishes that:

```text
not ready != dead != restart required
```

Authorization readiness is now a first-class safety signal independent of process liveness.

A loss of control-state certainty does not silently preserve authorization capability beyond the configured freshness boundary. Recovery does not require process replacement.

## Public Documentation Boundary

This report intentionally records verified architecture, externally relevant behavior, and regression status only.

Environment-specific diagnostic paths, A/B experiments, workstation-specific observations, and forensic transport investigation are maintained separately as restricted engineering material.
