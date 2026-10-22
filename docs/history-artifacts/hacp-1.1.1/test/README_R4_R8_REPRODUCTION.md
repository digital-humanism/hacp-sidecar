# HACP 1.1.1 — SIDECAR-A2 Verification Reproduction Record

Status:

```text
ROLE=HISTORICAL_VERIFICATION_NAVIGATION
NORMATIVE_AUTHORITY=NO
PRODUCTION_RUNTIME_INPUT=NO
RELEASE_AUTHORITY=NO
HISTORICAL_REWRITE=NO
```

This document records the temporary verification surfaces created during the
HACP 1.1.1 SIDECAR-A2 controlled reproduction work.

The primary purpose is reproducibility.

A future reviewer should be able to read this file and determine:

```text
what was tested
why the additional verification files were created
where those files live
which repository identities were used
which production surfaces were exercised
which evidence was produced
which hashes identify the exact verification artifacts
how the tests can be rerun
which conclusions may and may not be drawn
```

This directory does not duplicate the actual verification files.

The verification files remain in their original repository-local locations
because those locations are part of the execution context in which they were
created and run.

---

# 1. Controlled candidate identity

Repository:

```text
<SIDECAR_REPO>
```

Branch:

```text
admit/hacp-1.1.1-pre-release
```

Controlled candidate HEAD:

```text
0fcaff2603e4a1f7f683bda81f06ac25f6d68edc
```

During the described verification work:

```text
STAGING=NOT_PERFORMED
COMMIT=NOT_PERFORMED
PUSH=NOT_PERFORMED
```

The additional verification files were deliberately kept untracked while the
test work was in progress.

---

# 2. Verification-file ownership

The following files belong to the HACP 1.1.1 SIDECAR-A2 controlled
verification surface.

They are not production runtime implementation files.

## R8.2 Rust HTTP verification runner

Repository path:

```text
benchmarks/rust-http-reproduction/main.go
```

SHA-256:

```text
C52125CEA8CE959FBE613063E9370AE1E63F92F2CC755CEEAC1AD338B5FAB895
```

Role:

```text
ROLE=R8_2_RUST_HTTP_VERIFICATION_RUNNER
PRODUCTION_RUNTIME_INPUT=NO
NORMATIVE_AUTHORITY=NO
```

---

## R8.2 Rust HTTP reproduction procedure

Repository path:

```text
docs/verification/rust-http-local-reproduction.md
```

SHA-256:

```text
C442494DA324FBE3D526C4CB0E5CD0C086F6AE0DA5DCFBD2509443D61A6B86C7
```

Role:

```text
ROLE=R8_2_RUST_HTTP_REPRODUCTION_PROCEDURE
PRODUCTION_RUNTIME_INPUT=NO
NORMATIVE_AUTHORITY=NO
```

---

## R4.3 H-01 / H-02 provenance verification witness

Repository path:

```text
internal/proxy/r4_provenance_verification_test.go
```

SHA-256:

```text
C2CF4C78B3349366D30EE237DF74A03784127145FB215EFF43C0384255CA68F1
```

Role:

```text
ROLE=R4_3_H01_H02_VERIFICATION_TEST
PRODUCTION_SOURCE_SEMANTICS=UNCHANGED
NORMATIVE_AUTHORITY=NO
```

This file is located under `internal/proxy` because it is a Go package-local
verification test and reuses existing package-local signing and binding test
helpers.

Its repository location is therefore intentional.

---

# 3. External H-03 verification harness

H-03 was executed as an external process-level verification because the test
must exercise the actual sidecar process lifecycle.

Original external evidence path:

```text
tools/reproduction/graceful-shutdown/main.go
```

SHA-256:

```text
A216FA2115991937F8A710FA8A2F3D5460655BD91D8D9330DCAA52436AE7FB49
```

Role:

```text
ROLE=R4_3_H03_EXTERNAL_PROCESS_VERIFICATION_HARNESS
REPOSITORY_RUNTIME_INPUT=NO
NORMATIVE_AUTHORITY=NO
```

