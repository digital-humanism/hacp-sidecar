# Anti-Bypass Deployment Hardening

**Status:** PH-1B implementation hardening
**Scope:** deployment and infrastructure enforcement
**Normative HACP impact:** none

## Purpose

HACP authorization is meaningful only when protected execution cannot bypass the enforcement point.

The sidecar is an explicit enforcement point. It does not claim transparent operating-system interception. A deployment that requires bypass resistance must therefore preserve both:

```text
authorization correctness
+
execution-path control
=
meaningful enforcement
````

Loss of deployment certainty must never make protected execution more permissive.

## Deployment Classes

The production-hardening model distinguishes four deployment classes:

* **D0 — Advisory Only**
* **D1 — In-Process Enforcement**
* **D2 — External Sidecar Enforcement**
* **D3 — Sidecar + Infrastructure Anti-Bypass**

Only D3 claims infrastructure-level prevention of direct protected-upstream access.

## D3 Reference Topology

The Docker Compose reference topology separates the untrusted and protected execution zones:

```text
                 untrusted-net
        ┌─────────────────────────┐

        agent ───────→ sidecar
                         │
        └────────────────│────────┘
                         │
                         │
        ┌────────────────│────────┐
                         ▼
                      upstream

                 protected-net
                 internal: true
        └─────────────────────────┘
```

Network membership:

```text
agent:
  - untrusted-net

sidecar:
  - untrusted-net
  - protected-net
  - control-net

upstream:
  - protected-net

control-plane:
  - control-net
```

The sidecar is the only container that bridges the untrusted and protected execution zones.

The protected upstream has no host-published port.

## D3 Security Properties

A deployment may claim the D3 reference property only when all applicable conditions hold.

### Exclusive protected path

Protected execution must be reachable through the enforcement path.

An untrusted workload must not have a parallel direct route to the protected upstream.

### No protected-upstream host exposure

The protected upstream must not expose a host port, public listener, ingress, load balancer, service route, or equivalent path that bypasses enforcement.

### Network separation

The deployment must preserve an equivalent of:

```text
agent   → sidecar     allowed
sidecar → upstream    allowed
agent   → upstream    denied
host    → upstream    denied by default
```

The exact production mechanism may differ from Docker networking.

Equivalent controls may include:

* Kubernetes NetworkPolicy;
* firewall policy;
* service-mesh authorization;
* namespace or network isolation;
* equivalent infrastructure controls.

### Default deny

Anti-bypass security must not depend only on callers voluntarily using the sidecar.

Infrastructure must prevent unauthorized direct access to the protected execution surface.

### No permissive fallback

Sidecar failure, restart, timeout, unavailability, or loss of authorization readiness must not cause protected traffic to be routed directly to the upstream.

The required behavior is:

```text
enforcement unavailable
→ protected execution unavailable
```

The following behavior is forbidden:

```text
enforcement unavailable
→ direct upstream fallback
```

### No alternate origin route

Secondary service names, alternate listeners, direct origin addresses, management routes, or parallel proxies must not expose an uncontrolled path to the protected execution surface.

### Configuration integrity

Security-relevant deployment configuration must be protected against unauthorized modification.

This includes configuration controlling:

* sidecar listener exposure;
* upstream destination;
* network membership;
* ingress and service routing;
* fallback targets;
* protected-service exposure.

## Liveness and Authorization Readiness

Process liveness and authorization readiness are distinct properties.

### Liveness

Liveness answers:

```text
Is the sidecar process alive?
```

The existing `/healthz` endpoint is suitable as a basic process-liveness signal.

### Authorization readiness

Authorization readiness answers:

```text
Can this sidecar currently act as a safe enforcement point?
```

A future readiness surface may reflect security-relevant runtime state without changing HACP authorization semantics.

Loss of authorization readiness must never enable a direct or fallback route to the protected upstream.

Readiness hardening is a separate PH-1B implementation slice.

## Automated Anti-Bypass Verification

The repository provides deployment-level verification scripts:

```text
scripts/test-anti-bypass.ps1
scripts/test-anti-bypass.sh
```

These tests are intentionally separate from Go unit and protocol-conformance tests because they validate infrastructure properties rather than HACP-Core semantics.

The current automated matrix verifies:

| Test        | Property                                              |
| ----------- | ----------------------------------------------------- |
| `AB-E2E-01` | agent can reach the sidecar enforcement point         |
| `AB-E2E-02` | agent cannot directly reach the protected upstream    |
| `AB-E2E-03` | protected upstream has no host port bindings          |
| `AB-E2E-04` | sidecar can reach the protected upstream              |
| `AB-E2E-05` | network membership preserves the anti-bypass boundary |

Expected result:

```text
Result: 5 passed, 0 failed
```

The network-membership assertion verifies:

```text
agent:
  untrusted-net     yes
  protected-net     no

sidecar:
  untrusted-net     yes
  protected-net     yes

upstream:
  untrusted-net     no
  protected-net     yes
```

A topology regression that reconnects the agent to the protected network, exposes the upstream to the untrusted network, or publishes an upstream host port must cause verification failure.

## CI Verification

The GitHub Actions test workflow contains a dedicated anti-bypass deployment job.

The job:

1. builds the Docker reference deployment;
2. starts the isolated topology;
3. runs the anti-bypass verification matrix;
4. emits deployment state and logs on failure;
5. removes the deployment and volumes after execution.

Deployment verification remains separate from the Gate E and full Go regression jobs.

## Production Interpretation

The Docker Compose topology is a reproducible reference implementation of the D3 network property.

It is not a universal production deployment prescription.

Production environments must reproduce the same security property using controls appropriate to their infrastructure:

```text
untrusted workload
        |
        v
enforcement point
        |
        v
protected resource
```

with no uncontrolled parallel route:

```text
untrusted workload ------X------> protected resource
```

A deployment must not claim D3 solely because HACP Sidecar is present.

The execution-path isolation property must also be enforced.

## Scope Boundary

PH-1B deployment hardening does not introduce:

* new authority objects;
* new HACP authorization meanings;
* new checkpoint outcomes;
* new delegation semantics;
* new normative action fields;
* new wire semantics;
* new canonical vectors defining authorization behavior.

Changes that require those semantics belong in the protocol and architecture evolution track rather than PH-1B implementation hardening.

## Current Verification Status

The PH-1B reference topology has been validated with:

```text
agent   → sidecar     PASS
agent   → upstream    BLOCKED
sidecar → upstream    PASS
host    → upstream    BLOCKED
```

Automated anti-bypass verification:

```text
AB-E2E-01  PASS
AB-E2E-02  PASS
AB-E2E-03  PASS
AB-E2E-04  PASS
AB-E2E-05  PASS

Result: 5 passed, 0 failed
```

This establishes the D3 reference network topology without changing HACP-Core normative semantics.
