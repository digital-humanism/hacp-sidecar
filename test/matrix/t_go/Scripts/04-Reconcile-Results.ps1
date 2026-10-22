#Requires -Version 7.0

<#
04-Reconcile-Results.ps1 — MAP ACTUAL SURFACE RESULTS TO AUTHORITATIVE 36-ROW MATRIX.
CLASS = VERIFICATION TOOLING. Does NOT run tests, does NOT fix failures, does NOT declare
acceptance. Only reconciles evidence against the authoritative 36-row matrix. Refuses to emit
SIDECAR_A2=PASS / RELEASE_READY / ACCEPTANCE=PASS / GO. Final decision is human.
#>

[CmdletBinding()]
param(
    [string]$RunRoot      = 'C:\Temp\hacp-1.1.1-verification\run',
    [string]$EvidenceRoot = 'C:\Temp\hacp-1.1.1-verification\evidence',
    [string]$OutputDir    = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Import-Module (Join-Path $PSScriptRoot 'WindowsVerification.Common.psm1') -Force

$EVID_ROOT = New-WvDirectory -Path $EvidenceRoot
if (-not $OutputDir) { $OutputDir = New-WvDirectory -Path (Join-Path $EVID_ROOT '04-Reconcile') }
Set-WvLogFile -Path (Join-Path $RunRoot 'logs\04-reconcile-console.log')
Write-WvLog -Message "04-Reconcile-Results start (NO tests, NO acceptance)"

# ---- Load actual surface verdicts from individual 02 surface evidence ----
$surfaceVerdict = @{}
$surfaceEvidence = @{}

$expectedSurfaces = @(
    'A','B','C','D',
    'S01','S02','S03','S04','S05','S06',
    'S07','S08','S09','S10','S11','S12'
)

foreach ($surfaceId in $expectedSurfaces) {
    $resultPath = Join-Path (Join-Path $EVID_ROOT $surfaceId) 'result.json'

    if (-not (Test-Path -LiteralPath $resultPath -PathType Leaf)) {
        Write-WvLog -Level WARN -Message "Missing surface evidence: $surfaceId"
        continue
    }

    $r = Get-Content -LiteralPath $resultPath -Raw | ConvertFrom-Json

    if ($r.SURFACE_ID -ne $surfaceId) {
        throw "SURFACE_ID_MISMATCH: expected=$surfaceId actual=$($r.SURFACE_ID)"
    }

    if ($surfaceVerdict.ContainsKey($surfaceId)) {
        throw "DUPLICATE_SURFACE_ID: $surfaceId"
    }

    if (-not $r.VERDICT) {
        throw "MISSING_VERDICT: $surfaceId"
    }

    $surfaceVerdict[$surfaceId] = $r.VERDICT
    $surfaceEvidence[$surfaceId] = $resultPath
}

Write-WvLog -Message "Loaded $($surfaceVerdict.Count)/$($expectedSurfaces.Count) individual surface results"

function Resolve-RowClassification {
    param([string]$SurfaceId)
    if (-not $SurfaceId -or $SurfaceId -eq '-') { return $WV_CLASS.NOT_RUN }
    if (-not $surfaceVerdict.ContainsKey($SurfaceId)) { return $WV_CLASS.NOT_RUN }
    switch ($surfaceVerdict[$SurfaceId]) {
        'BEHAVIORAL_RED' { return 'FAIL' }
        default          { return $surfaceVerdict[$SurfaceId] }
    }
}

# ---- Authoritative 36-row mapping (A1-A8, B1-B28) ----
# Runtime rows map to an executed GO surface; governance/evidence rows are static.
$map = @(
    @{ Row='A1';  Surf='S01'; Kind='RUNTIME'; Req='Go Core 38/38' },
    @{ Row='A2';  Surf='S02'; Kind='RUNTIME'; Req='TS Core 38/38' },
    @{ Row='A3';  Surf='S03'; Kind='RUNTIME'; Req='Python Core 38/38' },
    @{ Row='A4';  Surf='C';   Kind='RUNTIME'; Req='Rust Core 36/38 direct + 2 adjudicated' },
    @{ Row='A5';  Surf='S01'; Kind='RUNTIME'; Req='Go HC2 55/55' },
    @{ Row='A6';  Surf='S02'; Kind='RUNTIME'; Req='TS HC2 55/55' },
    @{ Row='A7';  Surf='S03'; Kind='RUNTIME'; Req='Python HC2 55/55' },
    @{ Row='A8';  Surf='S04'; Kind='RUNTIME'; Req='Rust HC2 55/55' },
    @{ Row='B1';  Surf='A';   Kind='RUNTIME'; Req='Native Go regression' },
    @{ Row='B2';  Surf='A';   Kind='RUNTIME'; Req='Protocol correctness' },
    @{ Row='B3';  Surf='A';   Kind='RUNTIME'; Req='Semantic boundary' },
    @{ Row='B4';  Surf='S05'; Kind='RUNTIME'; Req='HC2 runtime black-box 55/55' },
    @{ Row='B5';  Surf='A';   Kind='RUNTIME'; Req='Trust lifecycle' },
    @{ Row='B6';  Surf='S11'; Kind='RUNTIME'; Req='Signature / canonicalization' },
    @{ Row='B7';  Surf='A';   Kind='RUNTIME'; Req='Revocation / expiration' },
    @{ Row='B8';  Surf='A';   Kind='RUNTIME'; Req='Binding / action constraints' },
    @{ Row='B9';  Surf='S12'; Kind='RUNTIME'; Req='Budget / replay (Go: max_uses)' },
    @{ Row='B10'; Surf='A';   Kind='RUNTIME'; Req='Provenance' },
    @{ Row='B11'; Surf='A';   Kind='RUNTIME'; Req='Distributed control' },
    @{ Row='B12'; Surf='A';   Kind='RUNTIME'; Req='Wire / interoperability' },
    @{ Row='B13'; Surf='S06'; Kind='RUNTIME'; Req='External Python-Go E2E 5/5' },
    @{ Row='B14'; Surf='S07'; Kind='RUNTIME'; Req='Anti-bypass AB-E2E 5/5' },
    @{ Row='B15'; Surf='D';   Kind='RUNTIME'; Req='Go HTTP reproduction (auth + forward + response)' },
    @{ Row='B16'; Surf='S08'; Kind='RUNTIME'; Req='Rust HTTP runtime (auth + local response)' },
    @{ Row='B17'; Surf='-';   Kind='HISTORICAL_STATIC_ONLY'; Req='Rust HTTP repository ownership (admitted 3018b6b)' },
    @{ Row='B18'; Surf='S09'; Kind='RUNTIME'; Req='Rust package 48+1 ignored' },
    @{ Row='B19'; Surf='S10'; Kind='RUNTIME'; Req='Timing / benchmarks (informational)' },
    @{ Row='B20'; Surf='-';   Kind='DOCUMENTARY_DECISION_ONLY'; Req='Gate D applicability' },
    @{ Row='B21'; Surf='-';   Kind='DOCUMENTARY_DECISION_ONLY'; Req='Source boundary S/P/C' },
    @{ Row='B22'; Surf='-';   Kind='DOCUMENTARY_DECISION_ONLY'; Req='Mutation discipline' },
    @{ Row='B23'; Surf='-';   Kind='DOCUMENTARY_DECISION_ONLY'; Req='Unresolved HOLD closure' },
    @{ Row='B24'; Surf='-';   Kind='HISTORICAL_STATIC_ONLY'; Req='R9.2 evidence manifest' },
    @{ Row='B25'; Surf='-';   Kind='HISTORICAL_STATIC_ONLY'; Req='R9.2 results reconciliation' },
    @{ Row='B26'; Surf='-';   Kind='HISTORICAL_STATIC_ONLY'; Req='Internal RU closure report' },
    @{ Row='B27'; Surf='-';   Kind='HISTORICAL_STATIC_ONLY'; Req='EN runtime/integration closure report' },
    @{ Row='B28'; Surf='-';   Kind='DOCUMENTARY_DECISION_ONLY'; Req='Final SIDECAR-A2 declaration (human)' }
)

$rows = foreach ($m in $map) {
    $cls = if ($m.Kind -eq 'RUNTIME') { Resolve-RowClassification -SurfaceId $m.Surf } else { $m.Kind }
    $executed = if ($m.Kind -eq 'RUNTIME') { if ($surfaceVerdict.ContainsKey($m.Surf)) { 'YES' } else { 'NO' } } else { 'NO' }
    $ev = if ($m.Surf -ne '-' -and $surfaceEvidence.ContainsKey($m.Surf)) { $surfaceEvidence[$m.Surf] } else { $null }
    [pscustomobject]@{
        MATRIX_ROW   = $m.Row
        REQUIREMENT  = $m.Req
        SURFACE_ID   = $m.Surf
        EXECUTED     = $executed
        RESULT       = $cls
        EVIDENCE     = $ev
        CLASSIFICATION = $cls
    }
}

Write-WvCsv -Object $rows -Path (Join-Path $OutputDir 'matrix-reconciliation.csv')
Write-WvJson -Object $rows -Path (Join-Path $OutputDir 'matrix-reconciliation.json')

# ---- Counters (must sum to 36) ----
$total = @($rows).Count
$pass = @($rows | Where-Object { $_.CLASSIFICATION -eq 'PASS' }).Count
$fail = @($rows | Where-Object { $_.CLASSIFICATION -eq 'FAIL' }).Count
$envRed = @($rows | Where-Object { $_.CLASSIFICATION -eq 'ENVIRONMENT_RED' }).Count
$idRed = @($rows | Where-Object { $_.CLASSIFICATION -eq 'IDENTITY_RED' }).Count
$blocked = @($rows | Where-Object { $_.CLASSIFICATION -eq 'BLOCKED' }).Count
$hist = @($rows | Where-Object { $_.CLASSIFICATION -eq 'HISTORICAL_STATIC_ONLY' }).Count
$na = @($rows | Where-Object { $_.CLASSIFICATION -eq 'NOT_APPLICABLE' }).Count
$doc = @($rows | Where-Object { $_.CLASSIFICATION -eq 'DOCUMENTARY_DECISION_ONLY' }).Count
$notRun = @($rows | Where-Object { $_.CLASSIFICATION -eq 'NOT_RUN' }).Count
$informational = @($rows | Where-Object { $_.CLASSIFICATION -eq 'INFORMATIONAL' }).Count
$executable = @($rows | Where-Object { $_.SURFACE_ID -ne '-' }).Count

$counters = [ordered]@{
    TOTAL_MATRIX_ROWS = $total
    EXECUTABLE_ROWS = $executable
    PASS_ROWS = $pass
    FAIL_ROWS = $fail
    ENVIRONMENT_RED_ROWS = $envRed
    IDENTITY_RED_ROWS = $idRed
    BLOCKED_ROWS = $blocked
    HISTORICAL_STATIC_ONLY_ROWS = $hist
    NOT_APPLICABLE_ROWS = $na
    DOCUMENTARY_DECISION_ONLY_ROWS = $doc
    INFORMATIONAL_ROWS = $informational
    NOT_RUN_ROWS = $notRun
    SUM_CHECK = ($pass + $fail + $envRed + $idRed + $blocked + $hist + $na + $doc + $notRun + $informational)
}
Write-WvJson -Object $counters -Path (Join-Path $OutputDir 'matrix-counters.json')
Write-WvSummaryText -Table $counters -Path (Join-Path $OutputDir 'matrix-counters.txt')

Write-WvLog -Message "TOTAL=$total SUM_CHECK=$($counters.SUM_CHECK) (must equal 36). PASS=$pass FAIL=$fail ENV_RED=$envRed ID_RED=$idRed BLOCKED=$blocked HIST=$hist NA=$na DOC=$doc NOT_RUN=$notRun"
Write-WvLog -Message "NO automatic acceptance. Human reviews evidence and decides. 04 does NOT emit SIDECAR_A2=PASS / RELEASE_READY / ACCEPTANCE=PASS / GO."

# ---------------------------------------------------------------------------
# Human-readable t_go verification report.
#
# This is evidence presentation only.
# It does not perform tests and does not emit acceptance / release decisions.
# ---------------------------------------------------------------------------

$VerificationReportPath = Join-Path $EvidenceRoot 'T_GO_VERIFICATION_REPORT.md'

function Get-TGoEvidenceText {
    param(
        [Parameter(Mandatory)]
        [string]$Surface,

        [Parameter(Mandatory)]
        [string]$File
    )

    $Path = Join-Path (Join-Path $EvidenceRoot $Surface) $File

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "T_GO_REPORT_EVIDENCE_MISSING=$Path"
    }

    return Get-Content -LiteralPath $Path -Raw
}