Unlike the three repository-local verification files, this harness remains in
the external evidence root.

It is not copied into this history directory.

---

# 4. Why additional verification was required

The controlled reproduction plan required R4 evidence covering:

```text
R4.1 HTTP process enforcement
R4.2 production trust lifecycle
R4.3 real provenance / graceful shutdown
R4.4 query-inclusive binding / forwarding
```

Existing tracked tests already covered substantial parts of R4.

Exact existing test names included:

```text
TestHTTPRequestPathBindingIncludesQueryString
TestHTTPRequestPathBindingRejectsDifferentQueryString
TestServeHTTPUsesQueryInclusivePathBinding
TestServeHTTPRejectsToolOutsideEnvelopeScope
```

These existing tests established, among other things:

```text
query-inclusive RequestURI binding
altered query rejection
authorized upstream forwarding
DENY does not forward
```

Existing trust and cmd/sidecar tests separately covered the production trust
lifecycle.

The remaining gap was R4.3 process/provenance evidence.

The production implementation already contained the required semantics, but no
single existing public test procedure provided the exact required witnesses:

```text
H-01 real provenance acceptance before forwarding
H-02 full provenance buffer fails closed and does not forward
H-03 graceful process shutdown flushes provenance and removes listener
```

Therefore the additional files were classified as:

```text
VERIFICATION_GAP_REMEDIATION
NOT_PRODUCTION_SEMANTIC_REMEDIATION
```

---

# 5. Production request path relevant to R4.3

The Go sidecar HTTP request path is:

```text
client request
    ->
HACP header extraction
    ->
body/action synthesis
    ->
RequestContext construction
    ->
Pipeline.Evaluate()
    ->
DENY and stop
        OR
    ->
forwardUpstream()
```

The critical ordering is:

```text
authorization / provenance acceptance
BEFORE
upstream forwarding
```

Forwarding is not attempted until `Pipeline.Evaluate()` returns an ALLOW
decision.

---

# 6. Real provenance implementation

R4.3 uses the actual production provenance writer:

```text
internal/provenance/ring.go
```

The writer is a bounded asynchronous `RingBuffer`.

Its acceptance rule is fail-closed.

Conceptually:

```text
if count >= capacity:
    return error
```

The implementation explicitly documents that inability to accept a provenance
record must result in:

```text
TRACEABILITY_FAILURE
```

Overwrite is not used as an escape path.

The ring buffer also contains:

```text
flush worker
periodic flush
non-blocking flush trigger
final flush on stop
Stop() waits for worker completion
```

---

# 7. H-01 — provenance accepted before forwarding

Verification file:

```text
internal/proxy/r4_provenance_verification_test.go
```

Test:

```text
TestR43H01ProvenanceAcceptedBeforeForward
```

Goal:

```text
prove that the real provenance writer has accepted the ALLOW event
before the upstream handler executes
```

The witness uses:

```text
real evaluate.Pipeline
real provenance.RingBuffer
real proxy.Handler
real signed HACP envelope
real signed HACP token
httptest upstream
```

It does not substitute `provenance.NoopWriter` for the provenance surface under
test.

At upstream execution time, the test requires that the provenance event is
either:

```text
still accepted in the real ring buffer
```

or:

```text
already asynchronously flushed to the provenance file
```

Controlled result:

```text
--- PASS: TestR43H01ProvenanceAcceptedBeforeForward
```

---

# 8. H-02 — full provenance buffer fails closed

Verification file:

```text
internal/proxy/r4_provenance_verification_test.go
```

Test:

```text
TestR43H02FullProvenanceBufferFailsClosedWithoutForward
```

Goal:

```text
provenance acceptance failure
    ->
TRACEABILITY_FAILURE
    ->
DENY
    ->
HTTP 403
    ->
no upstream forwarding
```

The witness uses a real `RingBuffer` with capacity zero.

