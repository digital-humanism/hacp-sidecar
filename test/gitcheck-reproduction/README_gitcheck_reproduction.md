# HACP 1.1.1 — GitCheck Conformance Reproduction (Windows)

## Purpose and status

This directory holds **seven working copies of historical R1.1.1 GitCheck shell scripts**, adapted where necessary to reproduce the available conformance checks on **native Windows using Git Bash** and existing development repositories. They are **reproduction tooling**, not the normative specification, a production runtime, or an independently completed release/admission gate.

The scripts are stored under `hacp-sidecar/test/gitcheck-reproduction/` for this local exercise. **Their location does not imply that the `hacp-sidecar` service is started, exercised, or certified by these runs.** The tested implementations and inputs are primarily in the neighboring `hacp-spec` checkout. None of the seven scripts was observed to execute or import `humanist-core` directly.

This document records **observed local runs**, not a claim that every historical Docker-based GitCheck step was reproduced. Core and HC2 are **separate contracts**; do not add their counts into a synthetic combined score.

## Repository layout and boundaries

The adapted scripts resolve `GC` to the workspace parent directory, with sibling checkouts:

```text
<workspace>\
  hacp-spec\
  hacp-sidecar\
    test\gitcheck-reproduction\
      30-hacp-go.sh
      40-build-ts.sh
      41-ts-cli-docker-bridge.sh
      42-typescript-conformance.sh
      50-build-rust.sh
      51-rust-conformance.sh
      60-python-conformance.sh
      results\                 # generated local evidence; not normative inputs
  humanist-core\
```

The branch observed for `hacp-sidecar` during the reproduction was `admit/hacp-1.1.1-pre-release`. Script adaptations were confined to working copies under this directory. The recorded runs did **not** involve changes to production implementations, the `hacp-spec` harness, normative vectors, or `humanist-core`; no commit or push was performed.

**Dependency terminology:** *Executed* means a runner, CLI, or harness was actually invoked. *Input* means vectors, tests, source, or metadata consumed or checked. *Storage only* means that `hacp-sidecar` contains the copied shell script and its generated evidence, rather than supplying tested sidecar behavior.

## Script-by-script repository dependency matrix

| Script | Purpose / execution status | `hacp-spec` — inputs and code used | `hacp-sidecar` — items used | `humanist-core` |
|---|---|---|---|---|
| `30-hacp-go.sh` | **Executed; PASS.** Builds the existing Go implementation locally and uses the spec harness for Core and HC2. | `hacp-go/` (Go build and runner); `harness/harness.py`; `vectors/*.json` (Core-38, excluding auxiliary E5C6 fixture); `vectors/enforcement-v2/*.json` (HC2-55). | This script and `results/hacp-go/` (binary, build log, per-family logs and hash). **No sidecar server executed.** | **Not used directly.** |
| `40-build-ts.sh` | Historical **Docker-based fresh TypeScript build** helper; **not executed** in this Windows series. | `hacp-ts/` source and package/build inputs if run. | Script stored here; historical derived build path outside this directory if run. **No sidecar server dependency.** | **Not used directly.** |
| `41-ts-cli-docker-bridge.sh` | **Executed indirectly by `42`.** Adapted from a Docker bridge to native Node; invokes the existing TypeScript CLI for HC2 vectors. | `hacp-ts/dist/src/cli.js` and its existing runtime dependencies; vector files passed by `42` from `vectors/enforcement-v2/`. | This helper script only, called from `42`; **no sidecar server executed**. | **Not used directly.** |
| `42-typescript-conformance.sh` | **Executed; PASS.** Verifies TypeScript source identity, runs Core tests through Node, and evaluates 55 HC2 vectors through `41`. Reuses the existing build. | `hacp-ts/src/cli.ts` and `hacp-ts/tests/conformance.test.ts` (SHA checks); `hacp-ts/dist/tests/conformance.test.js` (Core tests); `hacp-ts/dist/src/cli.js` (HC2 CLI); `hacp-ts/node_modules/`; `vectors/` Core inventory; `vectors/enforcement-v2/` HC2 inputs. | Orchestrator `42`, bridge `41`, and `results/typescript-conformance/`. `40` is referenced but **not executed** in the adapted run. **No sidecar server executed.** | **Not used directly.** |
| `50-build-rust.sh` | Historical **Docker-based fresh Rust runner build** helper; **not executed** in this Windows series. | `hacp-rs/Cargo.toml`, `Cargo.lock`, Rust source (including `src/runner.rs`), proto/build inputs, and `vectors/` if run. | Script stored here; historical build/result directories if run. **No sidecar server dependency.** | **Not used directly.** |
| `51-rust-conformance.sh` | **Executed; historical script FAIL** with known Core deviations and separately passing HC2. Exercises existing release runner; also performs raw-wire witness inspection. | `hacp-rs/target/release/hacp-rs-runner.exe`; `vectors/` Core and `vectors/enforcement-v2/` HC2 inputs; `vectors/core_inv5_006_negative.json` for raw-wire witness. | Script and `results/rust-conformance/` (corpus log, raw-wire request/output/error). **No sidecar server executed.** | **Not used directly.** |
| `60-python-conformance.sh` | **Executed; PASS.** Uses native Python and the historical harness in local mode for Core and HC2. | `harness/harness.py` (executed with `--mode local`); `harness/requirements.txt` and `harness/conformance_manifest.json` (existence/identity checks); `vectors/*.json` (Core-38, copied into a derived subset); `vectors/enforcement-v2/*.json` (HC2-55). | Script and `results/python-conformance/` (derived Core subset and separate stdout/stderr files). **No sidecar server executed.** | **Not used directly.** |

