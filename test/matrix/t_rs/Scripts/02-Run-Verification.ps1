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
    [string]$SpecRepo    = 'C:\Personal\GitHub\Dev\hacp-spec',
    [string]$SidecarRepo = 'C:\Personal\GitHub\Dev\hacp-sidecar',
    [string]$EvidenceRoot= 'C:\Temp\hacp-rust-verification\evidence',
    [string]$RunRoot     = 'C:\Temp\hacp-rust-verification\run',
    [string]$RunDir      = '',
    [ValidateSet('RS01','RS02','RS03','RS04','RS05','RS07')][string]$Surface = '',
    [switch]$ContinueIndependentSurfaces
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

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
            # RS01 does not re-run the historical strict Core harness.
            #
            # The current admission model inherits the already-proven Rust Core verdict:
            #   36/38 direct + 2 adjudicated
            #   adjudicated: CORE-INV1-002, CORE-RUNTIME-004
            #
            # This surface verifies continuity of the exact proven Rust source identity
            # and independently re-runs the CORE-INV5-006 raw-wire duplicate-key witness.
            # It must never manufacture 36/38 from the historical 13/38 strict harness.

            $expectedPHead =
                '03885b2b062905072040dc245c3d25d283c249c4'

            $expectedPTree =
                'b8341f7c114a4c51ea45d04352ee7ebdcc462e9a'

            $expectedEvaluateSha256 =
                '74580DE3A9E345B64E2548EDD07DDB4A4D7058C2F847CF0EBB9D8468C23FE041'

            $evaluatePath = Join-Path $SpecRepo 'hacp-rs\src\evaluate.rs'

            $actualEvaluateSha256 = $null
            if (Test-Path -LiteralPath $evaluatePath) {
                $actualEvaluateSha256 =
                    (Get-FileHash -LiteralPath $evaluatePath -Algorithm SHA256).Hash
            }

            $identityPass =
                ($P_HEAD -eq $expectedPHead) -and
                ($P_TREE -eq $expectedPTree) -and
                ($actualEvaluateSha256 -eq $expectedEvaluateSha256)

            $identityEvidence = Join-Path $evidence 'RS01-proven-baseline-identity.json'

            Write-RvJson -Object @{
                MODEL = 'INHERITED_PROVEN_RUST_CORE_BASELINE'
                STRICT_HARNESS_RERUN = 'NO'
                PROVEN_CORE_VERDICT = '36/38 direct + 2 adjudicated'
                ADJUDICATED = @(
                    'CORE-INV1-002',
                    'CORE-RUNTIME-004'
                )
                EXPECTED_P_HEAD = $expectedPHead
                ACTUAL_P_HEAD = $P_HEAD
                EXPECTED_P_TREE = $expectedPTree
                ACTUAL_P_TREE = $P_TREE
                EVALUATE_PATH = $evaluatePath
                EXPECTED_EVALUATE_SHA256 = $expectedEvaluateSha256
                ACTUAL_EVALUATE_SHA256 = $actualEvaluateSha256
                IDENTITY_MATCH = $identityPass
            } -Path $identityEvidence

            $witness = Invoke-RvNative `
                -FilePath 'cargo' `
                -ArgumentList @(
                    '+stable-x86_64-pc-windows-msvc',
                    'test',
                    '--locked',
                    '--test',
                    'json_ingress_v1_red',
                    '--',
                    '--nocapture'
                ) `
                -WorkDir (Join-Path $SpecRepo 'hacp-rs') `
                -StepId 'RS01-witness' `
                -SurfaceId 'RS01' `
                -EvidenceDir $evidence `
                -ExtraEnv $cargoEnv

            $exitCodes['RS01_WITNESS_RC'] = $witness.ProcessRc

            $witnessOut = Get-RvStdout -Run $witness

            $witnessPass =
                ($witness.ProcessRc -eq 0) -and
                ($witnessOut -match 'json_ingress_v1_must_reject_duplicate_proposed_action_key') -and
                ($witnessOut -match '1 passed') -and
                ($witnessOut -match '0 failed')

            if (-not $identityPass) {
                $cls = $RV_CLASS.ENVIRONMENT_RED
                $envRed = $true
                $observed =
                    'Proven Rust Core baseline identity mismatch; no behavioral verdict attempted'
            }
            elseif (-not $witnessPass) {
                if ($witness.ProcessRc -eq $null) {
                    $cls = $RV_CLASS.ENVIRONMENT_RED
                    $envRed = $true
                    $observed =
                        'CORE-INV5-006 raw-wire witness could not be executed'
                }
                else {
                    $cls = $RV_CLASS.BEHAVIORAL_RED
                    $behRed = $true
                    $observed =
                        'Proven Rust Core baseline identity matched, but CORE-INV5-006 raw-wire witness regressed'
                }
            }
            else {
                $cls = $RV_CLASS.PASS
                $observed =
                    'Proven Rust Core baseline identity matched: 36/38 direct + 2 adjudicated inherited; CORE-INV5-006 raw-wire witness PASS; historical strict harness not re-run'
            }

            Write-SurfaceResult `
                -Surface 'RS01' `
                -Rows @('A4') `
                -ExitCodes $exitCodes `
                -Classification $cls `
                -Expected 'Exact proven Rust Core source identity; inherited verdict 36/38 direct + 2 adjudicated (CORE-INV1-002, CORE-RUNTIME-004); CORE-INV5-006 raw-wire witness PASS; no historical strict harness rerun' `
                -Observed $observed `
                -Evidence @(
                    $identityEvidence,
                    $witness.StdOutPath
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
                '0E3D8A8B8A380829ACCC62ED8948EBA97019802FE098644F8E1FA9A887F4C158'

            $goldenExpectedSha =
                'C9E84D6BACB0E52E95FC2B05A3636BA2409764589E735D8286E354E49265FB90'

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
                Get-RvFileIdentity -Path $rs07Source

            $goldenSourceIdentity =
                Get-RvFileIdentity -Path $goldenSource

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
                Get-RvFileIdentity -Path $rs07OverlayTest

            $goldenOverlayIdentity =
                Get-RvFileIdentity -Path $goldenOverlay

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