This is intentional.

It creates a deterministic full-buffer state without depending on timing
against the asynchronous flush worker.

Observed controlled execution:

```text
provenance record failed ... ring buffer full: 0/0

DENY ...
reason=TRACEABILITY_FAILURE
err=provenance acceptance failed: ring buffer full: 0/0
```

Result:

```text
--- PASS: TestR43H02FullProvenanceBufferFailsClosedWithoutForward
```

The test explicitly verifies that upstream execution does not occur.

---

# 9. H-01 / H-02 execution record

Execution command:

```powershell
go test ./internal/proxy `
  -run '^TestR43H0[12]' `
  -count=1 `
  -v
```

Observed:

```text
TestR43H01ProvenanceAcceptedBeforeForward
PASS

TestR43H02FullProvenanceBufferFailsClosedWithoutForward
PASS

TEST_RC=0
```

Raw execution log SHA-256:

```text
19814AD5919F94AD9D2CC46AAFCA38F855BA6BDC7175533C46E2EB3C361B3B1D
```

Controlled evidence location:

```text
docs/history-artifacts/hacp-1.1.1/test/evidence/r4-3/H01_H02_provenance_proxy.log
```

---

# 10. H-03 — real process graceful shutdown

H-03 is intentionally process-level.

A unit test that directly invokes `RingBuffer.Stop()` is not sufficient for
this witness.

The production process handles:

```text
os.Interrupt
syscall.SIGTERM
```

using:

```text
signal.NotifyContext
```

The graceful path is:

```text
signal
    ->
ctx.Done()
    ->
server.Shutdown(...)
    ->
provLog.Stop()
    ->
ring-buffer final flush
    ->
clean process exit
```

---

# 11. H-03 topology

Controlled topology:

```text
external verification harness
    |
    | starts fresh sidecar process
    v
hacp-sidecar.exe
127.0.0.1:18082
    |
    | signed authorized request
    v
real Pipeline
    |
    v
real RingBuffer
    |
    | authorization ALLOW
    v
real forwarding attempt
    |
    v
127.0.0.1:18083
intentionally absent
```

Port `18083` is intentionally not serving an upstream.

This allows the test to distinguish:

```text
successful authorization
```

from:

```text
successful upstream execution
```

---

# 12. Meaning of ALLOW + HTTP 502

The H-03 trigger request intentionally results in:

```text
X-HACP-Decision: ALLOW
HTTP 502 Bad Gateway
```

This combination is expected for this topology.

The fields describe different layers.

```text
ALLOW
=
HACP authorization permitted the forwarding attempt
```

```text
HTTP 502
=
the permitted forwarding attempt failed because the upstream was absent
```

Therefore:

```text
ALLOW != successful execution
```

and:

```text
502 != HACP DENY
```

This distinction must be preserved when interpreting sidecar responses.

A consumer must not treat `X-HACP-Decision: ALLOW` alone as proof that the
upstream operation completed successfully.

---

# 13. H-03 Windows graceful signal

A forced process termination is not a valid substitute for graceful shutdown.

The external harness starts the sidecar using a distinct Windows process group:

```text
CREATE_NEW_PROCESS_GROUP
```

and sends:

```text
CTRL_BREAK_EVENT
```

using:

```text
GenerateConsoleCtrlEvent
```

Observed:

```text
CTRL_BREAK_RESULT=1
```

The process then follows its registered interrupt path.

Observed:

```text
PROCESS_EXIT_CODE=0
```

---

# 14. H-03 environment

Controlled environment:

```text
HACP_TRUST_MODE=test
HACP_CONTROL_MODE=standalone
HACP_SIDECAR_PORT=18082
HACP_UPSTREAM=http://127.0.0.1:18083
HACP_PROVENANCE_FLUSH_PATH=<external R4_3 evidence path>
```

`HACP_TRUST_MODE=test` is explicit.

The built-in conformance test key is not implicitly trusted in production
mode.

---

# 15. H-03 fresh build

Command:

```powershell
go build -o <evidence>\hacp-sidecar-h03.exe .\cmd\sidecar
```

Controlled build result:

```text
BUILD_RC=0
```

Executed binary SHA-256:

```text
5C02C1C42BF00707D6AEFEC65CE2CDDAC6752892C7AD67A073D4AE1B7E6233C5
```

Future reproductions should perform a fresh build and record their own binary
SHA together with source HEAD and Go toolchain identity.

---

# 16. H-03 signed fixture

The trigger request was generated from the tracked repository-owned token
generator:

```text
benchmarks/generate-tokens/main.go
```

Command shape:

```powershell
go run .\benchmarks\generate-tokens `
  -count 1 `
  -out <evidence>\h03-token.jsonl `
  -method GET `
  -path /api/test
