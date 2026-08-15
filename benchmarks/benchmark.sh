#!/usr/bin/env bash
# Benchmark: upstream directly vs through sidecar.
# Requires: hey (https://github.com/rakyll/hey) or wrk.

set -euo pipefail

SIDECAR="http://localhost:8080"
UPSTREAM="http://localhost:8000"
TOKENS_FILE="benchmarks/tokens.jsonl"
RESULTS_DIR="benchmarks/results"

mkdir -p "$RESULTS_DIR"

# Generate tokens
echo "Generating tokens..."
go run ./benchmarks/generate-tokens.go -count 10000 -out "$TOKENS_FILE"

# Read first token for load test
FIRST_TOKEN=$(head -n 1 "$TOKENS_FILE")
ENV=$(echo "$FIRST_TOKEN" | jq -r '.env')
TOK=$(echo "$FIRST_TOKEN" | jq -r '.tok')

echo ""
echo "============================================================"
echo " Baseline: upstream directly"
echo "============================================================"
hey -n 10000 -c 50 \
    -H "Connection: keep-alive" \
    "$UPSTREAM/api/test" | tee "$RESULTS_DIR/baseline.txt"

echo ""
echo "============================================================"
echo " Sidecar: through HACP enforcement"
echo "============================================================"
hey -n 10000 -c 50 \
    -H "X-HACP-Intent-Envelope: $ENV" \
    -H "X-HACP-Decision-Token: $TOK" \
    -H "Connection: keep-alive" \
    "$SIDECAR/api/test" | tee "$RESULTS_DIR/sidecar.txt"

echo ""
echo "============================================================"
echo " Comparison"
echo "============================================================"
echo "Baseline p99:"
grep "99%" "$RESULTS_DIR/baseline.txt" || grep "Slowest" "$RESULTS_DIR/baseline.txt"
echo ""
echo "Sidecar p99:"
grep "99%" "$RESULTS_DIR/sidecar.txt" || grep "Slowest" "$RESULTS_DIR/sidecar.txt"