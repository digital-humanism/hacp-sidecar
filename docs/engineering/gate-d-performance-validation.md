# Gate D — Performance Validation Report

## HACP Sidecar

---

**Version:** 1.1  
**Date:** 2026-08-15  
**Gate:** D (Performance)  
**Status:** ✅ PASS  
**Acceptance Criterion:** p99 HACP enforcement overhead < 5 ms

---

## Executive Summary

Gate D validates that HACP Sidecar enforces the full protocol without introducing unacceptable latency overhead on application traffic.

**Final confirmed results:**

| Scenario | Requests | Concurrency | Token Model | Result |
|----------|----------|-------------|-------------|--------|
| Serial | 1000 | 1 | unique | **p99 overhead 0.59 ms — PASS** |
| Concurrent PowerShell | 1000 | 5 | rotating unique | **avg overhead 0.53 ms, p95 overhead 1.10 ms — PASS** |
| High-rate Go | 5 × 1000 | 5 | rotating unique | **median p99 overhead 1.26 ms — PASS** |
| Shared-token `hey` | 1000 | 5 | shared | **sidecar p99 3.3 ms, ~2761 req/s** |

The primary Gate D acceptance criterion remains:

```text
p99 HACP enforcement overhead < 5 ms
```

The serial acceptance benchmark and repeated high-rate rotating-token characterization both remain comfortably below the target.

> **Note:** Historical investigation results are retained in this report because they explain the transport, benchmark, and runtime changes that led to the final production-like configuration.

---

## 1. Gate D Objective

Gate D verifies that HACP Sidecar provides protocol enforcement without unacceptable impact on application traffic latency.

The real execution path under test:

```text
Agent / Benchmark Client
        │
        ▼
HACP Sidecar :8080
        │
        ├── IntentEnvelope parsing
        ├── DecisionToken parsing
        ├── Ed25519 signature verification
        ├── Revocation checks
        ├── Envelope/token binding
        ├── action_hash verification
        ├── Request constraints
        ├── Semantic boundary / scope
        ├── Budget / replay control
        ├── Provenance acceptance
        │
        ▼
Mock Upstream :8000
```

Gate D measures the **additional cost** of HACP Sidecar relative to direct upstream access.

For an individual run:

```text
p99 delta = sidecar p99 latency − baseline p99 latency
```

However, baseline and sidecar are independent latency samples. Therefore, high-rate acceptance is based on repeated runs rather than a single isolated p99 delta.

**Acceptance threshold:**

```text
p99 overhead < 5 ms
```

---

## 2. Test Topology

Native Windows topology was used for final validation:

```text
                    ┌────────────────────┐
                    │   Control Plane    │
                    │      :5000         │
                    └────────────────────┘
                              │
                              ▼
Benchmark ────────► HACP Sidecar ────────► Mock Upstream
                    :8080                    :8000
```

Four terminal windows were used:

| Terminal | Component | Address |
|----------|-----------|---------|
| 1 | Mock upstream | `127.0.0.1:8000` |
| 2 | Control plane | `127.0.0.1:5000` |
| 3 | HACP Sidecar | `127.0.0.1:8080` |
| 4 | Benchmarks | — |

**Sidecar configuration:**

```powershell
$env:HACP_SIDECAR_PORT = "8080"
$env:HACP_UPSTREAM = "http://127.0.0.1:8000"
$env:HACP_PROVENANCE_FLUSH_PATH = "provenance.jsonl"
```

During diagnosis, temporary performance tracing was introduced to localize latency.

That instrumentation was removed from the final production path after diagnosis was complete.

---

## 3. Measured HACP Path Components

Gate D was not conducted on a no-op stub.

The final configuration included:

