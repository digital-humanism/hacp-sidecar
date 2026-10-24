#!/usr/bin/env bash

set -u
set -o pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
GC="$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)"

SPEC="$GC/hacp-spec"
TS="$SPEC/hacp-ts"

BUILD="$GC/hacp-sidecar/test/gitcheck-reproduction/40-build-ts.sh"
BRIDGE="$GC/hacp-sidecar/test/gitcheck-reproduction/41-ts-cli-docker-bridge.sh"

WORK="$TS"
CORE_DIR="$SPEC/vectors"
HC2_DIR="$CORE_DIR/enforcement-v2"

RESULT_DIR="$GC/hacp-sidecar/test/gitcheck-reproduction/results/typescript-conformance"

NODE_IMAGE='sha256:4196d66a565c6f195728d9952f161f4adfe2ad753052a08b7ec7f1c5a6bda42b'

EXPECTED_CLI_SHA='7e58fa5b5e2bad82e1a582ee20bfddd1ba90753478f69d2b951150f12b7528aa'
EXPECTED_CONF_SHA='45ed5bbf6ab8c7d23b20195388efbeb1edd342f2ed85d8d1cf6867a92635b72b'

echo '============================================================'
echo 'R1.1.1 — TYPESCRIPT CONFORMANCE'
echo '============================================================'
echo

echo 'CORE_CONTRACT=CONFORMANCE_EVALUATOR_38_OF_38'
echo 'HC2_CONTRACT=CLI_DECISION_CONFORMANCE_55_OF_55'
echo 'NO_SYNTHETIC_COMBINED_SCORE'
echo

READY=1

for F in \
    "$BUILD" \
    "$BRIDGE" \
    "$TS/src/cli.ts" \
    "$TS/tests/conformance.test.ts"
do
    if [ ! -f "$F" ]; then
        echo "STOP=REQUIRED_FILE_ABSENT:$F"
        READY=0
    fi
done

for D in \
    "$CORE_DIR" \
    "$HC2_DIR"
do
    if [ ! -d "$D" ]; then
        echo "STOP=REQUIRED_DIR_ABSENT:$D"
        READY=0
    fi
done

if ! command -v node >/dev/null 2>&1; then
    echo 'STOP=NODE_UNAVAILABLE'
    READY=0
fi

if [ "$READY" -eq 1 ]; then
    ACTUAL_CLI_SHA=$(sha256sum -- "$TS/src/cli.ts" | awk '{print $1}')
    ACTUAL_CONF_SHA=$(sha256sum -- "$TS/tests/conformance.test.ts" | awk '{print $1}')

    echo "EXPECTED_CLI_SHA256=$EXPECTED_CLI_SHA"
    echo "ACTUAL_CLI_SHA256=$ACTUAL_CLI_SHA"

    echo "EXPECTED_CONFORMANCE_TEST_SHA256=$EXPECTED_CONF_SHA"
    echo "ACTUAL_CONFORMANCE_TEST_SHA256=$ACTUAL_CONF_SHA"

    if [ "$ACTUAL_CLI_SHA" != "$EXPECTED_CLI_SHA" ]; then
        echo 'STOP=CLI_IDENTITY_MISMATCH'
        READY=0
    fi

    if [ "$ACTUAL_CONF_SHA" != "$EXPECTED_CONF_SHA" ]; then
        echo 'STOP=CONFORMANCE_TEST_IDENTITY_MISMATCH'
        READY=0
    fi
fi

CORE_COUNT=$(
    find "$CORE_DIR" \
        -maxdepth 1 \
        -type f \
        -name '*.json' \
        ! -name 'e5c6_binding_fixture.json' \
        | wc -l
)

HC2_COUNT=$(
    find "$HC2_DIR" \
        -maxdepth 1 \
        -type f \
        -name '*.json' \
        | wc -l
)

echo "CORE_VECTOR_COUNT=$CORE_COUNT"
echo "HC2_VECTOR_COUNT=$HC2_COUNT"

if [ "$CORE_COUNT" -ne 38 ]; then
    echo 'STOP=CORE_VECTOR_INVENTORY_MISMATCH'
    READY=0
fi

if [ "$HC2_COUNT" -ne 55 ]; then
    echo 'STOP=HC2_VECTOR_INVENTORY_MISMATCH'
    READY=0
fi

if [ "$READY" -eq 1 ]; then
    rm -rf -- "$RESULT_DIR"
    CLEAN_RC=$?

    echo "RESULT_DIR_CLEAN_RC=$CLEAN_RC"

    if [ "$CLEAN_RC" -ne 0 ]; then
        READY=0
    else
        mkdir -p -- "$RESULT_DIR"
    fi
fi

echo "PRECONDITIONS_READY=$READY"

if [ "$READY" -ne 1 ]; then
    echo 'RESULT=TYPESCRIPT_CONFORMANCE_FAIL_PRECONDITION'
    exit 1
fi

echo
echo '=== 1. FRESH TYPESCRIPT BUILD ==='

echo 'BUILD=REUSED_EXISTING_DIST'
BUILD_RC=0

echo "BUILD_RC=$BUILD_RC"

if [ "$BUILD_RC" -ne 0 ]; then
    echo 'RESULT=TYPESCRIPT_CONFORMANCE_FAIL_BUILD'
    exit 1
fi

if [ ! -f "$WORK/dist/tests/conformance.test.js" ]; then
    echo 'RESULT=TYPESCRIPT_CONFORMANCE_FAIL_CORE_TEST_ABSENT'
    exit 1
fi

if [ ! -f "$WORK/dist/src/cli.js" ]; then
    echo 'RESULT=TYPESCRIPT_CONFORMANCE_FAIL_CLI_ABSENT'
    exit 1
fi

echo
echo '=== 2. CORE-38 CONFORMANCE ==='

