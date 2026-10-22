#!/usr/bin/env bash

set -u
set -o pipefail

GC="$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)"

SPEC="$GC/hacp-spec"
GO_SRC="$SPEC/hacp-go"

RESULT_DIR="$GC/hacp-sidecar/test/gitcheck-reproduction/results/hacp-go"
BUILD_DIR="$RESULT_DIR/build"
BIN="$BUILD_DIR/hacp-go.exe"
TMP_CORE="$RESULT_DIR/core38-vectors"

echo '============================================================'
echo 'HACP R1.1.1 GitCheck — HACP-Go Windows Reproduction'
echo '============================================================'
echo 'SURFACE=hacp-spec/hacp-go'
echo 'SIDECAR_SURFACE=NOT_USED'
echo 'SCOPE=CORE38_AND_HC2_55'
echo 'E13=NOT_RUN'

READY=1

for F in \
    "$GO_SRC/go.mod" \
    "$GO_SRC/main.go" \
    "$SPEC/harness/harness.py"
do
    if [ ! -f "$F" ]; then
        echo "FAIL=REQUIRED_FILE_ABSENT:$F"
        READY=0
    fi
done

for D in \
    "$SPEC/vectors" \
    "$SPEC/vectors/enforcement-v2"
do
    if [ ! -d "$D" ]; then
        echo "FAIL=REQUIRED_DIR_ABSENT:$D"
        READY=0
    fi
done

for TOOL in go python cygpath sha256sum find sort cp grep; do
    if ! command -v "$TOOL" >/dev/null 2>&1; then
        echo "FAIL=COMMAND_ABSENT:$TOOL"
        READY=0
    fi
done

if [ "$READY" -ne 1 ]; then
    echo 'RESULT=HACP_GO_VERIFY_FAIL_PRECONDITION'
    exit 1
fi

mkdir -p -- "$RESULT_DIR" "$BUILD_DIR"

if [ -e "$TMP_CORE" ]; then
    echo "FAIL=CORE_TEMP_ALREADY_EXISTS:$TMP_CORE"
    exit 1
fi

mkdir -p -- "$TMP_CORE"

echo '=== TOOLCHAIN ==='
go version
TOOLCHAIN_RC=$?

if [ "$TOOLCHAIN_RC" -ne 0 ]; then
    echo 'RESULT=HACP_GO_VERIFY_FAIL'
    exit 1
fi

echo '=== LOCAL GO BUILD ==='

(
    cd -- "$GO_SRC" &&
    go build -trimpath -o "$(cygpath -w "$BIN")" .
) >"$RESULT_DIR/build.log" 2>&1

BUILD_RC=$?

cat "$RESULT_DIR/build.log"

if [ "$BUILD_RC" -ne 0 ] || [ ! -f "$BIN" ]; then
    echo "BUILD_RC=$BUILD_RC"
    echo 'RESULT=HACP_GO_VERIFY_FAIL'
    exit 1
fi

sha256sum -- "$BIN" | tee "$RESULT_DIR/binary.sha256"

echo 'PACKAGE_TEST=NOT_RUN_E13_OUT_OF_SCOPE'
echo
echo '=== CORE-38 MATERIALIZATION ==='

find "$SPEC/vectors" \
    -maxdepth 1 \
    -type f \
    -name '*.json' \
    ! -name 'e5c6_binding_fixture.json' \
    -print0 |
sort -z |
while IFS= read -r -d '' F; do
    cp -- "$F" "$TMP_CORE/"
done

CORE_COUNT=$(
    find "$TMP_CORE" \
        -maxdepth 1 \
        -type f \
        -name '*.json' \
        -printf '.' |
    wc -c
)

echo "CORE38_JSON_COUNT=$CORE_COUNT"

if [ "$CORE_COUNT" -ne 38 ]; then
    echo 'RESULT=HACP_GO_VERIFY_FAIL'
    exit 1
fi

echo
echo '=== CORE-38 CONFORMANCE ==='

python "$(cygpath -w "$SPEC/harness/harness.py")" \
    --mode cli \
    --binary-path "$(cygpath -w "$BIN")" \
    --vectors-dir "$(cygpath -w "$TMP_CORE")" \
    >"$RESULT_DIR/core38.log" 2>&1

CORE_RC=$?

cat "$RESULT_DIR/core38.log"

if [ "$CORE_RC" -ne 0 ]; then
    echo 'RESULT=HACP_GO_VERIFY_FAIL'
    exit 1
fi

if ! grep -Fq 'RESULTS: 38/38 passed' "$RESULT_DIR/core38.log"; then
    echo 'FAIL=CORE38_EXPECTED_RESULT_NOT_FOUND'
    echo 'RESULT=HACP_GO_VERIFY_FAIL'
    exit 1
fi

echo
echo '=== HC2-55 CONFORMANCE ==='

HC2_COUNT=$(
    find "$SPEC/vectors/enforcement-v2" \
        -maxdepth 1 \
        -type f \
        -name '*.json' \
        -printf '.' |
    wc -c
)

echo "HC2_JSON_COUNT=$HC2_COUNT"

if [ "$HC2_COUNT" -ne 55 ]; then
    echo 'RESULT=HACP_GO_VERIFY_FAIL'
    exit 1
fi

python "$(cygpath -w "$SPEC/harness/harness.py")" \
    --mode cli \
    --binary-path "$(cygpath -w "$BIN")" \
    --vectors-dir "$(cygpath -w "$SPEC/vectors/enforcement-v2")" \
    >"$RESULT_DIR/hc2-55.log" 2>&1

HC2_RC=$?

cat "$RESULT_DIR/hc2-55.log"

if [ "$HC2_RC" -ne 0 ]; then
    echo 'RESULT=HACP_GO_VERIFY_FAIL'
    exit 1
fi

if ! grep -Fq 'RESULTS: 55/55 passed' "$RESULT_DIR/hc2-55.log"; then
    echo 'FAIL=HC2_55_EXPECTED_RESULT_NOT_FOUND'
    echo 'RESULT=HACP_GO_VERIFY_FAIL'
    exit 1
fi

rm -rf -- "$TMP_CORE"

echo
echo '============================================================'
echo 'HACP-GO VERIFICATION SUMMARY'
echo '============================================================'
echo 'BUILD=PASS'
echo 'PACKAGE_TEST=NOT_RUN_E13_OUT_OF_SCOPE'
echo 'CORE38=38/38 PASS'
echo 'HC2_55=55/55 PASS'
echo 'SIDECAR_NOT_USED'
echo 'RESULT=HACP_GO_VERIFY_PASS'