- ✓ Real IntentEnvelope parsing
- ✓ Real DecisionToken parsing
- ✓ Ed25519 signature verification
- ✓ Canonical payload processing
- ✓ Signer-key resolution
- ✓ Key revocation checks
- ✓ Envelope/token revocation checks
- ✓ Expiration validation
- ✓ action_hash verification
- ✓ Token-envelope binding
- ✓ Request constraints validation
- ✓ Semantic scope / boundary matrix
- ✓ Token budget consumption
- ✓ Replay protection
- ✓ Real provenance acceptance
- ✓ Real HTTP forwarding
- ✓ Real upstream response

The results characterize the **real HACP enforcement path**, not an artificially simplified benchmark mode.

---

## 4. Initial Observations

During early Gate D iterations, significantly higher p99 values were observed:

```text
Baseline:
  p99 = 3.40 ms

Sidecar:
  p99 = 14.55 ms

p99 overhead = 11.15 ms
```

This formally appeared as:

```text
Gate D NEEDS OPTIMIZATION
```

Subsequent diagnosis showed that most tail latency originated outside the normative HACP evaluation pipeline.

The issues were localized incrementally.

---

## 5. Token Budget Fix

**Issue:** The `max_uses` field was initially placed at the top level of DecisionToken:

```json
{
  "max_uses": 99999,
  "constraints": {
    "method": "GET",
    "path": "/api/test"
  }
}
```

**Root cause:** The evaluation pipeline reads:

```text
token.constraints.max_uses
```

When absent, evaluation defaults to:

```text
max_uses = 1
```

which caused `BUDGET_EXHAUSTED` after the first request.

**Fix:**

```json
{
  "decision": "ALLOW",
  "constraints": {
    "method": "GET",
    "path": "/api/test",
    "max_uses": 99999
  }
}
```

Tokens were regenerated after this fix.

The current token generator stores `max_uses` correctly inside `constraints`.

---

## 6. Real Upstream Forwarding

**Issue:** The initial `forwardUpstream()` implementation returned a locally generated response without making a real HTTP request to upstream.

This created an invalid comparison:

```text
Baseline:
client → mock upstream

Sidecar:
client → HACP enforcement → local response
```

**Fix:** Implement a real HTTP reverse-proxy path:

```text
Sidecar
   │
   ▼
http.Client
   │
   ▼
Mock upstream
```

The actual test path became:

```text
client
  → HACP enforcement
  → upstream
  → sidecar
  → client
```

---

## 7. Hardcoded Docker Hostname

**Issue:** The sidecar initially created the handler with:

```go
"http://upstream:8000"
```

This works inside a Docker network but fails in the native Windows topology:

```text
lookup upstream: no such host
```

**Fix:**

```go
upstream := os.Getenv("HACP_UPSTREAM")
if upstream == "" {
    upstream = "http://127.0.0.1:8000"
}
```

The same runtime can now operate with native or containerized upstream endpoints.

---

## 8. Windows `localhost` Latency Artifact

**Issue:** Early benchmark versions used:

```text
http://localhost:8000
```

and produced anomalous second-scale latency on Windows.

**Observed cause:** Local hostname resolution and IPv4/IPv6 fallback introduced connection artifacts in the test environment.

**Fix:** Use explicit loopback addresses:

```powershell
$sidecar = "http://127.0.0.1:8080"
$upstream = "http://127.0.0.1:8000"
```

This removed hostname-resolution noise from Gate D measurements.

---

## 9. Percentile Calculation Fix

**Issue:** Initial PowerShell percentile code could make p99 behave like max for small sample sizes.

**Fix:** Use nearest-rank percentile:

```text
rank = ceil(p / 100 × N)
index = rank − 1
```

`max` is reported separately.

This prevents a single worst-case request from being mislabeled as p99.

---

## 10. Synchronous ALLOW Logging Impact

A significant diagnostic finding was that synchronous logging of every successful request increased tail latency.

**Original pattern:**

```go
log.Printf(
    "ALLOW request_id=%s ... latency=%s",
    ...
)
```

Temporarily disabling per-request ALLOW logging reduced tail latency substantially.

**Final hot-path rule:**

