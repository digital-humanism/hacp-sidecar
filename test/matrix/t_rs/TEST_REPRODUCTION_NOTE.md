# TEST REPRODUCTION NOTE — t_rs

Status: INTERNAL TEST-REPRODUCTION NOTE

This note clarifies how HTTP-related results inside `test/matrix/t_rs` must be interpreted when reproducing the Rust test package.

## RS04 functional HTTP reproduction

The authoritative Rust HTTP functional reproduction is `RS04`.

Its execution path is:

```text
HTTP reproduction client
→ Rust proxy
→ local response
```

There is no upstream forwarding in this reproduction surface.

The workload is:

```text
warmup:   100 requests
measured: 5000 requests
concurrency: 1
```

A successful reproduction requires:

```text
negative fail-closed smoke:
403 + DENY

positive workload:
100/100 warmup
5000/5000 measured
R8_2_WORKLOAD=PASS
```

The functional claim is strictly:

```text
AUTHORIZATION + LOCAL RESPONSE
```

The following claim is not permitted:

```text
Rust upstream forwarding
```

## RS05 timing observation

`RS05` does not execute a second HTTP workload.

It consumes timing evidence produced by the successful `RS04` run.

Therefore:

```text
RS04
= functional reproduction

RS05
= informational timing derived from RS04
```

RS05 records metrics such as:

```text
p50
p95
p99
mean
max
RPS
```

but:

```text
INFORMATIONAL_ONLY=YES
THRESHOLD=NONE
```

Latency is not a functional PASS/FAIL gate.

## Interpretation rule

For test reproduction under `t_rs`:

```text
RS04
= authorization + local response
= 100 warmup + 5000 measured
= functional reproduction

RS05
= timing observation from the same RS04 workload
= no additional workload
= informational only
```

The Rust HTTP workload may be compared with earlier Rust runs for reproducibility.

It must not be interpreted as topology-equivalent to the Go D/B15 surface because the Go path includes real upstream forwarding while the Rust path does not.

No Go-vs-Rust latency ranking is established by this note.

---

## Fresh reproduction confirmation — 2026-10-22

A fresh isolated reproduction of the Rust HTTP functional surface `RS04` was completed after the matrix cleanup.

Fresh run identity:

```text
RUN_ROOT=C:\Temp\hacp-rust-http-repro-20261023-001155\run
EVIDENCE_ROOT=C:\Temp\hacp-rust-http-repro-20261023-001155\evidence
RUN_DIR=C:\Temp\hacp-rust-http-repro-20261023-001155\evidence\20261022-211155_2493580c
```

Preflight result:

```text
PREFLIGHT=PASS
BLOCKERS=[]
P_BRANCH=admit/hacp-1.1.1-pre-release
S_BRANCH=admit/hacp-1.1.1-pre-release
STAGED_FILES_OK=true
PORT_18081=FREE
```

Environment preparation:

```text
ENVIRONMENT_READY=PASS
PREPARE_BLOCKERS=0
```

Fresh execution result:

```text
SURFACE=RS04
MATRIX_ROWS=B16,B17
CLASSIFICATION=PASS
ENVIRONMENT_RED=false
BEHAVIORAL_RED=false

negative smoke:
status=403
decision=DENY
reason=INVALID_ENVELOPE
PASS=true

surface=AUTHORIZATION_PLUS_LOCAL_RESPONSE

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

R8_2_WORKLOAD=PASS
```

Observed measured timing:

```text
elapsed_ms=95542.8066
mean_ms=18.054499739999983
p50_ms=18.2019
p95_ms=24.5085
p99_ms=31.7202
max_ms=796.1778
rps=52.333
```

Postflight result:

```text
CLEANUP_COMPLETE=YES
GIT_GUARDS_PASS=YES
EVIDENCE_INVENTORY_COMPLETE=YES
```

The fresh reproduction therefore confirms that the Rust `5000/5000` result is a real serial HTTP workload.

The reproduced RS04 claim remains strictly:

```text
AUTHORIZATION + LOCAL RESPONSE
UPSTREAM_REQUIRED=NO
```

RS05 timing remains derived from this same RS04 workload and does not constitute a separate execution.

Timing remains informational:

```text
INFORMATIONAL_ONLY=YES
THRESHOLD=NONE
```

This confirmation does not establish a Go-vs-Rust performance ranking or transfer the Go upstream-forwarding claim to the Rust surface.
