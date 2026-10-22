# TEST REPRODUCTION NOTE — t_go

Status: INTERNAL TEST-REPRODUCTION NOTE

This note clarifies how HTTP-related results inside `test/matrix/t_go` must be interpreted when reproducing the test package.

## Functional HTTP reproduction

The authoritative Go HTTP functional reproduction is surface `D` / B15.

Its execution path is:

```text
HTTP reproduction client
→ Go sidecar
→ real upstream service
→ real upstream response
```

The workload is:

```text
warmup:   100 requests
measured: 5000 requests
concurrency: 1
```

A successful reproduction requires the measured workload to complete successfully and confirms:

```text
authorization
+
real upstream forwarding
+
real upstream response
```

The `5000/5000 successful` statement belongs to this functional reproduction surface.

It must not be inferred from an informational benchmark result.

## S10 benchmark surface

`S10` is a separate informational timing/benchmark surface.

It executes existing benchmark routes including:

```text
benchmarks/benchmark_serial.ps1
benchmarks/benchmark_rotating.ps1
benchmarks/benchmark-rotating/main.go
```

These benchmark tools have their own workload sizes and measurement models.

For example, the current `benchmark_serial.ps1` uses:

```text
1000 direct-upstream baseline requests
+
1000 sidecar requests
```

Therefore:

```text
D/B15 5000 measured requests
!=
S10 benchmark_serial 1000 sidecar requests
```

The two surfaces must not be merged or treated as the same reproduction.

## Interpretation rule

For test reproduction under `t_go`:

```text
D / B15
= functional HTTP reproduction
= authorization + real forwarding + upstream response
= 100 warmup + 5000 measured

S10
= informational performance characterization
= separate benchmark workloads
= no release PASS/FAIL derived from latency alone
```

Timing may be recorded for both, but workload counts, topology and purpose must remain explicit.

No Go-vs-Rust latency ranking is established by this note.

---

## Fresh reproduction confirmation — 2026-10-22

A fresh isolated reproduction of the Go HTTP functional surface `D` / B15 was completed after the matrix cleanup.

Fresh run roots:

```text
RUN_ROOT=C:\Temp\hacp-go-http-repro-20261023-000854\run
EVIDENCE_ROOT=C:\Temp\hacp-go-http-repro-20261023-000854\evidence
```

Repository identities recorded by preflight:

```text
hacp-sidecar:
HEAD=ff603ce00f0c07f95a7a5c7b4d91ea5daf3d6d21
TREE=fb89a49031305876c19c33093fee9956c158e873

hacp-spec:
HEAD=03885b2b062905072040dc245c3d25d283c249c4
TREE=b8341f7c114a4c51ea45d04352ee7ebdcc462e9a

humanist-core:
HEAD=91360a1f35585c078981880f95ba566a822e7dde
TREE=85af995d3cd2c336b8df1c4cd9901712bca9c4c0
```

Fresh execution result:

```text
SURFACE=D
MATRIX_ROW=B15
VERDICT=PASS
PROCESS_RC=0

surface=AUTHORIZATION_PLUS_REAL_FORWARDING

warmup:
requests=100
success=100
failed=0
concurrency=1

measured:
requests=5000
success=5000
failed=0
concurrency=1

GO_HTTP_REPRODUCTION=PASS
```

Observed measured timing:

```text
elapsed_ms=51410.6238
mean_ms=10.233904660000018
p50_ms=9.663
p95_ms=13.1227
p99_ms=18.0286
max_ms=78.5921
```

The fresh reproduction therefore confirms that the Go `5000/5000` result is a real serial HTTP workload and is not derived from the separate S10 informational benchmark surface.

The reproduced B15 claim remains:

```text
authorization
+
real upstream forwarding
+
real upstream response
```

Timing remains observational. This confirmation does not establish a Go-vs-Rust performance ranking or a new acceptance threshold.