| Path | Logging Strategy |
|------|------------------|
| DENY | Synchronous, immediate |
| ALLOW | No synchronous per-request logging |
| Performance diagnostics | Temporary instrumentation only during investigation |

**Conclusion:** Observability must not materially alter the normal enforcement path.

---

## 11. Provenance Isolation Test

**Method:** Temporarily replace the real RingBuffer provenance writer with a no-op writer.

**Purpose:** Determine whether provenance acceptance was the dominant tail-latency source.

**Result:** Provenance contributed some cost but did not explain the large high-rate spikes.

**Outcome:** Real provenance was restored for final Gate D results.

---

## 12. Serial Benchmark

**Parameters:**

| Parameter | Value |
|-----------|-------|
| Requests | 1000 |
| Concurrency | 1 |
| Token model | Unique per request |
| Real provenance | Enabled |
| Real upstream | Enabled |
| Diagnostic tracing | Disabled |

### Final validation run

**Baseline:**

```text
Total: 2089.57 ms
Success: 1000 / 1000

Average: 2.09 ms
p50:     1.83 ms
p95:     2.59 ms
p99:     4.04 ms
max:    55.89 ms
```

**Sidecar:**

```text
Total: 2851.92 ms
Success: 1000 / 1000

Average: 2.85 ms
p50:     2.55 ms
p95:     3.36 ms
p99:     4.63 ms
max:    68.29 ms
```

**Overhead:**

```text
Total overhead: 762.35 ms
Average overhead: 0.76 ms/request
p99 overhead: 0.59 ms
```

**Result:**

```text
GATE D PASS
0.59 ms < 5 ms
```

---

## 13. Shared-Token `hey` Benchmark

The `hey` benchmark exercises high-rate requests using one shared DecisionToken.

**Parameters:**

| Parameter | Value |
|-----------|-------|
| Requests | 1000 |
| Concurrency | 5 |
| Token model | Shared |
| Load generator | `hey.exe` |

Early runs showed substantial p99 tail:

```text
Baseline p99: 10.7 ms
Sidecar p99:  41.0 ms
```

Those results led to further transport investigation.

This benchmark is not used as the primary Gate D acceptance metric because it combines:

- shared-token state;
- high throughput;
- transport behavior;
- independent baseline/sidecar samples;
- load-generator effects.

### Final shared-token characterization

After transport symmetry and keep-alive behavior were corrected:

**Baseline:**

```text
Total:        0.3466 secs
Average:      1.7 ms
p99:         33.0 ms
Requests/sec: 2885.5
```

**Sidecar:**

```text
Total:        0.3622 secs
Average:      1.8 ms
p99:          3.3 ms
Requests/sec: 2760.6
```

**Success:**

```text
1000 / 1000
```

The baseline p99 spike is treated as stochastic system/upstream tail behavior.

The correct interpretation is **not** that HACP makes requests faster.

The useful conclusions are:

- shared-token workload completes successfully;
- sidecar sustained ~2761 req/s;
- average sidecar latency remained close to baseline;
- no systematic shared-token bottleneck was observed in this test.

---

## 14. Concurrent Rotating PowerShell Benchmark

**Implementation:** `ForEach-Object -Parallel`

**Parameters:**

```text
Requests:    1000
Concurrency: 5
Token model: unique rotating token/request
```

### Latest validation run

**Baseline:**

```text
Total:        3945.29 ms
Average:      2.90 ms
p50:          2.31 ms
p95:          3.95 ms
p99:         22.09 ms
max:         67.12 ms
Requests/sec: 253.47
```

**Sidecar:**

```text
Total:        4178.93 ms
Average:      3.43 ms
p50:          2.88 ms
p95:          5.05 ms
p99:         16.40 ms
max:        106.42 ms
Requests/sec: 239.30
```

**Overhead:**

```text
Average: +0.53 ms
p50:     +0.57 ms
p95:     +1.10 ms
p99:     -5.69 ms
```

The negative p99 delta is not interpreted as acceleration.

