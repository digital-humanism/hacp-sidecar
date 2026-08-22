# Key Lifecycle and Trust Configuration

**Status:** Production hardening profile  
**Normative effect:** None  
**Applies to:** `hacp-sidecar` production trust configuration

## Overview

HACP Sidecar resolves signer identities through explicit Ed25519 public-key trust.

Production trust is represented as a complete validated snapshot and activated atomically.

The production trust path is:

```text
trust source
    ↓
validated TrustSnapshot
    ↓
AtomicTrustStore
    ↓
KeyResolver
    ↓
evaluation pipeline
```

Revocation remains a separate authorization check. A key may resolve successfully and still be denied because its `signer_key_id` is revoked.

## Production startup

Production mode is the default.

```text
HACP_TRUST_MODE=production
```

Production mode requires:

```text
HACP_TRUST_KEYS_FILE=<path>
```

If the configured trust source is missing, malformed, empty, internally conflicting, or otherwise invalid, the sidecar does not start with an implicit fallback key.

The published HACP conformance key is not a production trust root.

## Explicit test mode

The built-in conformance key is available only through explicit test mode:

```text
HACP_TRUST_MODE=test
```

Test mode must not be combined with `HACP_TRUST_KEYS_FILE`.

This separation prevents test trust material from becoming an accidental production default.

## Trust file

The current implementation-level file format is:

```json
{
  "revision": 1,
  "keys": [
    {
      "key_id": "production-signer-01",
      "public_key_hex": "<32-byte Ed25519 public key as 64 hex characters>"
    }
  ]
}
```

This format is an implementation/deployment format. It is not a normative HACP wire object.

Requirements:

- `revision` must be greater than zero;
- the key set must be non-empty;
- each `key_id` must be non-empty;
- every public key must be a valid 32-byte Ed25519 public key;
- identical duplicate bindings are idempotent;
- the same `key_id` bound to different public keys is rejected.

## Trust-state invariants

### Explicit trust only

Unknown signer identities fail closed.

### Unambiguous signer identity

A `signer_key_id` cannot silently change its public-key binding.

### Atomic activation

Readers observe either the previous complete snapshot or the next complete snapshot.

A partially loaded trust state is never exposed.

### Rollback protection

A candidate trust revision lower than the active revision is rejected by default.

### Revision conflict protection

The same revision with different trust content is rejected.

The same revision with identical content is idempotent.

### Revocation remains independent

Trust replacement does not clear or override revocation state.

A revoked key remains denied even if it exists in the active trust snapshot.

## Planned rotation

A planned signer rotation uses overlapping trust snapshots:

```text
revision N
    old key

revision N+1
    old key
    new key

revision N+2
    new key
```

Safe sequence:

1. distribute the new public key;
2. activate the overlap snapshot;
3. verify the new signer is trusted;
4. switch new issuance to the new key;
5. allow the defined overlap window to drain;
6. activate the new-only snapshot.

A planned rotation is not a substitute for revocation.

If compromise is suspected, revoke the affected `signer_key_id` immediately.

## Runtime reload

Runtime trust reload is explicit.

When the optional local trust-admin surface is enabled:

```text
POST /trust/reload
```

reloads the configured `HACP_TRUST_KEYS_FILE`, validates the complete candidate snapshot, and atomically activates it only if all checks succeed.

Rejected reloads preserve the previous active safe state.

There is intentionally no automatic file watcher or polling loop in the current profile.

## Trust observability

When the trust-admin surface is enabled:

```text
GET /trust
```

returns operational trust metadata:

- revision;
- fingerprint;
- key count;
- source;
- loaded-at timestamp.

These values are operational diagnostics. They do not create authorization.

## Local admin surface

The trust-admin server is:

- disabled by default;
- enabled only with `HACP_TRUST_ADMIN_ADDR`;
- restricted to loopback addresses.

Examples:

```text
127.0.0.1:9081
localhost:9081
[::1]:9081
```

Non-loopback bind addresses are rejected.

The local admin surface assumes host-local administrative access is privileged.

It is not a public remote-management API and must not be exposed through ingress, reverse proxy, external service-mesh routing, or equivalent external publication.

## Current non-goals

The current implementation does not provide:

- automatic file watching;
- remote trust administration;
- KMS/HSM-specific HACP fields;
- distributed trust replication through the HACP control plane;
- normative key-lifecycle wire objects;
- normative signer-key expiry fields.

External KMS/HSM signing can be integrated later behind the signing boundary without changing current HACP verification semantics.

## Security rule

The production trust model is:

```text
validated immutable snapshot
        +
atomic activation
        +
read-only resolution
        +
separate revocation
```

not:

```text
mutable key map
        +
last write wins
```