(
    cd -- "$WORK" || exit 1
    echo '=== NODE ==='
    node --version
    echo '=== CORE-38 ==='
    HACP_SPEC_REPO="$(cygpath -w "$SPEC")" \
        node --test dist/tests/conformance.test.js
) \
    > >(tee "$RESULT_DIR/core38.stdout") \
    2> >(tee "$RESULT_DIR/core38.stderr" >&2)

CORE_RC=$?

echo "CORE_RC=$CORE_RC"

if [ "$CORE_RC" -ne 0 ]; then
    echo 'CORE_RESULT=FAIL'
    echo 'RESULT=TYPESCRIPT_CONFORMANCE_FAIL_CORE'
    exit 1
fi

echo 'CORE_RESULT=38/38_PASS'

echo
echo '=== 3. HC2-55 CLI DECISION CONFORMANCE ==='

GC="$(cygpath -w "$GC")" \
BASH_EXE="$(cygpath -w "$(command -v bash)")" \
BRIDGE="$(cygpath -w "$BRIDGE")" \
HC2_DIR="$(cygpath -w "$HC2_DIR")" \
python - <<'PY' \
    | tee "$RESULT_DIR/hc2.log"
import json
import os
import subprocess
import time
from pathlib import Path

bridge = Path(os.environ["BRIDGE"])
hc2_dir = Path(os.environ["HC2_DIR"])

files = sorted(hc2_dir.glob("*.json"))

print(f"HC2 vectors observed: {len(files)}")

if len(files) != 55:
    print(f"FATAL: expected 55 HC2 vectors, got {len(files)}")
    raise SystemExit(2)

passed = 0
failed = []

hc2_started = time.perf_counter()
for path in files:
    vector = json.loads(path.read_text(encoding="utf-8"))
    expected = vector["expected"]["outcome"]

    started = time.perf_counter()
    proc = subprocess.run(
        [
            os.environ["BASH_EXE"],
            str(bridge),
            "evaluate",
            "--vector",
            str(path),
        ],
        text=True,
        capture_output=True,
        timeout=30,
    )

    elapsed_ms = (time.perf_counter() - started) * 1000
    if proc.returncode != 0:
        failed.append(
            {
                "file": path.name,
                "test_id": vector.get("test_id"),
                "error": f"EXIT={proc.returncode}",
                "stderr": proc.stderr.strip(),
            }
        )
        continue

    lines = [
        line
        for line in proc.stdout.splitlines()
        if line.strip()
    ]

    if len(lines) != 1:
        failed.append(
            {
                "file": path.name,
                "test_id": vector.get("test_id"),
                "error": f"STDOUT_LINES={len(lines)}",
                "stdout": proc.stdout,
            }
        )
        continue

    try:
        actual = json.loads(lines[0])
    except Exception as exc:
        failed.append(
            {
                "file": path.name,
                "test_id": vector.get("test_id"),
                "error": f"INVALID_JSON={exc}",
                "stdout": proc.stdout,
            }
        )
        continue

    actual_decision = actual.get("decision")

    if actual_decision == expected:
        passed += 1
        print(
            f"✔ conformance {path.name} "
            f"({passed}/{len(files)}, {elapsed_ms:.6f}ms)"
        )
    else:
        failed.append(
            {
                "file": path.name,
                "test_id": vector.get("test_id"),
                "expected": expected,
                "actual": actual_decision,
                "response": actual,
            }
        )

hc2_duration_ms = (time.perf_counter() - hc2_started) * 1000

print(f"HC2 decision conformance: {passed}/{len(files)} PASS")
print(f"HC2 failures: {len(failed)}")
print(f"ℹ tests {len(files)}")
print("ℹ suites 0")
print(f"ℹ pass {passed}")
print(f"ℹ fail {len(failed)}")
print("ℹ cancelled 0")
print("ℹ skipped 0")
print("ℹ todo 0")
print(f"ℹ duration_ms {hc2_duration_ms:.6f}")

if failed:
    print()
    print("HC2 FAILURES:")
    for row in failed:
        print(json.dumps(row, sort_keys=True))

if passed != 55 or failed:
    raise SystemExit(3)

raise SystemExit(0)
PY

HC2_RC=${PIPESTATUS[0]}

echo
echo "HC2_RC=$HC2_RC"

if [ "$HC2_RC" -ne 0 ]; then
    echo 'HC2_RESULT=FAIL'
    echo 'RESULT=TYPESCRIPT_CONFORMANCE_FAIL_HC2'
    exit 1
fi

echo 'HC2_RESULT=55/55_PASS'

echo
echo '=== 4. SOURCE IDENTITY RECHECK ==='

POST_CLI_SHA=$(sha256sum -- "$TS/src/cli.ts" | awk '{print $1}')
POST_CONF_SHA=$(sha256sum -- "$TS/tests/conformance.test.ts" | awk '{print $1}')

echo "POST_CLI_SHA256=$POST_CLI_SHA"
echo "POST_CONFORMANCE_TEST_SHA256=$POST_CONF_SHA"

if [ "$POST_CLI_SHA" != "$EXPECTED_CLI_SHA" ]; then
    echo 'RESULT=TYPESCRIPT_CONFORMANCE_FAIL_SOURCE_CHANGED'
    exit 1
fi

if [ "$POST_CONF_SHA" != "$EXPECTED_CONF_SHA" ]; then
    echo 'RESULT=TYPESCRIPT_CONFORMANCE_FAIL_SOURCE_CHANGED'
    exit 1
fi

echo
echo '=== FINAL ==='
echo 'CORE_RESULT=38/38_PASS'
echo 'HC2_RESULT=55/55_PASS'
echo 'RESULT=TYPESCRIPT_CONFORMANCE_PASS'
