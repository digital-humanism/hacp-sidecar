#!/bin/bash

set -u
set -o pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
GC="$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)"
SPEC="$GC/hacp-spec"

HARNESS="$SPEC/harness/harness.py"
REQ="$SPEC/harness/requirements.txt"

BUILD_ROOT="$GC/hacp-sidecar/test/gitcheck-reproduction/results/python-conformance/build"
SITE="$BUILD_ROOT/site"
CORE_DIR="$BUILD_ROOT/core38"

RESULT_DIR="$GC/hacp-sidecar/test/gitcheck-reproduction/results/python-conformance/output"

PY_IMAGE='sha256:8b9a8b28d9cc221c6ab5d40e9cfcd99429959f6a8f5171612a99147975ab043f'

EXPECTED_HARNESS_SHA='bc2f90b3a0516519f64e8c82707f963852487e1856a8c0eb05339c12e97c6549'
EXPECTED_REQ_SHA='d4cfb7542cad98541cbfcaade7b80ff569d2a98f06a2d579727f4dc71c1e79bc'
EXPECTED_MANIFEST_SHA='bcfc6cfe87cea4d7b07a73e1981e7e577be65f2bd89161bce2cac714ad96dac7'

HOST_UID=$(id -u)
HOST_GID=$(id -g)

echo '============================================================'
echo 'R1.1.1 — PYTHON DECLARED CONFORMANCE'
echo '============================================================'
echo

echo 'CORE_CONTRACT=HISTORICAL_LOCAL_MODE_38_OF_38'
echo 'HC2_CONTRACT=HISTORICAL_LOCAL_MODE_55_OF_55'
echo 'NO_SYNTHETIC_COMBINED_SCORE'
echo

READY=1

echo '=== 1. SOURCE / CONTRACT IDENTITY ==='

if [ ! -f "$HARNESS" ]; then
    echo "STOP=HARNESS_ABSENT:$HARNESS"
    READY=0
fi

if [ ! -f "$REQ" ]; then
    echo "STOP=REQUIREMENTS_ABSENT:$REQ"
    READY=0
fi

if [ ! -f "$SPEC/harness/conformance_manifest.json" ]; then
    echo 'STOP=CONFORMANCE_MANIFEST_ABSENT'
    READY=0
fi

if [ "$READY" -eq 1 ]; then
    ACTUAL_HARNESS_SHA=$(sha256sum -- "$HARNESS" | awk '{print $1}')
    ACTUAL_REQ_SHA=$(sha256sum -- "$REQ" | awk '{print $1}')
    ACTUAL_MANIFEST_SHA=$(
        sha256sum -- "$SPEC/harness/conformance_manifest.json" |
        awk '{print $1}'
    )

    echo "EXPECTED_HARNESS_SHA256=$EXPECTED_HARNESS_SHA"
    echo "ACTUAL_HARNESS_SHA256=$ACTUAL_HARNESS_SHA"

    echo "EXPECTED_REQUIREMENTS_SHA256=$EXPECTED_REQ_SHA"
    echo "ACTUAL_REQUIREMENTS_SHA256=$ACTUAL_REQ_SHA"

    echo "EXPECTED_MANIFEST_SHA256=$EXPECTED_MANIFEST_SHA"
    echo "ACTUAL_MANIFEST_SHA256=$ACTUAL_MANIFEST_SHA"

    if [ "$ACTUAL_HARNESS_SHA" != "$EXPECTED_HARNESS_SHA" ]; then
        echo 'STOP=HARNESS_IDENTITY_MISMATCH'
        READY=0
    fi

    if [ "$ACTUAL_REQ_SHA" != "$EXPECTED_REQ_SHA" ]; then
        echo 'STOP=REQUIREMENTS_IDENTITY_MISMATCH'
        READY=0
    fi

    if [ "$ACTUAL_MANIFEST_SHA" != "$EXPECTED_MANIFEST_SHA" ]; then
        echo 'STOP=MANIFEST_IDENTITY_MISMATCH'
        READY=0
    fi
fi

echo
echo '=== 2. VECTOR INVENTORY ==='

CORE_COUNT=$(
    find "$SPEC/vectors" \
        -maxdepth 1 \
        -type f \
        -name '*.json' \
        ! -name 'e5c6_binding_fixture.json' \
        -printf '.' \
        2>/dev/null |
    wc -c
)

AUX_COUNT=$(
    find "$SPEC/vectors" \
        -maxdepth 1 \
        -type f \
        -name 'e5c6_binding_fixture.json' \
        -printf '.' \
        2>/dev/null |
    wc -c
)

HC2_COUNT=$(
    find "$SPEC/vectors/enforcement-v2" \
        -maxdepth 1 \
        -type f \
        -name '*.json' \
        -printf '.' \
        2>/dev/null |
    wc -c
)

