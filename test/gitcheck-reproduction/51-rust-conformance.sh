#!/usr/bin/env bash

set -u
set -o pipefail

GC="$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)"

SPEC="$GC/hacp-spec"
RUNNER="$SPEC/hacp-rs/target/release/hacp-rs-runner.exe"

CORE_DIR="$SPEC/vectors"
HC2_DIR="$CORE_DIR/enforcement-v2"
DUP_VECTOR="$CORE_DIR/core_inv5_006_negative.json"

RESULT_DIR="$GC/hacp-sidecar/test/gitcheck-reproduction/results/rust-conformance"

EXPECTED_RUNNER_SHA='391372a02bb914bf41ce9ae49a521b01339e3a0c179f83a077ee5f5dc6ab63d6'
EXPECTED_DUP_VECTOR_SHA='c686eee312e6f380ebd553eef82dda532b5833c28a3d623d0f351f729ba8cbe6'

echo '============================================================'
echo 'R1.1.1 — RUST CONFORMANCE'
echo '============================================================'
echo
echo 'PROVENANCE=37_CORE_CORPUS_PLUS_CORRECTED_RAW_WIRE_CORE_INV5_006'
echo 'CORE_TARGET=38/38'
echo 'HC2_TARGET=55/55'
echo 'NO_SYNTHETIC_COMBINED_SCORE'
echo

READY=1

for P in \
    "$SPEC" \
    "$CORE_DIR" \
    "$HC2_DIR"
do
    if [ ! -d "$P" ]; then
        echo "STOP=REQUIRED_DIR_ABSENT:$P"
        READY=0
    fi
done

for F in \
    "$RUNNER" \
    "$DUP_VECTOR"
do
    if [ ! -f "$F" ]; then
        echo "STOP=REQUIRED_FILE_ABSENT:$F"
        READY=0
    fi
done

if [ "$READY" -eq 1 ]; then
    ACTUAL_RUNNER_SHA=$(sha256sum -- "$RUNNER" | awk '{print $1}')
    ACTUAL_DUP_VECTOR_SHA=$(sha256sum -- "$DUP_VECTOR" | awk '{print $1}')

    echo "EXPECTED_RUNNER_SHA256=$EXPECTED_RUNNER_SHA"
    echo "ACTUAL_RUNNER_SHA256=$ACTUAL_RUNNER_SHA"
    echo "EXPECTED_DUP_VECTOR_SHA256=$EXPECTED_DUP_VECTOR_SHA"
    echo "ACTUAL_DUP_VECTOR_SHA256=$ACTUAL_DUP_VECTOR_SHA"

    if [ "$ACTUAL_RUNNER_SHA" != "$EXPECTED_RUNNER_SHA" ]; then
        echo 'STOP=RUNNER_IDENTITY_MISMATCH'
        READY=0
    fi

    if [ "$ACTUAL_DUP_VECTOR_SHA" != "$EXPECTED_DUP_VECTOR_SHA" ]; then
        echo 'STOP=DUP_VECTOR_IDENTITY_MISMATCH'
        READY=0
    fi
fi

if [ "$READY" -eq 1 ]; then
    rm -rf -- "$RESULT_DIR"
    CLEAN_RC=$?

    echo "RESULT_DIR_CLEAN_RC=$CLEAN_RC"

    if [ "$CLEAN_RC" -ne 0 ]; then
        echo 'STOP=RESULT_DIR_CLEAN_FAIL'
        READY=0
    else
        mkdir -p -- "$RESULT_DIR"
    fi
fi

echo "PRECONDITIONS_READY=$READY"

if [ "$READY" -ne 1 ]; then
    echo 'RESULT=RUST_CONFORMANCE_FAIL_PRECONDITION'
    exit 1
fi

echo
echo '=== 1. CORE-37 + HC2-55 CORPUS EXECUTION ==='

ROOT="$(cygpath -w "$SPEC")" RUNNER="$(cygpath -w "$RUNNER")" python - <<'PY' \
    | tee "$RESULT_DIR/corpus.log"
import json
import os
import subprocess
from pathlib import Path

ROOT = Path(os.environ["ROOT"])
RUNNER = Path(os.environ["RUNNER"])

CORE_DIR = ROOT / "vectors"
HC2_DIR = CORE_DIR / "enforcement-v2"

CORE_EXCLUDE = {
    "e5c6_binding_fixture.json",
    "core_inv5_006_negative.json",
}