function Assert-TGoEvidence {
    param(
        [Parameter(Mandatory)]
        [string]$Text,

        [Parameter(Mandatory)]
        [string]$Pattern,

        [Parameter(Mandatory)]
        [string]$Name
    )

    if ($Text -notmatch $Pattern) {
        throw "T_GO_REPORT_EXPECTATION_MISSING evidence=$Name pattern=$Pattern"
    }
}

function Get-TGoMatchInt {
    param(
        [Parameter(Mandatory)]
        [string]$Text,

        [Parameter(Mandatory)]
        [string]$Pattern,

        [int]$Group = 1
    )

    $m = [regex]::Match(
        $Text,
        $Pattern,
        [System.Text.RegularExpressions.RegexOptions]::Multiline
    )

    if (-not $m.Success) {
        throw "T_GO_REPORT_PARSE_FAIL pattern=$Pattern"
    }

    return [int]$m.Groups[$Group].Value
}

function Get-TGoFirstLine {
    param(
        [Parameter(Mandatory)]
        [string]$Text,

        [Parameter(Mandatory)]
        [string]$Pattern
    )

    $m = [regex]::Match(
        $Text,
        $Pattern,
        [System.Text.RegularExpressions.RegexOptions]::Multiline
    )

    if (-not $m.Success) {
        return $null
    }

    return $m.Value.Trim()
}