### How `humanist-core` relates to these tests

The `humanist-core` repository was present in the development workspace, but **the inspected scripts do not directly reference its source files, import its SDK, or invoke its tests**. The Python conformance run executes **`hacp-spec/harness/harness.py`**, **not** `humanist-core/tests/conformance/...`. Therefore its 38/38 and 55/55 results must not be represented as a `humanist-core` SDK certification. Indirect conceptual or architectural relationships are **not** evidence of execution.

### How `hacp-sidecar` relates to these tests

The working shell scripts and `results/` are housed in `hacp-sidecar`, but the recorded Go, TypeScript, Rust, and Python checks **did not launch the `hacp-sidecar` Go service, run sidecar integration/E2E tests, or exercise live sidecar enforcement**. These results verify the specific `hacp-spec` implementation/harness surfaces named above, not sidecar runtime parity or deployment readiness.

## Observed results

| Language | Core contract | HC2 contract | Script outcome | Qualification |
|---|---|---|---|---|
| **Go** | **38/38 PASS** | **55/55 PASS** | **Exit 0 / PASS** | Local Go build and spec-harness execution; E13-specific package/preflight checks were outside this reproduction scope. |
| **TypeScript** | **38/38 PASS** | **55/55 PASS** | **Exit 0 / PASS** | Node Core runner reported 39 passing tests: 38 vectors **plus one inventory check**. HC2 invoked the existing CLI 55 times. Existing `dist` and `node_modules` reused; no fresh build. |
| **Python** | **38/38 PASS** | **55/55 PASS** | **Exit 0 / PASS** | `hacp-spec` harness v0.9.2, `--mode local`; existing Python installation, no pip install. |
| **Rust** | **Historical Core deviations; not a 38/38 PASS** | **55/55 PASS** | **Exit 1 / FAIL** | Observed ordinary Core Family A **35/37** with two known deviations (`CORE-INV1-002`, `CORE-RUNTIME-004`); separate Family B **12/37**. The historical script's checks and final failure verdict were deliberately preserved. |

