#!/bin/sh

set -u

PATH='/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin'
export PATH

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
GC="$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)"

SOURCE="$GC/hacp-spec/hacp-ts"
BUILD_ROOT="$GC/build"
WORK="$BUILD_ROOT/hacp-ts-work"

NODE_IMAGE='sha256:4196d66a565c6f195728d9952f161f4adfe2ad753052a08b7ec7f1c5a6bda42b'

HOST_UID=$(id -u)
HOST_GID=$(id -g)

printf '%s\n' '============================================================'
printf '%s\n' 'HACP R1.1.1 — Fresh TypeScript Build'
printf '%s\n' '============================================================'
printf '\n'

printf 'ROOT=%s\n' "$GC"
printf 'SOURCE=%s\n' "$SOURCE"
printf 'BUILD_ROOT=%s\n' "$BUILD_ROOT"
printf 'WORK=%s\n' "$WORK"
printf 'NODE_IMAGE=%s\n' "$NODE_IMAGE"
printf 'HOST_UID=%s\n' "$HOST_UID"
printf 'HOST_GID=%s\n' "$HOST_GID"
printf '\n'

if [ ! -d "$SOURCE" ]; then
    printf '%s\n' 'RESULT=TS_BUILD_SOURCE_ABSENT'
    exit 1
fi

if ! docker image inspect "$NODE_IMAGE" >/dev/null 2>&1; then
    printf '%s\n' 'RESULT=TS_BUILD_NODE_IMAGE_ABSENT'
    exit 1
fi

SOURCE_PACKAGE_SHA=$(sha256sum -- "$SOURCE/package.json" | awk '{print $1}')
SOURCE_LOCK_SHA=$(sha256sum -- "$SOURCE/package-lock.json" | awk '{print $1}')
SOURCE_TSCONFIG_SHA=$(sha256sum -- "$SOURCE/tsconfig.json" | awk '{print $1}')

printf 'SOURCE_PACKAGE_SHA256=%s\n' "$SOURCE_PACKAGE_SHA"
printf 'SOURCE_LOCK_SHA256=%s\n' "$SOURCE_LOCK_SHA"
printf 'SOURCE_TSCONFIG_SHA256=%s\n' "$SOURCE_TSCONFIG_SHA"

mkdir -p -- "$BUILD_ROOT"

printf '\n'
printf '%s\n' '=== FAIL-CLOSED DERIVED WORK CLEANUP ==='

docker run --rm \
    --network none \
    --mount "type=bind,src=$BUILD_ROOT,dst=/build" \
    "$NODE_IMAGE" \
    sh -c '
        rm -rf -- /build/hacp-ts-work
    '

CLEAN_RC=$?

printf 'CLEANUP_DOCKER_RC=%s\n' "$CLEAN_RC"

if [ "$CLEAN_RC" -ne 0 ]; then
    printf '%s\n' 'RESULT=TS_BUILD_WORK_CLEANUP_FAIL'
    exit 1
fi

if [ -e "$WORK" ]; then
    printf '%s\n' 'RESULT=TS_BUILD_WORK_CLEANUP_INCOMPLETE'
    exit 1
fi

mkdir -p -- "$WORK"
cp -a "$SOURCE/." "$WORK/"

printf '\n'
printf '%s\n' '=== DERIVED WORK TREE CREATED ==='

WORK_PACKAGE_SHA=$(sha256sum -- "$WORK/package.json" | awk '{print $1}')
WORK_LOCK_SHA=$(sha256sum -- "$WORK/package-lock.json" | awk '{print $1}')
WORK_TSCONFIG_SHA=$(sha256sum -- "$WORK/tsconfig.json" | awk '{print $1}')

printf 'WORK_PACKAGE_SHA256=%s\n' "$WORK_PACKAGE_SHA"
printf 'WORK_LOCK_SHA256=%s\n' "$WORK_LOCK_SHA"
printf 'WORK_TSCONFIG_SHA256=%s\n' "$WORK_TSCONFIG_SHA"

if [ "$SOURCE_PACKAGE_SHA" != "$WORK_PACKAGE_SHA" ] || \
   [ "$SOURCE_LOCK_SHA" != "$WORK_LOCK_SHA" ] || \
   [ "$SOURCE_TSCONFIG_SHA" != "$WORK_TSCONFIG_SHA" ]; then
    printf '%s\n' 'RESULT=TS_BUILD_WORK_COPY_IDENTITY_FAIL'
    exit 1