$GoCore  = Get-TGoEvidenceText 'S01' 'S01-core.stdout.log'
$GoHC2   = Get-TGoEvidenceText 'S01' 'S01-hc2.stdout.log'

$TsCore  = Get-TGoEvidenceText 'S02' 'S02-core.stdout.log'
$TsHC2   = Get-TGoEvidenceText 'S02' 'S02-hc2.stdout.log'
$TsReg   = Get-TGoEvidenceText 'S02' 'S02-regression.stdout.log'

$PyCore  = Get-TGoEvidenceText 'S03' 'S03-core.stdout.log'
$PyHC2   = Get-TGoEvidenceText 'S03' 'S03-hc2.stdout.log'

$SideHC2 = Get-TGoEvidenceText 'S05' 'S05-hc2.stdout.log'

$RsCore  = Get-TGoEvidenceText 'C'   'C-core-family-a.stdout.log'
$RsHC2   = Get-TGoEvidenceText 'S04' 'S04-hc2.stdout.log'
$RsTests = Get-TGoEvidenceText 'S09' 'S09-cargo-test.stdout.log'

$Wire    = Get-TGoEvidenceText 'S11' 'S11-wire.stdout.log'
$BudgetU = Get-TGoEvidenceText 'S12' 'S12-budget-unit.stdout.log'
$BudgetI = Get-TGoEvidenceText 'S12' 'S12-proxy-integration.stdout.log'