```

Controlled fixture SHA-256:

```text
755CB1633E3FD638F095785999569C4E96D3181A9E4F2AEC38100AAA64BEFE6F
```

A future reproduction should generate a fresh fixture rather than silently
reuse this historical file.

---

# 17. H-03 controlled result

Observed:

```text
SIDECAR_PID=12360

LISTENER_READY=PASS

AUTHORIZED_HTTP_STATUS=502
AUTHORIZED_DECISION=ALLOW
AUTHORIZED_BODY_BYTES=84
AUTHORIZED_PROVENANCE_TRIGGER=PASS

CTRL_BREAK_RESULT=1

PROCESS_EXIT_CODE=0

LISTENER_REMOVED=PASS

PROVENANCE_BYTES=300
PROVENANCE_RECORDS=1
PROVENANCE_PERSISTED_AFTER_SHUTDOWN=PASS

R4_3_H03=PASS
H03_RC=0
```

This establishes:

```text
real process started
real authorization executed
real provenance writer used
graceful signal observed
process exited cleanly
sidecar listener disappeared
provenance persisted
```

---

# 18. H-03 evidence identities

```text
hacp-sidecar-h03.exe
SHA256=
5C02C1C42BF00707D6AEFEC65CE2CDDAC6752892C7AD67A073D4AE1B7E6233C5
```

```text
h03-token.jsonl
SHA256=
755CB1633E3FD638F095785999569C4E96D3181A9E4F2AEC38100AAA64BEFE6F
```

```text
h03_graceful_shutdown.go
SHA256=
A216FA2115991937F8A710FA8A2F3D5460655BD91D8D9330DCAA52436AE7FB49
```

```text
H03_graceful_shutdown.log
SHA256=
175D3B62249775BF510BDB5DF4DDE20EC71DA2A8B46EAF507DCFBCB6AEE677B9
```

```text
H03_sidecar_stdout.log
SHA256=
E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855
```

```text
H03_sidecar_stderr.log
SHA256=
E3F167E2DC1E03C5EA6C396ADBD097AC5C2372090FABB1533949F2A2339F38A4
```

```text
H03_provenance.jsonl
SHA256=
528220317885F1C353D3E60BDAC7376C5AE3CA980F43CA320490E06057C52A70
```

The empty stdout evidence is expected because the sidecar runtime diagnostics
were written to stderr.

---

# 19. R4.3 final disposition

Controlled results:

```text
H-01=PASS
H-02=PASS
H-03=PASS
```

Disposition:

```text
R4_3_REAL_PROVENANCE_AND_SHUTDOWN=PASS
```

The conclusion applies to the exact controlled candidate and verification
identities recorded in this document.

---

# 20. Rust HTTP R8.2 boundary

The Rust HTTP surface used during R8.2 is intentionally different from the Go
sidecar forwarding path.

Rust source repository:

```text
<HACP_SPEC_REPO>
```

Branch:

```text
admit/hacp-1.1.1-pre-release
```

HEAD:

```text
03885b2b062905072040dc245c3d25d283c249c4
```

Tree:

```text
b8341f7c114a4c51ea45d04352ee7ebdcc462e9a
```

Rust proxy entrypoint:

```text
hacp-rs/src/bin/hacp-rs-proxy.rs
```

Observed source SHA-256:

```text
6CFF4DE92056B6643510F704D622A7911EAE1027EBB972347B47EEC0C8FECC7D
```

Related implementation:

```text
hacp-rs/src/proxy.rs
```

Observed SHA-256:

```text
34A60CCEB8DC77D2B7D5C7CFE122B6372919A1774F1BF911DA690996FEC19503
```

---

# 21. Go HTTP versus Rust HTTP architecture

Go sidecar:

```text
client
    ->