fi

docker run --rm \
    --user "$HOST_UID:$HOST_GID" \
    --env HOME=/tmp/hacp-home \
    --env npm_config_cache=/tmp/npm-cache \
    --mount "type=bind,src=$WORK,dst=/work" \
    --workdir /work \
    "$NODE_IMAGE" \
    sh -c '
        set -u

        mkdir -p "$HOME" "$npm_config_cache"

        echo "=== TOOLCHAIN ==="
        node --version
        npm --version

        echo
        echo "=== CONTAINER IDENTITY ==="
        id

        echo
        echo "=== NPM CI ==="
        npm ci
        NPM_CI_RC=$?

        echo "NPM_CI_RC=$NPM_CI_RC"

        if [ "$NPM_CI_RC" -ne 0 ]; then
            echo "RESULT=TS_NPM_CI_FAIL"
            exit "$NPM_CI_RC"
        fi

        echo
        echo "=== TYPESCRIPT BUILD ==="
        npm run build
        TS_BUILD_RC=$?

        echo "TS_BUILD_RC=$TS_BUILD_RC"

        if [ "$TS_BUILD_RC" -ne 0 ]; then
            echo "RESULT=TS_BUILD_COMMAND_FAIL"
            exit "$TS_BUILD_RC"
        fi
    '

RC=$?

printf '\n'
printf 'DOCKER_RC=%s\n' "$RC"

if [ "$RC" -ne 0 ]; then
    printf '%s\n' 'RESULT=TS_FRESH_BUILD_FAIL'
    exit "$RC"
fi

printf '\n'
printf '%s\n' '=== BUILD OUTPUT ==='

if [ ! -d "$WORK/dist" ]; then
    printf '%s\n' 'RESULT=TS_FRESH_BUILD_DIST_ABSENT'
    exit 1
fi

DIST_FILES=$(
    find "$WORK/dist" \
        -type f \
        -printf '.' \
        2>/dev/null |
    wc -c
)

printf 'DIST_FILE_COUNT=%s\n' "$DIST_FILES"

find "$WORK/dist" \
    -type f \
    -printf '%P|%s bytes\n' \
    2>/dev/null |
sort

printf '\n'
printf '%s\n' '=== DERIVED WORK OWNERSHIP CHECK ==='

NON_HOST_COUNT=$(
    find "$WORK" \
        \( ! -uid "$HOST_UID" -o ! -gid "$HOST_GID" \) \
        -print \
        2>/dev/null |
    wc -l
)

printf 'NON_HOST_OWNED_COUNT=%s\n' "$NON_HOST_COUNT"

if [ "$NON_HOST_COUNT" -ne 0 ]; then
    find "$WORK" \
        \( ! -uid "$HOST_UID" -o ! -gid "$HOST_GID" \) \
        -printf '%u|%g|%m|%p\n' \
        2>/dev/null |
    sort |
    head -n 100

    printf '%s\n' 'RESULT=TS_FRESH_BUILD_OWNERSHIP_FAIL'
    exit 1
fi

printf '\n'
printf '%s\n' '=== SOURCE IDENTITY RECHECK ==='

POST_PACKAGE_SHA=$(sha256sum -- "$SOURCE/package.json" | awk '{print $1}')
POST_LOCK_SHA=$(sha256sum -- "$SOURCE/package-lock.json" | awk '{print $1}')
POST_TSCONFIG_SHA=$(sha256sum -- "$SOURCE/tsconfig.json" | awk '{print $1}')

printf 'POST_SOURCE_PACKAGE_SHA256=%s\n' "$POST_PACKAGE_SHA"
printf 'POST_SOURCE_LOCK_SHA256=%s\n' "$POST_LOCK_SHA"
printf 'POST_SOURCE_TSCONFIG_SHA256=%s\n' "$POST_TSCONFIG_SHA"

if [ "$SOURCE_PACKAGE_SHA" != "$POST_PACKAGE_SHA" ] || \
   [ "$SOURCE_LOCK_SHA" != "$POST_LOCK_SHA" ] || \
   [ "$SOURCE_TSCONFIG_SHA" != "$POST_TSCONFIG_SHA" ]; then
    printf '%s\n' 'RESULT=TS_SOURCE_IDENTITY_CHANGED'
    exit 1
fi

printf '%s\n' 'RESULT=TS_FRESH_BUILD_PASS'