echo "CORE_VECTOR_COUNT=$CORE_COUNT"
echo "AUXILIARY_E5C6_COUNT=$AUX_COUNT"
echo "HC2_VECTOR_COUNT=$HC2_COUNT"

if [ "$CORE_COUNT" -ne 38 ]; then
    echo 'STOP=CORE_VECTOR_COUNT_NOT_38'
    READY=0
fi

echo 'AUXILIARY_E5C6=OPTIONAL_NOT_USED_IN_CORE_HC2'

if [ "$HC2_COUNT" -ne 55 ]; then
    echo 'STOP=HC2_VECTOR_COUNT_NOT_55'
    READY=0
fi

echo
echo '=== 3. WINDOWS PYTHON ==='

if ! command -v python >/dev/null 2>&1; then
    echo 'STOP=PYTHON_UNAVAILABLE'
    READY=0
else
    python --version
fi

if ! command -v cygpath >/dev/null 2>&1; then
    echo 'STOP=CYGPATH_UNAVAILABLE'
    READY=0
fi

echo "PRECONDITIONS_READY=$READY"

if [ "$READY" -ne 1 ]; then
    echo 'RESULT=PYTHON_DECLARED_CONFORMANCE_PRECONDITION_FAIL'
    exit 1
fi

echo
echo '=== 4. CLEAN DERIVED / RESULT AREAS ==='

rm -rf -- "$BUILD_ROOT"
BUILD_CLEAN_RC=$?

rm -rf -- "$RESULT_DIR"
RESULT_CLEAN_RC=$?

echo "BUILD_CLEAN_RC=$BUILD_CLEAN_RC"
echo "RESULT_CLEAN_RC=$RESULT_CLEAN_RC"

if [ "$BUILD_CLEAN_RC" -ne 0 ] || [ "$RESULT_CLEAN_RC" -ne 0 ]; then
    echo 'RESULT=PYTHON_DERIVED_CLEANUP_FAIL'
    exit 1
fi

mkdir -p -- "$SITE" "$CORE_DIR" "$RESULT_DIR"

echo
echo '=== 5. MATERIALIZE DERIVED CORE-38 VECTOR SET ==='

while IFS= read -r F; do
    cp -- "$F" "$CORE_DIR/"
done < <(
    find "$SPEC/vectors" \
        -maxdepth 1 \
        -type f \
        -name '*.json' \
        ! -name 'e5c6_binding_fixture.json' \
        -print |
    sort
)

DERIVED_CORE_COUNT=$(
    find "$CORE_DIR" \
        -maxdepth 1 \
        -type f \
        -name '*.json' \
        -printf '.' |
    wc -c
)

echo "DERIVED_CORE_VECTOR_COUNT=$DERIVED_CORE_COUNT"

if [ "$DERIVED_CORE_COUNT" -ne 38 ]; then
    echo 'RESULT=PYTHON_DERIVED_CORE_VECTOR_COUNT_FAIL'
    exit 1
fi

echo
echo '=== 6. EXISTING PYTHON ENVIRONMENT ==='

python --version
python -m pip --version
DEP_RC=$?

echo "DEPENDENCY_ENVIRONMENT_RC=$DEP_RC"

if [ "$DEP_RC" -ne 0 ]; then
    echo 'RESULT=PYTHON_ENVIRONMENT_FAIL'
    exit "$DEP_RC"
fi

echo 'PIP_INSTALL=NOT_RUN_EXISTING_ENVIRONMENT'

echo '=== 7. CORE-38 — HISTORICAL LOCAL MODE ==='

CORE_STARTED_NS=$(date +%s%N)

PYTHONDONTWRITEBYTECODE=1 PYTHONIOENCODING=utf-8 \
    python "$(cygpath -w "$HARNESS")" \
        --mode local \
        --vectors-dir "$(cygpath -w "$CORE_DIR")" \
    >"$RESULT_DIR/core38.stdout" \
    2>"$RESULT_DIR/core38.stderr"
CORE_RC=$?
CORE_ENDED_NS=$(date +%s%N)
CORE_DURATION_NS=$((CORE_ENDED_NS - CORE_STARTED_NS))
CORE_DURATION_MS_INT=$((CORE_DURATION_NS / 1000000))
CORE_DURATION_MS_FRAC=$((CORE_DURATION_NS % 1000000))

cat "$RESULT_DIR/core38.stdout"

if [ -s "$RESULT_DIR/core38.stderr" ]; then
    echo
    echo '=== CORE-38 STDERR ==='
    cat "$RESULT_DIR/core38.stderr"
fi

echo "CORE_RC=$CORE_RC"