class DuplicateKeyError(ValueError):
    pass


def reject_duplicates(pairs):
    out = {}
    for k, v in pairs:
        if k in out:
            raise DuplicateKeyError(k)
        out[k] = v
    return out


def strict_load(path):
    return json.loads(
        path.read_text(encoding="utf-8"),
        object_pairs_hook=reject_duplicates,
    )


def normal_request(vector):
    return json.dumps(
        {
            "protocol_version": "1",
            "operation": "evaluate",
            "vector_id": vector["test_id"],
            "input": vector["inputs"],
            "policy_context": vector.get("policy_context", {}),
        },
        separators=(",", ":"),
    )


def evaluate(request):
    p = subprocess.run(
        [str(RUNNER)],
        input=request + "\n",
        text=True,
        capture_output=True,
        timeout=10,
    )

    if p.returncode != 0:
        return None, f"EXIT={p.returncode} STDERR={p.stderr.strip()}"

    lines = [x for x in p.stdout.splitlines() if x.strip()]

    if len(lines) != 1:
        return None, f"STDOUT_LINES={len(lines)} STDOUT={p.stdout!r}"

    try:
        return json.loads(lines[0]), None
    except Exception as e:
        return None, f"INVALID_RESPONSE_JSON: {e}"


def family_a(vector, response):
    if response is None:
        return False

    return response.get("decision") == vector["expected"]["outcome"]


def family_b(vector, response):
    if response is None:
        return False, "NO_RESPONSE"

    expected = vector["expected"]
    actual_decision = response.get("decision")

    if actual_decision != expected["outcome"]:
        return (
            False,
            f"DECISION expected={expected['outcome']} actual={actual_decision}",
        )

    expected_reasons = expected.get("reason_codes")

    if expected_reasons is not None:
        actual_reasons = response.get("reason_codes")

        if actual_reasons is None:
            return False, "MISSING"

        if actual_reasons != expected_reasons:
            return (
                False,
                f"WRONG expected={expected_reasons} actual={actual_reasons}",
            )

    return True, "MATCH"


def load_suite(directory, exclude=None):
    exclude = exclude or set()
    rows = []

    for path in sorted(directory.glob("*.json")):
        if path.name in exclude:
            continue

        vector = strict_load(path)
        rows.append((path, vector))

    return rows


def run_suite(label, suite):
    results = []

    for path, vector in suite:
        request = normal_request(vector)
        response, error = evaluate(request)

        fa = family_a(vector, response)
        fb, detail = family_b(vector, response)

        results.append(
            {
                "file": path.name,
                "test_id": vector["test_id"],
                "family_a": fa,
                "family_b": fb,
                "detail": detail,
                "response": response,
                "error": error,
            }
        )

    fa_pass = sum(1 for r in results if r["family_a"])
    fb_pass = sum(1 for r in results if r["family_b"])

    print()
    print("=" * 72)
    print(label)
    print(f"inventory:                 {len(results)}")
    print(f"Family A decision-only:    {fa_pass}/{len(results)} PASS")
    print(f"Family B strict:           {fb_pass}/{len(results)} PASS")

    for family, key in (
        ("Family A failures", "family_a"),
        ("Family B failures", "family_b"),
    ):
        failures = [r for r in results if not r[key]]

        if failures:
            print()
            print(family + ":")

            for r in failures:
                print(
                    f"  {r['test_id']} ({r['file']}): "
                    f"{r['error'] or r['detail'] or r['response']}"
                )

    return results


core = load_suite(CORE_DIR, exclude=CORE_EXCLUDE)
hc2 = load_suite(HC2_DIR)

print(f"Core normal-case inventory observed: {len(core)}")
print(f"HC2 inventory observed:              {len(hc2)}")

if len(core) != 37:
    print(f"FATAL: Core normal-case inventory expected 37, got {len(core)}")
    raise SystemExit(2)

if len(hc2) != 55:
    print(f"FATAL: HC2 inventory expected 55, got {len(hc2)}")
    raise SystemExit(2)

core_results = run_suite("RUST CORE NORMAL 37", core)
hc2_results = run_suite("RUST HC2-55", hc2)

core_fa = sum(r["family_a"] for r in core_results)
core_fb = sum(r["family_b"] for r in core_results)
hc2_fa = sum(r["family_a"] for r in hc2_results)
hc2_fb = sum(r["family_b"] for r in hc2_results)

errors = [
    r
    for r in core_results + hc2_results
    if r["error"] is not None
]

