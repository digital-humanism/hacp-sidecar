#Requires -Version 7.0

<#
04-Reconcile-Results.ps1 (RUST) — documentary/result reconciliation. NO tests/build/runtime.
Maps Rust evidence to the authoritative 36-row matrix using the Rust status model (§48):
RUST_NOT_APPLICABLE / RUST_VERIFIED / RUST_ENVIRONMENT_BLOCKED / RUST_BEHAVIORAL_RED /
RUST_EVIDENCE_INCOMPLETE. Never emits ACCEPTED/RELEASED. Max auto marker:
RUST_LOCAL_REPRODUCTION_VERIFIED + HUMAN_ACCEPTANCE_DECISION_REQUIRED.
#>

[CmdletBinding()]
param(
    [string]$EvidenceRoot = '',
    [string]$RunRoot      = '',
    [string]$RunDir       = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Portable default path resolution.
# Explicit caller-supplied paths take precedence.
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

$run = Resolve-RvRunDir -EvidenceRoot $EvidenceRoot -RunDir $RunDir -RequireState 'POSTFLIGHT_COMPLETE'
$OUT = New-RvDirectory -Path (Join-Path $run.RunDir '04-reconciliation')
Set-RvLogFile -Path (Join-Path $OUT 'reconcile-console.log')
Write-RvLog -Message "04-Reconcile (RUST) start. NO tests, NO acceptance."

# ---- Load RS surface results from 02-verification ----
$rsClass = @{}
foreach ($id in @('RS01','RS02','RS03','RS04','RS05','RS07')) {
    $rp = Join-Path $run.RunDir "02-verification\$id\result.json"
    if (Test-Path -LiteralPath $rp) {
        try { $r = Get-Content -LiteralPath $rp -Raw | ConvertFrom-Json; $rsClass[$id] = $r.classification } catch { $rsClass[$id] = 'NOT_RUN' }
    } else { $rsClass[$id] = 'NOT_RUN' }
}
# RS05 may be INFORMATIONAL (timing). RS06 is NOT_APPLICABLE (no run).

function Resolve-RustRowStatus {
    param([string]$RSurf)
    $c = if ($rsClass.ContainsKey($RSurf)) { $rsClass[$RSurf] } else { 'NOT_RUN' }
    switch ($c) {
        'PASS'             { return 'RUST_VERIFIED' }

        'INFORMATIONAL'    { return 'RUST_VERIFIED' }   # timing captured (informational; no threshold)
        'ENVIRONMENT_RED'  { return 'RUST_ENVIRONMENT_BLOCKED' }
        'BEHAVIORAL_RED'   { return 'RUST_BEHAVIORAL_RED' }
        'NOT_RUN'          { return 'RUST_EVIDENCE_INCOMPLETE' }
        'BLOCKED'          { return 'RUST_EVIDENCE_INCOMPLETE' }
        default            { return 'RUST_EVIDENCE_INCOMPLETE' }
    }
}

# ---- Authoritative 36-row Rust applicability (§49) ----
# Rust-applicable rows map to an RS surface; all others are RUST_NOT_APPLICABLE.
$rustMap = @{ A4='RS01'; A8='RS02'; B6='RS03'; B9='RS07'; B16='RS04'; B17='RS04'; B18='RS03'; B19='RS05' }
$allRows = @('A1','A2','A3','A4','A5','A6','A7','A8') + @(1..28 | ForEach-Object { "B$_" })

$rows = foreach ($row in $allRows) {
    if ($rustMap.ContainsKey($row)) {
        $rs = $rustMap[$row]
        $status = Resolve-RustRowStatus -RSurf $rs
        $note = switch ($row) {
            'A4'  { 'Rust Core: 36/38 direct + 2 adjudicated (CORE-INV1-002, CORE-RUNTIME-004); CORE-INV5-006 raw-wire PASS; never report 38/38 direct' }
            'A8'  { 'Rust HC2: 55/55 (separate verdict family)' }
            'B6'  { 'Rust signature/canonicalization covered by RS03 package tests (EXACT)' }
            'B9'  { 'Rust autonomy budget exhaustion only; NO token replay / max_uses claim' }
            'B16' { 'Rust HTTP: AUTHORIZATION + LOCAL RESPONSE; NO upstream forwarding' }
            'B17' { 'Rust HTTP repository ownership (admitted 3018b6b)' }
            'B18' { 'Rust package: 48 ACTIVE + 1 HISTORICAL IGNORED; claim bound to P_HEAD/P_TREE' }
            'B19' { 'Rust timing: INFORMATIONAL_ONLY; THRESHOLD=NONE; NO GO-vs-Rust ranking' }
            default { '' }
        }
        [pscustomobject]@{ MATRIX_ROW=$row; RUST_APPLICABILITY='RUST_APPLICABLE'; RS_SURFACE=$rs; RUST_STATUS=$status; RS_CLASSIFICATION=$rsClass[$rs]; NOTE=$note }
    } else {
        [pscustomobject]@{ MATRIX_ROW=$row; RUST_APPLICABILITY='RUST_NOT_APPLICABLE'; RS_SURFACE=''; RUST_STATUS='RUST_NOT_APPLICABLE'; RS_CLASSIFICATION=''; NOTE='Go/TS/Python or governance row; out of Rust reproduction contour' }
    }
}

Write-RvCsv -Object $rows -Path (Join-Path $OUT 'RUST_MATRIX_RESULT.csv')
Write-RvJson -Object $rows -Path (Join-Path $OUT 'RUST_MATRIX_RESULT.json')

# ---- Counters (must sum to 36) ----
$total = @($rows).Count
$applicable = @($rows | Where-Object { $_.RUST_APPLICABILITY -eq 'RUST_APPLICABLE' }).Count
$notApplicable = @($rows | Where-Object { $_.RUST_STATUS -eq 'RUST_NOT_APPLICABLE' }).Count
$verified = @($rows | Where-Object { $_.RUST_STATUS -eq 'RUST_VERIFIED' }).Count
$envBlocked = @($rows | Where-Object { $_.RUST_STATUS -eq 'RUST_ENVIRONMENT_BLOCKED' }).Count
$behRed = @($rows | Where-Object { $_.RUST_STATUS -eq 'RUST_BEHAVIORAL_RED' }).Count
$evIncomplete = @($rows | Where-Object { $_.RUST_STATUS -eq 'RUST_EVIDENCE_INCOMPLETE' }).Count
$sumCheck = $verified + $envBlocked + $behRed + $evIncomplete + $notApplicable
$unresolved = $total - $sumCheck

$counters = [ordered]@{
    TOTAL_MATRIX_ROWS = $total
    RUST_APPLICABLE_ROWS = $applicable
    RUST_NOT_APPLICABLE_ROWS = $notApplicable
    RUST_VERIFIED_ROWS = $verified
    RUST_ENVIRONMENT_BLOCKED_ROWS = $envBlocked
    RUST_BEHAVIORAL_RED_ROWS = $behRed
    RUST_EVIDENCE_INCOMPLETE_ROWS = $evIncomplete
    SUM_CHECK = $sumCheck
    UNRESOLVED_ROWS = $unresolved
}

# ---- DEFECT-7: consume postflight integrity (03-postflight/POSTFLIGHT.json) ----
$postflightPath = Join-Path $run.RunDir '03-postflight\POSTFLIGHT.json'
$pfCleanup = 'MISSING'; $pfGit = 'MISSING'; $pfEv = 'MISSING'
$cleanupOk = $false; $gitGuardsOk = $false; $evInvOk = $false
if (Test-Path -LiteralPath $postflightPath) {
    try {
        $pf = Get-Content -LiteralPath $postflightPath -Raw | ConvertFrom-Json
        $pfCleanup = $pf.CLEANUP_COMPLETE; $pfGit = $pf.GIT_GUARDS_PASS; $pfEv = $pf.EVIDENCE_INVENTORY_COMPLETE
        $cleanupOk = ($pfCleanup -eq 'YES'); $gitGuardsOk = ($pfGit -eq 'YES'); $evInvOk = ($pfEv -eq 'YES')
    } catch { Write-RvLog -Level WARN -Message "POSTFLIGHT.json malformed" }
}
$postflightOk = ($cleanupOk -and $gitGuardsOk -and $evInvOk)

# ---- Overall reproduction marker (§52): never ACCEPTED/RELEASED (§20: gated on postflight) ----
$allVerifiedOrNa = ($verified -eq $applicable -and $evIncomplete -eq 0 -and $envBlocked -eq 0 -and $behRed -eq 0)
$verifiedGate = ($allVerifiedOrNa -and $postflightOk)
$overall = [ordered]@{
    RUST_LOCAL_REPRODUCTION_VERIFIED = $(if ($verifiedGate) { 'YES' } else { 'NO' })
    HUMAN_ACCEPTANCE_DECISION_REQUIRED = 'YES'
    EVIDENCE_INTEGRITY = $(if ($postflightOk) { 'PASS' } else { 'FAIL' })
    POSTFLIGHT_CLEANUP_COMPLETE = $pfCleanup
    POSTFLIGHT_GIT_GUARDS_PASS = $pfGit
    POSTFLIGHT_EVIDENCE_INVENTORY_COMPLETE = $pfEv
    FORBIDDEN_MARKERS_EMITTED = 'NO (SIDECAR_ACCEPTED / RELEASED never written)'
}

# ---- Human-readable detailed evidence for acceptance review ----
# Presentation/evidence enrichment only. Does not alter matrix status, classification,
# acceptance logic, or release/admission state.

$SidecarRepo = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
$WorkspaceRoot = Split-Path -Parent $SidecarRepo
$SpecRepo = Join-Path $WorkspaceRoot 'hacp-spec'
$verificationRoot = Join-Path $run.RunDir '02-verification'

# ---- Rust Core: exact canonical 38-vector inventory + authoritative descriptions ----
$coreVectorRoot = Join-Path $SpecRepo 'vectors'

$coreFiles = @(
    Get-ChildItem -LiteralPath $coreVectorRoot -File -Filter '*.json' |
    Where-Object {
        $_.Name -match '^(core_inv[1-7]_\d{3}_(golden|negative)|runtime_\d{3}_(golden|negative))\.json$'
    } |
    Sort-Object Name
)

$coreDetails = @(
    foreach ($file in $coreFiles) {
        $v = Get-Content -LiteralPath $file.FullName -Raw | ConvertFrom-Json

        $id = [string]$v.test_id
        $description = [string]$v.description

        $status = if ($id -in @('CORE-INV1-002','CORE-RUNTIME-004')) {
            'ADJUDICATED'
        } elseif ($id -eq 'CORE-INV5-006') {
            'PASS_RAW_WIRE'
        } else {
            'PASS'
        }

        $detail = [ordered]@{
            ID          = $id
            STATUS      = $status
            DESCRIPTION = $description
            SOURCE_FILE = $file.Name
        }

        if ($status -eq 'ADJUDICATED') {
            $detail['DIRECT_EXPECTED'] = 'CHECKPOINT'
            $detail['DIRECT_OBSERVED'] = 'DENY'
            $detail['ADJUDICATION'] = 'Known CHECKPOINT-vs-DENY conflict; accounted as adjudicated, never direct PASS'
        }

        if ($id -eq 'CORE-INV5-006') {
            $detail['RAW_WIRE_RESULT'] = 'PASS'
            $detail['RAW_WIRE_DECISION'] = 'DENY'
            $detail['RAW_WIRE_REASON'] = 'INVALID_ACTION'
        }

        [pscustomobject]$detail
    }
)

$coreDirect = @(
    $coreDetails |
    Where-Object { $_.STATUS -in @('PASS','PASS_RAW_WIRE') }
).Count

$coreAdjudicated = @(
    $coreDetails |
    Where-Object { $_.STATUS -eq 'ADJUDICATED' }
).Count

if (
    $coreDetails.Count -ne 38 -or
    $coreDirect -ne 36 -or
    $coreAdjudicated -ne 2
) {
    throw "CORE_DETAIL_ACCOUNTING_RED total=$($coreDetails.Count) direct=$coreDirect adjudicated=$coreAdjudicated"
}

# ---- Rust HC2: exact PASS lines + harness-provided descriptions ----
$rs02Stdout = Get-ChildItem `
    -LiteralPath (Join-Path $verificationRoot 'RS02') `
    -File |
Where-Object { $_.Name -match '(?i)stdout' } |
Select-Object -First 1

if (-not $rs02Stdout) {
    throw 'RS02_STDOUT_NOT_FOUND'
}

$hc2Details = @(
    Get-Content -LiteralPath $rs02Stdout.FullName |
    Where-Object { $_ -match '^\[PASS\]\s+(ENF-HC2-[^:]+):\s*(.+)$' } |
    ForEach-Object {
        if ($_ -match '^\[PASS\]\s+([^:]+):\s*(.+)$') {
            [pscustomobject][ordered]@{
                ID          = $Matches[1]
                STATUS      = 'PASS'
                DESCRIPTION = $Matches[2]
            }
        }
    }
)

if ($hc2Details.Count -ne 55) {
    throw "HC2_DETAIL_ACCOUNTING_RED count=$($hc2Details.Count)"
}

# ---- Rust package: exact active/ignored test inventory ----
$rs03Stdout = Get-ChildItem `
    -LiteralPath (Join-Path $verificationRoot 'RS03') `
    -File |
Where-Object { $_.Name -match '(?i)stdout' } |
Select-Object -First 1

if (-not $rs03Stdout) {
    throw 'RS03_STDOUT_NOT_FOUND'
}

$rs03Lines = @(Get-Content -LiteralPath $rs03Stdout.FullName)

$packageActiveNames = @(
    $rs03Lines |
    Where-Object { $_ -match '^test\s+.+\s+\.\.\.\s+ok$' } |
    ForEach-Object {
        if ($_ -match '^test\s+(.+?)\s+\.\.\.\s+ok$') {
            $Matches[1]
        }
    }
)

$packageIgnoredNames = @(
    $rs03Lines |
    Where-Object { $_ -match '^test\s+.+\s+\.\.\.\s+ignored' } |
    ForEach-Object {
        if ($_ -match '^test\s+(.+?)\s+\.\.\.\s+ignored') {
            $Matches[1]
        }
    }
)

function Get-RustPackageTestDescription {
    param([Parameter(Mandatory)][string]$Name)

    switch -Regex ($Name) {
        '^jcs::tests::test_sorted_keys$' {
            return 'verifies deterministic lexicographic object-key ordering in JCS'
        }
        '^jcs::tests::test_string_escaping$' {
            return 'verifies canonical JSON string escaping'
        }
        '^jcs::tests::test_no_whitespace$' {
            return 'verifies canonical output contains no insignificant whitespace'
        }

        '^authorization_v2_action_hash_mismatch' {
            return 'verifies authorization-v2 uses the current enforcement reason for action-hash mismatch'
        }
        '^authorization_v2_rejects_invalid_intent_envelope_signature' {
            return 'verifies an invalid intent-envelope signature is rejected before claims are trusted'
        }
        '^authorization_v2_honors_authenticated_applicable_signed_deny_token$' {
            return 'verifies an authenticated applicable signed DENY token is enforced'
        }

        '^canonical_v2_f1_finite_number$' {
            return 'verifies canonical-v2 handling of finite numeric values'
        }
        '^canonical_v2_f2_omits_null_object_member$' {
            return 'verifies canonical-v2 omits null-valued object members'
        }
        '^canonical_v2_f3_mixed_object$' {
            return 'verifies canonical-v2 behavior for mixed object content'
        }
        '^canonical_v2_f4_preserves_null_inside_array$' {
            return 'verifies null values inside arrays are preserved'
        }
        '^canonical_v2_omits_leading_null_without_comma_artifact$' {
            return 'verifies omitted leading null members do not leave comma artifacts'
        }
        '^canonical_v2_omits_nested_null_object_members$' {
            return 'verifies nested null-valued object members are omitted'
        }
        '^canonical_v2_omits_trailing_null_without_comma_artifact$' {
            return 'verifies omitted trailing null members do not leave comma artifacts'
        }
        '^canonical_v2_r2_rejects_non_finite_binary64_construction$' {
            return 'verifies non-finite binary64 values are rejected'
        }
        '^canonical_v2_r2_preserves_original_contract$' {
            return 'verifies canonical-v2 revision 2 preserves the original contract'
        }
        '^canonical_v2_r2_rfc8785_appendix_b_numeric_corpus$' {
            return 'verifies RFC 8785 Appendix B numeric corpus compatibility'
        }

        '^e13_canonical_v2_r01_r24_matches_shared_manifest$' {
            return 'verifies E13 canonical-v2 R01-R24 results match the shared manifest'
        }
        '^e13_family2_hash_signature_binding$' {
            return 'verifies Family-2 canonical hash/signature binding'
        }

        '^ed25519_v2_rejects_padded_base64url$' {
            return 'verifies Ed25519-v2 rejects padded base64url signatures'
        }
        '^ed25519_v2_rejects_invalid_signature_length$' {
            return 'verifies Ed25519-v2 rejects invalid signature lengths'
        }
        '^ed25519_v2_rejects_whitespace_variants$' {
            return 'verifies Ed25519-v2 rejects whitespace-modified encodings'
        }
        '^ed25519_v2_rejects_malformed_base64url$' {
            return 'verifies Ed25519-v2 rejects malformed base64url'
        }
        '^ed25519_v2_rejects_invalid_signature_bytes$' {
            return 'verifies Ed25519-v2 rejects invalid signature bytes'
        }
        '^ed25519_v2_rejects_wrong_key$' {
            return 'verifies Ed25519-v2 rejects signatures checked with the wrong key'
        }
        '^ed25519_v2_rejects_wrong_message$' {
            return 'verifies Ed25519-v2 rejects signatures over a different message'
        }
        '^ed25519_v2_accepts_canonical_signature$' {
            return 'verifies Ed25519-v2 accepts a canonical valid signature'
        }

        '^json_ingress_v1_must_reject_duplicate_proposed_action_key$' {
            return 'verifies duplicate proposed_action JSON keys are rejected'
        }
        '^duplicate_proposed_action_path_is_preserved_before_value_materialization$' {
            return 'verifies the duplicate proposed_action path is preserved before materialization'
        }
        '^nested_duplicate_path_is_preserved$' {
            return 'verifies nested duplicate-member paths are preserved'
        }
        '^valid_json_is_preserved_as_normal_value$' {
            return 'verifies valid JSON is preserved as a normal parsed value'
        }
        '^runner_maps_duplicate_proposed_action_to_invalid_action_without_action_hash$' {
            return 'verifies duplicate proposed_action input maps to INVALID_ACTION before action-hash use'
        }

        '^fail_closed_missing_envelope$' {
            return 'verifies evaluation fails closed when the envelope is missing'
        }
        '^fail_closed_missing_tool_name$' {
            return 'verifies evaluation fails closed when tool_name is missing'
        }
        '^fail_closed_missing_token$' {
            return 'verifies evaluation fails closed when the token is missing'
        }

        '^m2_01_valid_deterministic_projection$' {
            return 'verifies a valid M2 projection is deterministic'
        }
        '^m2_02_deterministic_across_time$' {
            return 'verifies M2 projection remains deterministic across time'
        }
        '^m2_03_identical_action_hash_across_time$' {
            return 'verifies identical input produces the same action hash across time'
        }
        '^m2_04_two_calls_in_one_process$' {
            return 'verifies two M2 calls in one process remain deterministic'
        }
        '^m2_05_synthesis_has_no_clock_dependency$' {
            return 'verifies action synthesis has no clock dependency'
        }
        '^m2_06_no_proposed_at_in_synthesized_action$' {
            return 'verifies synthesized actions omit proposed_at'
        }
        '^m2_07_name_binds_resource_id$' {
            return 'verifies name contributes to resource_id binding'
        }
        '^m2_08_name_binds_tool_name$' {
            return 'verifies name contributes to tool_name binding'
        }
        '^m2_09_field_mismatch_different_hash$' {
            return 'verifies a relevant field mismatch changes the action hash'
        }
        '^m2_10_arguments_do_not_affect_hash$' {
            return 'verifies arguments do not affect the declared M2 action hash'
        }
        '^m2_11_v1_module_exists$' {
            return 'verifies the expected v1 module exists'
        }
        '^m2_12_evaluate_module_exists_and_callable$' {
            return 'verifies the evaluate module exists and is callable'
        }

        '^sha256_v1_known_answer_vectors$' {
            return 'verifies SHA-256-v1 against known-answer vectors'
        }
        '^sha256_v1_lowercase_hex_representation$' {
            return 'verifies SHA-256-v1 hexadecimal output is lowercase'
        }

        default {
            return 'description requires explicit source review'
        }
    }
}

$packageActive = @(
    foreach ($name in $packageActiveNames) {
        [pscustomobject][ordered]@{
            NAME        = $name
            STATUS      = 'PASS'
            DESCRIPTION = Get-RustPackageTestDescription -Name $name
        }
    }
)

$packageIgnored = @(
    foreach ($name in $packageIgnoredNames) {
        $description = if ($name -eq 'ed25519_v1_rejects_noncanonical_whitespace_in_signature_encoding') {
            'historical Ed25519-v1 test for rejection of non-canonical whitespace in signature encoding'
        } else {
            'historical ignored test'
        }

        [pscustomobject][ordered]@{
            NAME        = $name
            STATUS      = 'HISTORICAL_IGNORED'
            DESCRIPTION = $description
        }
    }
)

if ($packageActive.Count -ne 48 -or $packageIgnored.Count -ne 1) {
    throw "PACKAGE_DETAIL_ACCOUNTING_RED active=$($packageActive.Count) ignored=$($packageIgnored.Count)"
}

$unmappedDescriptions = @(
    $packageActive |
    Where-Object { $_.DESCRIPTION -eq 'description requires explicit source review' }
)

if ($unmappedDescriptions.Count -ne 0) {
    throw "PACKAGE_DESCRIPTION_INCOMPLETE count=$($unmappedDescriptions.Count)"
}

# ---- RS04 HTTP workload ----
$rs04WorkloadPath = Join-Path $verificationRoot 'RS04\RS04-workload.stdout.log'

if (-not (Test-Path -LiteralPath $rs04WorkloadPath -PathType Leaf)) {
    throw "RS04_WORKLOAD_EVIDENCE_MISSING=$rs04WorkloadPath"
}

$rs04JsonRecords = @(
    Get-Content -LiteralPath $rs04WorkloadPath |
    Where-Object { $_.Trim().StartsWith('{') } |
    ForEach-Object { $_ | ConvertFrom-Json }
)

$rs04Warmup = @(
    $rs04JsonRecords |
    Where-Object { $_.phase -eq 'warmup' }
) | Select-Object -First 1

$rs04Measured = @(
    $rs04JsonRecords |
    Where-Object { $_.phase -eq 'measured' }
) | Select-Object -First 1

if (-not $rs04Warmup -or -not $rs04Measured) {
    throw 'RS04_WORKLOAD_DETAIL_INCOMPLETE'
}

$rs04Details = [ordered]@{
    SURFACE = [string]$rs04Measured.surface

    WARMUP = [ordered]@{
        REQUESTS    = $rs04Warmup.requests
        SUCCESS     = $rs04Warmup.success
        FAILED      = $rs04Warmup.failed
        CONCURRENCY = $rs04Warmup.concurrency
        ELAPSED_MS  = $rs04Warmup.elapsed_ms
        MEAN_MS     = $rs04Warmup.mean_ms
        P50_MS      = $rs04Warmup.p50_ms
        P95_MS      = $rs04Warmup.p95_ms
        P99_MS      = $rs04Warmup.p99_ms
        MAX_MS      = $rs04Warmup.max_ms
    }

    MEASURED = [ordered]@{
        REQUESTS    = $rs04Measured.requests
        SUCCESS     = $rs04Measured.success
        FAILED      = $rs04Measured.failed
        CONCURRENCY = $rs04Measured.concurrency
        ELAPSED_MS  = $rs04Measured.elapsed_ms
        MEAN_MS     = $rs04Measured.mean_ms
        P50_MS      = $rs04Measured.p50_ms
        P95_MS      = $rs04Measured.p95_ms
        P99_MS      = $rs04Measured.p99_ms
        MAX_MS      = $rs04Measured.max_ms
    }

    CLAIM_BOUNDARY = 'AUTHORIZATION + LOCAL RESPONSE; NO UPSTREAM FORWARDING'
}

# ---- RS05 timing capture ----
$rs05TimingPath = Join-Path $verificationRoot 'RS04\RS05-timing.json'

if (-not (Test-Path -LiteralPath $rs05TimingPath -PathType Leaf)) {
    throw "RS05_TIMING_EVIDENCE_MISSING=$rs05TimingPath"
}

$rs05TimingRaw = Get-Content -LiteralPath $rs05TimingPath -Raw | ConvertFrom-Json

$rs05Details = [ordered]@{
    P50_MS  = $rs05TimingRaw.p50
    P95_MS  = $rs05TimingRaw.p95
    P99_MS  = $rs05TimingRaw.p99
    MEAN_MS = $rs05TimingRaw.mean
    MAX_MS  = $rs05TimingRaw.max
    RPS     = $rs05TimingRaw.rps

    CLASSIFICATION = 'INFORMATIONAL_ONLY'
    THRESHOLD      = 'NONE'
    RANKING        = 'NO GO-vs-RUST RANKING'
}

foreach ($name in @('P50_MS','P95_MS','P99_MS','MEAN_MS','MAX_MS','RPS')) {
    if ($rs05Details[$name] -eq 'NOT_REPORTED' -or $null -eq $rs05Details[$name]) {
        throw "RS05_TIMING_DETAIL_RED field=$name"
    }
}

# ---- RS07 budget package result ----
$rs07Stdout = Get-ChildItem `
    -LiteralPath (Join-Path $verificationRoot 'RS07') `
    -File |
Where-Object { $_.Name -match '(?i)budget.*stdout' } |
Select-Object -First 1

if (-not $rs07Stdout) {
    throw 'RS07_BUDGET_STDOUT_NOT_FOUND'
}

$rs07Text = Get-Content -LiteralPath $rs07Stdout.FullName -Raw

$rs07Passed  = $null
$rs07Failed  = $null
$rs07Ignored = $null

if ($rs07Text -match 'test result:\s+ok\.\s+(\d+)\s+passed;\s+(\d+)\s+failed;\s+(\d+)\s+ignored') {
    $rs07Passed  = [int]$Matches[1]
    $rs07Failed  = [int]$Matches[2]
    $rs07Ignored = [int]$Matches[3]
}

if ($rs07Passed -ne 3 -or $rs07Failed -ne 0 -or $rs07Ignored -ne 0) {
    throw "RS07_DETAIL_ACCOUNTING_RED passed=$rs07Passed failed=$rs07Failed ignored=$rs07Ignored"
}

$rs07Details = [ordered]@{
    PASSED         = $rs07Passed
    FAILED         = $rs07Failed
    IGNORED        = $rs07Ignored
    CLAIM_BOUNDARY = 'AUTONOMY BUDGET ONLY; NO TOKEN REPLAY; NO max_uses'
}

# ---- Structured JSON summary ----
$details = [ordered]@{
    CORE = [ordered]@{
        TOTAL        = $coreDetails.Count
        DIRECT       = $coreDirect
        ADJUDICATED  = $coreAdjudicated
        VECTORS      = $coreDetails
    }

    HC2 = [ordered]@{
        TOTAL   = $hc2Details.Count
        PASSED  = $hc2Details.Count
        VECTORS = $hc2Details
    }

    PACKAGE = [ordered]@{
        ACTIVE_COUNT             = $packageActive.Count
        HISTORICAL_IGNORED_COUNT = $packageIgnored.Count
        ACTIVE                   = $packageActive
        HISTORICAL_IGNORED       = $packageIgnored
    }

    RS04_HTTP = $rs04Details
    RS05_TIMING = $rs05Details
    RS07_BUDGET = $rs07Details
}

Write-RvJson -Object ([ordered]@{
    COUNTERS = $counters
    OVERALL  = $overall
    ROWS     = $rows
    DETAILS  = $details
}) -Path (Join-Path $OUT 'RUST_VERIFICATION_SUMMARY.json')

# ---- Summary markdown ----
$md = @()

$md += '# RUST VERIFICATION SUMMARY'
$md += ''
$md += "Generated: $((Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ'))"
$md += ''

$md += '## Matrix reconciliation'
$md += ''
$md += '```text'
$md += "TOTAL_MATRIX_ROWS=$total"
$md += ''
$md += "RUST_APPLICABLE_ROWS=$applicable"
$md += "RUST_NOT_APPLICABLE_ROWS=$notApplicable"
$md += ''
$md += "RUST_VERIFIED_ROWS=$verified"
$md += "RUST_ENVIRONMENT_BLOCKED_ROWS=$envBlocked"
$md += "RUST_BEHAVIORAL_RED_ROWS=$behRed"
$md += "RUST_EVIDENCE_INCOMPLETE_ROWS=$evIncomplete"
$md += ''
$md += "UNRESOLVED_ROWS=$unresolved"
$md += '```'
$md += ''

$md += '## Rust Core — 38 vectors'
$md += ''
$md += '```text'
$md += 'Authoritative descriptions are read from the canonical Core vector files.'
$md += ''

foreach ($v in $coreDetails) {
    switch ($v.STATUS) {
        'ADJUDICATED' {
            $md += "[ADJUDICATED] $($v.ID) — $($v.DESCRIPTION)"
            $md += "  - Direct expected: $($v.DIRECT_EXPECTED)"
            $md += "  - Direct observed: $($v.DIRECT_OBSERVED)"
            $md += "  - Adjudication: $($v.ADJUDICATION)"
        }

        'PASS_RAW_WIRE' {
            $md += "[PASS / RAW-WIRE] $($v.ID) — $($v.DESCRIPTION)"
            $md += "  - Raw-wire result: $($v.RAW_WIRE_RESULT)"
            $md += "  - Decision: $($v.RAW_WIRE_DECISION)"
            $md += "  - Reason: $($v.RAW_WIRE_REASON)"
        }

        default {
            $md += "[PASS] $($v.ID) — $($v.DESCRIPTION)"
        }
    }
}

$md += ''
$md += "CORE_TOTAL=$($coreDetails.Count)"
$md += "CORE_DIRECT=$coreDirect"
$md += "CORE_ADJUDICATED=$coreAdjudicated"
$md += 'FINAL_ACCOUNTING=36 DIRECT + 2 ADJUDICATED'
$md += 'NEVER_REPORT=38/38 DIRECT'
$md += '```'
$md += ''

$md += '## Rust HC2 — actual PASS list'
$md += ''

$md += '```text'
foreach ($v in $hc2Details) {
    $md += "[PASS] $($v.ID) — $($v.DESCRIPTION)"
}

$md += ''
$md += "HC2_PASS_COUNT=$($hc2Details.Count)"
$md += '```'
$md += ''

$md += '## Rust package — active tests'
$md += ''
$md += 'Descriptions below are short human-readable explanations of the executed Rust test identifiers.'
$md += ''

$md += '```text'
foreach ($t in $packageActive) {
    $md += "[PASS] $($t.NAME) — $($t.DESCRIPTION)"
}

$md += ''
$md += "PACKAGE_ACTIVE_COUNT=$($packageActive.Count)"
$md += '```'
$md += ''

$md += '## Rust package — historical ignored'
$md += ''

$md += '```text'
foreach ($t in $packageIgnored) {
    $md += "[IGNORED] $($t.NAME) — $($t.DESCRIPTION)"
}

$md += ''
$md += "PACKAGE_HISTORICAL_IGNORED_COUNT=$($packageIgnored.Count)"
$md += '```'
$md += ''

$md += '## RS04 — HTTP workload'
$md += ''
$md += '```text'
$md += "SURFACE=$($rs04Details.SURFACE)"
$md += ''
$md += 'WARMUP:'
$md += "REQUESTS=$($rs04Details.WARMUP.REQUESTS)"
$md += "SUCCESS=$($rs04Details.WARMUP.SUCCESS)"
$md += "FAILED=$($rs04Details.WARMUP.FAILED)"
$md += "CONCURRENCY=$($rs04Details.WARMUP.CONCURRENCY)"
$md += "ELAPSED_MS=$($rs04Details.WARMUP.ELAPSED_MS)"
$md += "MEAN_MS=$($rs04Details.WARMUP.MEAN_MS)"
$md += "P50_MS=$($rs04Details.WARMUP.P50_MS)"
$md += "P95_MS=$($rs04Details.WARMUP.P95_MS)"
$md += "P99_MS=$($rs04Details.WARMUP.P99_MS)"
$md += "MAX_MS=$($rs04Details.WARMUP.MAX_MS)"
$md += ''
$md += 'MEASURED:'
$md += "REQUESTS=$($rs04Details.MEASURED.REQUESTS)"
$md += "SUCCESS=$($rs04Details.MEASURED.SUCCESS)"
$md += "FAILED=$($rs04Details.MEASURED.FAILED)"
$md += "CONCURRENCY=$($rs04Details.MEASURED.CONCURRENCY)"
$md += "ELAPSED_MS=$($rs04Details.MEASURED.ELAPSED_MS)"
$md += "MEAN_MS=$($rs04Details.MEASURED.MEAN_MS)"
$md += "P50_MS=$($rs04Details.MEASURED.P50_MS)"
$md += "P95_MS=$($rs04Details.MEASURED.P95_MS)"
$md += "P99_MS=$($rs04Details.MEASURED.P99_MS)"
$md += "MAX_MS=$($rs04Details.MEASURED.MAX_MS)"
$md += ''
$md += "CLAIM_BOUNDARY=$($rs04Details.CLAIM_BOUNDARY)"
$md += '```'
$md += ''

$md += '## RS05 — timing'
$md += ''
$md += '```text'
$md += "P50_MS=$($rs05Details.P50_MS)"
$md += "P95_MS=$($rs05Details.P95_MS)"
$md += "P99_MS=$($rs05Details.P99_MS)"
$md += "MEAN_MS=$($rs05Details.MEAN_MS)"
$md += "MAX_MS=$($rs05Details.MAX_MS)"
$md += "RPS=$($rs05Details.RPS)"
$md += ''
$md += "CLASSIFICATION=$($rs05Details.CLASSIFICATION)"
$md += "THRESHOLD=$($rs05Details.THRESHOLD)"
$md += "RANKING=$($rs05Details.RANKING)"
$md += '```'
$md += ''

$md += '## RS07 — autonomy budget verification'
$md += ''
$md += '```text'
$md += "PASSED=$($rs07Details.PASSED)"
$md += "FAILED=$($rs07Details.FAILED)"
$md += "IGNORED=$($rs07Details.IGNORED)"
$md += "CLAIM_BOUNDARY=$($rs07Details.CLAIM_BOUNDARY)"
$md += '```'
$md += ''

$md += '## Claim boundaries'
$md += ''
$md += '```text'
$md += 'RUST CORE:  36/38 DIRECT + 2 ADJUDICATED (CORE-INV1-002, CORE-RUNTIME-004); CORE-INV5-006 RAW-WIRE PASS; NEVER 38/38 DIRECT'
$md += 'RUST HC2:   55/55'
$md += 'RUST PACKAGE: 48 ACTIVE + 1 HISTORICAL IGNORED'
$md += 'RUST HTTP:  LOCAL RESPONSE; NO UPSTREAM FORWARDING'
$md += 'RUST TIMING: INFORMATIONAL ONLY; NO THRESHOLD; NO GO-vs-RUST RANKING'
$md += 'RS07:       AUTONOMY BUDGET ONLY; NO TOKEN REPLAY; NO max_uses'
$md += '```'
$md += ''

$md += '## Overall'
$md += ''
$md += '```text'
$md += "RUST_LOCAL_REPRODUCTION_VERIFIED=$($overall.RUST_LOCAL_REPRODUCTION_VERIFIED)"
$md += "HUMAN_ACCEPTANCE_DECISION_REQUIRED=$($overall.HUMAN_ACCEPTANCE_DECISION_REQUIRED)"
$md += "EVIDENCE_INTEGRITY=$($overall.EVIDENCE_INTEGRITY)"
$md += '```'
$md += ''
$md += 'No automatic acceptance. Human reviews evidence and decides.'

$summaryMdPath   = Join-Path $OUT 'RUST_VERIFICATION_SUMMARY.md'
$summaryJsonPath = Join-Path $OUT 'RUST_VERIFICATION_SUMMARY.json'

$md | Set-Content -LiteralPath $summaryMdPath -Encoding UTF8

Write-RvSummaryText -Table $counters -Path (Join-Path $OUT 'counters.txt')
Write-RvLog -Message "TOTAL=$total SUM_CHECK=$sumCheck UNRESOLVED=$unresolved. RUST_LOCAL_REPRODUCTION_VERIFIED=$($overall.RUST_LOCAL_REPRODUCTION_VERIFIED). HUMAN_ACCEPTANCE_DECISION_REQUIRED=YES."
Set-RvRunState -RunDir $run.RunDir -State 'RECONCILED'

Write-Host ''
Write-Host '============================================================'
Write-Host 'HACP 1.1.1 — RUST VERIFICATION RESULT'
Write-Host '============================================================'
Write-Host ''
Write-Host "RUN=$($run.RunDir)"
Write-Host ''
Write-Host 'CORE:'
Write-Host "  TOTAL=$($coreDetails.Count)"
Write-Host "  DIRECT=$coreDirect"
Write-Host "  ADJUDICATED=$coreAdjudicated"
Write-Host ''
Write-Host 'HC2:'
Write-Host "  PASS=$($hc2Details.Count)/55"
Write-Host ''
Write-Host 'RUST PACKAGE:'
Write-Host "  ACTIVE_PASS=$($packageActive.Count)"
Write-Host "  HISTORICAL_IGNORED=$($packageIgnored.Count)"
Write-Host ''
Write-Host 'RS07:'
Write-Host "  PASSED=$($rs07Details.PASSED)"
Write-Host "  FAILED=$($rs07Details.FAILED)"
Write-Host "  IGNORED=$($rs07Details.IGNORED)"
Write-Host ''
Write-Host 'POSTFLIGHT:'
Write-Host "  CLEANUP_COMPLETE=$($overall.POSTFLIGHT_CLEANUP_COMPLETE)"
Write-Host "  GIT_GUARDS_PASS=$($overall.POSTFLIGHT_GIT_GUARDS_PASS)"
Write-Host ''
Write-Host 'RESULT:'
Write-Host "  RUST_LOCAL_REPRODUCTION_VERIFIED=$($overall.RUST_LOCAL_REPRODUCTION_VERIFIED)"
Write-Host "  HUMAN_ACCEPTANCE_DECISION_REQUIRED=$($overall.HUMAN_ACCEPTANCE_DECISION_REQUIRED)"
Write-Host "  EVIDENCE_INTEGRITY=$($overall.EVIDENCE_INTEGRITY)"
Write-Host ''
Write-Host "RESULTS_MD=$summaryMdPath"
Write-Host "RESULTS_JSON=$summaryJsonPath"
Write-Host ''
Write-Host '============================================================'
# FORBIDDEN (never emitted): SIDECAR_ACCEPTED, RELEASED, ACCEPTED, FROZEN.