if [ "$CORE_RC" -ne 0 ]; then
    echo 'RESULT=PYTHON_CORE38_EXECUTION_FAIL'
    exit "$CORE_RC"
fi

if ! grep -Eq 'RESULTS:[[:space:]]*38/38[[:space:]]+passed' \
    "$RESULT_DIR/core38.stdout"
then
    echo 'RESULT=PYTHON_CORE38_RESULT_MARKER_FAIL'
    exit 1
fi

echo 'CORE_RESULT=38/38_PASS'
echo 'ℹ tests 38'
echo 'ℹ suites 0'
echo 'ℹ pass 38'
echo 'ℹ fail 0'
echo 'ℹ cancelled 0'
echo 'ℹ skipped 0'
echo 'ℹ todo 0'
printf 'ℹ duration_ms %d.%06d\n' "$CORE_DURATION_MS_INT" "$CORE_DURATION_MS_FRAC"

echo
echo '=== 8. HC2-55 — HISTORICAL LOCAL MODE ==='

HC2_STARTED_NS=$(date +%s%N)

PYTHONDONTWRITEBYTECODE=1 PYTHONIOENCODING=utf-8 \
    python "$(cygpath -w "$HARNESS")" \
        --mode local \
        --vectors-dir "$(cygpath -w "$SPEC/vectors/enforcement-v2")" \
    >"$RESULT_DIR/hc2.stdout" \
    2>"$RESULT_DIR/hc2.stderr"
HC2_RC=$?
HC2_ENDED_NS=$(date +%s%N)
HC2_DURATION_NS=$((HC2_ENDED_NS - HC2_STARTED_NS))
HC2_DURATION_MS_INT=$((HC2_DURATION_NS / 1000000))
HC2_DURATION_MS_FRAC=$((HC2_DURATION_NS % 1000000))

cat "$RESULT_DIR/hc2.stdout"

if [ -s "$RESULT_DIR/hc2.stderr" ]; then
    echo
    echo '=== HC2-55 STDERR ==='
    cat "$RESULT_DIR/hc2.stderr"
fi

echo "HC2_RC=$HC2_RC"

if [ "$HC2_RC" -ne 0 ]; then
    echo 'RESULT=PYTHON_HC2_EXECUTION_FAIL'
    exit "$HC2_RC"
fi

if ! grep -Eq 'RESULTS:[[:space:]]*55/55[[:space:]]+passed' \
    "$RESULT_DIR/hc2.stdout"
then
    echo 'RESULT=PYTHON_HC2_RESULT_MARKER_FAIL'
    exit 1
fi

echo 'HC2_RESULT=55/55_PASS'
echo 'ℹ tests 55'
echo 'ℹ suites 0'
echo 'ℹ pass 55'
echo 'ℹ fail 0'
echo 'ℹ cancelled 0'
echo 'ℹ skipped 0'
echo 'ℹ todo 0'
printf 'ℹ duration_ms %d.%06d\n' "$HC2_DURATION_MS_INT" "$HC2_DURATION_MS_FRAC"

echo
echo '=== 9. DERIVED OWNERSHIP CHECK ==='

NON_HOST_COUNT=$(
    find "$BUILD_ROOT" "$RESULT_DIR" \
        \( ! -uid "$HOST_UID" -o ! -gid "$HOST_GID" \) \
        -print \
        2>/dev/null |
    wc -l
)

echo "NON_HOST_OWNED_COUNT=$NON_HOST_COUNT"

if [ "$NON_HOST_COUNT" -ne 0 ]; then
    echo 'RESULT=PYTHON_DERIVED_OWNERSHIP_FAIL'
    exit 1
fi

echo
echo '=== 10. SOURCE IDENTITY RECHECK ==='

POST_HARNESS_SHA=$(sha256sum -- "$HARNESS" | awk '{print $1}')
POST_REQ_SHA=$(sha256sum -- "$REQ" | awk '{print $1}')

echo "POST_HARNESS_SHA256=$POST_HARNESS_SHA"
echo "POST_REQUIREMENTS_SHA256=$POST_REQ_SHA"

if [ "$POST_HARNESS_SHA" != "$EXPECTED_HARNESS_SHA" ]; then
    echo 'RESULT=PYTHON_HARNESS_IDENTITY_CHANGED'
    exit 1
fi

if [ "$POST_REQ_SHA" != "$EXPECTED_REQ_SHA" ]; then
    echo 'RESULT=PYTHON_REQUIREMENTS_IDENTITY_CHANGED'
    exit 1
fi

echo
echo '=== FINAL ==='
echo 'CORE_RESULT=38/38_PASS'
echo 'HC2_RESULT=55/55_PASS'
echo 'RESULT=PYTHON_DECLARED_CONFORMANCE_PASS'