**Rust interpretation.** Project-level prior adjudication uses a separate **36+2** Rust Core accounting model. This does **not** override the observed historical script output, does **not** convert `35/37` into a pass, and must **not** be reported as synthetic `38/38`. Both Rust HC2 families were observed to pass **55/55**. The raw-wire `CORE-INV5-006` witness produced the expected rejection (`DENY`, `INVALID_ACTION`), but its emitted raw-output hash did not match the historical fixed hash in this Windows run. Consequently **`51-rust-conformance.sh` exited 1**; this series did not close the Rust discrepancy.

## Execution environment and adaptations

- **OS / shell:** native Windows, PowerShell 7 as the launcher, and Git Bash from the active Git for Windows environment for shell scripts. `cygpath -w` bridges Git Bash paths to Windows-native executables.
- **Go:** native Go 1.26.5; a local `hacp-go.exe` was built into `results/hacp-go/build/` from existing `hacp-spec/hacp-go` source. No production source change.
- **TypeScript:** Node.js 24.19.0; existing `hacp-ts/dist/` and `node_modules/` reused. `41` uses native Node rather than Docker; `42` runs the existing Node tests and CLI. Windows Python invokes the Git Bash executable resolved from the active Git for Windows environment for the bridge.
- **Python:** native Python 3.13.14; `60` uses the existing `hacp-spec` harness and installed dependencies rather than a Docker image or `pip install`. `PYTHONIOENCODING=utf-8` avoids Windows console encoding errors. The `e5c6_binding_fixture.json` auxiliary file is absent and excluded from Core-38; it is not needed for the two executed Python conformance families.
- **Rust:** existing `hacp-rs-runner.exe` reused; `50-build-rust.sh` was not run. Windows path handling and native Python were used for the witness/corpus orchestration.
- **Docker historical steps:** `40-build-ts.sh` and `50-build-rust.sh` remain as historical build helpers and **were not executed**. Do not infer fresh-container reproducibility from the successful native-language conformance runs.

The adapted copies differ from their historical versions in local path bindings, native process invocation, and environment handling. SHA checks in adapted scripts refer to the **actual files present in these local `Dev` checkouts**; updating an expected hash in a temporary script is not a claim that the checkout matches the original historical GitCheck snapshot.

## Evidence locations

The following paths are relative to this README and identify **locally generated** output, not normative reference artifacts:

| Area | Recorded evidence |
|---|---|
| `results/hacp-go/` | `build.log`, `binary.sha256`, `core38.log`, `hc2-55.log`, `build/hacp-go.exe` |
| `results/typescript-conformance/` | `core38.stdout`, `core38.stderr`, `hc2.log` |
| `results/python-conformance/output/` | `core38.stdout`, `core38.stderr`, `hc2.stdout`, `hc2.stderr` |
| `results/python-conformance/build/core38/` | Derived copies of the 38 Core JSON vectors, excluding the auxiliary E5C6 fixture |
| `results/rust-conformance/` | `corpus.log`, `core_inv5_006_request.jsonl`, `core_inv5_006.stdout`, `core_inv5_006.stderr` |

These files can be overwritten by reruns and are **not an immutable evidence archive**. Run-specific stdout, stderr, exit codes, toolchain versions, source identities, and any required evidence hashes should be captured separately before making a formal release or admission claim.

## What this reproduction establishes — and does not establish

**Established by the recorded runs:** local Go, TypeScript, and Python Core/HC2 conformance results in the stated configurations; Rust HC2 success; the described Rust deviations and nonzero historical script verdict; and the absence of production-source edits, commits, and pushes during these test executions.

**Not established:** a full original Docker GitCheck rerun; fresh Rust/TypeScript container builds; E13 auxiliary/E2E coverage; `hacp-sidecar` live service or cross-repository integration behavior; direct `humanist-core` SDK conformance; a Rust Core 38/38 PASS; or final HACP 1.1.1 release/admission authorization.

**Policy:** retain the observed exit status and each contract's independent score. Do not weaken a test, rewrite normative vectors, fabricate missing inputs, replace existing files merely because hashes differ, or treat a successful local reproduction as release approval.