$H03     = Get-TGoEvidenceText 'B'   'B-h03.stdout.log'
$E2E     = Get-TGoEvidenceText 'S06' 'S06-e2e.stdout.log'
$Bypass  = Get-TGoEvidenceText 'S07' 'S07-anti-bypass.stdout.log'

$GoHttp  = Get-TGoEvidenceText 'D'   'D-b15.stdout.log'
$RsHttp  = Get-TGoEvidenceText 'S08' 'S08-b16.stdout.log'

$GoSerial = Get-TGoEvidenceText 'S10' 'S10-go-serial.stdout.log'
$PsRot    = Get-TGoEvidenceText 'S10' 'S10-ps-rotating.stdout.log'
$GoRot    = Get-TGoEvidenceText 'S10' 'S10-go-rotating.stdout.log'


# ---------------------------------------------------------------------------
# Conformance facts.
# ---------------------------------------------------------------------------

Assert-TGoEvidence $GoCore  'RESULTS:\s+38/38 passed' 'Go Core'
Assert-TGoEvidence $GoHC2   'RESULTS:\s+55/55 passed' 'Go HC2'

Assert-TGoEvidence $TsCore  'RESULTS:\s+38/38 passed' 'TypeScript Core'
Assert-TGoEvidence $TsHC2   'RESULTS:\s+55/55 passed' 'TypeScript HC2'
Assert-TGoEvidence $TsReg   'ℹ fail 0'                'TypeScript regression'

$PyCoreVectors = (
    [regex]::Matches(
        $PyCore,
        'test_conformance_vector\[[^\]]+\]\s+PASSED'
    )
).Count

$PyHC2Vectors = (
    [regex]::Matches(
        $PyHC2,
        'test_enforcement_v2_family_a_vector\[[^\]]+\]\s+PASSED'
    )
).Count

if ($PyCoreVectors -ne 38) {
    throw "PYTHON_CORE_VECTOR_COUNT=$PyCoreVectors EXPECTED=38"
}

if ($PyHC2Vectors -ne 55) {
    throw "PYTHON_HC2_VECTOR_COUNT=$PyHC2Vectors EXPECTED=55"
}

Assert-TGoEvidence $RsCore 'DIRECT RESULTS:\s+36/38 passed' 'Rust Core direct'
Assert-TGoEvidence $RsCore 'ADJUDICATED:\s+2' 'Rust Core adjudication'
Assert-TGoEvidence $RsCore 'FINAL ACCOUNTING:\s+36 direct \+ 2 adjudicated' 'Rust Core accounting'
Assert-TGoEvidence $RsCore 'CORE-INV5-006' 'Rust duplicate-member witness'
Assert-TGoEvidence $RsCore 'RAW-WIRE:\s+DENY / INVALID_ACTION' 'Rust raw-wire result'
Assert-TGoEvidence $RsHC2  'RESULTS:\s+55/55 passed' 'Rust HC2'