print()
print("=" * 72)
print("CORPUS RESULT")
print(f"Core normal  Family A: {core_fa}/37")
print(f"Core normal  Family B: {core_fb}/37")
print(f"HC2-55       Family A: {hc2_fa}/55")
print(f"HC2-55       Family B: {hc2_fb}/55")
print(f"Evaluator/infrastructure errors: {len(errors)}")

if errors:
    raise SystemExit(3)

if (
    core_fa != 37
    or core_fb != 37
    or hc2_fa != 55
    or hc2_fb != 55
):
    raise SystemExit(4)

raise SystemExit(0)
PY

CORPUS_RC=${PIPESTATUS[0]}

echo
echo "CORPUS_RC=$CORPUS_RC"

echo
echo '=== 2. CORE-INV5-006 CORRECTED SINGLE-LINE RAW-WIRE ==='

REQ="$RESULT_DIR/core_inv5_006_request.jsonl"
OUT="$RESULT_DIR/core_inv5_006.stdout"
ERR="$RESULT_DIR/core_inv5_006.stderr"

ROOT="$(cygpath -w "$SPEC")" REQ="$(cygpath -w "$REQ")" python - <<'PY'
import json
import os
from pathlib import Path

root = Path(os.environ["ROOT"])
out = Path(os.environ["REQ"])
src = root / "vectors/core_inv5_006_negative.json"

raw = src.read_text(encoding="utf-8")


def extract_top_level_raw_value(raw, key):
    decoder = json.JSONDecoder()
    i = 0
    n = len(raw)

    def ws(p):
        while p < n and raw[p].isspace():
            p += 1
        return p

    i = ws(i)
    assert raw[i] == "{"
    i += 1

    while True:
        i = ws(i)

        if raw[i] == "}":
            break

        k, end = decoder.raw_decode(raw, i)
        i = ws(end)

        assert raw[i] == ":"
        i += 1
        i = ws(i)

        start = i

        if raw[i] in "{[":
            stack = [raw[i]]
            i += 1
            in_string = False
            escaped = False

            while stack:
                c = raw[i]

                if in_string:
                    if escaped:
                        escaped = False
                    elif c == "\\":
                        escaped = True
                    elif c == '"':
                        in_string = False
                else:
                    if c == '"':
                        in_string = True
                    elif c in "{[":
                        stack.append(c)
                    elif c == "}":
                        assert stack[-1] == "{"
                        stack.pop()
                    elif c == "]":
                        assert stack[-1] == "["
                        stack.pop()

                i += 1

            end = i
        else:
            _, end = decoder.raw_decode(raw, i)
            i = end

        if k == key:
            return raw[start:end]

        i = ws(i)

        if raw[i] == ",":
            i += 1

    raise KeyError(key)


def minify_raw_json(raw):
    out = []
    in_string = False
    escaped = False

    for c in raw:
        if in_string:
            out.append(c)

            if escaped:
                escaped = False
            elif c == "\\":
                escaped = True
            elif c == '"':
                in_string = False
        else:
            if c == '"':
                in_string = True
                out.append(c)
            elif c in " \t\r\n":
                continue
            else:
                out.append(c)

    return "".join(out)


v = json.loads(raw)

raw_inputs = minify_raw_json(
    extract_top_level_raw_value(raw, "inputs")
)

raw_policy = minify_raw_json(
    extract_top_level_raw_value(raw, "policy_context")
)

request = (
    '{"protocol_version":"1",'
    '"operation":"evaluate",'
    f'"vector_id":{json.dumps(v["test_id"])},'
    f'"input":{raw_inputs},'
    f'"policy_context":{raw_policy}'
    '}'
)

assert "\n" not in request
assert request.count('"verb":"read"') >= 1
assert request.count('"verb":"delete"') >= 1

out.write_text(request + "\n", encoding="utf-8")

print("request_file:", out)
print("request_bytes:", len(request.encode("utf-8")))
print("physical_lines:", request.count("\n") + 1)
print("verb_member_count:", request.count('"verb":'))
print('contains "verb":"read":', '"verb":"read"' in request)
print('contains "verb":"delete":', '"verb":"delete"' in request)
PY

REQUEST_BUILD_RC=$?

echo "REQUEST_BUILD_RC=$REQUEST_BUILD_RC"

