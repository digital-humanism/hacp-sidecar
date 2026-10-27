#Requires -Version 7.0

<#
02-Run-Verification.ps1 (RUST) — executes RS surfaces strictly sequentially:
RS01 → RS02 → RS03 → RS04 → RS05 → RS07 (RS06 = NOT_APPLICABLE, no separate run).
Per-surface evidence + result.json (§38). NO admission/acceptance automation. No synthetic
combined scores. Failure policy (§39): ENVIRONMENT_RED stops dependents; BEHAVIORAL_RED is
recorded (never repaired); RS01 Core result is 36/38 direct + 2 adjudicated, never 38/38 direct.
#>

[CmdletBinding()]
param(
    [string]$SpecRepo    = '',
    [string]$SidecarRepo = '',
    [string]$EvidenceRoot= '',
    [string]$RunRoot     = '',
    [string]$RunDir      = '',
    [ValidateSet('RS01','RS02','RS03','RS04','RS05','RS07')][string]$Surface = '',
    [switch]$ContinueIndependentSurfaces
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Portable default path resolution.
# Explicit caller-supplied paths take precedence.
if (-not $SidecarRepo) {
    $SidecarRepo = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
}

$WorkspaceRoot = Split-Path -Parent $SidecarRepo

if (-not $SpecRepo) {
    $SpecRepo = Join-Path $WorkspaceRoot 'hacp-spec'
}

if (-not $RunRoot -or -not $EvidenceRoot) {
    $VerificationRoot = Join-Path ([System.IO.Path]::GetTempPath()) 'hacp-rust-verification'
}

if (-not $RunRoot) {
    $RunRoot = Join-Path $VerificationRoot 'run'
}

if (-not $EvidenceRoot) {
    $EvidenceRoot = Join-Path $VerificationRoot 'evidence'
}

Import-Module (Join-Path $PSScriptRoot 'WindowsVerification.Rust.Common.psm1') -Force

$run = Resolve-RvRunDir -EvidenceRoot $EvidenceRoot -RunDir $RunDir -RequireState 'PREPARED'

# DEFECT-3B: hard 01 -> 02 gate (must run BEFORE any RS surface execution).
$prepPath = Join-Path $run.RunDir '01-prepare\PREPARE_RESULT.json'
$prepReady = $false
if (-not (Test-Path -LiteralPath $prepPath)) {
    Write-RvLog -Level RED -Message "PREPARE_RESULT missing: $prepPath"
} else {
    try {
        $prep = Get-Content -LiteralPath $prepPath -Raw | ConvertFrom-Json
        if ($prep.ENVIRONMENT_READY -eq 'PASS') { $prepReady = $true } else { Write-RvLog -Level RED -Message "ENVIRONMENT_READY=$($prep.ENVIRONMENT_READY) != PASS" }
    } catch { Write-RvLog -Level RED -Message "PREPARE_RESULT malformed: $($_.Exception.Message)" }
}
if (-not $prepReady) {
    Write-RvLog -Level RED -Message "01->02 gate FAILED. NO RS SURFACE EXECUTION. Exiting nonzero."
    exit 2
}

# DEFECT-8: run-owned work root (fixtures, cargo-target, PID registry are run-specific).
$workRoot = Get-RvWorkRoot -RunRoot $RunRoot -RunDir $run.RunDir
$CARGO_TARGET = Join-Path $workRoot 'cargo-target'
$FIXTURES = Join-Path $workRoot 'fixtures'
$OUT_ROOT = New-RvDirectory -Path (Join-Path $run.RunDir '02-verification')
Set-RvLogFile -Path (Join-Path $run.RunDir '01-prepare\run-console.log')
Write-RvLog -Message "02-Run (RUST) start. RunDir=$($run.RunDir) WorkRoot=$workRoot"
$cargoEnv = @{ CARGO_TARGET_DIR = $CARGO_TARGET }

$RUNNER = Join-Path (Join-Path $CARGO_TARGET 'release') 'hacp-rs-runner.exe'
$PROXY  = Join-Path (Join-Path $CARGO_TARGET 'debug') 'hacp-rs-proxy.exe'
$P_HEAD = (Get-RvGitIdentity -RepoRoot $SpecRepo).Head
$P_TREE = (Get-RvGitIdentity -RepoRoot $SpecRepo).Tree

$ORDER = @('RS01','RS02','RS03','RS04','RS05','RS07')
if ($Surface) { $QUEUE = @($Surface) } else { $QUEUE = $ORDER }
$MATRIX = @{ RS01='A4'; RS02='A8'; RS03='B18,B6'; RS04='B16,B17'; RS05='B19'; RS07='B9' }

function Write-SurfaceResult {
    param([string]$Surface,[string[]]$Rows,[hashtable]$ExitCodes,[string]$Classification,[string]$Expected,[string]$Observed,[string[]]$Evidence,[bool]$EnvRed,[bool]$BehRed,[hashtable]$CargoSummary=$null)
    $r = New-RvResult -Surface $Surface -MatrixRows $Rows -ProcessExitCodes $ExitCodes -Classification $Classification `
        -ExpectedResult $Expected -ObservedResult $Observed -EvidenceFiles $Evidence -EnvironmentRed $EnvRed -BehavioralRed $BehRed
    if ($CargoSummary) { $r | Add-Member -NotePropertyName cargo_summary -NotePropertyValue $CargoSummary -Force }
    Write-RvJson -Object $r -Path (Join-Path $OUT_ROOT "$Surface\result.json")
    $ident = @{ P_HEAD=$P_HEAD; P_TREE=$P_TREE; CARGO_TARGET_DIR=$CARGO_TARGET }
    Write-RvJson -Object $ident -Path (Join-Path $OUT_ROOT "$Surface\identity.json")
    return $r
}

function Invoke-Surface {
    param([string]$Id)
    $evidence = New-RvDirectory -Path (Join-Path $OUT_ROOT $Id)
    Write-RvLog -Level STEP -Message "=== $Id (rows $($MATRIX[$Id])) ==="
    $cls = $RV_CLASS.PASS; $envRed = $false; $behRed = $false; $exitCodes = @{}; $observed = ''

    switch ($Id) {
        'RS01' {
            # Fresh execution of current Rust Core Family A.
            # 37 ordinary vectors plus CORE-INV5-006 raw-wire witness.
            # Test implementation is reused from t_go Surface C.
            # Historical source identity is recorded, not used as a verdict gate.

            $corePython = @"
import hashlib
import json
import subprocess
import sys
from pathlib import Path

root = Path(sys.argv[1])
runner = Path(sys.argv[2])
evidence = Path(sys.argv[3])

vectors = root / "vectors"

SPECIAL_RAW_FILE = "core_inv5_006_negative.json"
AUXILIARY_EXCLUDE = {"e5c6_binding_fixture.json"}

EXPECTED_ADJUDICATED = {
    "CORE-INV1-002",
    "CORE-RUNTIME-004",
}


class DuplicateKeyError(ValueError):
    pass


def reject_duplicates(pairs):
    out = {}
    for key, value in pairs:
        if key in out:
            raise DuplicateKeyError(key)
        out[key] = value
    return out


def strict_load(path):
    return json.loads(
        path.read_text(encoding="utf-8"),
        object_pairs_hook=reject_duplicates,
    )


def title_of(vector):
    return (
        vector.get("description")
        or vector.get("name")
        or vector.get("title")
        or ""
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
    process = subprocess.run(
        [str(runner)],
        input=request + "\n",
        text=True,
        capture_output=True,
        timeout=10,
    )

    if process.returncode != 0:
        raise RuntimeError(
            f"runner exit={process.returncode}; stderr={process.stderr.strip()}"
        )

    lines = [line for line in process.stdout.splitlines() if line.strip()]

    if len(lines) != 1:
        raise RuntimeError(
            f"runner stdout line count={len(lines)}; stdout={process.stdout!r}"
        )

    try:
        return json.loads(lines[0])
    except Exception as exc:
        raise RuntimeError(f"invalid runner JSON: {exc}") from exc


def extract_top_level_raw_value(raw_text, key):
    decoder = json.JSONDecoder()
    i = 0
    n = len(raw_text)

    def ws(pos):
        while pos < n and raw_text[pos].isspace():
            pos += 1
        return pos

    i = ws(i)
    assert raw_text[i] == "{"
    i += 1

    while True:
        i = ws(i)

        if raw_text[i] == "}":
            break

        current_key, end = decoder.raw_decode(raw_text, i)
        i = ws(end)

        assert raw_text[i] == ":"
        i += 1
        i = ws(i)

        start = i

        if raw_text[i] in "{[":
            stack = [raw_text[i]]
            i += 1
            in_string = False
            escaped = False

            while stack:
                char = raw_text[i]

                if in_string:
                    if escaped:
                        escaped = False
                    elif char == "\\":
                        escaped = True
                    elif char == '"':
                        in_string = False
                else:
                    if char == '"':
                        in_string = True
                    elif char in "{[":
                        stack.append(char)
                    elif char == "}":
                        assert stack[-1] == "{"
                        stack.pop()
                    elif char == "]":
                        assert stack[-1] == "["
                        stack.pop()

                i += 1

            end = i
        else:
            _, end = decoder.raw_decode(raw_text, i)
            i = end

        if current_key == key:
            return raw_text[start:end]

        i = ws(i)

        if raw_text[i] == ",":
            i += 1

    raise KeyError(key)


def minify_raw_json(raw_text):
    out = []
    in_string = False
    escaped = False

    for char in raw_text:
        if in_string:
            out.append(char)

            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                in_string = False
        else:
            if char == '"':
                in_string = True
                out.append(char)
            elif char in " \t\r\n":
                continue
            else:
                out.append(char)

    return "".join(out)


def execute_raw_inv5_006(path):
    raw = path.read_text(encoding="utf-8")

    vector = json.loads(raw)

    raw_inputs = minify_raw_json(
        extract_top_level_raw_value(raw, "inputs")
    )

    raw_policy = minify_raw_json(
        extract_top_level_raw_value(raw, "policy_context")
    )

    request = (
        '{"protocol_version":"1",'
        '"operation":"evaluate",'
        f'"vector_id":{json.dumps(vector["test_id"])},'
        f'"input":{raw_inputs},'
        f'"policy_context":{raw_policy}'
        '}'
    )

    if "\n" in request or "\r" in request:
        raise RuntimeError("CORE-INV5-006 request is not single-line")

    if request.count('"verb":"read"') < 1:
        raise RuntimeError("CORE-INV5-006 missing raw verb=read member")

    if request.count('"verb":"delete"') < 1:
        raise RuntimeError("CORE-INV5-006 missing raw verb=delete member")

    request_bytes = (request + "\n").encode("utf-8")

    request_path = evidence / "core_inv5_006_request.jsonl"
    stdout_path = evidence / "core_inv5_006.stdout"
    stderr_path = evidence / "core_inv5_006.stderr"

    request_path.write_bytes(request_bytes)

    process = subprocess.run(
        [str(runner)],
        input=request + "\n",
        text=True,
        capture_output=True,
        timeout=10,
    )

    stdout_path.write_text(process.stdout, encoding="utf-8")
    stderr_path.write_text(process.stderr, encoding="utf-8")

    if process.returncode != 0:
        raise RuntimeError(
            f"CORE-INV5-006 runner exit={process.returncode}"
        )

    if process.stderr:
        raise RuntimeError(
            f"CORE-INV5-006 stderr not empty: {process.stderr!r}"
        )

    lines = [
        line
        for line in process.stdout.splitlines()
        if line.strip()
    ]

    if len(lines) != 1:
        raise RuntimeError(
            f"CORE-INV5-006 response line count={len(lines)}"
        )

    response = json.loads(lines[0])

    decision_ok = response.get("decision") == "DENY"
    reasons_ok = response.get("reason_codes") == ["INVALID_ACTION"]

    if not decision_ok or not reasons_ok:
        raise RuntimeError(
            "CORE-INV5-006 expected DENY / INVALID_ACTION; "
            f"actual={response!r}"
        )

    sha256 = hashlib.sha256(request_bytes).hexdigest()

    return vector, response, sha256


paths = [
    path
    for path in sorted(vectors.glob("*.json"))
    if path.name not in AUXILIARY_EXCLUDE
]

if len(paths) != 38:
    print(f"FATAL: Core inventory expected 38, got {len(paths)}")
    raise SystemExit(2)


print("=" * 60)
print("HACP 1.1.1 Rust Core Conformance - Family A")
print("Protocol version: 1")
print(f"Runner: {runner}")
print("Spec: HACP-Core")
print("Contract: decision-level")
print("=" * 60)
print()
print(f"Vector source: {vectors}")
print("Inventory: Core-38")
print()

results = []
raw_sha = None

for path in paths:
    try:
        if path.name == SPECIAL_RAW_FILE:
            vector, response, raw_sha = execute_raw_inv5_006(path)
            expected = vector["expected"]["outcome"]
            actual = response.get("decision")

            passed = (
                response.get("decision") == "DENY"
                and response.get("reason_codes") == ["INVALID_ACTION"]
            )

            raw_wire = True
        else:
            vector = strict_load(path)
            request = normal_request(vector)
            response = evaluate(request)

            expected = vector["expected"]["outcome"]
            actual = response.get("decision")
            passed = actual == expected
            raw_wire = False

    except Exception as exc:
        print(f"[ERROR] {path.name}: {exc}")
        raise SystemExit(3)

    test_id = vector["test_id"]
    description = title_of(vector)

    status = "PASS" if passed else "FAIL"

    if description:
        print(f"[{status}] {test_id}: {description}")
    else:
        print(f"[{status}] {test_id}")

    if raw_wire:
        print("       RAW-WIRE: DENY / INVALID_ACTION")
        print(f"       Request SHA-256: {raw_sha}")

    elif not passed:
        print(
            "       Details: "
            + repr(
                {
                    "outcome_mismatch": {
                        "expected": expected,
                        "actual": actual,
                    }
                }
            )
        )

        if test_id in EXPECTED_ADJUDICATED:
            print("       ADJUDICATED: CHECKPOINT-vs-DENY")

    results.append(
        {
            "test_id": test_id,
            "expected": expected,
            "actual": actual,
            "passed": passed,
            "raw_wire": raw_wire,
        }
    )


direct_pass = sum(1 for result in results if result["passed"])

failures = [
    result
    for result in results
    if not result["passed"]
]

failure_ids = {
    result["test_id"]
    for result in failures
}

adjudicated_shape_ok = all(
    result["expected"] == "CHECKPOINT"
    and result["actual"] == "DENY"
    for result in failures
)

raw_result = next(
    (
        result
        for result in results
        if result["test_id"] == "CORE-INV5-006"
    ),
    None,
)

raw_ok = (
    raw_result is not None
    and raw_result["passed"]
    and raw_result["raw_wire"]
)


print()
print("=" * 60)
print(f"DIRECT RESULTS: {direct_pass}/38 passed")
print(f"ADJUDICATED: {len(failures)}")

print("=" * 60)


if direct_pass != 36:
    print(f"FATAL: expected 36 direct PASS, got {direct_pass}")
    raise SystemExit(4)

if failure_ids != EXPECTED_ADJUDICATED:
    print(
        "FATAL: adjudicated ID set mismatch: "
        + ",".join(sorted(failure_ids))
    )
    raise SystemExit(4)

if not adjudicated_shape_ok:
    print("FATAL: adjudicated outcome shape mismatch")
    raise SystemExit(4)

if not raw_ok:
    print("FATAL: CORE-INV5-006 raw-wire witness mismatch")
    raise SystemExit(4)

print("FINAL ACCOUNTING: 36 direct + 2 adjudicated")
raise SystemExit(0)
"@

            $corePy = Join-Path $evidence 'rust-core-family-a.py'

            [System.IO.File]::WriteAllText(
                $corePy,
                $corePython,
                [System.Text.UTF8Encoding]::new($false)
            )

            $r = Invoke-RvNative `
                -FilePath 'python' `
                -ArgumentList @(
                    $corePy,
                    $SpecRepo,
                    $RUNNER,
                    $evidence
                ) `
                -WorkDir $SpecRepo `
                -StepId 'RS01-core-family-a' `
                -SurfaceId 'RS01' `
                -EvidenceDir $evidence

            $exitCodes['RS01_CORE_FAMILY_A_RC'] = $r.ProcessRc
            $output = Get-RvStdout -Run $r

            $required = @(
                'DIRECT RESULTS: 36/38 passed',
                'ADJUDICATED: 2',
                'FINAL ACCOUNTING: 36 direct + 2 adjudicated',
                '[FAIL] CORE-INV1-002:',
                '[FAIL] CORE-RUNTIME-004:',
                '[PASS] CORE-INV5-006:',
                'RAW-WIRE: DENY / INVALID_ACTION'
            )

            $markersPass = $true
            foreach ($marker in $required) {
                if (-not $output.Contains($marker)) {
                    $markersPass = $false
                }
            }

            if ($null -eq $r.ProcessRc) {
                $cls = $RV_CLASS.ENVIRONMENT_RED
                $envRed = $true
                $observed = 'Rust Core Family A execution unavailable'
            }
            elseif ($r.ProcessRc -eq 0 -and $markersPass) {
                $cls = $RV_CLASS.PASS
                $observed = 'Fresh Core-38: 35 ordinary direct + 1 raw-wire direct + 2 adjudicated'
                Write-RvLog -Message 'RS01 Rust Core result: 36 direct + 2 adjudicated'
            }
            else {
                $cls = $RV_CLASS.BEHAVIORAL_RED
                $behRed = $true
                $observed = "Rust Core Family A failed: RC=$($r.ProcessRc); markers=$markersPass"
            }

            Write-SurfaceResult `
                -Surface 'RS01' `
                -Rows @('A4') `
                -ExitCodes $exitCodes `
                -Classification $cls `
                -Expected 'Fresh execution: 35/37 ordinary direct + 2 adjudicated; CORE-INV5-006 raw-wire PASS; 36 direct + 2 adjudicated total' `
                -Observed $observed `
                -Evidence @(
                    $corePy,
                    $r.StdOutPath,
                    $r.StdErrPath
                ) `
                -EnvRed $envRed `
                -BehRed $behRed
        }
        'RS02' {
            $r = Invoke-RvNative -FilePath 'python' -ArgumentList @((Join-Path $SpecRepo 'harness\enforcement_v2_runner.py'),'--runner',$RUNNER,'--vectors-dir',(Join-Path $SpecRepo 'vectors\enforcement-v2'),'--implementation-name','hacp-rs','--output','console') -WorkDir (Join-Path $SpecRepo 'harness') -StepId 'RS02-hc2' -SurfaceId 'RS02' -EvidenceDir $evidence
            $exitCodes['RS02_HARNESS_RC'] = $r.ProcessRc
            $o = Get-RvStdout -Run $r
            if ($r.ProcessRc -eq 0 -and $o -match '55/55') { $cls = $RV_CLASS.PASS; $observed = 'RESULTS: 55/55' }
            elseif ($r.ProcessRc -eq $null) { $cls = $RV_CLASS.ENVIRONMENT_RED; $envRed = $true }
            else { $cls = $RV_CLASS.BEHAVIORAL_RED; $behRed = $true; $observed = 'HC2 != 55/55' }
            Write-SurfaceResult -Surface 'RS02' -Rows @('A8') -ExitCodes $exitCodes -Classification $cls -Expected 'RESULTS: 55/55 passed (separate verdict family)' -Observed $observed -Evidence @($r.StdOutPath) -EnvRed $envRed -BehRed $behRed
        }
        'RS03' {
            # RS03 package qualification requires the shared E13 R01-R24 corpus.
            # The Rust test reads its absolute path from HACP_E13_CORPUS.
            # Missing corpus/path is an environment failure, never a behavioral verdict.

            $e13Corpus = Join-Path $SpecRepo 'vectors\family1\E13_R01_R24_NORMALIZED_CORPUS.json'
            $e13Family2Fixture = Join-Path $SpecRepo 'vectors\family2\e5c6_binding_fixture.json'
            $e13Family2PublicKey = Join-Path $SpecRepo 'harness\keys\test-ed25519-001.pub'

            $rs03MissingInputs = @()

            if (-not (Test-Path -LiteralPath $e13Corpus -PathType Leaf)) {
                $rs03MissingInputs += "HACP_E13_CORPUS=$e13Corpus"
            }

            if (-not (Test-Path -LiteralPath $e13Family2Fixture -PathType Leaf)) {
                $rs03MissingInputs += "HACP_E13_FAMILY2_FIXTURE=$e13Family2Fixture"
            }

            if (-not (Test-Path -LiteralPath $e13Family2PublicKey -PathType Leaf)) {
                $rs03MissingInputs += "HACP_E13_FAMILY2_PUBLIC_KEY=$e13Family2PublicKey"
            }

            if ($rs03MissingInputs.Count -ne 0) {
                $cls = $RV_CLASS.ENVIRONMENT_RED
                $envRed = $true
                $observed = "RS03 required input missing: $($rs03MissingInputs -join '; ')"

                Write-SurfaceResult `
                    -Surface 'RS03' `
                    -Rows @('B18','B6') `
                    -ExitCodes $exitCodes `
                    -Classification $cls `
                    -Expected '48 passed, 1 ignored (ed25519_v1_red), 0 failed; shared E13 corpus available; claim bound to exact P_HEAD/P_TREE' `
                    -Observed $observed `
                    -Evidence @() `
                    -EnvRed $envRed `
                    -BehRed $behRed
            }
            else {
                $rs03Env = @{}

                foreach ($key in $cargoEnv.Keys) {
                    $rs03Env[$key] = $cargoEnv[$key]
                }

                $rs03Env['HACP_E13_CORPUS'] = $e13Corpus
                $rs03Env['HACP_E13_FAMILY2_FIXTURE'] = $e13Family2Fixture
                $rs03Env['HACP_E13_FAMILY2_PUBLIC_KEY'] = $e13Family2PublicKey

                $r = Invoke-RvNative `
                    -FilePath 'cargo' `
                    -ArgumentList @(
                        '+stable-x86_64-pc-windows-msvc',
                        'test',
                        '--locked'
                    ) `
                    -WorkDir (Join-Path $SpecRepo 'hacp-rs') `
                    -StepId 'RS03-cargo-test' `
                    -SurfaceId 'RS03' `
                    -EvidenceDir $evidence `
                    -ExtraEnv $rs03Env

                $exitCodes['RS03_CARGO_RC'] = $r.ProcessRc
                $o = Get-RvStdout -Run $r

                # Sum every Cargo `test result:` line because the package contains
                # multiple unit/integration test binaries.
                $sumPassed = 0
                $sumFailed = 0
                $sumIgnored = 0

                foreach ($line in ($o -split "`n")) {
                    $l = $line.TrimEnd("`r")

                    if ($l -match 'test result:') {
                        if ($l -match '(\d+)\s+passed') {
                            $sumPassed += [int]$Matches[1]
                        }

                        if ($l -match '(\d+)\s+failed') {
                            $sumFailed += [int]$Matches[1]
                        }

                        if ($l -match '(\d+)\s+ignored') {
                            $sumIgnored += [int]$Matches[1]
                        }
                    }
                }

                $cargoSummary = @{
                    passed  = $sumPassed
                    failed  = $sumFailed
                    ignored = $sumIgnored
                }

                $rs03Ok =
                    ($r.ProcessRc -eq 0) -and
                    ($sumPassed -eq 48) -and
                    ($sumFailed -eq 0) -and
                    ($sumIgnored -eq 1)

                if ($rs03Ok) {
                    $cls = $RV_CLASS.PASS
                    $observed =
                        "cargo_summary passed=48 failed=0 ignored=1; HACP_E13_CORPUS=$e13Corpus; claim bound to P_HEAD=$P_HEAD"
                }
                elseif ($r.ProcessRc -eq $null) {
                    $cls = $RV_CLASS.ENVIRONMENT_RED
                    $envRed = $true
                    $observed = 'RS03 cargo test could not be executed'
                }
                else {
                    $cls = $RV_CLASS.BEHAVIORAL_RED
                    $behRed = $true
                    $observed =
                        "cargo_summary passed=$sumPassed failed=$sumFailed ignored=$sumIgnored != 48/0/1"
                }

                Write-SurfaceResult `
                    -Surface 'RS03' `
                    -Rows @('B18','B6') `
                    -ExitCodes $exitCodes `
                    -Classification $cls `
                    -Expected '48 passed, 1 ignored (ed25519_v1_red), 0 failed; shared E13 corpus supplied; claim bound to exact P_HEAD/P_TREE' `
                    -Observed $observed `
                    -Evidence @($r.StdOutPath,$r.StdErrPath) `
                    -EnvRed $envRed `
                    -BehRed $behRed `
                    -CargoSummary $cargoSummary
            }
        }

        'RS04' {
            $exitCodes['RS04_PROXY_PID'] = $null
            # D3 §8/§9: capture exact previous RS04 temp-env state (EXISTED/NOT_EXISTED + VALUE); restore in finally.
            $rs04Env = Save-RvEnvironment -Names @('HACP_TRUST_MODE','HACP_TRUST_KEYS_FILE','HACP_SIDECAR_PORT','NO_PROXY','HTTP_PROXY','HTTPS_PROXY','ALL_PROXY')
            try {
            # start owned Rust proxy :18081 (UPSTREAM_REQUIRED=NO; no upstream started)
            $portFree = (Test-RvPortState -Port 18081).State -eq 'FREE'
            if (-not $portFree) {
                $cls = $RV_CLASS.ENVIRONMENT_RED; $envRed = $true; $observed = 'port 18081 busy'
            } else {
                $svcOut = Join-Path $evidence 'RS04-proxy.out.log'; $svcErr = Join-Path $evidence 'RS04-proxy.err.log'
                $env:HACP_TRUST_MODE = 'test'; Remove-Item Env:HACP_TRUST_KEYS_FILE -ErrorAction SilentlyContinue; $env:HACP_SIDECAR_PORT = '18081'
                # §21: ensure the localhost probe/workload bypasses any system proxy (HTTP_PROXY/ALL_PROXY).
                $env:NO_PROXY = $(if ($env:NO_PROXY) { "$($env:NO_PROXY),localhost,127.0.0.1" } else { 'localhost,127.0.0.1' })
                $registered = $true
                $proxyProcess = $null; $proxyStart = $null   # F2 §9: StrictMode-safe init before try (Start-Process may throw before assignment)
                try {
                    $proxyProcess = Start-Process -FilePath $PROXY -ArgumentList @() -WorkingDirectory (Join-Path $SpecRepo 'hacp-rs') -RedirectStandardOutput $svcOut -RedirectStandardError $svcErr -PassThru -WindowStyle Hidden
                    $proxyStart = $null
                    try { $proxyStart = $proxyProcess.StartTime.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ') } catch { $proxyStart = $null }
                    Register-RvOwnedProcess -WorkRoot $workRoot -Pid $proxyProcess.Id -Command "hacp-rs-proxy.exe (owned; local response)" -SurfaceOwner 'RS04' -Port 18081 | Out-Null
                    $exitCodes['RS04_PROXY_PID'] = $proxyProcess.Id
                } catch {
                    # R2 §11: registration without a reliable StartTime = PROCESS_OWNERSHIP_REGISTRATION=FAIL => ENVIRONMENT_RED
                    $registered = $false
                    $cls = $RV_CLASS.ENVIRONMENT_RED; $envRed = $true; $observed = "PROCESS_OWNERSHIP_REGISTRATION=FAIL: $($_.Exception.Message)"
                    # D5 §19/§20: NO raw PID-only kill. Verify ownership (PID+StartTime) before any Stop-Process.
                    $killed = $false
                    if ($proxyProcess -and $proxyStart) {
                        $chk = Get-Process -Id $proxyProcess.Id -ErrorAction SilentlyContinue
                        $obsStart = $null; if ($chk) { try { $obsStart = $chk.StartTime.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ') } catch {} }
                        if ($chk -and $obsStart -eq $proxyStart) { try { Stop-Process -Id $proxyProcess.Id -Force -ErrorAction Stop; $killed = $true } catch {} }
                    }
                    if (-not $killed) { $observed += ' [PROCESS_NOT_KILLED: ownership unverifiable or mismatch; fail-safe; cleanup-integrity-fail]' }
                }
                if ($registered) {
                $deadline = (Get-Date).AddSeconds(30); $owned = $false
                while ((Get-Date) -lt $deadline) { if (Test-RvPortOwned -Port 18081 -OwningPid $proxyProcess.Id) { $owned = $true; break }; Start-Sleep -Milliseconds 500 }

                if (-not $owned) {
                    $cls = $RV_CLASS.ENVIRONMENT_RED; $envRed = $true; $observed = 'proxy did not own port 18081'
                } else {
                    # negative fail-closed smoke (separate step; NOT counted in workload)
                    Start-Sleep -Milliseconds 500
                    $probe = Invoke-RvLocalProbe -Uri 'http://127.0.0.1:18081/api/test' -Method GET
                    $smokePass = ($probe.StatusCode -eq 403 -and $probe.Decision -eq 'DENY')
                    Write-RvJson -Object @{ EXPECTED='403 + X-HACP-Decision: DENY'; OBSERVED_STATUS=$probe.StatusCode; OBSERVED_DECISION=$probe.Decision; OBSERVED_REASON=$probe.Reason; PASS=$smokePass } -Path (Join-Path $evidence 'RS04-negative-smoke.json')
                    $exitCodes['RS04_SMOKE'] = $(if ($smokePass) { 0 } else { 1 })

                    # positive workload (fresh fixtures; serial; 100 warmup + 5000 measured)
                    $wl = Invoke-RvNative -FilePath 'go' -ArgumentList @('run','.\benchmarks\rust-http-reproduction','-url','http://127.0.0.1:18081/api/test','-warmup-file',(Join-Path $FIXTURES 'tokens-warmup-100.jsonl'),'-measured-file',(Join-Path $FIXTURES 'tokens-5000.jsonl')) -WorkDir $SidecarRepo -StepId 'RS04-workload' -SurfaceId 'RS04' -EvidenceDir $evidence
                    $exitCodes['RS04_WORKLOAD_RC'] = $wl.ProcessRc
                    $wlOut = Get-RvStdout -Run $wl
                    $workloadPass = ($wl.ProcessRc -eq 0 -and $wlOut -match 'R8_2_WORKLOAD=PASS')

                    if ($smokePass -and $workloadPass) { $cls = $RV_CLASS.PASS; $observed = 'smoke 403+DENY; R8_2_WORKLOAD=PASS (warmup 100 + measured 5000); local response contract' }
                    elseif (-not $smokePass) { $cls = $RV_CLASS.BEHAVIORAL_RED; $behRed = $true; $observed = 'negative smoke did not return 403+DENY' }
                    else { $cls = $RV_CLASS.BEHAVIORAL_RED; $behRed = $true; $observed = 'workload not PASS' }

                    # capture RS05 timing from the measured JSONL workload record
                    $timing = [ordered]@{
                        p50='NOT_REPORTED'
                        p95='NOT_REPORTED'
                        p99='NOT_REPORTED'
                        max='NOT_REPORTED'
                        rps='NOT_REPORTED'
                        mean='NOT_REPORTED'
                    }

                    $measuredRecord = @(
                        $wlOut -split "`r?`n" |
                        Where-Object { $_.Trim().StartsWith('{') } |
                        ForEach-Object { $_ | ConvertFrom-Json } |
                        Where-Object { $_.phase -eq 'measured' }
                    ) | Select-Object -First 1

                    if ($measuredRecord) {
                        $timing['p50']  = $measuredRecord.p50_ms
                        $timing['p95']  = $measuredRecord.p95_ms
                        $timing['p99']  = $measuredRecord.p99_ms
                        $timing['mean'] = $measuredRecord.mean_ms
                        $timing['max']  = $measuredRecord.max_ms

                        if ($measuredRecord.elapsed_ms -gt 0) {
                            $timing['rps'] = [math]::Round(
                                $measuredRecord.requests / ($measuredRecord.elapsed_ms / 1000.0),
                                3
                            )
                        }
                    }

                    Write-RvJson -Object $timing -Path (Join-Path $evidence 'RS05-timing.json')
                }
                $cleanup = Stop-RvOwnedProcesses -WorkRoot $workRoot
                if (-not $cleanup.CleanupComplete) {
                    $cls = $RV_CLASS.ENVIRONMENT_RED
                    $envRed = $true
                    $observed += ' [PROCESS_CLEANUP_INCOMPLETE]'
                }
                }
            }
            Write-SurfaceResult -Surface 'RS04' -Rows @('B16','B17') -ExitCodes $exitCodes -Classification $cls -Expected 'negative smoke 403+DENY; warmup 100/100; measured 5000/5000; R8_2_WORKLOAD=PASS; AUTHORIZATION + LOCAL RESPONSE (UPSTREAM_REQUIRED=NO)' -Observed $observed -Evidence @((Join-Path $evidence 'RS04-negative-smoke.json')) -EnvRed $envRed -BehRed $behRed
            } finally {
                Restore-RvEnvironment -Saved $rs04Env
            }
        }
        'RS05' {
            # DEFECT-9: RS05 requires valid RS04 evidence (result.json exists + classification=PASS).
            $rs04ResultPath = Join-Path $OUT_ROOT 'RS04\result.json'
            $rs04Ok = $false
            if (Test-Path -LiteralPath $rs04ResultPath) {
                try { $rs04r = Get-Content -LiteralPath $rs04ResultPath -Raw | ConvertFrom-Json; $rs04Ok = ($rs04r.classification -eq 'PASS') } catch { $rs04Ok = $false }
            }
            if (-not $rs04Ok) {
                $cls = $RV_CLASS.BLOCKED   # reconciliation -> B19 = RUST_EVIDENCE_INCOMPLETE
                Write-RvLog -Level WARN -Message "RS05 BLOCKED: RS04 missing or not PASS (dependency)."
                Write-SurfaceResult -Surface 'RS05' -Rows @('B19') -ExitCodes @{} -Classification $cls -Expected 'TIMING_CAPTURED after RS04 PASS' -Observed 'RS04 evidence missing or not PASS' -Evidence @() -EnvRed $false -BehRed $false
            } else {
                # timing is extracted from RS04 evidence; no separate workload (§35)
                $tPath = Join-Path $OUT_ROOT 'RS04\RS05-timing.json'
                $timing = [ordered]@{ p50='NOT_REPORTED'; p95='NOT_REPORTED'; p99='NOT_REPORTED'; max='NOT_REPORTED'; rps='NOT_REPORTED'; mean='NOT_REPORTED' }
                if (Test-Path -LiteralPath $tPath) {
                    try { $t = Get-Content -LiteralPath $tPath -Raw | ConvertFrom-Json; foreach ($k in @('p50','p95','p99','max','rps','mean')) { if ($t.$k) { $timing[$k] = $t.$k } } } catch {}
                }
                $timing['INFORMATIONAL_ONLY'] = 'YES'; $timing['THRESHOLD'] = 'NONE'
                Write-RvJson -Object $timing -Path (Join-Path $evidence 'RS05-timing.json')
                $cls = $RV_CLASS.INFORMATIONAL
                Write-SurfaceResult -Surface 'RS05' -Rows @('B19') -ExitCodes @{} -Classification $cls -Expected 'TIMING_CAPTURED; INFORMATIONAL_ONLY=YES; THRESHOLD=NONE; NO GO-vs-Rust ranking' -Observed ("p50=$($timing.p50) p95=$($timing.p95) p99=$($timing.p99) max=$($timing.max) rps=$($timing.rps)") -Evidence @((Join-Path $evidence 'RS05-timing.json')) -EnvRed $false -BehRed $false
            }
        }
        'RS07' {
            # RS07 remains a verification artifact owned by t_rs.
            # Execution occurs in a run-owned temporary Cargo crate.
            # hacp-spec/hacp-rs is used read-only as a path dependency.

            $rs07ExpectedSha =
                'E97E732546F6D1688F0EEB464259F4F0B7B53C82BE3F37B97EEC776C94859FB2'

            $goldenExpectedSha =
                'FA2CAE774C39F9C7C6174B956C55F62F43DA1CD507887E858342B34B3B62086C'

            $rs07Source =
                Join-Path `
                    (Split-Path -Parent $PSScriptRoot) `
                    'RS07_budget_replay_verification.rs'

            $goldenSource =
                Join-Path `
                    $SpecRepo `
                    'vectors\core_inv1_001_golden.json'

            $hacpRsSource =
                Join-Path `
                    $SpecRepo `
                    'hacp-rs'

            $rs07OverlayRoot =
                Join-Path `
                    $workRoot `
                    'rs07-overlay'

            $rs07OverlayCrate =
                Join-Path `
                    $rs07OverlayRoot `
                    'hacp-rs'

            $rs07OverlayTests =
                Join-Path `
                    $rs07OverlayCrate `
                    'tests'

            $rs07OverlayVectors =
                Join-Path `
                    $rs07OverlayRoot `
                    'vectors'

            New-RvDirectory -Path $rs07OverlayRoot | Out-Null
            New-RvDirectory -Path $rs07OverlayCrate | Out-Null
            New-RvDirectory -Path $rs07OverlayTests | Out-Null
            New-RvDirectory -Path $rs07OverlayVectors | Out-Null

            $rs07SourceIdentity =
                Get-RvCanonicalTextIdentity -Path $rs07Source

            $goldenSourceIdentity =
                Get-RvCanonicalTextIdentity -Path $goldenSource

            $rs07SourceOk =
                $rs07SourceIdentity.Exists -and
                (
                    $rs07SourceIdentity.Sha256.ToUpperInvariant() `
                    -eq $rs07ExpectedSha
                )

            $goldenSourceOk =
                $goldenSourceIdentity.Exists -and
                (
                    $goldenSourceIdentity.Sha256.ToUpperInvariant() `
                    -eq $goldenExpectedSha
                )

            $sourceIdentityEvidence = [ordered]@{
                RS07_SOURCE = $rs07Source
                RS07_EXPECTED_SHA256 = $rs07ExpectedSha
                RS07_ACTUAL_SHA256 = $rs07SourceIdentity.Sha256
                RS07_SOURCE_MATCH = $rs07SourceOk
                GOLDEN_SOURCE = $goldenSource
                GOLDEN_EXPECTED_SHA256 = $goldenExpectedSha
                GOLDEN_ACTUAL_SHA256 = $goldenSourceIdentity.Sha256
                GOLDEN_SOURCE_MATCH = $goldenSourceOk
                HACP_RS_PATH_DEPENDENCY = $hacpRsSource
                HACP_RS_HEAD = $P_HEAD
                HACP_RS_TREE = $P_TREE
                OVERLAY_ROOT = $rs07OverlayRoot
                HACP_SPEC_MUTATION = 'NO'
                HUMANIST_CORE_MUTATION = 'NO'
            }

            $sourceIdentityEvidencePath =
                Join-Path `
                    $evidence `
                    'RS07-overlay-source-identity.json'

            Write-RvJson `
                -Object $sourceIdentityEvidence `
                -Path $sourceIdentityEvidencePath

            if (-not $rs07SourceOk -or -not $goldenSourceOk) {
                $cls = $RV_CLASS.ENVIRONMENT_RED
                $envRed = $true

                $problems = @()

                if (-not $rs07SourceOk) {
                    $problems += 'RS07_SOURCE_IDENTITY_MISMATCH'
                }

                if (-not $goldenSourceOk) {
                    $problems += 'GOLDEN_VECTOR_IDENTITY_MISMATCH'
                }

                $observed = $problems -join '; '
                $exitCodes['RS07_OVERLAY_PREP_RC'] = 1

                Write-SurfaceResult `
                    -Surface 'RS07' `
                    -Rows @('B9') `
                    -ExitCodes $exitCodes `
                    -Classification $cls `
                    -Expected 'Rust autonomy budget verification: 3 passed, 0 failed, 0 ignored; run-owned overlay; hacp-spec read-only' `
                    -Observed $observed `
                    -Evidence @($sourceIdentityEvidencePath) `
                    -EnvRed $envRed `
                    -BehRed $behRed

                break
            }

            $rs07OverlayTest =
                Join-Path `
                    $rs07OverlayTests `
                    'budget_replay_verification.rs'

            $goldenOverlay =
                Join-Path `
                    $rs07OverlayVectors `
                    'core_inv1_001_golden.json'

            Copy-Item `
                -LiteralPath $rs07Source `
                -Destination $rs07OverlayTest `
                -Force

            Copy-Item `
                -LiteralPath $goldenSource `
                -Destination $goldenOverlay `
                -Force

            $hacpRsTomlPath =
                $hacpRsSource.Replace('\','/')

            $overlayManifest = @"
[package]
name = "hacp-rs-rs07-overlay"
version = "0.0.0"
edition = "2021"
publish = false

[dependencies]
hacp-rs = { path = "$hacpRsTomlPath" }
ed25519-dalek = { version = "2", features = ["alloc"] }
sha2 = "0.10"
serde_json = "1"
base64 = "0.22"
hex = "0.4"
"@

            $overlayManifestPath =
                Join-Path `
                    $rs07OverlayCrate `
                    'Cargo.toml'

            Set-Content `
                -LiteralPath $overlayManifestPath `
                -Value $overlayManifest `
                -Encoding utf8

            $rs07OverlayIdentity =
                Get-RvCanonicalTextIdentity -Path $rs07OverlayTest

            $goldenOverlayIdentity =
                Get-RvCanonicalTextIdentity -Path $goldenOverlay

            $overlayFilesOk =
                (
                    $rs07OverlayIdentity.Sha256.ToUpperInvariant() `
                    -eq $rs07ExpectedSha
                ) -and
                (
                    $goldenOverlayIdentity.Sha256.ToUpperInvariant() `
                    -eq $goldenExpectedSha
                )

            $overlayPrepared = [ordered]@{
                MANIFEST = $overlayManifestPath
                TEST_FILE = $rs07OverlayTest
                TEST_SHA256 = $rs07OverlayIdentity.Sha256
                GOLDEN_FILE = $goldenOverlay
                GOLDEN_SHA256 = $goldenOverlayIdentity.Sha256
                PATH_DEPENDENCY = $hacpRsSource
                INPUT_IDENTITIES_MATCH = $overlayFilesOk
                HACP_SPEC_MUTATION = 'NO'
            }

            $overlayPreparedPath =
                Join-Path `
                    $evidence `
                    'RS07-overlay-prepared.json'

            Write-RvJson `
                -Object $overlayPrepared `
                -Path $overlayPreparedPath

            if (-not $overlayFilesOk) {
                $cls = $RV_CLASS.ENVIRONMENT_RED
                $envRed = $true
                $observed = 'run-owned overlay input identity mismatch'
                $exitCodes['RS07_OVERLAY_PREP_RC'] = 1

                Write-SurfaceResult `
                    -Surface 'RS07' `
                    -Rows @('B9') `
                    -ExitCodes $exitCodes `
                    -Classification $cls `
                    -Expected 'Rust autonomy budget verification: 3 passed, 0 failed, 0 ignored; run-owned overlay; hacp-spec read-only' `
                    -Observed $observed `
                    -Evidence @(
                        $sourceIdentityEvidencePath,
                        $overlayPreparedPath
                    ) `
                    -EnvRed $envRed `
                    -BehRed $behRed

                break
            }

            $exitCodes['RS07_OVERLAY_PREP_RC'] = 0

            $lock = Invoke-RvNative `
                -FilePath 'cargo' `
                -ArgumentList @(
                    '+stable-x86_64-pc-windows-msvc',
                    'generate-lockfile',
                    '--offline',
                    '--manifest-path',
                    $overlayManifestPath
                ) `
                -WorkDir $rs07OverlayCrate `
                -StepId 'RS07-lockfile' `
                -SurfaceId 'RS07' `
                -EvidenceDir $evidence `
                -ExtraEnv $cargoEnv

            $exitCodes['RS07_LOCKFILE_RC'] =
                $lock.ProcessRc

            if ($lock.ProcessRc -ne 0) {
                $cls = $RV_CLASS.ENVIRONMENT_RED
                $envRed = $true
                $observed =
                    "run-owned overlay lockfile generation failed rc=$($lock.ProcessRc)"

                Write-SurfaceResult `
                    -Surface 'RS07' `
                    -Rows @('B9') `
                    -ExitCodes $exitCodes `
                    -Classification $cls `
                    -Expected 'Rust autonomy budget verification: 3 passed, 0 failed, 0 ignored; run-owned overlay; hacp-spec read-only' `
                    -Observed $observed `
                    -Evidence @(
                        $sourceIdentityEvidencePath,
                        $overlayPreparedPath,
                        $lock.StdOutPath,
                        $lock.StdErrPath
                    ) `
                    -EnvRed $envRed `
                    -BehRed $behRed

                break
            }

            $r = Invoke-RvNative `
                -FilePath 'cargo' `
                -ArgumentList @(
                    '+stable-x86_64-pc-windows-msvc',
                    'test',
                    '--locked',
                    '--offline',
                    '--manifest-path',
                    $overlayManifestPath,
                    '--test',
                    'budget_replay_verification'
                ) `
                -WorkDir $rs07OverlayCrate `
                -StepId 'RS07-budget' `
                -SurfaceId 'RS07' `
                -EvidenceDir $evidence `
                -ExtraEnv $cargoEnv

            $exitCodes['RS07_CARGO_RC'] =
                $r.ProcessRc

            $o = Get-RvStdout -Run $r

            $sumPassed = 0
            $sumFailed = 0
            $sumIgnored = 0

            foreach ($line in ($o -split "`n")) {
                $l = $line.TrimEnd("`r")

                if ($l -match 'test result:') {
                    if ($l -match '(\d+)\s+passed') {
                        $sumPassed += [int]$Matches[1]
                    }

                    if ($l -match '(\d+)\s+failed') {
                        $sumFailed += [int]$Matches[1]
                    }

                    if ($l -match '(\d+)\s+ignored') {
                        $sumIgnored += [int]$Matches[1]
                    }
                }
            }

            $cargoSummary = @{
                passed = $sumPassed
                failed = $sumFailed
                ignored = $sumIgnored
            }

            $rs07Ok =
                ($r.ProcessRc -eq 0) -and
                ($sumPassed -eq 3) -and
                ($sumFailed -eq 0) -and
                ($sumIgnored -eq 0)

            if ($rs07Ok) {
                $cls = $RV_CLASS.PASS
                $observed =
                    'cargo_summary passed=3 failed=0 ignored=0; run-owned overlay; autonomy budget exhaustion only'
            }
            elseif ($r.ProcessRc -eq $null) {
                $cls = $RV_CLASS.ENVIRONMENT_RED
                $envRed = $true
                $observed =
                    'RS07 Cargo process did not produce an exit code'
            }
            else {
                $cls = $RV_CLASS.BEHAVIORAL_RED
                $behRed = $true
                $observed =
                    "cargo_summary passed=$sumPassed failed=$sumFailed ignored=$sumIgnored != 3/0/0"
            }

            Write-SurfaceResult `
                -Surface 'RS07' `
                -Rows @('B9') `
                -ExitCodes $exitCodes `
                -Classification $cls `
                -Expected 'Rust autonomy budget verification: 3 passed, 0 failed, 0 ignored; run-owned overlay; NO token replay / max_uses / default maxUses=1 claim' `
                -Observed $observed `
                -Evidence @(
                    $sourceIdentityEvidencePath,
                    $overlayPreparedPath,
                    $r.StdOutPath
                ) `
                -EnvRed $envRed `
                -BehRed $behRed `
                -CargoSummary $cargoSummary
        }
    }
    return $cls
}

# ---- sequential execution with failure policy ----
$results = @(); $materialRed = $false
foreach ($id in $QUEUE) {
    if ($materialRed -and -not $ContinueIndependentSurfaces) {
        $results += New-RvResult -Surface $id -MatrixRows (@($MATRIX[$id] -split ',')) -ProcessExitCodes @{} -Classification $RV_CLASS.NOT_RUN -ExpectedResult '—' -ObservedResult 'NOT_RUN (stopped after material RED)'
        Write-RvLog -Level WARN -Message "$id NOT_RUN (stopped after material RED)"
        continue
    }
    $c = Invoke-Surface -Id $id
    if ($c -in @($RV_CLASS.ENVIRONMENT_RED,$RV_CLASS.BEHAVIORAL_RED)) {
        $materialRed = $true
        Write-RvLog -Level RED -Message "$id -> $c"
        if (-not $ContinueIndependentSurfaces) { Write-RvLog -Level WARN -Message "STOP ON MATERIAL RED. Use -ContinueIndependentSurfaces to continue independent surfaces." }
    }
    # reload result for summary
    $rp = Join-Path $OUT_ROOT "$id\result.json"
    if (Test-Path -LiteralPath $rp) { $results += (Get-Content -LiteralPath $rp -Raw | ConvertFrom-Json) }
}

# ---- run summary (NO acceptance; §7 max = VERIFICATION_EXECUTION_COMPLETE) ----
$summary = [ordered]@{
    VERIFICATION_EXECUTION_COMPLETE = 'YES'
    RESULTS_RECONCILED = 'PENDING_04'
    HUMAN_REVIEW_REQUIRED = 'YES'
    RS01 = $(if ($results | Where-Object { $_.surface -eq 'RS01' }) { ($results | Where-Object { $_.surface -eq 'RS01' })[0].classification } else { 'NOT_RUN' })
    RS02 = $(if ($results | Where-Object { $_.surface -eq 'RS02' }) { ($results | Where-Object { $_.surface -eq 'RS02' })[0].classification } else { 'NOT_RUN' })
    RS03 = $(if ($results | Where-Object { $_.surface -eq 'RS03' }) { ($results | Where-Object { $_.surface -eq 'RS03' })[0].classification } else { 'NOT_RUN' })
    RS04 = $(if ($results | Where-Object { $_.surface -eq 'RS04' }) { ($results | Where-Object { $_.surface -eq 'RS04' })[0].classification } else { 'NOT_RUN' })
    RS05 = $(if ($results | Where-Object { $_.surface -eq 'RS05' }) { ($results | Where-Object { $_.surface -eq 'RS05' })[0].classification } else { 'NOT_RUN' })
    RS06 = 'NOT_APPLICABLE'
    RS07 = $(if ($results | Where-Object { $_.surface -eq 'RS07' }) { ($results | Where-Object { $_.surface -eq 'RS07' })[0].classification } else { 'NOT_RUN' })
    RUN_DIR = $run.RunDir
}
Write-RvJson -Object $summary -Path (Join-Path $run.RunDir '02-verification\run-summary.json')
Write-RvSummaryText -Table $summary -Path (Join-Path $run.RunDir '02-verification\run-summary.txt')
Write-RvLog -Message "02 complete. Review $($run.RunDir)\02-verification. No acceptance declared."
Set-RvRunState -RunDir $run.RunDir -State 'VERIFICATION_COMPLETE'
# NO admission, NO acceptance, NO synthetic combined score, NO "38/38 DIRECT".