HACP authorization
    ->
provenance
    ->
real upstream HTTP forwarding
    ->
real upstream response
    ->
client
```

Rust HTTP surface used for R8.2:

```text
client
    ->
HACP authorization
    ->
local ALLOW or DENY HTTP response
    ->
client
```

Therefore:

```text
GO_HTTP_SURFACE=
AUTHORIZATION_PLUS_REAL_UPSTREAM_FORWARDING
```

```text
RUST_HTTP_SURFACE=
AUTHORIZATION_PLUS_LOCAL_RESPONSE
```

and:

```text
DIRECT_GO_VS_RUST_LATENCY_RANKING=INVALID
```

The Rust adapter must not be described as performing real upstream forwarding.

Its current boundary is intentional and accepted for this controlled
verification.

---

# 22. R8.2 fresh Rust build

The controlled build used:

```text
cargo +stable-x86_64-pc-windows-msvc build --locked --offline
```

with an external `CARGO_TARGET_DIR`.

Fresh Rust proxy binary SHA-256:

```text
39279B9BCF9555D1754CCBCADBBE72BB13A76DBDB58088F9C9A94972D6166ADA
```

Build log SHA-256:

```text
2D835139ECD7ADB45C575B527A32CD7AF2A92FF4A3215DD53D3DCA02FCD7351F
```

---

# 23. R8.2 fresh fixtures

Warmup fixture:

```text
records=100
SHA256=
5002C8348C459A0EDE4E3F630F5E8F92DDF4722DC7FD9CFBAA82D080BC082639
```

Measured fixture:

```text
records=5000
SHA256=
62C8AD205F71F8A7879CB0C0D9B53C3B26840F06712F2F2DCF80867716FFAD02
```

Generation log SHA-256:

```text
DBCA8E47FBE06BB88BCF32F23220BF4EF37784146106B623F01E2555640CDF06
```

No historical G/M fixture was used.

---

# 24. R8.2 negative fail-closed smoke

Rust proxy listener:

```text
127.0.0.1:18081
```

A request without HACP headers returned:

```text
HTTP 403
X-HACP-Decision: DENY
X-HACP-Reason: INVALID_ENVELOPE
```

Disposition:

```text
R8_2_FAIL_CLOSED_SMOKE=PASS
```

---

# 25. R8.2 workload

Warmup:

```text
requests=100
success=100
failed=0
concurrency=1

mean=19.470437 ms
p50=19.6215 ms
p95=22.2985 ms
p99=25.7155 ms
max=37.3308 ms
```

Measured:

```text
requests=5000
success=5000
failed=0
concurrency=1