echo
echo '=== REQUEST IDENTITY ==='
wc -l -c "$REQ"
REQUEST_SHA=$(sha256sum -- "$REQ" | awk '{print $1}')
echo "REQUEST_SHA256=$REQUEST_SHA"

echo
echo '=== DUPLICATE VERB EVIDENCE ==='
grep -o '"verb":"[^"]*"' "$REQ"

echo
echo '=== RUN FRESH RUST RUNNER ==='

"$RUNNER" <"$REQ" >"$OUT" 2>"$ERR"
RAW_RC=$?

echo "RAW_RUNNER_RC=$RAW_RC"

echo
echo '=== STDOUT ==='
cat "$OUT"

echo
echo '=== STDERR ==='
cat "$ERR"

echo
echo '=== OUTPUT SIZES ==='
wc -c "$OUT" "$ERR"

echo
echo '=== 3. RAW-WIRE RESULT VALIDATION ==='

RAW_VALIDATION_RC=0

REQ_LINES=$(wc -l < "$REQ")
ERR_BYTES=$(wc -c < "$ERR")

if [ "$REQUEST_SHA" != '8d95b2d935f24b74e26ad9e81200d87b2cbbc092713ba78ac1ead3c011888fe4' ]; then
    echo 'RAW_REQUEST_SHA_MATCH=NO'
    RAW_VALIDATION_RC=1
else
    echo 'RAW_REQUEST_SHA_MATCH=YES'
fi

if [ "$REQ_LINES" -ne 1 ]; then
    echo "RAW_REQUEST_SINGLE_LINE=NO lines=$REQ_LINES"
    RAW_VALIDATION_RC=1
else
    echo 'RAW_REQUEST_SINGLE_LINE=YES'
fi

if [ "$ERR_BYTES" -ne 0 ]; then
    echo "RAW_STDERR_EMPTY=NO bytes=$ERR_BYTES"
    RAW_VALIDATION_RC=1
else
    echo 'RAW_STDERR_EMPTY=YES'
fi

ROOT_OUT="$(cygpath -w "$OUT")" python - <<'PY'
import json
import os
from pathlib import Path

path = Path(os.environ["ROOT_OUT"])

lines = [
    line
    for line in path.read_text(encoding="utf-8").splitlines()
    if line.strip()
]

if len(lines) != 1:
    print(f"RAW_RESPONSE_VALID=NO lines={len(lines)}")
    raise SystemExit(1)

try:
    response = json.loads(lines[0])
except Exception as exc:
    print(f"RAW_RESPONSE_VALID=NO json_error={exc}")
    raise SystemExit(1)

decision_ok = response.get("decision") == "DENY"
reasons_ok = response.get("reason_codes") == ["INVALID_ACTION"]

print(f"RAW_DECISION_MATCH={'YES' if decision_ok else 'NO'}")
print(f"RAW_REASON_CODES_MATCH={'YES' if reasons_ok else 'NO'}")

if not (decision_ok and reasons_ok):
    raise SystemExit(1)
PY

RAW_RESPONSE_RC=$?

if [ "$RAW_RESPONSE_RC" -ne 0 ]; then
    RAW_VALIDATION_RC=1
fi

echo "RAW_VALIDATION_RC=$RAW_VALIDATION_RC"

echo
echo '=== 4. FINAL RUST CONFORMANCE ==='

echo 'Core provenance:'
echo '  37 normal corpus cases'
echo '  + corrected raw-wire CORE-INV5-006'
echo '  = Core-38'
echo

echo 'HC2 provenance:'
echo '  55-vector corpus execution'
echo '  = HC2-55'
echo

echo 'Expected independent evidence families:'
echo '  Core Family A = 38/38'
echo '  Core Family B = 38/38'
echo '  HC2  Family A = 55/55'
echo '  HC2  Family B = 55/55'
echo

echo '=== RUNNER IDENTITY ==='
sha256sum -- "$RUNNER"

echo
echo '=== FINAL ==='

if [ "$CORPUS_RC" -eq 0 ] && \
   [ "$REQUEST_BUILD_RC" -eq 0 ] && \
   [ "$RAW_RC" -eq 0 ] && \
   [ "$RAW_VALIDATION_RC" -eq 0 ]; then
    echo 'CORE_RESULT=38/38_PASS'
    echo 'HC2_RESULT=55/55_PASS'
    echo 'RESULT=RUST_CONFORMANCE_PASS'
    exit 0
else
    echo 'RESULT=RUST_CONFORMANCE_FAIL'
    exit 1
fi