It only means the baseline sample contained a larger stochastic tail in this particular run.

For this benchmark, the more stable average/p50/p95 metrics are used for characterization.

**Result:**

```text
PASS
```

**Limitation:** The PowerShell runspace implementation reaches only approximately 200–250 req/s in this environment.

A native Go benchmark was therefore added for high-rate characterization.

---

## 15. Temporary Performance Tracing

Temporary diagnostic instrumentation was introduced during Gate D to separate HACP evaluation time from upstream transport time.

Trace categories included:

| Trace | Purpose |
|-------|---------|
| `request_trace` | Full request breakdown |
| `evaluate_trace` | Pipeline timing |
| `upstream_trace` | HTTP connection/TTFB behavior |

Example:

```text
total=75.595ms
evaluate=513µs
upstream=75.081ms
```

**Key finding:** Even when total request latency was tens of milliseconds, HACP evaluation frequently remained below 1 ms.

After diagnosis, this tracing instrumentation was removed from the production hot path.

---

## 16. Pipeline Profiling

Temporary timers were added around evaluation stages such as:

- envelope;
- token;
- binding;
- scope;
- budget;
- provenance.

Hypotheses tested included:

- crypto bottleneck;
- budget mutex contention;
- provenance bottleneck;
- scheduler pauses.

Typical measurements:

```text
evaluate=514.3µs
evaluate=506.7µs
evaluate=550.5µs
evaluate=999.5µs
evaluate=702.3µs
```

**Conclusion:** The normative HACP evaluation pipeline was not the primary cause of observed multi-millisecond high-rate tail spikes.

---

## 17. HTTP Trace

Temporary `net/http/httptrace` instrumentation was added to the sidecar → upstream path.

Measured components included:

- DNS resolution;
- connection establishment;
- connection reuse;
- request write;
- time to first byte;
- response read.

Example slow request:

```text
total=75.08 ms

connect=10.57 ms
ttfb=63.33 ms
read=1.19 ms

evaluate=0.51 ms
```

**Conclusion:** The majority of the spike occurred outside HACP evaluation.

---

## 18. HTTP/1.0 Mock Upstream Discovery

The mock upstream used Python `BaseHTTPRequestHandler` without an explicit protocol version.

The default behavior was HTTP/1.0.

Responses also lacked `Content-Length`.

**Impact:** The Go transport could not reliably reuse upstream connections.

Diagnostic tracing showed frequent:

```text
reused=false
```

---

## 19. Mock Upstream Fix

The reference upstream was changed to HTTP/1.1:

```python
protocol_version = "HTTP/1.1"
```

Responses now include:

```http
Content-Length
```

The server uses a threaded HTTP implementation.

After the fix, persistent connection reuse became available.

This significantly reduced transport-related tail latency.

---

## 20. Hop-by-Hop Header Handling

The initial proxy implementation copied client headers directly to the upstream request.

For a reverse proxy, transport-level hop-by-hop headers must not be forwarded unchanged.

The sidecar now excludes:

- `Connection`
- `Keep-Alive`
- `Proxy-Connection`
- `Proxy-Authenticate`
- `Proxy-Authorization`
- `TE`
- `Trailer`
- `Transfer-Encoding`
- `Upgrade`

Headers named by the `Connection` header are also excluded.

The same filtering principle is applied to upstream response headers.

---

## 21. High-Rate Go Benchmark

PowerShell rotating-token testing was useful for functional concurrency but introduced substantial scheduler/runspace overhead.

A native Go benchmark was therefore implemented under:

```text
benchmarks/benchmark-rotating/main.go
```

Features:

- ✓ Reads `tokens.jsonl`
- ✓ Uses unique token pairs
- ✓ Concurrency = 5
- ✓ Worker pool
- ✓ Dedicated HTTP clients
- ✓ Separate baseline and sidecar measurements
- ✓ avg/p50/p95/p99/max
- ✓ requests/sec
- ✓ warm-up phase

