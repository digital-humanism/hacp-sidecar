#!/bin/sh

set -u

# Windows Git Bash: preserve inherited PATH
export PATH

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
GC="$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)"

WORK="$GC/hacp-spec/hacp-ts"
TSCLI="$(cygpath -w "$WORK/dist/src/cli.js")"

NODE_IMAGE='sha256:4196d66a565c6f195728d9952f161f4adfe2ad753052a08b7ec7f1c5a6bda42b'

if [ "${1-}" != "evaluate" ]; then
    printf '%s\n' '{"error":"USAGE","message":"expected: evaluate --vector <path>"}' >&2
    exit 2
fi

shift

VECTOR=''
PUBKEY=''

while [ "$#" -gt 0 ]; do
    case "$1" in
        --vector)
            shift
            VECTOR="${1-}"
            ;;
        --public-key)
            shift
            PUBKEY="${1-}"
            ;;
        *)
            printf '%s\n' '{"error":"USAGE","message":"unexpected argument"}' >&2
            exit 2
            ;;
    esac
    shift
done

if [ -z "$VECTOR" ] || [ ! -f "$VECTOR" ]; then
    printf '%s\n' '{"error":"USAGE","message":"valid --vector path required"}' >&2
    exit 2
fi

if [ ! -f "$TSCLI" ]; then
    printf '%s\n' '{"error":"BRIDGE","message":"fresh TypeScript CLI absent"}' >&2
    exit 2
fi

if ! command -v node >/dev/null 2>&1; then
    printf '%s\n' '{"error":"BRIDGE","message":"Node unavailable"}' >&2
    exit 2
fi

if [ -n "$PUBKEY" ]; then
    if [ ! -f "$PUBKEY" ]; then
        printf '%s\n' '{"error":"USAGE","message":"public key path absent"}' >&2
        exit 2
    fi

    node "$TSCLI" \
        evaluate \
        --vector "$(cygpath -w "$VECTOR")" \
        --public-key "$(cygpath -w "$PUBKEY")"
else
    node "$TSCLI" \
        evaluate \
        --vector "$(cygpath -w "$VECTOR")"
fi