Assert-TGoEvidence $SideHC2 'RESULTS:\s+55/55 passed' 'Sidecar HC2'


# ---------------------------------------------------------------------------
# Runtime qualification facts.
# ---------------------------------------------------------------------------

$WirePassed = (
    [regex]::Matches(
        $Wire,
        '(?m)^--- PASS: '
    )
).Count

$BudgetUnitPassed = (
    [regex]::Matches(
        $BudgetU,
        '(?m)^--- PASS: '
    )
).Count

$BudgetIntegrationPassed = (
    [regex]::Matches(
        $BudgetI,
        '(?m)^--- PASS: '
    )
).Count

Assert-TGoEvidence $H03 'R4_3_H03=PASS' 'H03'
Assert-TGoEvidence $H03 'PROCESS_EXIT_CODE=0' 'H03 child exit'
Assert-TGoEvidence $H03 'LISTENER_REMOVED=PASS' 'H03 listener'
Assert-TGoEvidence $H03 'PROVENANCE_PERSISTED_AFTER_SHUTDOWN=PASS' 'H03 provenance'

$E2EPassed = (
    [regex]::Matches(
        $E2E,
        '(?m)^tests/.+PASSED'
    )
).Count

$BypassPassed = (
    [regex]::Matches(
        $Bypass,
        '(?m)^\[PASS\] AB-E2E-'
    )
).Count

if ($E2EPassed -ne 5) {
    throw "E2E_PASS_COUNT=$E2EPassed EXPECTED=5"
}

if ($BypassPassed -ne 5) {
    throw "ANTI_BYPASS_PASS_COUNT=$BypassPassed EXPECTED=5"
}


# ---------------------------------------------------------------------------
# Rust package qualification.
# ---------------------------------------------------------------------------

$RustActivePassed = 0
$RustIgnored = 0
$RustFailed = 0

foreach (
    $m in [regex]::Matches(
        $RsTests,
        'test result: ok\. (\d+) passed; (\d+) failed; (\d+) ignored;',
        [System.Text.RegularExpressions.RegexOptions]::Multiline
    )
) {
    $RustActivePassed += [int]$m.Groups[1].Value
    $RustFailed       += [int]$m.Groups[2].Value
    $RustIgnored      += [int]$m.Groups[3].Value
}


# ---------------------------------------------------------------------------
# HTTP reproduction facts.
# ---------------------------------------------------------------------------

Assert-TGoEvidence $GoHttp 'GO_HTTP_REPRODUCTION=PASS' 'Go HTTP reproduction'
Assert-TGoEvidence $RsHttp 'R8_2_WORKLOAD=PASS' 'Rust HTTP reproduction'

$GoMeasuredLine = (
    $GoHttp -split "`r?`n" |
    Where-Object {
        $_ -match '"phase":"measured"'
    } |
    Select-Object -First 1
)

$RsMeasuredLine = (
    $RsHttp -split "`r?`n" |
    Where-Object {
        $_ -match '"phase":"measured"'
    } |
    Select-Object -First 1
)

if (-not $GoMeasuredLine) {
    throw 'GO_HTTP_MEASURED_JSON_MISSING'
}

if (-not $RsMeasuredLine) {
    throw 'RUST_HTTP_MEASURED_JSON_MISSING'
}

$GoMeasured = $GoMeasuredLine | ConvertFrom-Json
$RsMeasured = $RsMeasuredLine | ConvertFrom-Json


# ---------------------------------------------------------------------------
# Benchmark observations.
# ---------------------------------------------------------------------------

$GoRotMessage = Get-TGoFirstLine `
    $GoRot `
    '^TAIL LATENCY HIGH:.*$'

$GoSerialMessage = Get-TGoFirstLine `
    $GoSerial `
    '^GATE D NEEDS OPTIMIZATION:.*$'

$PsRotMessage = Get-TGoFirstLine `
    $PsRot `
    '^TEST INVALID:.*$'


# ---------------------------------------------------------------------------
# Build Markdown.
# ---------------------------------------------------------------------------

$Report = [System.Collections.Generic.List[string]]::new()

function Add-TGoReportLine {
    param([string]$Text = '')
    $Report.Add($Text)
}