This benchmark reaches several thousand requests per second on the reference workstation.

---

## 22. Initial High-Rate Go Result

Before final sidecar transport tuning:

**Baseline:**

```text
Average:      1.30 ms
p50:          1.07 ms
p95:          1.85 ms
p99:          3.86 ms
Requests/sec: 3839.86
```

**Sidecar:**

```text
Average:      2.89 ms
p50:          2.24 ms
p95:          4.12 ms
p99:         18.06 ms
Requests/sec: 1722.41
```

**Delta:**

```text
Average: +1.59 ms
p50:     +1.17 ms
p95:     +2.28 ms
p99:    +14.20 ms
```

A repeated run produced:

```text
p99 delta = 11.87 ms
```

This was sufficient to justify deeper transport analysis.

---

## 23. High-Rate Tail Localization

Temporary performance tracing during the native Go benchmark showed:

```text
evaluate ≈ 0.5–1 ms
upstream ≈ 20–80 ms on slow requests
```

Example:

```text
total=75.45 ms
evaluate=0.55 ms
upstream=74.90 ms
```

Connection traces showed both reused and non-reused connections.

This localized the largest spikes to sidecar → upstream transport behavior rather than HACP evaluation.

---

## 24. `http.DefaultClient` Discovery

The sidecar originally forwarded with:

```go
http.DefaultClient.Do(upstreamReq)
```

Meanwhile, the high-rate baseline benchmark used a dedicated transport configured for concurrency.

This created a methodological asymmetry.

For a reverse-proxy workload, a dedicated long-lived client/transport provides explicit connection-pool control.

---

## 25. Dedicated Pooled Transport

The handler was changed to own a long-lived `*http.Client` backed by a dedicated `*http.Transport`.

Representative configuration:

```go
MaxIdleConns:        100
MaxIdleConnsPerHost: 32
MaxConnsPerHost:     0
IdleConnTimeout:     90s
DisableKeepAlives:   false
ForceAttemptHTTP2:   false
```

Design principle:

```text
one Handler
→ one long-lived HTTP Client
→ one long-lived Transport
→ reusable connection pool
```

The HTTP client is not created per request.

---

## 26. Representative High-Rate Clean Run

After switching to the dedicated pooled client and removing temporary diagnostics from the hot path, one clean run produced:

**Baseline:**

```text
Total:        204.67 ms
Success:      1000 / 1000

Average:      1.02 ms
p50:          1.05 ms
p95:          1.68 ms
p99:          2.59 ms
max:         13.39 ms

Requests/sec: 4885.84
```

**Sidecar:**

```text
Total:        403.09 ms
Success:      1000 / 1000

Average:      2.00 ms
p50:          1.83 ms
p95:          2.96 ms
p99:          5.03 ms
max:         13.90 ms

Requests/sec: 2480.83
```

**Delta:**

```text
Average: +0.99 ms
p50:     +0.78 ms
p95:     +1.28 ms
p99:     +2.44 ms
```

This run passed the Gate D target.

However, because independent p99 samples exhibited significant stochastic variation, this single run is treated as a representative result rather than the sole final acceptance basis.

---

## 27. Repeated High-Rate Validation

Five consecutive clean runs were executed without restarting the sidecar or upstream between runs.

**Parameters:**

```text
Requests:    1000 per run
Concurrency: 5
Token model: unique DecisionToken per request
Runs:        5
```

Results:

| Run | Baseline p99 | Sidecar p99 | p99 Delta |
|---|---:|---:|---:|
| 1 | 19.08 ms | 6.09 ms | -12.99 ms |
| 2 | 2.47 ms | 3.73 ms | +1.26 ms |
| 3 | 1.85 ms | 5.36 ms | +3.51 ms |
| 4 | 2.14 ms | 12.99 ms | +10.85 ms |
| 5 | 30.48 ms | 11.09 ms | -19.38 ms |

Sorted p99 deltas:

```text
-19.38
-12.99
+1.26
+3.51
+10.85
```

Median:

```text
median p99 delta = 1.26 ms
```

**Result:**

```text
PASS
1.26 ms < 5 ms
```

Negative values do not indicate that the sidecar accelerates traffic.

They indicate that the independent baseline sample experienced a larger stochastic tail during that run.

The repeated-run median is therefore used as the high-rate characterization result.

---

## 28. Why Single-Run p99 Is Insufficient

Gate D investigation demonstrated substantial variability in p99 across independent baseline and sidecar samples.

Examples observed during development included:

```text
+14 ms
+12 ms
+10 ms
+6 ms
-1 ms
+2.44 ms
```

Meanwhile:

```text
average
p50
p95
```

were considerably more stable.

This leads to an important methodological distinction:

```text
algorithmic/runtime overhead
vs.
scheduler/transport/environment tail
```

For high-rate characterization, repeated runs are more reliable than a single subtraction of two independently sampled p99 values.

---

## 29. Benchmark Infrastructure Is Part of the Result

Gate D demonstrated that benchmark infrastructure can materially alter observed p99.

Significant factors included:

- `localhost` vs `127.0.0.1`;
- HTTP/1.0 vs HTTP/1.1;
- `Content-Length`;
- connection reuse;
- keep-alive symmetry;
- connection-pool sizing;
- synchronous logging;
- PowerShell runspace scheduling;
- Python mock-server behavior.

**Conclusion:** The benchmark topology and transport configuration are part of the system under test.

---

## 30. Observability Must Not Alter the Hot Path

A major finding was that:

```text
synchronous log.Printf on every ALLOW
```

could create measurable tail latency.

Production-oriented observability should prefer:

- asynchronous metrics;
- aggregation;
- sampling;
- slow-request tracing;
- event buffering.

It should avoid synchronous per-request logging on the successful hot path.

---

## 31. Final Architectural Configuration

Final request path:

```text
             ┌─────────────────┐
             │ Benchmark/Agent │
             └────────┬────────┘
                      │
                      ▼
             ┌─────────────────┐
             │   HACP Sidecar  │
             │                 │
             │ Parse           │
             │ Crypto          │
             │ Binding         │
             │ Boundary        │
             │ Budget          │
             │ Provenance      │
             └────────┬────────┘
                      │
                      │ pooled HTTP/1.1
                      │ persistent connections
                      ▼
             ┌─────────────────┐
             │    Upstream     │
             │    HTTP/1.1     │
             └─────────────────┘
```

---

## 32. Production Recommendations

### HTTP Transport

**Do:**

- ✓ use a dedicated long-lived `http.Client`;
- ✓ use a dedicated `http.Transport`;
- ✓ enable connection pooling;
- ✓ use persistent HTTP/1.1 connections;
- ✓ provide correct `Content-Length`;
- ✓ strip hop-by-hop headers.

**Do not:**

- ✗ create a client per request;
- ✗ blindly proxy `Connection` headers;
- ✗ use HTTP/1.0 mock servers for latency validation.

### Logging

| Path | Strategy |
|------|----------|
| DENY | Immediate logging |
| ALLOW | No synchronous per-request logging |
| Performance diagnostics | Temporary instrumentation when required |

### Windows Benchmarking

For deterministic loopback testing:

```text
Use:
127.0.0.1

Avoid:
localhost
```

when local IPv4/IPv6 behavior could affect measurements.

### Percentiles

Always report separately:

- p50
- p95
- p99
- max

Use a consistent percentile algorithm.

---

## 33. Final Acceptance Summary

Final Gate D reference results:

| Scenario | Requests | Concurrency | Token Model | Final Characterization | Result |
|----------|----------|-------------|-------------|------------------------|--------|
| Serial | 1000 | 1 | unique | p99 overhead **0.59 ms** | ✅ PASS |
| Concurrent PowerShell | 1000 | 5 | rotating unique | avg **+0.53 ms**, p95 **+1.10 ms** | ✅ PASS |
| High-rate Go | 5 × 1000 | 5 | rotating unique | median p99 delta **1.26 ms** | ✅ PASS |
| Shared-token `hey` | 1000 | 5 | shared | sidecar p99 **3.3 ms**, ~**2761 req/s** | ✅ Characterization |

Primary acceptance target:

```text
p99 HACP enforcement overhead < 5 ms
```

Primary serial result:

```text
0.59 ms < 5 ms
```

Repeated high-rate characterization:

```text
median p99 delta = 1.26 ms < 5 ms
```

### Gate D — PASS ✅

---

## 34. Validation Integrity

Final validation used:

| Parameter | Value |
|-----------|-------|
| Real signature verification | Yes |
| Real canonicalization | Yes |
| Real revocation checks | Yes |
| Real boundary evaluation | Yes |
| Real budget enforcement | Yes |
| Real replay state | Yes |
| Real provenance | Yes |
| Real upstream forwarding | Yes |
| HTTP/1.1 | Yes |
| Connection pooling | Yes |
| Synchronous ALLOW logging | No |
| Diagnostic PERF tracing in production hot path | No |

The results therefore characterize the actual HACP enforcement path.

---

## 35. Gate Status After Completion

| Gate | Focus | Status |
|------|-------|--------|
| A | Protocol correctness / conformance | ✅ PASS |
| B | Semantic boundary matrix | ✅ PASS |
| C | Reference deployment topology | ✅ PASS |
| D | Performance / operational viability | ✅ PASS |
| E | Distributed control | ⏳ NEXT |

---

## Reproduction Checklist

Before running Gate D benchmarks:

- [ ] Mock upstream uses HTTP/1.1
- [ ] Mock upstream returns `Content-Length`
- [ ] Sidecar uses a dedicated long-lived HTTP client
- [ ] Connection pooling is enabled
- [ ] URLs use `127.0.0.1`
- [ ] No synchronous ALLOW logging in the hot path
- [ ] `max_uses` is inside `constraints`
- [ ] No temporary performance tracing is enabled in the final runtime path
- [ ] Nearest-rank percentile calculation is used where applicable
- [ ] `max` is reported separately from p99
- [ ] Baseline and sidecar use symmetric transport settings
- [ ] High-rate acceptance is based on repeated runs rather than one isolated p99 delta

---

## Related Files

- `internal/evaluate/pipeline.go` — Full HACP evaluation pipeline
- `internal/proxy/handler.go` — HTTP enforcement proxy with real upstream forwarding
- `internal/proxy/proposed_action.go` — ProposedAction synthesis from HTTP requests
- `internal/scope/matrix.go` — Data-driven semantic boundary matrix
- `benchmarks/generate-tokens/main.go` — Pre-signed token generator
- `benchmarks/benchmark_serial.ps1` — Serial unique-token benchmark
- `benchmarks/benchmark_rotating.ps1` — Concurrent PowerShell rotating-token benchmark
- `benchmarks/benchmark-rotating/main.go` — Native Go high-rate rotating-token benchmark
- `benchmarks/benchmark.ps1` — Shared-token `hey` benchmark
- `benchmarks/benchmark.sh` — Shell benchmark helper
- `deployments/upstream/server.py` — Reference HTTP/1.1 upstream
- `deployments/control-plane/` — Reference control-plane implementation
- `docs/architecture/architecture.md` — HACP Sidecar architecture

---

## Related Documentation

- [`README.md`](../../README.md)
- [`docs/architecture/architecture.md`](../architecture/architecture.md)
- [`hacp-spec`](https://github.com/digital-humanism/hacp-spec)

---

**Document maintained by:** HACP Engineering Team  
**Last updated:** 2026-08-15  
**Next gate:** Gate E — Distributed Management

---

**Contact:** [digital.humanism.collective@protonmail.com](mailto:digital.humanism.collective@protonmail.com)