mean=19.91901754 ms
p50=20.0561 ms
p95=24.4273 ms
p99=32.182 ms
max=88.6014 ms
```

Disposition:

```text
R8_2_WORKLOAD=PASS
R8_2_RUNTIME_WORKLOAD=PASS
```

These timing results are informational.

They must not be directly ranked against the full Go sidecar forwarding path
because the request boundaries are different.

---

# 26. R8.2 evidence identities

```text
03_proxy_stdout.log
SHA256=
E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855
```

```text
03_proxy_stderr.log
SHA256=
D791A1027993F427B827B2433158821051DD98A030692039D86675AC9D39C731
```

```text
04_fail_closed_smoke.json
SHA256=
8C36602F35162A0A525F01F358BA572D9B5995D3CBF98CF70CEDDB5656544CE9
```

```text
05_rust_http_workload.log
SHA256=
EA0507FF6CE53FF07A95C263CA517335DDA9BEA257BC2CC12704900CCA4DE705
```

---

# 27. Reproducing H-01 and H-02

The verification file already remains at its execution path:

```text
internal/proxy/r4_provenance_verification_test.go
```

Verify its SHA-256 before use:

```text
C2CF4C78B3349366D30EE237DF74A03784127145FB215EFF43C0384255CA68F1
```

From the repository root:

```powershell
go test ./internal/proxy `
  -run '^TestR43H0[12]' `
  -count=1 `
  -v
```

Expected test names:

```text
TestR43H01ProvenanceAcceptedBeforeForward
TestR43H02FullProvenanceBufferFailsClosedWithoutForward
```

A reproduction must capture:

```text
repository HEAD
Go toolchain
verification-file SHA
exact command
stdout/stderr
numeric RC
PASS/FAIL/SKIP
post-run repository status
```

---

# 28. Reproducing H-03

The H-03 harness remains in the controlled external evidence directory.

Build a fresh sidecar:

```powershell
go build `
  -o <external-evidence>\hacp-sidecar-h03.exe `
  .\cmd\sidecar
```

Generate one fresh signed fixture:

```powershell
go run .\benchmarks\generate-tokens `
  -count 1 `
  -out <external-evidence>\h03-token.jsonl `
  -method GET `
  -path /api/test
```

Run the external harness using:

```text
h03_graceful_shutdown.go
```

with:

```text
fresh sidecar executable
fresh token JSONL
stdout evidence path
stderr evidence path
provenance evidence path
```

Required result markers:

```text
LISTENER_READY=PASS
AUTHORIZED_PROVENANCE_TRIGGER=PASS
CTRL_BREAK_RESULT=1
PROCESS_EXIT_CODE=0
LISTENER_REMOVED=PASS
PROVENANCE_PERSISTED_AFTER_SHUTDOWN=PASS
R4_3_H03=PASS
```

Do not replace the graceful signal with forced process termination.

---

# 29. Reproducing R8.2

The verification runner remains at:

```text
benchmarks/rust-http-reproduction/main.go
```

The procedure remains at:

```text
docs/verification/rust-http-local-reproduction.md
```

Follow that procedure against the exact admitted `hacp-spec` source identity or
a future explicitly recorded candidate identity.

A reproduction must use:

```text
fresh Rust build
external CARGO_TARGET_DIR
--locked
fresh generated fixtures
negative fail-closed smoke
100 warmup requests
5000 measured requests
concurrency=1
```

Do not reuse old G/M binaries or fixtures.

---

# 30. Source boundary

Allowed controlled source boundary:

```text
S = admitted public hacp-sidecar source
P = admitted public hacp-spec source
C = admitted public humanist-core source where applicable
fresh clones of those exact identities
```

Forbidden implicit execution dependencies include:

```text
G/M historical working trees
HACP_RUST laboratory
old binaries
private fixtures
manual copied vectors
unidentified external source files
```

The external evidence directory is an evidence-output location.

It is not automatically a source authority.

---

# 31. Why the verification files remain in place

The repository-local verification files are intentionally not moved into this
history directory.

Reason:

```text
their original repository paths are part of the reproducible execution context
```

Specifically:

```text
benchmarks/rust-http-reproduction/main.go
```

belongs to the benchmark/verification execution surface.

```text
docs/verification/rust-http-local-reproduction.md
```

belongs to the verification-procedure documentation surface.

```text
internal/proxy/r4_provenance_verification_test.go
```

belongs to the Go `proxy` test package and requires package-local test helpers.

Moving those files solely for archival appearance would make reproduction less
direct.

This README provides ownership and navigation without changing execution
locations.

---

# 32. Interpretation boundary

These verification artifacts do not establish:

```text
new normative HACP semantics
new protocol behavior
new production architecture
Rust upstream forwarding
Go/Rust latency equivalence
automatic release authorization
```

They establish bounded controlled verification claims only.

---

# 33. Preservation rule

Once these verification artifacts become admitted historical evidence,
preserve their exact identities.

If a later verification generation needs changed logic:

```text
do not silently rewrite the historical witness
preserve the existing identity
create a successor verification artifact
record the source identity
record the procedural or behavioral delta
```

---

# 34. Summary

The current ownership mapping is:

```text
benchmarks/rust-http-reproduction/main.go
    -> R8.2 verification runner

docs/verification/rust-http-local-reproduction.md
    -> R8.2 reproduction procedure

internal/proxy/r4_provenance_verification_test.go
    -> R4.3 H-01 / H-02 verification witness

external R4_3/h03_graceful_shutdown.go
    -> R4.3 H-03 real-process harness

docs/history-artifacts/hacp-1.1.1/test/
README_R4_R8_REPRODUCTION.md
    -> ownership, provenance, execution record and reproduction navigation
```

The actual verification files remain at their original execution locations.

This document is the repository-owned historical navigation record explaining
why they exist and how their results can be reproduced.

---

# Public Repository Ownership of H-03 Reproduction Evidence

The H-03 reproduction is no longer dependent on an external private evidence
directory for its executable verification logic.

The process harness used by the controlled run is preserved byte-identically at:

```text
tools/reproduction/graceful-shutdown/main.go
```

Its SHA-256 is:

```text
A216FA2115991937F8A710FA8A2F3D5460655BD91D8D9330DCAA52436AE7FB49
```

The following controlled evidence is preserved byte-identically under:

```text
docs/history-artifacts/hacp-1.1.1/test/evidence/r4-3/
```

```text
H01_H02_provenance_proxy.log
SHA256=19814AD5919F94AD9D2CC46AAFCA38F855BA6BDC7175533C46E2EB3C361B3B1D

H03_graceful_shutdown.log
SHA256=175D3B62249775BF510BDB5DF4DDE20EC71DA2A8B46EAF507DCFBCB6AEE677B9

H03_provenance.jsonl
SHA256=528220317885F1C353D3E60BDAC7376C5AE3CA980F43CA320490E06057C52A70

H03_sidecar_stdout.log
SHA256=E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855

h03-token.jsonl
SHA256=755CB1633E3FD638F095785999569C4E96D3181A9E4F2AEC38100AAA64BEFE6F
```

`H03_sidecar_stderr.log` contained one environment-local Windows path and is
therefore published as a sanitized historical derivative.

```text
ORIGINAL_INTERNAL_SHA256=
E3F167E2DC1E03C5EA6C396ADBD097AC5C2372090FABB1533949F2A2339F38A4

PUBLICATION_FORM=
SANITIZED_HISTORICAL_DERIVATIVE

SANITIZATION_SCOPE=
LOCAL_PROVENANCE_PATH_ONLY
```

The original local provenance path was replaced with:

```text
<EVIDENCE_ROOT>\H03_provenance.jsonl
```

No runtime conclusion, authorization result, HTTP status, process result,
listener result, provenance content, or test outcome was changed.

The controlled executable itself is not stored in Git.

Its executed historical identity remains:

```text
hacp-sidecar-h03.exe
SHA256=
5C02C1C42BF00707D6AEFEC65CE2CDDAC6752892C7AD67A073D4AE1B7E6233C5
```

A reproduction must build a fresh executable from the pinned sidecar source
rather than reuse the historical binary.

Therefore the repository-owned reproduction boundary is:

```text
SOURCE / PROCEDURE / HARNESS
    -> public repository

HISTORICAL TEXTUAL EVIDENCE
    -> public history-artifacts surface

HISTORICAL EXECUTABLE
    -> identity recorded only; fresh build required
```

No private evidence directory is required to understand or rerun H-03.
