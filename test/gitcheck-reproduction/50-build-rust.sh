#!/bin/bash

set -u
set -o pipefail

PATH='/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin'
export PATH

HOST_UID=$(id -u)
HOST_GID=$(id -g)

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
GC="$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)"

SPEC="$GC/hacp-spec"
RS="$SPEC/hacp-rs"
PROTO="$SPEC/proto"
VECTORS="$SPEC/vectors"

WORKROOT="$GC/build/hacp-rs-work"
WORKRS="$WORKROOT/hacp-rs"
WORKPROTO="$WORKROOT/proto"
WORKVECTORS="$WORKROOT/vectors"

TOOLROOT="$GC/build/rust-toolchain"
PROTOC="$TOOLROOT/protoc"

OUT="$GC/build/rust"
RESULTS="$GC/results/rust-build"

RUST_IMAGE='sha256:c49256cbe5ea0188bc658a689500d70c41eb51f009a7a7be209caf60a944f3ec'
EXPECTED_PROTOC_SHA='d2c65c3ea5eeb59427f684ed3e0a0cd458386122fc9f1b98de946a7242b53d31'
SOURCE_PROTOC="$GC/verification/e16/container-b/protoc/bin/protoc"

echo '============================================================'
echo 'R1.1.1 — RUST FRESH BUILD'
echo '============================================================'
echo

READY=1

for F in \
    "$RS/Cargo.toml" \
    "$RS/Cargo.lock" \
    "$RS/build.rs" \
    "$SOURCE_PROTOC"
do
    if [ ! -f "$F" ]; then
        echo "STOP=REQUIRED_FILE_ABSENT:$F"
        READY=0
    fi
done

if [ ! -d "$PROTO" ]; then
    echo 'STOP=PROTO_TREE_ABSENT'
    READY=0
fi

if [ ! -d "$VECTORS" ]; then
    echo 'STOP=VECTOR_TREE_ABSENT'
    READY=0
fi

if ! docker image inspect "$RUST_IMAGE" >/dev/null 2>&1; then
    echo 'STOP=RUST_IMAGE_ABSENT'
    READY=0
fi

ACTUAL_PROTOC_SHA=$(
    sha256sum -- "$SOURCE_PROTOC" 2>/dev/null |
    awk '{print $1}'
)

echo "PRESERVED_PROTOC_SHA256=$ACTUAL_PROTOC_SHA"

if [ "$ACTUAL_PROTOC_SHA" != "$EXPECTED_PROTOC_SHA" ]; then
    echo 'STOP=PRESERVED_PROTOC_SHA_MISMATCH'
    READY=0
fi

echo "PRECONDITIONS_READY=$READY"

if [ "$READY" -ne 1 ]; then
    echo 'RESULT=RUST_FRESH_BUILD_NOT_STARTED'
    exit 1
fi

echo
echo '=== SOURCE IDENTITIES BEFORE ==='

sha256sum -- \
    "$RS/Cargo.toml" \
    "$RS/Cargo.lock" \
    "$RS/build.rs" \
    "$RS/src/lib.rs" \
    "$RS/src/evaluate.rs" \
    "$RS/src/runner.rs"

echo
echo '=== PREPARE DERIVED WORKSPACE ==='

rm -rf -- "$WORKROOT" "$TOOLROOT" "$OUT" "$RESULTS"
CLEAN_RC=$?

echo "CLEAN_RC=$CLEAN_RC"

if [ "$CLEAN_RC" -ne 0 ]; then
    echo 'RESULT=RUST_FRESH_BUILD_FAIL_CLEANUP'
    exit 1
fi


mkdir -p -- \
    "$WORKROOT" \
    "$TOOLROOT" \
    "$OUT" \
    "$RESULTS"

cp -a -- "$RS" "$WORKRS"
cp -a -- "$PROTO" "$WORKPROTO"
cp -a -- "$VECTORS" "$WORKVECTORS"