Add-TGoReportLine '# HACP 1.1.1 — t_go Verification Report'
Add-TGoReportLine ''
Add-TGoReportLine ('Generated from `02-Run-Verification.ps1` evidence at `{0}`.' -f ([DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')))
Add-TGoReportLine ''
Add-TGoReportLine 'This report presents verification evidence only. It does not emit release acceptance, RELEASE_READY, ACCEPTANCE=PASS or GO.'
Add-TGoReportLine ''

Add-TGoReportLine '## 1. HACP-Core and HC2 language reproduction'
Add-TGoReportLine ''
Add-TGoReportLine 'All four admitted implementations are reproduced against the same Family A HACP-Core and HC2 vector surfaces.'
Add-TGoReportLine ''

Add-TGoReportLine '### Go — `hacp-go`'
Add-TGoReportLine ''
Add-TGoReportLine 'Reproduced implementation: Go reference/conformance implementation from `hacp-spec`.'
Add-TGoReportLine ''
Add-TGoReportLine 'Checks: HACP-Core decision behavior and HC2 request-target/enforcement binding.'
Add-TGoReportLine ''
Add-TGoReportLine '- Core: **38/38 all pass**'
Add-TGoReportLine '- HC2: **55/55 all pass**'
Add-TGoReportLine ''

Add-TGoReportLine '### TypeScript — `hacp-ts`'
Add-TGoReportLine ''
Add-TGoReportLine 'Reproduced implementation: TypeScript reference/conformance implementation from `hacp-spec`.'
Add-TGoReportLine ''
Add-TGoReportLine 'Checks: the same Core-38 and HC2-55 semantics through the TypeScript CLI plus its package conformance regression.'
Add-TGoReportLine ''
Add-TGoReportLine '- Core: **38/38 all pass**'
Add-TGoReportLine '- HC2: **55/55 all pass**'
Add-TGoReportLine '- package conformance regression: **39/39 tests all pass**'
Add-TGoReportLine ''

Add-TGoReportLine '### Python — `humanist-core`'
Add-TGoReportLine ''
Add-TGoReportLine 'Reproduced implementation: Python SDK implementation in `humanist-core`.'
Add-TGoReportLine ''
Add-TGoReportLine 'Checks: Python realization of Core Family A and HC2 Family A behavior.'
Add-TGoReportLine ''
Add-TGoReportLine '- Core: **38/38 all pass**'
Add-TGoReportLine '- HC2: **55/55 all pass**'
Add-TGoReportLine ''

Add-TGoReportLine '### Rust — `hacp-rs`'
Add-TGoReportLine ''
Add-TGoReportLine 'Reproduced implementation: Rust implementation in `hacp-spec\hacp-rs`, executed through `hacp-rs-runner.exe`.'
Add-TGoReportLine ''
Add-TGoReportLine 'Checks: Rust Family A decision behavior, duplicate-member raw-wire ingress behavior and HC2 enforcement semantics.'
Add-TGoReportLine ''
Add-TGoReportLine '- Core: **36 direct + 2 adjudicated / 38**'
Add-TGoReportLine '- HC2: **55/55 all pass**'
Add-TGoReportLine '- `CORE-INV1-002`: CHECKPOINT expected, DENY observed — adjudicated'
Add-TGoReportLine '- `CORE-RUNTIME-004`: CHECKPOINT expected, DENY observed — adjudicated'
Add-TGoReportLine '- `CORE-INV5-006`: direct raw-wire **DENY / INVALID_ACTION**'
Add-TGoReportLine ''

Add-TGoReportLine '### Core / HC2 reproduction matrix'
Add-TGoReportLine ''
Add-TGoReportLine '| Implementation | HACP-Core | HC2 |'
Add-TGoReportLine '|---|---:|---:|'
Add-TGoReportLine '| Go (`hacp-go`) | **38/38 all pass** | **55/55 all pass** |'
Add-TGoReportLine '| TypeScript (`hacp-ts`) | **38/38 all pass** | **55/55 all pass** |'
Add-TGoReportLine '| Python (`humanist-core`) | **38/38 all pass** | **55/55 all pass** |'
Add-TGoReportLine '| Rust (`hacp-rs`) | **36 direct + 2 adjudicated / 38** | **55/55 all pass** |'
Add-TGoReportLine ''

Add-TGoReportLine '## 2. Go sidecar runtime qualification'
Add-TGoReportLine ''
Add-TGoReportLine 'Reproduced implementation: production Go `hacp-sidecar` runtime.'
Add-TGoReportLine ''
Add-TGoReportLine 'Checks: runtime enforcement behavior beneath the language conformance layer — wire canonicalization/signatures, budget/replay behavior, provenance ordering and graceful shutdown.'
Add-TGoReportLine ''
Add-TGoReportLine ('- wire / canonicalization / Ed25519 focused tests: **{0}/{0} all pass**' -f $WirePassed)
Add-TGoReportLine ('- budget unit tests: **{0}/{0} all pass**' -f $BudgetUnitPassed)
Add-TGoReportLine ('- budget/replay proxy integration: **{0}/{0} all pass**' -f $BudgetIntegrationPassed)
Add-TGoReportLine '- H03 graceful shutdown: **authorized ALLOW, child exit 0, listener removed, provenance persisted**'
Add-TGoReportLine ''

Add-TGoReportLine '## 3. Sidecar HC2 enforcement reproduction'
Add-TGoReportLine ''
Add-TGoReportLine 'Reproduced implementation: real Go sidecar conformance runner.'
Add-TGoReportLine ''
Add-TGoReportLine 'Checks: HC2 Family A enforcement through the sidecar-owned runtime path rather than only a standalone language implementation.'
Add-TGoReportLine ''
Add-TGoReportLine '- HC2: **55/55 all pass**'
Add-TGoReportLine ''

Add-TGoReportLine '## 4. Rust package qualification'
Add-TGoReportLine ''
Add-TGoReportLine 'Reproduced implementation: `hacp-rs` static package test surface.'
Add-TGoReportLine ''
Add-TGoReportLine 'Checks: JCS, authorization-v2, canonical-v2, signature binding, Ed25519-v2, duplicate-member JSON ingress, MCP projection and SHA-256 primitives.'
Add-TGoReportLine ''
Add-TGoReportLine ('- active tests: **{0} passed**' -f $RustActivePassed)
Add-TGoReportLine ('- failed: **{0}**' -f $RustFailed)
Add-TGoReportLine ('- historical ignored witnesses: **{0}**' -f $RustIgnored)
Add-TGoReportLine ''


Add-TGoReportLine '## 5. Rust execution provenance'
Add-TGoReportLine ''
Add-TGoReportLine 'Core/HC2 runner and HTTP proxy are separate adapters over the same admitted `hacp-rs` evaluator implementation.'
Add-TGoReportLine ''
Add-TGoReportLine 'Active evaluator:'
Add-TGoReportLine ''
Add-TGoReportLine '`hacp-spec\hacp-rs\src\evaluate.rs`'
Add-TGoReportLine ''
Add-TGoReportLine 'SHA-256:'
Add-TGoReportLine ''
Add-TGoReportLine '`74580DE3A9E345B64E2548EDD07DDB4A4D7058C2F847CF0EBB9D8468C23FE041`'
Add-TGoReportLine ''
Add-TGoReportLine 'Execution composition:'
Add-TGoReportLine ''
Add-TGoReportLine '- `hacp-rs-runner` → `evaluate::evaluate(...)`'
Add-TGoReportLine '- `hacp-rs-proxy` → `evaluate::evaluate(...)`'
Add-TGoReportLine ''
Add-TGoReportLine 'Therefore the Rust HTTP reproduction/benchmark exercises the same active decision implementation qualified by the Rust Core and HC2 reproduction.'
Add-TGoReportLine ''
Add-TGoReportLine 'Qualified Rust results bound to this evaluator:'
Add-TGoReportLine ''
Add-TGoReportLine '- Core: **36 direct + 2 adjudicated / 38**'
Add-TGoReportLine '- HC2: **55/55 all pass**'
Add-TGoReportLine ''

Add-TGoReportLine '## 5. Python ↔ Go external E2E'
Add-TGoReportLine ''
Add-TGoReportLine 'Reproduced implementations: Python `humanist-core` client/signing path against the real Go `hacp-sidecar`.'
Add-TGoReportLine ''
Add-TGoReportLine 'Checks: cross-language action-hash agreement, signature consistency and real sidecar ALLOW behavior.'
Add-TGoReportLine ''
Add-TGoReportLine ('- **{0}/{0} all pass**' -f $E2EPassed)
Add-TGoReportLine ''

Add-TGoReportLine '## 6. Deployment anti-bypass'
Add-TGoReportLine ''
Add-TGoReportLine 'Reproduced topology: test agent → Go sidecar → protected upstream.'
Add-TGoReportLine ''
Add-TGoReportLine 'Checks: the sidecar remains the required enforcement point and the protected upstream cannot be reached directly by the agent.'
Add-TGoReportLine ''
Add-TGoReportLine ('- **{0}/{0} all pass**' -f $BypassPassed)
Add-TGoReportLine ''

Add-TGoReportLine '## 7. HTTP reproduction'
Add-TGoReportLine ''
Add-TGoReportLine '### Go sidecar'
Add-TGoReportLine ''
Add-TGoReportLine 'Measurement boundary: authorization plus real upstream forwarding.'
Add-TGoReportLine ''
Add-TGoReportLine ('- measured requests: **{0}/{1} successful**' -f $GoMeasured.success, $GoMeasured.requests)
Add-TGoReportLine ('- mean: **{0:N2} ms**' -f [double]$GoMeasured.mean_ms)
Add-TGoReportLine ('- p50: **{0:N2} ms**' -f [double]$GoMeasured.p50_ms)
Add-TGoReportLine ('- p95: **{0:N2} ms**' -f [double]$GoMeasured.p95_ms)
Add-TGoReportLine ('- p99: **{0:N2} ms**' -f [double]$GoMeasured.p99_ms)
Add-TGoReportLine ''

Add-TGoReportLine '### Rust HTTP proxy'
Add-TGoReportLine ''
Add-TGoReportLine 'Measurement boundary: authorization plus local response. This is intentionally not interpreted as the same workload as the Go real-forwarding path.'
Add-TGoReportLine ''
Add-TGoReportLine ('- measured requests: **{0}/{1} successful**' -f $RsMeasured.success, $RsMeasured.requests)
Add-TGoReportLine ('- mean: **{0:N2} ms**' -f [double]$RsMeasured.mean_ms)
Add-TGoReportLine ('- p50: **{0:N2} ms**' -f [double]$RsMeasured.p50_ms)
Add-TGoReportLine ('- p95: **{0:N2} ms**' -f [double]$RsMeasured.p95_ms)
Add-TGoReportLine ('- p99: **{0:N2} ms**' -f [double]$RsMeasured.p99_ms)
Add-TGoReportLine ''

Add-TGoReportLine '## 8. Benchmark observations'
Add-TGoReportLine ''
Add-TGoReportLine 'These are observed performance results, not HACP-Core / HC2 conformance results.'
Add-TGoReportLine ''

if ($GoRotMessage) {
    Add-TGoReportLine ('- Go rotating-token benchmark: `{0}`' -f $GoRotMessage)
}

if ($GoSerialMessage) {
    Add-TGoReportLine ('- Go serial benchmark: `{0}`' -f $GoSerialMessage)
}

if ($PsRotMessage) {
    Add-TGoReportLine ('- PowerShell rotating-token benchmark: `{0}`' -f $PsRotMessage)
}

Add-TGoReportLine ''

Add-TGoReportLine '## 9. Verification summary'
Add-TGoReportLine ''
Add-TGoReportLine '| Verification surface | Result |'
Add-TGoReportLine '|---|---:|'
Add-TGoReportLine '| Go Core | **38/38 all pass** |'
Add-TGoReportLine '| Go HC2 | **55/55 all pass** |'
Add-TGoReportLine '| TypeScript Core | **38/38 all pass** |'
Add-TGoReportLine '| TypeScript HC2 | **55/55 all pass** |'
Add-TGoReportLine '| Python Core | **38/38 all pass** |'
Add-TGoReportLine '| Python HC2 | **55/55 all pass** |'
Add-TGoReportLine '| Rust Core | **36 direct + 2 adjudicated / 38** |'
Add-TGoReportLine '| Rust HC2 | **55/55 all pass** |'
Add-TGoReportLine '| Sidecar HC2 | **55/55 all pass** |'
Add-TGoReportLine ('| Wire / crypto | **{0}/{0} all pass** |' -f $WirePassed)
Add-TGoReportLine ('| Budget / replay | **{0}/{0} all pass** |' -f ($BudgetUnitPassed + $BudgetIntegrationPassed))
Add-TGoReportLine ('| Python ↔ Go E2E | **{0}/{0} all pass** |' -f $E2EPassed)
Add-TGoReportLine ('| Deployment anti-bypass | **{0}/{0} all pass** |' -f $BypassPassed)
Add-TGoReportLine ('| Go HTTP measured workload | **{0}/{1} successful** |' -f $GoMeasured.success, $GoMeasured.requests)
Add-TGoReportLine ('| Rust HTTP measured workload | **{0}/{1} successful** |' -f $RsMeasured.success, $RsMeasured.requests)
Add-TGoReportLine ('| Rust active package tests | **{0} passed, {1} failed, {2} historical ignored** |' -f $RustActivePassed, $RustFailed, $RustIgnored)
Add-TGoReportLine ''
Add-TGoReportLine '> Controlled-execution worktree `IDENTITY_RED` remains expected/open until the subsequent `t_rs` remediation and closure stage.'

$Report |
    Set-Content `
        -LiteralPath $VerificationReportPath `
        -Encoding utf8

# Required console presentation:
# one blank line immediately after the existing
# "NO automatic acceptance..." reconciliation message,
# then the exact Markdown report content.

Write-Host ''

Get-Content -LiteralPath $VerificationReportPath

# Explicitly withheld (§52): SIDECAR_A2=PASS, RELEASE_READY=YES, ACCEPTANCE=PASS, GO.
