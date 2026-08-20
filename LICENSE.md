## License

This repository contains the HACP Enforcement Sidecar implementation, licensed under the **GNU Affero General Public License v3.0 (AGPLv3)**, with Commercial Dual Licensing available for enterprise deployments, closed-source embedding, and OEM integration.

For commercial licensing inquiries, contact: `digital.humanism.collective@protonmail.com`.

### Relationship to other HACP repositories

| Repository | Purpose | License |
|---|---|---|
| [`hacp-spec`](https://github.com/digital-humanism/hacp-spec) | Open standard (specification, schemas, conformance suite) | CC BY 4.0 |
| [`humanist-core`](https://github.com/digital-humanism/humanist-core) | Reference SDK (Python) | AGPLv3 + commercial dual |
| [`hacp-sidecar`](https://github.com/digital-humanism/hacp-sidecar) | Enforcement sidecar (Go) | AGPLv3 + commercial dual |

This deliberate separation ensures the protocol remains a vendor-neutral open standard, while reference tooling, enforcement implementations, and enterprise integrations maintain a sustainable commercial model.

### AGPLv3 notice for network deployment

Under AGPLv3 §13, any network-accessible deployment of this sidecar that serves external users requires the source code of the deployed version (including any modifications) to be made available to those users. Commercial licenses waive this requirement.
