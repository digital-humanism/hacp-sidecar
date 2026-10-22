#Requires -Version 7.0

<#
02-Run-Verification.ps1 — MAIN EXECUTION ORCHESTRATOR (PHASE 1-6).
CLASS = VERIFICATION TOOLING. Executes the 16 surfaces strictly sequentially in the
approved manifest order, keeping a separate verdict per surface. NO admission, NO
acceptance automation, NO synthetic combined scores. By default STOPS on material
IDENTITY_RED / BEHAVIORAL_RED / ENVIRONMENT_RED; independent surfaces continue only
with -ContinueIndependentSurfaces. No automatic retry of BEHAVIORAL_RED.
#>

[CmdletBinding()]
param(
    [string]$SidecarRoot = 'C:\Personal\GitHub\Dev\hacp-sidecar',
    [string]$SpecRoot    = 'C:\Personal\GitHub\Dev\hacp-spec',
    [string]$CoreRoot    = 'C:\Personal\GitHub\Dev\humanist-core',
    [string]$RunRoot     = 'C:\Temp\hacp-1.1.1-verification\run',
    [string]$EvidenceRoot= 'C:\Temp\hacp-1.1.1-verification\evidence',
    [string]$Surface     = '',          # optional single-surface resume
    [switch]$ContinueIndependentSurfaces
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Import-Module (Join-Path $PSScriptRoot 'WindowsVerification.Common.psm1') -Force

$BUILD_ROOT   = Join-Path $RunRoot 'build'
$FIXTURE_ROOT = Join-Path $RunRoot 'fixtures'
$LOG_ROOT     = New-WvDirectory -Path (Join-Path $RunRoot 'logs')
$EVID_ROOT    = New-WvDirectory -Path $EvidenceRoot
Set-WvLogFile -Path (Join-Path $LOG_ROOT '02-run-console.log')
Write-WvLog -Message "02-Run-Verification start"

# Approved execution order (manifest §execution order).
$ALL_SURFACES = @('A','S11','S12','B','S01','S02','S03','S05','C','S04','S09','S06','S07','D','S08','S10')
if ($Surface) { $SURFACE_QUEUE = @($Surface) } else { $SURFACE_QUEUE = $ALL_SURFACES }

$MATRIX_ROWS = @{
    A='B1,B2,B3,B5,B7,B8,B10,B11,B12'; S11='B6'; S12='B9'; B='B10(H03)'
    S01='A1,A5'; S02='A2,A6'; S03='A3,A7'; S05='B4'; C='A4'; S04='A8'; S09='B18'
    S06='B13'; S07='B14'; D='B15'; S08='B16,B19-subset'; S10='B19'
}

$identityBefore = [ordered]@{
    S = Get-WvGitIdentity -RepoRoot $SidecarRoot
    P = Get-WvGitIdentity -RepoRoot $SpecRoot
    C = Get-WvGitIdentity -RepoRoot $CoreRoot
}

function New-WvService {
    param([string]$Name,[string]$FilePath,[string[]]$ArgumentList,[string]$WorkDir,[int]$Port,[string]$SurfaceOwner)
    $st = Test-WvPortState -Port $Port
    if ($st.State -ne 'FREE') {
        return [pscustomobject]@{ Ok=$false; Pid=$null; Error="PORT_BUSY:$Port" }
    }
    $svcOut = Join-Path $LOG_ROOT "$SurfaceOwner-$Name.service.out.log"
    $svcErr = Join-Path $LOG_ROOT "$SurfaceOwner-$Name.service.err.log"
    $p = Start-Process -FilePath $FilePath -ArgumentList $ArgumentList -WorkingDirectory $WorkDir `
            -RedirectStandardOutput $svcOut -RedirectStandardError $svcErr -PassThru -WindowStyle Hidden
    Register-WvOwnedProcess -RunRoot $RunRoot -Pid $p.Id -Command "$FilePath $($ArgumentList -join ' ')" -SurfaceOwner $SurfaceOwner -Port $Port | Out-Null
    $deadline = (Get-Date).AddSeconds(30)
    while ((Get-Date) -lt $deadline) {
        if (Test-WvPortOwned -Port $Port -OwningPid $p.Id) { break }
        Start-Sleep -Milliseconds 500
    }
    $owned = Test-WvPortOwned -Port $Port -OwningPid $p.Id
    [pscustomobject]@{ Ok=$owned; Pid=$p.Id; Error=$(if ($owned) { $null } else { "PORT_NOT_OWNED:$Port" }) }
}

function Set-WvServiceEnvSidecar {
    param([int]$Port,[string]$Upstream)
    $env:HACP_SIDECAR_PORT = "$Port"
    if ($Upstream) { $env:HACP_UPSTREAM = $Upstream }
    $env:HACP_TRUST_MODE = 'test'
    Remove-Item Env:HACP_TRUST_KEYS_FILE -ErrorAction SilentlyContinue
    $env:HACP_CONTROL_MODE = 'standalone'
}

function Invoke-WvSurface {
    param([string]$Id)

    $evidenceDir = New-WvSurfaceEvidenceDir -EvidenceRoot $EVID_ROOT -SurfaceId $Id
    Write-WvLog -Level STEP -Message "=== SURFACE $Id (rows: $($MATRIX_ROWS[$Id])) ==="
    $steps = @()
    $verdict = $WV_CLASS.PASS
    $sidecarExe = Join-Path $BUILD_ROOT 'hacp-sidecar.exe'
    $genWork = Join-Path $SidecarRoot 'benchmarks\generate-tokens'

    switch ($Id) {
        'A' {
            $targets = @(
                @{ Sid='A-TestR43'; Args=@('test','.\internal\proxy','-run','TestR43','-count=1','-v') },
                @{ Sid='A-proxy';   Args=@('test','.\internal\proxy','-count=1','-v') },
                @{ Sid='A-full';    Args=@('test','./...','-count=1') }
            )
            foreach ($t in $targets) {
                $r = Invoke-WvNative -FilePath 'go' -ArgumentList $t.Args -WorkDir $SidecarRoot -StepId $t.Sid -SurfaceId 'A' -EvidenceDir $evidenceDir
                $cls = Resolve-WvStepClassification -Run $r -ForbiddenMarkers @('--- FAIL','panic:')
                $steps += New-WvStepResult -StepId $t.Sid -SurfaceId 'A' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir
                if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
            }
        }
        'S11' {
            $source = Join-Path $SidecarRoot 'test\matrix\t_go\S11_signature_canonicalization_test.go.overlay'
            $fi = Get-WvFileIdentity -Path $source

            $virtual = Join-Path $SidecarRoot 'internal\wire\s11_matrix_verification_test.go'
            $overlay = Join-Path $evidenceDir 'S11.overlay.json'

            $overlayDocument = [ordered]@{
                Replace = [ordered]@{
                    $virtual = $source
                }
            }

            $utf8NoBom = [System.Text.UTF8Encoding]::new($false)
            [System.IO.File]::WriteAllText(
                $overlay,
                ($overlayDocument | ConvertTo-Json -Depth 5),
                $utf8NoBom
            )

            $runRegex = '^(TestCanonicalizeJSONSortsTopLevelKeys|TestCanonicalizeJSONSortsNestedKeys|TestCanonicalizeJSONStripsWhitespace|TestCanonicalizeJSONPreservesArrayOrder|TestCanonicalizeJSONNumberLargeInteger|TestCanonicalizeJSONKeepsSolidusUnescaped|TestCanonicalizeJSONKeepsEscapedLineFeed|TestCanonicalizeJSONRejectsInvalidInput|TestVerifyCanonicalizationDetectsEquivalence|TestRemoveSignatureFieldStripsSignature|TestVerifyEd25519AcceptsValidSignature|TestVerifyEd25519RejectsTamperedMessage|TestVerifyEd25519RejectsCorruptedSignature|TestVerifyEd25519RejectsWrongKey|TestSignatureBindsCanonicalForm)$'

            $r = Invoke-WvNative `
                -FilePath 'go' `
                -ArgumentList @(
                    'test',
                    '-overlay', $overlay,
                    '.\internal\wire',
                    '-run', $runRegex,
                    '-count=1',
                    '-v'
                ) `
                -WorkDir $SidecarRoot `
                -StepId 'S11-wire' `
                -SurfaceId 'S11' `
                -EvidenceDir $evidenceDir

            $cls = Resolve-WvStepClassification `
                -Run $r `
                -ForbiddenMarkers @('--- FAIL','panic:')

            $steps += New-WvStepResult `
                -StepId 'S11-wire' `
                -SurfaceId 'S11' `
                -Command $r.Command `
                -StartTimeUtc $r.StartTimeUtc `
                -EndTimeUtc $r.EndTimeUtc `
                -ProcessRc $r.ProcessRc `
                -Classification $cls `
                -EvidencePath $evidenceDir `
                -Status "S11_SHA256=$($fi.Sha256)"

            if ($cls -ne $WV_CLASS.PASS) {
                $verdict = $cls
            }
        }

        'S12' {
            $budgetSource = Join-Path $SidecarRoot 'test\matrix\t_go\S12_budget_ledger_test.go.overlay'
            $proxySource = Join-Path $SidecarRoot 'test\matrix\t_go\S12_budget_replay_verification_test.go.overlay'

            $budgetFi = Get-WvFileIdentity -Path $budgetSource
            $proxyFi = Get-WvFileIdentity -Path $proxySource

            $budgetVirtual = Join-Path $SidecarRoot 'internal\budget\s12_matrix_budget_ledger_test.go'
            $proxyVirtual = Join-Path $SidecarRoot 'internal\proxy\s12_matrix_budget_replay_verification_test.go'

            $budgetOverlay = Join-Path $evidenceDir 'S12-budget.overlay.json'
            $proxyOverlay = Join-Path $evidenceDir 'S12-proxy.overlay.json'

            $utf8NoBom = [System.Text.UTF8Encoding]::new($false)

            $budgetOverlayDocument = [ordered]@{
                Replace = [ordered]@{
                    $budgetVirtual = $budgetSource
                }
            }

            [System.IO.File]::WriteAllText(
                $budgetOverlay,
                ($budgetOverlayDocument | ConvertTo-Json -Depth 5),
                $utf8NoBom
            )

            $proxyOverlayDocument = [ordered]@{
                Replace = [ordered]@{
                    $proxyVirtual = $proxySource
                }
            }

            [System.IO.File]::WriteAllText(
                $proxyOverlay,
                ($proxyOverlayDocument | ConvertTo-Json -Depth 5),
                $utf8NoBom
            )

            $targets = @(
                @{
                    Sid = 'S12-budget-unit'
                    Args = @(
                        'test',
                        '-overlay', $budgetOverlay,
                        '.\internal\budget',
                        '-run', '^TestConsumeToken',
                        '-count=1',
                        '-v'
                    )
                    Status = "S12_BUDGET_SHA256=$($budgetFi.Sha256)"
                },
                @{
                    Sid = 'S12-proxy-integration'
                    Args = @(
                        'test',
                        '-overlay', $proxyOverlay,
                        '.\internal\proxy',
                        '-run', '^(TestBudgetDefaultMaxUsesOneDeniesSecondUse|TestBudgetMaxUsesExhaustionDeniesWithBudgetExhausted|TestBudgetReplayRemainsDeniedAfterExhaustion)$',
                        '-count=1',
                        '-v'
                    )
                    Status = "S12_PROXY_SHA256=$($proxyFi.Sha256)"
                }
            )

            foreach ($t in $targets) {
                $r = Invoke-WvNative `
                    -FilePath 'go' `
                    -ArgumentList $t.Args `
                    -WorkDir $SidecarRoot `
                    -StepId $t.Sid `
                    -SurfaceId 'S12' `
                    -EvidenceDir $evidenceDir

                $cls = Resolve-WvStepClassification `
                    -Run $r `
                    -ForbiddenMarkers @('--- FAIL','panic:')

                $steps += New-WvStepResult `
                    -StepId $t.Sid `
                    -SurfaceId 'S12' `
                    -Command $r.Command `
                    -StartTimeUtc $r.StartTimeUtc `
                    -EndTimeUtc $r.EndTimeUtc `
                    -ProcessRc $r.ProcessRc `
                    -Classification $cls `
                    -EvidencePath $evidenceDir `
                    -Status $t.Status

                if ($cls -ne $WV_CLASS.PASS) {
                    $verdict = $cls
                }
            }
        }

        'B' {
            $fixture = Join-Path $FIXTURE_ROOT 'tokens-h03.jsonl'
            $out = Join-Path $evidenceDir 'B-stdout.log'; $err = Join-Path $evidenceDir 'B-stderr.log'; $prov = Join-Path $evidenceDir 'B-provenance.jsonl'
            $r = Invoke-WvNative -FilePath (Join-Path $BUILD_ROOT 'graceful-shutdown.exe') `
                -ArgumentList @($sidecarExe,$fixture,$out,$err,$prov) -WorkDir $SidecarRoot -StepId 'B-h03' -SurfaceId 'B' -EvidenceDir $evidenceDir
            # 502+ALLOW is the expected H03 topology — not a failure.
            $cls = Resolve-WvStepClassification -Run $r -RequiredMarkers @('R4_3_H03=PASS')
            $steps += New-WvStepResult -StepId 'B-h03' -SurfaceId 'B' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir -Status 'EXPECTED 502+ALLOW (H03 topology)'
            if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
        }
        'S01' {
            $harness = Join-Path $SpecRoot 'harness\harness.py'
            $core = Invoke-WvNative -FilePath 'python' -ArgumentList @($harness,'--mode','cli','--binary-path',(Join-Path $BUILD_ROOT 'hacp-go.exe'),'--vectors-dir',(Join-Path $SpecRoot 'vectors'),'--output','console') -WorkDir $SpecRoot -StepId 'S01-core' -SurfaceId 'S01' -EvidenceDir $evidenceDir
            $cc = Resolve-WvStepClassification -Run $core -RequiredMarkers @('38/38')
            $steps += New-WvStepResult -StepId 'S01-core' -SurfaceId 'S01' -Command $core.Command -StartTimeUtc $core.StartTimeUtc -EndTimeUtc $core.EndTimeUtc -ProcessRc $core.ProcessRc -Classification $cc -EvidencePath $evidenceDir
            if ($cc -ne $WV_CLASS.PASS) { $verdict = $cc }
            $hc2 = Invoke-WvNative -FilePath 'python' -ArgumentList @($harness,'--mode','cli','--binary-path',(Join-Path $BUILD_ROOT 'hacp-go.exe'),'--vectors-dir',(Join-Path $SpecRoot 'vectors\enforcement-v2'),'--output','console') -WorkDir $SpecRoot -StepId 'S01-hc2' -SurfaceId 'S01' -EvidenceDir $evidenceDir
            $hc = Resolve-WvStepClassification -Run $hc2 -RequiredMarkers @('55/55')
            $steps += New-WvStepResult -StepId 'S01-hc2' -SurfaceId 'S01' -Command $hc2.Command -StartTimeUtc $hc2.StartTimeUtc -EndTimeUtc $hc2.EndTimeUtc -ProcessRc $hc2.ProcessRc -Classification $hc -EvidencePath $evidenceDir
            if ($hc -ne $WV_CLASS.PASS) { $verdict = $hc }
        }
        'S02' {
            $harness = Join-Path $SpecRoot 'harness\harness.py'
            $cli = Join-Path $SpecRoot 'hacp-ts\dist\src\cli.js'
            $core = Invoke-WvNative -FilePath 'python' -ArgumentList @($harness,'--mode','cli','--binary-path',$cli,'--vectors-dir',(Join-Path $SpecRoot 'vectors'),'--output','console') -WorkDir $SpecRoot -StepId 'S02-core' -SurfaceId 'S02' -EvidenceDir $evidenceDir
            $cc = Resolve-WvStepClassification -Run $core -RequiredMarkers @('38/38')
            $steps += New-WvStepResult -StepId 'S02-core' -SurfaceId 'S02' -Command $core.Command -StartTimeUtc $core.StartTimeUtc -EndTimeUtc $core.EndTimeUtc -ProcessRc $core.ProcessRc -Classification $cc -EvidencePath $evidenceDir
            if ($cc -ne $WV_CLASS.PASS) { $verdict = $cc }
            $hc2 = Invoke-WvNative -FilePath 'python' -ArgumentList @($harness,'--mode','cli','--binary-path',$cli,'--vectors-dir',(Join-Path $SpecRoot 'vectors\enforcement-v2'),'--output','console') -WorkDir $SpecRoot -StepId 'S02-hc2' -SurfaceId 'S02' -EvidenceDir $evidenceDir
            $hc = Resolve-WvStepClassification -Run $hc2 -RequiredMarkers @('55/55')
            $steps += New-WvStepResult -StepId 'S02-hc2' -SurfaceId 'S02' -Command $hc2.Command -StartTimeUtc $hc2.StartTimeUtc -EndTimeUtc $hc2.EndTimeUtc -ProcessRc $hc2.ProcessRc -Classification $hc -EvidencePath $evidenceDir
            if ($hc -ne $WV_CLASS.PASS) { $verdict = $hc }
            $reg = Invoke-WvNative -FilePath 'npm.cmd' -ArgumentList @('run','test:conformance') -WorkDir (Join-Path $SpecRoot 'hacp-ts') -StepId 'S02-regression' -SurfaceId 'S02' -EvidenceDir $evidenceDir
            $rc = Resolve-WvStepClassification -Run $reg
            $steps += New-WvStepResult -StepId 'S02-regression' -SurfaceId 'S02' -Command $reg.Command -StartTimeUtc $reg.StartTimeUtc -EndTimeUtc $reg.EndTimeUtc -ProcessRc $reg.ProcessRc -Classification $rc -EvidencePath $evidenceDir -Status 'package regression (not Core-claim)'
            if ($rc -ne $WV_CLASS.PASS) { $verdict = $rc }
        }
        'S03' {
            $targets = @(
                @{ Sid='S03-core'; Args=@('-m','pytest','tests\conformance\test_core_vectors.py','-v','-rs') },
                @{ Sid='S03-hc2';  Args=@('-m','pytest','tests\conformance\test_enforcement_v2_vectors.py','-v','-rs') }
            )
            foreach ($t in $targets) {
                $r = Invoke-WvNative -FilePath 'python' -ArgumentList $t.Args -WorkDir $CoreRoot -StepId $t.Sid -SurfaceId 'S03' -EvidenceDir $evidenceDir
                $cls = Resolve-WvStepClassification -Run $r -ForbiddenMarkers @('failed')
                $steps += New-WvStepResult -StepId $t.Sid -SurfaceId 'S03' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir
                if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
            }
        }
        'S05' {
            $runner = Join-Path $BUILD_ROOT 'hacp-conformance-runner.exe'
            $r = Invoke-WvNative -FilePath 'python' -ArgumentList @((Join-Path $SpecRoot 'harness\enforcement_v2_runner.py'),'--runner',$runner,'--vectors-dir',(Join-Path $SpecRoot 'vectors\enforcement-v2'),'--implementation-name','hacp-sidecar','--implementation-version','0.5.0','--output','console','--verbose') -WorkDir (Join-Path $SpecRoot 'harness') -StepId 'S05-hc2' -SurfaceId 'S05' -EvidenceDir $evidenceDir
            $cls = Resolve-WvStepClassification -Run $r -RequiredMarkers @('55/55')
            $steps += New-WvStepResult -StepId 'S05-hc2' -SurfaceId 'S05' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir
            if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
        }
        'C' {
            $rsRunner = Join-Path $SpecRoot 'hacp-rs\target\release\hacp-rs-runner.exe'
            $r = Invoke-WvNative -FilePath 'python' -ArgumentList @((Join-Path $SpecRoot 'harness\harness_runner.py'),'--runner',$rsRunner,'--vectors-dir',(Join-Path $SpecRoot 'vectors'),'--manifest',(Join-Path $SpecRoot 'harness\conformance_manifest.json'),'--implementation-name','hacp-rs','--implementation-version','0.1.0-1.1.0','--output','console') -WorkDir (Join-Path $SpecRoot 'harness') -StepId 'C-core-strict' -SurfaceId 'C' -EvidenceDir $evidenceDir
            $cls = Resolve-WvStepClassification -Run $r -RequiredMarkers @('13/38') -ExpectedNonZero
            $steps += New-WvStepResult -StepId 'C-core-strict' -SurfaceId 'C' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir -Status 'Rust Core = 36/38 direct + 2 adjudicated (NOT 38/38 direct)'
            if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
            $w = Invoke-WvNative -FilePath 'cargo' -ArgumentList @('+stable-x86_64-pc-windows-msvc','test','--locked','--test','json_ingress_v1_red','--','--nocapture') -WorkDir (Join-Path $SpecRoot 'hacp-rs') -StepId 'C-witness' -SurfaceId 'C' -EvidenceDir $evidenceDir
            $wc = Resolve-WvStepClassification -Run $w -RequiredMarkers @('1 passed; 0 failed') -ForbiddenMarkers @('test result: FAILED','panic:')
            $steps += New-WvStepResult -StepId 'C-witness' -SurfaceId 'C' -Command $w.Command -StartTimeUtc $w.StartTimeUtc -EndTimeUtc $w.EndTimeUtc -ProcessRc $w.ProcessRc -Classification $wc -EvidencePath $evidenceDir
            if ($wc -ne $WV_CLASS.PASS) { $verdict = $wc }
        }
        'S04' {
            $rsRunner = Join-Path $SpecRoot 'hacp-rs\target\release\hacp-rs-runner.exe'
            $r = Invoke-WvNative -FilePath 'python' -ArgumentList @((Join-Path $SpecRoot 'harness\enforcement_v2_runner.py'),'--runner',$rsRunner,'--vectors-dir',(Join-Path $SpecRoot 'vectors\enforcement-v2'),'--implementation-name','hacp-rs','--output','console') -WorkDir (Join-Path $SpecRoot 'harness') -StepId 'S04-hc2' -SurfaceId 'S04' -EvidenceDir $evidenceDir
            $cls = Resolve-WvStepClassification -Run $r -RequiredMarkers @('55/55')
            $steps += New-WvStepResult -StepId 'S04-hc2' -SurfaceId 'S04' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir -Status 'Rust HC2 = 55/55 (separate family)'
            if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
        }
        'S09' {
            $e13EnvNames = @(
    'HACP_E13_CORPUS',
    'HACP_E13_FAMILY2_FIXTURE',
    'HACP_E13_FAMILY2_PUBLIC_KEY'
)
$savedE13Env = Save-WvEnvironment -Names $e13EnvNames
try {
    $env:HACP_E13_CORPUS = Join-Path $SpecRoot 'vectors\family1\E13_R01_R24_NORMALIZED_CORPUS.json'
    $env:HACP_E13_FAMILY2_FIXTURE = Join-Path $SpecRoot 'vectors\family2\e5c6_binding_fixture.json'
    $env:HACP_E13_FAMILY2_PUBLIC_KEY = Join-Path $SpecRoot 'harness\keys\test-ed25519-001.pub'

    $r = Invoke-WvNative -FilePath 'cargo' -ArgumentList @('+stable-x86_64-pc-windows-msvc','test','--locked') -WorkDir (Join-Path $SpecRoot 'hacp-rs') -StepId 'S09-cargo-test' -SurfaceId 'S09' -EvidenceDir $evidenceDir
}
finally {
    Restore-WvEnvironment -Saved $savedE13Env
}
            $cls = Resolve-WvStepClassification -Run $r -ForbiddenMarkers @('test result: FAILED','panic:')
            $steps += New-WvStepResult -StepId 'S09-cargo-test' -SurfaceId 'S09' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir -Status "48 ACTIVE + 1 IGNORED; claim bound to P_HEAD=$($identityBefore.P.Head)"
            if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
        }
        'S06' {
            Set-WvServiceEnvSidecar -Port 18080 -Upstream 'http://127.0.0.1:18084'
            $env:HACP_TEST_PRIVATE_KEY = Join-Path $RunRoot 'keys\hacp-test-key.pem'
            $env:HACP_TEST_SIGNER_KEY_ID = 'key-ed25519-test-001'
            $env:HACP_SIDECAR_EXTERNAL = '1'
            $env:HACP_SIDECAR_URL = 'http://127.0.0.1:18080'
            $env:HACP_SIDECAR_BIN = $sidecarExe
            Remove-Item Env:HACP_SIDECAR_REPO -ErrorAction SilentlyContinue
            $upstream = New-WvService -Name 'upstream' -FilePath 'python' -ArgumentList @((Join-Path $SidecarRoot 'deployments\upstream\server.py'),'18084') -WorkDir $SidecarRoot -Port 18084 -SurfaceOwner 'S06'
            $sidecar  = New-WvService -Name 'sidecar' -FilePath $sidecarExe -ArgumentList @() -WorkDir $SidecarRoot -Port 18080 -SurfaceOwner 'S06'
            if ($upstream.Ok -and $sidecar.Ok) {
                $r = Invoke-WvNative -FilePath 'python' -ArgumentList @('-m','pytest','tests\test_hacp_sidecar_integration.py','-v','-rs') -WorkDir $CoreRoot -StepId 'S06-e2e' -SurfaceId 'S06' -EvidenceDir $evidenceDir
                $cls = Resolve-WvStepClassification -Run $r -RequiredMarkers @('5 passed')
                $steps += New-WvStepResult -StepId 'S06-e2e' -SurfaceId 'S06' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir -Status 'EXTERNAL mode; SKIP!=PASS; expected 5 passed, 0 skipped'
            } else {
                $cls = $WV_CLASS.ENVIRONMENT_RED
                $steps += New-WvStepResult -StepId 'S06-startup' -SurfaceId 'S06' -Command 'start sidecar/upstream' -StartTimeUtc (Get-Date).ToUniversalTime().ToString('o') -EndTimeUtc (Get-Date).ToUniversalTime().ToString('o') -ProcessRc $null -Classification $cls -EvidencePath $evidenceDir -Status "SERVICE_START_FAILED:$($upstream.Error),$($sidecar.Error)"
            }
            if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
            Stop-WvOwnedProcesses -RunRoot $RunRoot | Out-Null
        }
        'S07' {
            $composeFile = Join-Path $SidecarRoot 'deployments\docker-compose.yml'
            $testComposeFile = Join-Path $SidecarRoot 'test\deployments\docker-compose.yml'

            $prepUp = Invoke-WvNative -FilePath 'docker' -ArgumentList @('compose','-f',$composeFile,'-f',$testComposeFile,'up','-d','--build') -WorkDir $SidecarRoot -StepId 'S07-prep-compose-up' -SurfaceId 'S07' -EvidenceDir $evidenceDir
            $prepUpCls = Resolve-WvStepClassification -Run $prepUp
            $steps += New-WvStepResult -StepId 'S07-prep-compose-up' -SurfaceId 'S07' -Command $prepUp.Command -StartTimeUtc $prepUp.StartTimeUtc -EndTimeUtc $prepUp.EndTimeUtc -ProcessRc $prepUp.ProcessRc -Classification $prepUpCls -EvidencePath $evidenceDir -Status 'package-owned prerequisite: start declared anti-bypass topology'

            $prepCls = $prepUpCls

            if ($prepUpCls -eq $WV_CLASS.PASS) {
                $prepCurl = Invoke-WvNative -FilePath 'docker' -ArgumentList @('compose','-f',$composeFile,'-f',$testComposeFile,'exec','-T','agent','sh','-c','"for i in $(seq 1 30); do command -v curl >/dev/null 2>&1 && exit 0; sleep 1; done; exit 1"') -WorkDir $SidecarRoot -StepId 'S07-prep-agent-curl' -SurfaceId 'S07' -EvidenceDir $evidenceDir
                $prepCurlCls = Resolve-WvStepClassification -Run $prepCurl
                $steps += New-WvStepResult -StepId 'S07-prep-agent-curl' -SurfaceId 'S07' -Command $prepCurl.Command -StartTimeUtc $prepCurl.StartTimeUtc -EndTimeUtc $prepCurl.EndTimeUtc -ProcessRc $prepCurl.ProcessRc -Classification $prepCurlCls -EvidencePath $evidenceDir -Status 'package-owned prerequisite: agent curl provisioning ready before existing anti-bypass procedure'
                $prepCls = $prepCurlCls
            }

            if ($prepCls -eq $WV_CLASS.PASS) {
                $r = Invoke-WvNative -FilePath 'powershell' -ArgumentList @('-NoProfile','-File',(Join-Path $SidecarRoot 'scripts\test-anti-bypass.ps1')) -WorkDir $SidecarRoot -StepId 'S07-anti-bypass' -SurfaceId 'S07' -EvidenceDir $evidenceDir
                $cls = Resolve-WvStepClassification -Run $r -RequiredMarkers @('Result: 5 passed, 0 failed') -ForbiddenMarkers @('\[FAIL\]')
                $steps += New-WvStepResult -StepId 'S07-anti-bypass' -SurfaceId 'S07' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir -Status 'docker compose anti-bypass verification'

                if ($cls -ne $WV_CLASS.PASS) {
                    $verdict = $cls
                }
            }
            else {
                $verdict = $prepCls
            }

            $cleanup = Invoke-WvNative -FilePath 'docker' -ArgumentList @('compose','-f',$composeFile,'-f',$testComposeFile,'down','-v') -WorkDir $SidecarRoot -StepId 'S07-cleanup' -SurfaceId 'S07' -EvidenceDir $evidenceDir
            $cleanupCls = Resolve-WvStepClassification -Run $cleanup
            $steps += New-WvStepResult -StepId 'S07-cleanup' -SurfaceId 'S07' -Command $cleanup.Command -StartTimeUtc $cleanup.StartTimeUtc -EndTimeUtc $cleanup.EndTimeUtc -ProcessRc $cleanup.ProcessRc -Classification $cleanupCls -EvidencePath $evidenceDir -Status 'required cleanup: docker compose down -v; declared compose file only'

            if ($verdict -eq $WV_CLASS.PASS -and $cleanupCls -ne $WV_CLASS.PASS) {
                $verdict = $cleanupCls
            }
        }
        'D' {
            Set-WvServiceEnvSidecar -Port 18082 -Upstream 'http://127.0.0.1:18080'
            $upstream = New-WvService -Name 'upstream' -FilePath 'python' -ArgumentList @((Join-Path $SidecarRoot 'deployments\upstream\server.py'),'18080') -WorkDir $SidecarRoot -Port 18080 -SurfaceOwner 'D'
            $sidecar  = New-WvService -Name 'sidecar' -FilePath $sidecarExe -ArgumentList @() -WorkDir $SidecarRoot -Port 18082 -SurfaceOwner 'D'
            if ($upstream.Ok -and $sidecar.Ok) {
                $r = Invoke-WvNative -FilePath (Join-Path $BUILD_ROOT 'go-http-reproduction.exe') -ArgumentList @('-url','http://127.0.0.1:18082/api/test','-warmup-file',(Join-Path $FIXTURE_ROOT 'd-warmup.jsonl'),'-measured-file',(Join-Path $FIXTURE_ROOT 'd-measured.jsonl')) -WorkDir $SidecarRoot -StepId 'D-b15' -SurfaceId 'D' -EvidenceDir $evidenceDir
                $cls = Resolve-WvStepClassification -Run $r -RequiredMarkers @('GO_HTTP_REPRODUCTION=PASS')
                $steps += New-WvStepResult -StepId 'D-b15' -SurfaceId 'D' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir -Status 'B15 = authorization + real upstream forwarding + real upstream response; negative 403 DENY is safety precondition (not B15 verdict)'
            } else {
                $cls = $WV_CLASS.ENVIRONMENT_RED
                $steps += New-WvStepResult -StepId 'D-startup' -SurfaceId 'D' -Command 'start sidecar/upstream' -StartTimeUtc (Get-Date).ToUniversalTime().ToString('o') -EndTimeUtc (Get-Date).ToUniversalTime().ToString('o') -ProcessRc $null -Classification $cls -EvidencePath $evidenceDir -Status "SERVICE_START_FAILED:$($upstream.Error),$($sidecar.Error)"
            }
            if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
            Stop-WvOwnedProcesses -RunRoot $RunRoot | Out-Null
        }
        'S08' {
            Set-WvServiceEnvSidecar -Port 18081 -Upstream ''
            $env:HACP_SIDECAR_PORT = '18081'
            $proxyExe = Join-Path (Join-Path $RunRoot 'cargo-target-s08\debug') 'hacp-rs-proxy.exe'
            $proxy = New-WvService -Name 'proxy' -FilePath $proxyExe -ArgumentList @() -WorkDir (Join-Path $SpecRoot 'hacp-rs') -Port 18081 -SurfaceOwner 'S08'
            if ($proxy.Ok) {
                $r = Invoke-WvNative -FilePath 'go' -ArgumentList @('run','.\benchmarks\rust-http-reproduction','-url','http://127.0.0.1:18081/api/test','-warmup-file',(Join-Path $FIXTURE_ROOT 's08-warmup.jsonl'),'-measured-file',(Join-Path $FIXTURE_ROOT 's08-measured.jsonl')) -WorkDir $SidecarRoot -StepId 'S08-b16' -SurfaceId 'S08' -EvidenceDir $evidenceDir
                $cls = Resolve-WvStepClassification -Run $r -RequiredMarkers @('R8_2_WORKLOAD=PASS')
                $steps += New-WvStepResult -StepId 'S08-b16' -SurfaceId 'S08' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir -Status 'B16 = authorization + local response; NO upstream-forwarding claim; timing INFORMATIONAL'
            } else {
                $cls = $WV_CLASS.ENVIRONMENT_RED
                $steps += New-WvStepResult -StepId 'S08-startup' -SurfaceId 'S08' -Command 'start proxy' -StartTimeUtc (Get-Date).ToUniversalTime().ToString('o') -EndTimeUtc (Get-Date).ToUniversalTime().ToString('o') -ProcessRc $null -Classification $cls -EvidencePath $evidenceDir -Status "SERVICE_START_FAILED:$($proxy.Error)"
            }
            if ($cls -ne $WV_CLASS.PASS) { $verdict = $cls }
            Stop-WvOwnedProcesses -RunRoot $RunRoot | Out-Null
        }
        'S10' {
            Set-WvServiceEnvSidecar -Port 8080 -Upstream 'http://127.0.0.1:8000'
            $upstream = New-WvService -Name 'upstream' -FilePath 'python' -ArgumentList @((Join-Path $SidecarRoot 'deployments\upstream\server.py'),'8000') -WorkDir $SidecarRoot -Port 8000 -SurfaceOwner 'S10'
            $sidecar  = New-WvService -Name 'sidecar' -FilePath $sidecarExe -ArgumentList @() -WorkDir $SidecarRoot -Port 8080 -SurfaceOwner 'S10'
            if ($upstream.Ok -and $sidecar.Ok) {
                $tgt = @(
                    @{ Sid='S10-go-serial'; Exe='powershell'; Args=@('-NoProfile','-File',(Join-Path $SidecarRoot 'benchmarks\benchmark_serial.ps1')) },
                    @{ Sid='S10-ps-rotating'; Exe='pwsh'; Args=@('-NoProfile','-File',(Join-Path $SidecarRoot 'benchmarks\benchmark_rotating.ps1')) },
                    @{ Sid='S10-go-rotating'; Exe='go'; Args=@('run','.\benchmarks\benchmark-rotating') }
                )
                foreach ($t in $tgt) {
                    $r = Invoke-WvNative -FilePath $t.Exe -ArgumentList $t.Args -WorkDir $SidecarRoot -StepId $t.Sid -SurfaceId 'S10' -EvidenceDir $evidenceDir
                    $cls = $WV_CLASS.INFORMATIONAL   # timing: no threshold, no latency PASS/FAIL
                    $steps += New-WvStepResult -StepId $t.Sid -SurfaceId 'S10' -Command $r.Command -StartTimeUtc $r.StartTimeUtc -EndTimeUtc $r.EndTimeUtc -ProcessRc $r.ProcessRc -Classification $cls -EvidencePath $evidenceDir -Status 'TIMING INFORMATIONAL; GATE D lines = HISTORICAL_MARKER_ONLY'
                }
                $verdict = $WV_CLASS.INFORMATIONAL
            } else {
                $verdict = $WV_CLASS.ENVIRONMENT_RED
                $steps += New-WvStepResult -StepId 'S10-startup' -SurfaceId 'S10' -Command 'start sidecar/upstream' -StartTimeUtc (Get-Date).ToUniversalTime().ToString('o') -EndTimeUtc (Get-Date).ToUniversalTime().ToString('o') -ProcessRc $null -Classification $verdict -EvidencePath $evidenceDir -Status "SERVICE_START_FAILED:$($upstream.Error),$($sidecar.Error)"
            }
            Stop-WvOwnedProcesses -RunRoot $RunRoot | Out-Null
        }
    }

    $result = [pscustomobject]@{
        SURFACE_ID       = $Id
        MATRIX_ROWS      = $MATRIX_ROWS[$Id]
        VERDICT          = $verdict
        STEPS            = $steps
        ENVIRONMENT_STATUS = $(if ($verdict -eq $WV_CLASS.PASS) { 'READY' } else { $verdict })
        EVIDENCE_DIRECTORY = $evidenceDir
    }
    Write-WvJson -Object $result -Path (Join-Path $evidenceDir 'result.json')
    return $result
}