rm -rf -- "$WORKRS/target"

cp -- "$SOURCE_PROTOC" "$PROTOC"
chmod 755 -- "$PROTOC"

echo "WORKROOT=$WORKROOT"
echo "WORKRS=$WORKRS"
echo "WORKPROTO=$WORKPROTO"
echo "WORKVECTORS=$WORKVECTORS"
echo "PROTOC=$PROTOC"

echo
echo '=== DERIVED PROTOC IDENTITY ==='

stat -c 'FILE=%n SIZE=%s MODE=%a' -- "$PROTOC"
sha256sum -- "$PROTOC"

DERIVED_PROTOC_SHA=$(
    sha256sum -- "$PROTOC" |
    awk '{print $1}'
)

if [ "$DERIVED_PROTOC_SHA" != "$EXPECTED_PROTOC_SHA" ]; then
    echo 'RESULT=RUST_FRESH_BUILD_FAIL_PROTOC_COPY'
    exit 1
fi

echo
echo '=== TOOLCHAIN ==='

docker run --rm \
    --mount "type=bind,src=$WORKROOT,dst=/work" \
    --mount "type=bind,src=$PROTOC,dst=/toolchain/protoc,readonly" \
    --workdir /work/hacp-rs \
    --env PROTOC=/toolchain/protoc \
    "$RUST_IMAGE" \
    sh -c '
        set -u
        echo "rustc=$(rustc --version)"
        echo "cargo=$(cargo --version)"
        echo "protoc=$($PROTOC --version)"
    '

TOOLCHAIN_RC=$?
echo "TOOLCHAIN_RC=$TOOLCHAIN_RC"

if [ "$TOOLCHAIN_RC" -ne 0 ]; then
    echo 'RESULT=RUST_FRESH_BUILD_FAIL_TOOLCHAIN'
    exit 1
fi

echo
echo '=== CARGO BUILD ==='

docker run --rm \
    --user "$HOST_UID:$HOST_GID" \
    --mount "type=bind,src=$WORKROOT,dst=/work" \
    --mount "type=bind,src=$PROTOC,dst=/toolchain/protoc,readonly" \
    --workdir /work/hacp-rs \
    --env PROTOC=/toolchain/protoc \
    "$RUST_IMAGE" \
    cargo build \
        --locked \
        --release \
        --bin hacp-rs-runner \
    > "$RESULTS/cargo-build.log" 2>&1

BUILD_RC=$?

cat "$RESULTS/cargo-build.log"

echo
echo "BUILD_RC=$BUILD_RC"

if [ "$BUILD_RC" -ne 0 ]; then
    echo 'RESULT=RUST_FRESH_BUILD_FAIL'
    exit 1
fi

BUILT="$WORKRS/target/release/hacp-rs-runner"

if [ ! -f "$BUILT" ]; then
    echo 'RESULT=RUST_FRESH_BUILD_FAIL_BINARY_ABSENT'
    exit 1
fi

cp -- "$BUILT" "$OUT/hacp-rs-runner"
chmod 755 -- "$OUT/hacp-rs-runner"

echo
echo '=== FRESH RUNNER IDENTITY ==='

stat -c 'FILE=%n SIZE=%s MODE=%a' -- "$OUT/hacp-rs-runner"
sha256sum -- "$OUT/hacp-rs-runner"

echo
echo '=== SOURCE IDENTITIES AFTER ==='

sha256sum -- \
    "$RS/Cargo.toml" \
    "$RS/Cargo.lock" \
    "$RS/build.rs" \
    "$RS/src/lib.rs" \
    "$RS/src/evaluate.rs" \
    "$RS/src/runner.rs"

echo
echo '=== RESULT ==='
echo 'RESULT=RUST_FRESH_BUILD_PASS'
echo 'NO SOURCE FILES MODIFIED'
echo 'NO TEST PERFORMED'
echo 'E16_REMAINS_CLOSED'
echo 'E17_NOT_STARTED'
