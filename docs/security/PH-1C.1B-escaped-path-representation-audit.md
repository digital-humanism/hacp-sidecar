# PH-1C.1B — Escaped Path Representation Audit

## Status

PH-1C.1B examined whether percent-encoded HTTP paths could cause HACP authorization binding to collapse distinct request targets.

The audit found:

```text
Implementation defect:             No
Authorization bypass demonstrated: No
Authorization/execution mismatch:  No
Specification ambiguity:           Yes
```

The normative finding is tracked as:

```text
SPEC-REP-01
```

No production sidecar change is required by this audit result.

## Scope

The audit examined the representation used for:

```text
DecisionToken.constraints.path
```

The initial adversarial case was:

```text
/a/b
/a%2Fb
```

The audit did not redefine:

- `ProposedAction.resource_id`;
- `ProposedAction` semantics;
- `action_hash`;
- authority objects;
- checkpoint semantics;
- delegation semantics.

`resource_id` remains a semantic action field and is not treated as an HTTP request-target representation.

## Security invariant

The audit used the following enforcement invariant:

> An enforcement point must not authorize one execution-boundary request identity while forwarding or executing a different request identity unless the HACP specification explicitly defines the two representations as equivalent.

Authorization binding and execution identity therefore need a common, protocol-defined representation model.

## Encoded slash result

Two HTTP request targets were examined:

```text
/a/b
/a%2Fb
```

The decoded path representation can collapse both targets to:

```text
/a/b
```

while the escaped/request-target representations remain distinct.

The tested reverse-proxy execution path preserved the distinction:

```text
/a/b
→ /a/b

/a%2Fb
→ /a%2Fb
```

The current sidecar request binding also preserved the same distinction.

Therefore, no authorization/execution mismatch was found for the encoded-slash case.

## Specification finding

The HACP Enforcement Profile defines request-level constraints including:

```text
method
path
tool_name
payload_hash
```

It also requires the enforcement point to verify that the current request is inside token scope, and request-binding mismatches fail with:

```text
SCOPE_EXCEEDED
```

However, the profile does not currently define which representation must be used for `constraints.path`.

In particular, it does not specify whether the binding representation is:

- a decoded URI path;
- an escaped URI path;
- an HTTP request-target path representation;
- a canonicalized URI representation.

This creates an interoperability and security ambiguity because distinct HTTP request targets may decode to the same path.

## SPEC-REP-01

The audit records the following normative finding:

> HACP Enforcement defines HTTP path binding but does not define the representation against which `constraints.path` is compared.
>
> This distinction is security-relevant because distinct HTTP request targets may decode to the same path.
>
> The current sidecar preserves the distinction between `/a/b` and `/a%2Fb` consistently across authorization binding and upstream execution.
>
> No implementation defect was demonstrated. Resolution requires normative specification clarification rather than a production-code change.

## Additional representation checks

The audit also examined:

```text
/a%2Fb
/a%2fb
/a%252Fb
```

The HTTP execution path preserved all three representations independently.

No recursive percent-decoding was observed for the double-encoded form.

These results are implementation evidence only. They do not by themselves define HACP protocol semantics.

## Normative direction

A separate HC2 specification review is required to define the request-binding representation.

The proposed direction is to bind authorization to the HTTP request-target path representation observed at the enforcement boundary, excluding scheme and authority and applying only explicitly defined HACP normalization rules.

The proposal also considers percent-encoding hexadecimal digit case as a candidate explicit equivalence:

```text
/a%2Fb
/a%2fb
```

while preserving distinctions such as:

```text
/a/b
!=
/a%2Fb
```

and:

```text
/a%2Fb
!=
/a%252Fb
```

These semantics remain subject to normative HC2 approval.

## Query binding

PH-1C.1A previously established query-inclusive request binding in the sidecar.

For example:

```text
/transfer?account=A
```

and:

```text
/transfer?account=B
```

remain distinct authorization targets.

Whether this representation becomes the normative definition of HACP `constraints.path` is part of the specification clarification and must not be inferred solely from the current implementation.

## Deferred cases

The following representation questions remain outside the completed PH-1C.1B audit:

```text
dot segments
duplicate slashes
encoded question mark
encoded percent sign
encoded unreserved characters
UTF-8 percent encoding
```

No normative equivalence should be inferred from implementation-specific HTTP normalization behavior.

## Disposition

PH-1C.1B is classified as:

```text
C. Specification ambiguity / architectural escalation
```

Current disposition:

```text
Production sidecar change:
None

Required action:
HC2 normative clarification

Implementation changes after HC2:
Only if required by the accepted normative representation
```

Production behavior must not be changed merely to encode a protocol interpretation that has not yet been accepted normatively.