# V7A External E2E Successor Binding Evidence

Status:

CLOSED — PASS

Target repository:

```text
C:\Personal\GitHub\Mount\hacp-sidecar
```

Target baseline:

```text
HEAD:
1bc10acbe79620a165cee573c5b260cd8639280f
```

## 1. Identity model

Historical identity:

```text
key_id:
key-ed25519-test-001

status:
HISTORICAL
IMMUTABLE
COMPATIBILITY IDENTITY
```

Successor identity:

```text
key_id:
key-ed25519-test-002

derivation:
SHA-256 of the deterministic successor test-key label

private material:
EPHEMERAL
NOT RECORDED
NOT COMMITTED
public_key:
ba55cfea3098a712b5c77374b33dd95e9b6d2df3ebe9ab1653ee3a5997c497fa

status:
SUCCESSOR
ADDITIVE
```

The historical `001` identity was not renamed, overwritten, rebound,
rebaked, or otherwise reinterpreted.

## 2. Successor positive verification

V7A-5 result:

```text
002 positive signature verification:
PASS

Ed25519 signature length:
64 bytes
```

Classification:

```text
SUCCESSOR_002_CRYPTOGRAPHIC_IDENTITY=PASS
```

## 3. Cross-version negative verification

V7A-6:

```text
001 signature against 002 public identity:
DENY
```

V7A-7:

```text
002 signature against 001 public identity:
DENY
```

Combined result:

```text
CROSS_VERSION_FAIL_CLOSED=PASS
```

These results establish the cryptographic identity boundary between the
historical `001` identity and the additive `002` successor identity.

## 4. Runtime trust binding

No successor key was embedded into the sidecar source.

The existing production trust path was used:

```text
HACP_TRUST_MODE=production
HACP_TRUST_KEYS_FILE=<ephemeral trust snapshot>
```

The ephemeral runtime trust snapshot bound:

```text
key-ed25519-test-002
```

to:

```text
ba55cfea3098a712b5c77374b33dd95e9b6d2df3ebe9ab1653ee3a5997c497fa
```

Result:

```text
TARGET_SIDECAR_RUNTIME=PASS
SUCCESSOR_002_RUNTIME_BINDING=PASS
```

No production Go source change was required to activate the successor
identity through the existing production trust mechanism.

## 5. Fresh Python <-> Go external E2E

Execution surfaces:

```text
Go target:
C:\Personal\GitHub\Mount\hacp-sidecar

Python SDK:
C:\Personal\GitHub\Dev\humanist-core

sidecar:
http://127.0.0.1:8080

ephemeral upstream:
http://127.0.0.1:8000
```

Python environment:

```text
Python 3.13.14
humanist-core project .venv
```

Final external E2E result:

```text
test_real_sidecar_is_fail_closed_without_hacp_headers        PASS
test_http_action_hash_matches_sidecar_shape                  PASS
test_python_envelope_and_token_signatures_are_self_consistent PASS
test_real_sidecar_allows_python_signed_request               PASS
test_sidecar_client_gets_allow                               PASS

TOTAL:
5/5 PASS
```

Final markers:

```text
UPSTREAM_RUNTIME=PASS
TARGET_SIDECAR_RUNTIME=PASS
SUCCESSOR_002_RUNTIME_BINDING=PASS
PYTHON_GO_EXTERNAL_E2E=PASS
```

## 6. Superseded intermediate executions

Two intermediate executions are explicitly excluded from final PASS
evidence.

### Intermediate execution A

Pytest collection stopped before any tests executed because the system
Python environment did not contain the declared `jcs` dependency.

Observed:

```text
collected 0 items / 1 error
ModuleNotFoundError: No module named 'jcs'
```

Classification:

```text
PYTHON_EXECUTION_ENVIRONMENT_ERROR
NOT HACP FAILURE
NOT SIDECAR FAILURE
NOT KEY FAILURE
```

Any PASS marker printed after that STOP is VOID.

### Intermediate execution B

Using the correct humanist-core project `.venv`, five tests executed.

Three passed and two signed-request tests reached an HACP `ALLOW`
decision, but returned HTTP 502 because no upstream process was listening
on `127.0.0.1:8000`.

Observed sidecar condition:

```text
upstream request failed
dial tcp 127.0.0.1:8000
connection refused
```

Classification:

```text
UPSTREAM_RUNTIME_ABSENT
HACP_ALLOW_PATH_REACHED
NOT KEY FAILURE
NOT TRUST_BINDING_FAILURE
```

Any final PASS marker printed after that STOP is VOID.

The subsequent clean execution with an ephemeral upstream supersedes this
intermediate run and produced 5/5 PASS.

## 7. Behavioral classification

The V7A successor-key work did not change:

```text
Ed25519 algorithm
SHA-256 algorithm
JCS behavior
HACP envelope signature semantics
HACP token signature semantics
key resolution semantics
trust snapshot format
runtime evaluation pipeline
historical 001 identity
```

The successor work adds a distinct test/conformance identity and proves
that the existing explicit production trust mechanism can bind that
identity without changing protocol behavior.

Classification:

```text
ADDITIVE SUCCESSOR IDENTITY
NO GLOBAL 001 -> 002 REPLACEMENT
NO HISTORICAL REINTERPRETATION
NO PRODUCTION CRYPTOGRAPHIC SEMANTIC CHANGE
```

## 8. V7A closure

Closed slices:

```text
V7A-1  successor surface capture                   PASS
V7A-2  deterministic successor derivation          PASS
V7A-3  additive successor fixture materialization  PASS
V7A-4  historical 001 compatibility                PASS
V7A-5  successor 002 positive verification         PASS
V7A-6  001 -> 002 cryptographic negative           PASS
V7A-7  002 -> 001 cryptographic negative           PASS
V7A-8  fresh Python <-> Go external E2E             5/5 PASS
V7A-9  evidence reconciliation                     PASS
```

Overall:

```text
V7A=CLOSED
V7A_RESULT=PASS
SUCCESSOR_002_IDENTITY=PROVEN
CROSS_VERSION_BOUNDARY=PROVEN
TARGET_RUNTIME_BINDING=PROVEN
PYTHON_GO_EXTERNAL_E2E=5/5_PASS
```

Publication status:

```text
PUBLICATION_AUTHORIZED=NO
```

Reason:

```text
V8 composition / HOLD reconciliation subsequently closed PASS; publication remains subject to V9 final candidate review and the controlled transfer decision.
```

No publication authorization is implied by V7A closure.



---

## Executed V7A Verification Evidence

The successor identity was exercised through the existing explicit trust
mechanism without any production Go source change.

```text
SUCCESSOR_002_RUNTIME_BINDING=PASS

002 signer -> 002-only trust:
Python <-> Go external E2E = 5/5 PASS
repeat verification         = 5/5 PASS

001 signer -> 002-only trust:
HTTP_STATUS=403
X_HACP_DECISION=DENY
REASON=SIGNATURE_FAILURE
FAIL_CLOSED=PASS

002 signer -> 001-only trust:
HTTP_STATUS=403
X_HACP_DECISION=DENY
REASON=SIGNATURE_FAILURE
FAIL_CLOSED=PASS

CROSS_VERSION_FAIL_CLOSED=PASS
```

The historical `key-ed25519-test-001` identity remains immutable.
The successor `key-ed25519-test-002` identity is additive.

```text
NO GLOBAL 001 -> 002 REPLACEMENT
NO HISTORICAL REINTERPRETATION
NO PRODUCTION CRYPTOGRAPHIC SEMANTIC CHANGE
```
