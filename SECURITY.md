# Security Policy

`hacp-sidecar` is a security-sensitive enforcement component for the Human Agency Continuity Protocol (HACP).

Security reports are welcome and should be handled privately until a fix or coordinated disclosure plan is available.

## Supported Versions

The project is currently preparing its first public release candidate.

| Version | Supported |
|---|---|
| `0.5.0-rc.1` | Yes |
| Earlier development snapshots | No |

Support policy will be updated as stable releases are published.

## Reporting a Vulnerability

Please report suspected security vulnerabilities privately by email:

`digital.humanism.collective@protonmail.com`

Include, where possible:

- affected component and version;
- a clear description of the issue;
- reproduction steps or proof of concept;
- expected and observed behavior;
- potential security impact;
- suggested remediation, if known.

Please do not open a public GitHub issue for vulnerabilities that could enable bypass of HACP enforcement, signature validation, revocation, authority boundaries, replay protection, provenance integrity, or fail-closed behavior.

## Security-Relevant Areas

Reports are especially important for issues involving:

- signature or canonicalization bypass;
- `IntentEnvelope` or `DecisionToken` validation;
- action binding and `action_hash`;
- replay protection and autonomy-budget enforcement;
- revocation propagation and distributed control state;
- revision gaps, stale control state, or reconnect recovery;
- scope-boundary enforcement;
- fail-open behavior;
- provenance integrity;
- authorization bypass through malformed wire inputs;
- sidecar deployment or configuration that permits enforcement bypass.

## Disclosure Process

After receiving a report, maintainers will:

1. acknowledge receipt;
2. assess reproducibility and impact;
3. determine affected versions;
4. prepare and validate a fix where required;
5. coordinate disclosure timing with the reporter when appropriate;
6. publish remediation guidance or a patched release.

No guaranteed response or remediation timeline is currently committed for the release-candidate phase.

## Security Model and Limitations

Passing conformance, regression, coverage, and interoperability tests does not constitute a formal security proof.

The current release candidate should be treated as a validated implementation under active hardening rather than as a formally verified enforcement system.

Architecture, threat-model, conformance, and release documentation should be reviewed before production deployment.

---

**Contact:** [digital.humanism.collective@protonmail.com](mailto:digital.humanism.collective@protonmail.com)
