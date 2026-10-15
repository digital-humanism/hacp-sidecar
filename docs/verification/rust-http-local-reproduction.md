# Rust HTTP Local-Response Reproduction — R8.2

**Status:** REPRODUCED / PENDING REPOSITORY ADMISSION. **Type:** S-owned verification tooling; not a protocol specification or product-runtime change.

## Source and behavior boundaries

- S: public repository root `hacp-sidecar`, branch `admit/hacp-1.1.1-pre-release`; repository-owned procedure and verification client live in this repository.
- P: public peer repository `hacp-spec`; Rust proxy sources are consumed from the exact immutable specification identity selected by the active HACP 1.1.1 composition/reproduction contract.
- C: frozen, no mutation.
- Rust server: tracked `P/hacp-rs/src/bin/hacp-rs-proxy.rs`, SHA-256 `6CFF4DE92056B6643510F704D622A7911EAE1027EBB972347B47EEC0C8FECC7D`.
- Rust evaluation: tracked `P/hacp-rs/src/proxy.rs`, SHA-256 `34A60CCEB8DC77D2B7D5C7CFE122B6372919A1774F1BF911DA690996FEC19503`.
- Token generator: tracked `S/benchmarks/generate-tokens/main.go`, SHA-256 `BAFD98089B1165EBF7390818DA4A71FE935CDBE01E62A90AC9B9A92717C10608`.
- Rust surface: `AUTHORIZATION_PLUS_LOCAL_RESPONSE`. **No upstream forwarding** and no Go-vs-Rust direct latency ranking.
- Excluded: G/M, historical `70-http-performance.py`, historical binaries, old fixtures, lab/private build or test inputs. External directories may contain **outputs** only, never source inputs.

## Repository-ownership disposition

The R8.2 Rust HTTP runtime reproduction has been executed successfully from the S/P public-source boundary using this procedure and the S-owned verification client. The remaining repository-ownership action is to admit this procedure and client into tracked `hacp-sidecar` history through the controlled signed-commit workflow. This document does not authorize an automatic push, merge, tag, release, protocol change, or product-runtime change.

## Accepted procedure requirements

1. Verify S/P/C branch, HEAD, tree, status, tracked inputs and hashes. Preserve the known P untracked release-passport exception.
2. Create an external evidence directory and an external `CARGO_TARGET_DIR`; do not put generated inputs or artifacts in S/P/C.
3. Fresh-build `hacp-rs-proxy` from immutable P using `cargo +stable-x86_64-pc-windows-msvc build --locked --offline --manifest-path <P>\hacp-rs\Cargo.toml --bin hacp-rs-proxy`, with `CARGO_TARGET_DIR` pointing outside repositories. Record cargo/rustc/protoc versions, exact invocation, RC and binary SHA-256. This invocation was exercised during the controlled A2 reproduction and produced a successful fresh Rust proxy build.
4. Generate two **fresh** signing-token JSONL fixtures from tracked S generator with `go run .\benchmarks\generate-tokens -count 100 -out <external>\tokens-warmup-100.jsonl -method GET -path /api/test` and repeat with `-count 5000`. Verify 100 and 5000 lines and record SHA-256. Reconfirm flags against tracked generator before execution.
5. Start the freshly built Rust proxy with `HACP_SIDECAR_PORT=18081` on `127.0.0.1`; capture stdout/stderr separately and process PID. Verify listener belongs to that process; avoid conflicts with an existing service.
6. Run a pre-load **negative fail-closed smoke**: `GET http://127.0.0.1:18081/api/test` **without HACP headers** must return HTTP 403 and `X-HACP-Decision: DENY` (reason is captured, not fabricated). This must not be counted in workload samples.
7. Run the separately tracked S client with fresh fixture paths: `go run .\benchmarks\rust-http-reproduction -url http://127.0.0.1:18081/api/test -warmup-file <external>\tokens-warmup-100.jsonl -measured-file <external>\tokens-5000.jsonl`. Its pinned contract is 100 warm-up then 5,000 measured **serial** `GET /api/test` requests, one fresh signed token pair per request, expected HTTP 200 and `X-HACP-Decision: ALLOW`; it records elapsed, mean, p50, p95, p99, max and exact counts. If warm-up fails, do not run measured. Persist raw outputs and numeric RC.
8. Stop **only the launched proxy PID**, verify exit and listener shutdown. Record cleanup and S/P/C post-guards. No process left running.
9. Capture SHA-256 for all final evidence and summarize each required success/negative outcome. This is **informational** timing, not a Gate D performance threshold or independent conformance claim.
