# Benchmarking HACP Enforcement: Closing Gate D

A postmortem on measuring the performance overhead of the HACP Sidecar enforcement proxy.

---

## Executive Summary

**Gate D** required demonstrating that the HACP Sidecar adds less than **5 ms p99 overhead** to each enforced request. The final measurement achieved **1.56 ms p99 overhead** (average 1.78 ms), exceeding the target by a 3.2× margin.

Reaching this number required fixing seven separate issues — none of which were in the HACP evaluation pipeline itself. The dominant sources of apparent latency were benchmark infrastructure artifacts: incorrect token generation, missing upstream forwarding, Windows-specific DNS resolution, flawed percentile calculation, and synchronous logging in the hot path.

---

## Background

The HACP Sidecar is a reverse proxy that enforces Human-Authorized Control Protocol policies on every request passing through it. Each request triggers a full evaluation pipeline:

```
HTTP parsing
→ ProposedAction synthesis
→ Envelope Ed25519 signature verification
→ DecisionToken Ed25519 signature verification
→ Revocation checks (tokens, envelopes, keys)
→ Envelope/token binding
→ action_hash binding
→ Token constraints validation
→ Boundary matrix evaluation
→ Budget consumption
→ Provenance event acceptance
→ Upstream forwarding
```

Gate D's acceptance criterion: **p99 overhead < 5 ms** (target), **< 10 ms** (acceptable fallback).

Initial benchmarks appeared to violate this target by a large margin. The investigation revealed that the measurement methodology, not the implementation, was broken.

---

## Initial Symptom

The first concurrent benchmark (10,000 requests via `hey`, concurrency 10) reported:

```
Baseline:   200 OK responses
Sidecar:    9,999 × 403 BUDGET_EXHAUSTED
              1 × 200 OK
```

Sidecar latency appeared **lower** than baseline — a classic signal that the two measurements were not comparable. The sidecar was returning fast-path denies, not enforcing real requests.

---

## Root Cause Analysis

Seven distinct issues were identified and fixed in sequence.

### 1. `max_uses` placed at the wrong JSON level

In `benchmarks/generate-tokens.go`, the field was generated at the top level of the decision token:

```go
"max_uses": 99999,
"constraints": {
    "method": *method,
    "path":   *path,
},
```

The evaluation pipeline (`internal/evaluate/pipeline.go`, Step 17) reads it from `tok.Constraints.MaxUses`:

```go
maxUses := 1
if tok.Constraints != nil && tok.Constraints.MaxUses != nil {
    maxUses = *tok.Constraints.MaxUses
}
```

When `Constraints.MaxUses` was `nil`, the pipeline defaulted to `maxUses = 1`. The first request consumed the token; the next 9,999 received `BUDGET_EXHAUSTED`. The benchmark was measuring deny-path latency, not enforcement overhead.

**Fix:** Move `max_uses` inside `constraints`:

```go
"constraints": {
    "method":   *method,
    "path":     *path,
    "max_uses": 99999,
},
```

### 2. Sidecar did not actually proxy to upstream

`forwardUpstream()` originally returned a local echo with a `200 OK` response, bypassing the real upstream. The benchmark was comparing:

```
Baseline: client → mock upstream
Sidecar:  client → HACP enforcement → local echo
```

These paths are not comparable.

**Fix:** Replace the echo with real HTTP forwarding:

```
Baseline: client → upstream
Measured: client → HACP sidecar → upstream
```

### 3. Docker hostname used in native mode

After enabling real forwarding, the first ALLOW request returned `502 Bad Gateway`. Sidecar logs revealed:

```
upstream=http://upstream:8000/api/test
lookup upstream: no such host
```

The upstream URL was hardcoded to `http://upstream:8000` — a Docker Compose hostname that does not resolve in native Windows execution.

**Fix:** Read the upstream URL from an environment variable with a localhost default:

```go
upstream := os.Getenv("HACP_UPSTREAM")
if upstream == "" {
    upstream = "http://127.0.0.1:8000"
}
```

### 4. `localhost` vs `127.0.0.1` on Windows

Serial benchmark showed baseline latency around **2050 ms/request**, while a single `curl` request completed in ~8.5 ms. The discrepancy was caused by using `http://localhost:8000` in PowerShell.

On Windows, `localhost` resolves first to the IPv6 loopback `::1`. Many servers (including the Python mock upstream) listen only on IPv4, so each connection attempt waits for IPv6 to time out before falling back to IPv4.

**Fix:** Use `127.0.0.1` explicitly:

```powershell
$sidecar  = "http://127.0.0.1:8080"
$upstream = "http://127.0.0.1:8000"
```

### 5. Asymmetric connection reuse

Baseline and sidecar measurements differed in connection pooling behavior due to differences between PowerShell, Python's HTTP server, and Go's HTTP server.

**Fix:** Add `-DisableKeepAlive` to `Invoke-WebRequest` for both paths, ensuring symmetric connection behavior.

### 6. Flawed percentile calculation on small samples

The initial percentile implementation used `Floor(count * p / 100)`. With 100 measurements, p99 landed at index 99 — effectively the sample maximum, artificially inflating tail latency.

**Fix:** Use the nearest-rank method:

```powershell
function Get-Percentile($arr, $p) {
    $sorted = @($arr | Sort-Object)
    if ($sorted.Count -eq 0) { return 0 }
    $rank = [math]::Ceiling(($p / 100.0) * $sorted.Count)
    $idx  = [math]::Max(0, $rank - 1)
    return [math]::Round($sorted[$idx], 2)
}
```

The report now separately shows `max` so that outliers are not conflated with percentiles.

### 7. Synchronous ALLOW logging in the hot path

With methodology issues resolved, 1,000-request serial benchmark showed:

```
Baseline p99:  3.40 ms
Sidecar p99:  14.55 ms
p99 overhead: 11.15 ms
```

Average overhead was only ~2.23 ms and p50/p95 were ~2 ms — but a tail dominated p99.

**Diagnostic steps:**

- Replaced the provenance writer with a `NoopWriter` → p99 overhead dropped to **7.79 ms**. Provenance contributed but was not the primary source.
- Disabled synchronous per-request `log.Printf("ALLOW ...")` on the success path → p99 overhead dropped to **0.04 ms**.

The dominant source of tail latency was **synchronous stdout writes on every successful request**.

**Fix:** Remove ALLOW logging from the hot path. Retain synchronous logging only for DENY and error paths (which are low-frequency and operationally valuable).

---

## Final Results

Serial benchmark: 1,000 requests, 1,000 unique tokens, 100% success rate on both paths.

| Metric | Baseline | Sidecar | Overhead |
|--------|----------|---------|----------|
| Average | 2.02 ms | 3.80 ms | **1.78 ms** |
| p50 | 1.84 ms | 3.54 ms | 1.70 ms |
| p95 | 2.45 ms | 4.60 ms | 2.15 ms |
| p99 | 3.87 ms | 5.43 ms | **1.56 ms** |
| Max | 64.64 ms | 74.47 ms | 9.83 ms |

```
GATE D PASS
p99 enforcement overhead = 1.56 ms
target < 5 ms
achieved = 3.2× under target
```

---

## Lessons Learned

1. **Baseline must measure an identical path.** You cannot compare `client → upstream` with `client → sidecar → upstream + local echo`. Both measurements must traverse the same downstream.

2. **On Windows, `localhost` ≠ `127.0.0.1`.** `localhost` resolves to `::1` first; many servers bind only to IPv4. Always use `127.0.0.1` in benchmarks for reliable measurements.

3. **`hey` does not rotate headers per-request.** For load testing with unique tokens, use either a serial benchmark with per-request tokens, a tool like `vegeta`/`k6` with request scripts, or pre-generated per-request files.

4. **Synchronous I/O in the hot path creates tail latency.** Any write to stdout, file, or database on every request inflates the p99. ALLOW-path logging should be asynchronous (ring buffer + flusher); DENY logging can remain synchronous.

5. **Default `max_uses = 1` is fail-safe but dangerous for benchmarks.** If a token generator forgets `max_uses`, the pipeline silently limits the token to a single use. Production benefits; benchmarks quietly fail.

6. **Percentile math matters on small samples.** `Floor(n * p / 100)` returns the sample maximum at high percentiles for small `n`. Use nearest-rank and always report `max` separately.

7. **Performance regressions are usually methodology, not code.** Seven distinct measurement artifacts had to be eliminated before the real pipeline performance was visible. Trust the measurement infrastructure last.

---

## Reproduction

```powershell
cd C:\Personal\GitHub\Dev\hacp-sidecar

# Terminal 1: upstream
python deployments/upstream/server.py 8000

# Terminal 2: control plane
python deployments/control-plane/server.py 5000

# Terminal 3: sidecar
$env:HACP_SIDECAR_PORT = "8080"
$env:HACP_UPSTREAM     = "http://127.0.0.1:8000"
$env:HACP_PROVENANCE_FLUSH_PATH = "provenance.jsonl"
.\hacp-sidecar.exe

# Terminal 4: benchmark
$env:Path += ";C:\Users\$env:USERNAME\go\bin"
.\benchmarks\benchmark_serial.ps1
```

---

## Related Files

- `internal/evaluate/pipeline.go` — full evaluation pipeline (Step 17: budget)
- `internal/proxy/handler.go` — HTTP enforcement proxy with real upstream forwarding
- `internal/proxy/proposed_action.go` — `ProposedAction` synthesis from HTTP request
- `internal/scope/matrix.go` — data-driven boundary matrix
- `benchmarks/generate-tokens.go` — pre-signed token generator
- `benchmarks/benchmark_serial.ps1` — serial benchmark (1000 unique tokens)
- `benchmarks/benchmark.ps1` — concurrent benchmark (single token, for deny-path measurement)
- `scripts/demo.ps1` — three enforcement scenarios (ALLOW / SCOPE_EXCEEDED / SIGNATURE_FAILURE)
- `deployments/` — Docker Compose reference deployment

---

## Conclusion

The HACP Sidecar enforces the full protocol — cryptographic verification, revocation, action-hash binding, boundary matrix evaluation, budget consumption, and provenance acceptance — while adding **1.56 ms** of p99 latency, well under the 5 ms target.

The dominant lesson from Gate D is not about Go performance optimization. It is that **performance measurement is itself a distributed system** — with its own failure modes, its own silent errors, and its own need for careful instrumentation. The pipeline was never the bottleneck; the measurement was.

---

**Contact:** [digital.humanism.collective@protonmail.com](mailto:digital.humanism.collective@protonmail.com)