# ---- Sequential execution with stop-on-material-RED ----
$results = @()
$materialRed = $false
foreach ($id in $SURFACE_QUEUE) {
    if ($materialRed -and -not $ContinueIndependentSurfaces) {
        $results += [pscustomobject]@{ SURFACE_ID=$id; MATRIX_ROWS=$MATRIX_ROWS[$id]; VERDICT=$WV_CLASS.NOT_RUN; STEPS=@(); ENVIRONMENT_STATUS='NOT_RUN'; EVIDENCE_DIRECTORY=$null }
        Write-WvLog -Level WARN -Message "SURFACE $id NOT_RUN (stopped after material RED)"
        continue
    }
    # D-3: scoped environment mutation (SAVE -> surface sets exact env -> RESTORE in finally).
    # Distinguishes SET vs UNSET (UNSET restored as UNSET); restoration guaranteed on exceptions (§26).
    $envSaved = Save-WvEnvironment -Names @('HACP_SIDECAR_PORT','HACP_UPSTREAM','HACP_TRUST_MODE','HACP_TRUST_KEYS_FILE','HACP_CONTROL_MODE','HACP_SIDECAR_EXTERNAL','HACP_SIDECAR_URL','HACP_SIDECAR_BIN','HACP_SIDECAR_REPO','HACP_TEST_PRIVATE_KEY','HACP_TEST_SIGNER_KEY_ID','HACP_TIME_TOLERANCE')
    try {
        $res = Invoke-WvSurface -Id $id
    } finally {
        Restore-WvEnvironment -Saved $envSaved
    }
    $results += $res
    if ($res.VERDICT -in @($WV_CLASS.BEHAVIORAL_RED,$WV_CLASS.IDENTITY_RED,$WV_CLASS.ENVIRONMENT_RED)) {
        $materialRed = $true
        Write-WvLog -Level RED -Message "SURFACE $id -> $($res.VERDICT)"
        if (-not $ContinueIndependentSurfaces) {
            Write-WvLog -Level WARN -Message "STOP ON MATERIAL RED (default). Use -ContinueIndependentSurfaces to continue independent surfaces."
        }
    }
}

# ---- Result outputs (§43) ----
$rows = foreach ($r in $results) {
    [pscustomobject]@{
        SURFACE_ID        = $r.SURFACE_ID
        MATRIX_ROWS       = $r.MATRIX_ROWS
        PROCESS_RC        = $(if ($r.STEPS -and @($r.STEPS).Count -gt 0) { (@($r.STEPS)[0]).PROCESS_RC } else { $null })
        VERDICT           = $r.VERDICT
        ENVIRONMENT_STATUS= $r.ENVIRONMENT_STATUS
        START_TIME        = $(if ($r.STEPS -and @($r.STEPS).Count -gt 0) { (@($r.STEPS)[0]).START_TIME_UTC } else { $null })
        END_TIME          = $(if ($r.STEPS -and @($r.STEPS).Count -gt 0) { (@($r.STEPS)[-1]).END_TIME_UTC } else { $null })
        EVIDENCE_DIRECTORY= $r.EVIDENCE_DIRECTORY
    }
}
Write-WvJson -Object $rows -Path (Join-Path $EVID_ROOT 'verification-results.json')
Write-WvCsv -Object $rows -Path (Join-Path $EVID_ROOT 'verification-results.csv')

$summary = [ordered]@{
    TOTAL_SURFACES_RUN = @($rows).Count
    PASS = @($rows | Where-Object { $_.VERDICT -eq $WV_CLASS.PASS }).Count
    ENVIRONMENT_RED = @($rows | Where-Object { $_.VERDICT -eq $WV_CLASS.ENVIRONMENT_RED }).Count
    BEHAVIORAL_RED = @($rows | Where-Object { $_.VERDICT -eq $WV_CLASS.BEHAVIORAL_RED }).Count
    IDENTITY_RED = @($rows | Where-Object { $_.VERDICT -eq $WV_CLASS.IDENTITY_RED }).Count
    INFORMATIONAL = @($rows | Where-Object { $_.VERDICT -eq $WV_CLASS.INFORMATIONAL }).Count
    NOT_RUN = @($rows | Where-Object { $_.VERDICT -eq $WV_CLASS.NOT_RUN }).Count
    MATERIAL_RED_STOPPED = $materialRed
}
Write-WvJson -Object $summary -Path (Join-Path $EVID_ROOT 'verification-summary.json')
Write-WvLog -Message "02-Run-Verification complete. Results in $EVID_ROOT"
# NO admission, NO acceptance, NO synthetic combined score.