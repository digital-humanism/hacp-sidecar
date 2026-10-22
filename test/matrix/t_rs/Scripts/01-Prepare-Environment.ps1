#Requires -Version 7.0

<#
01-Prepare-Environment.ps1 (RUST) — controlled MUTATING preparation. Only contract-approved
actions: temp dirs, RS07 artifact validation, repository branch validation, dependency prep,
RS01/RS02 runner build, RS04 proxy build (fresh),
RS04 fixtures. NO tests, NO services left running, NO reset/clean/stash.
#>

[CmdletBinding()]
param(
    [string]$SpecRepo    = 'C:\Personal\GitHub\Dev\hacp-spec',
    [string]$SidecarRepo = 'C:\Personal\GitHub\Dev\hacp-sidecar',
    [string]$EvidenceRoot= 'C:\Temp\hacp-rust-verification\evidence',
    [string]$RunRoot     = 'C:\Temp\hacp-rust-verification\run',
    [string]$RunDir      = '',
    [switch]$SkipDependencyPrep,
    [switch]$SkipBuilds
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Import-Module (Join-Path $PSScriptRoot 'WindowsVerification.Rust.Common.psm1') -Force

$run = Resolve-RvRunDir -EvidenceRoot $EvidenceRoot -RunDir $RunDir -RequireState 'PREFLIGHT_COMPLETE'
$OUT = Join-Path $run.RunDir '01-prepare'
New-RvDirectory -Path $OUT | Out-Null
Set-RvLogFile -Path (Join-Path $OUT 'prepare-console.log')
Write-RvLog -Message "01-Prepare (RUST) start. RunDir=$($run.RunDir)"

# DEFECT-3: hard 00 -> 01 gate (must run BEFORE any preparation action).
$precheckPath = Join-Path $run.RunDir '00-preflight\PRECHECK.json'
$precheckReady = $false
if (-not (Test-Path -LiteralPath $precheckPath)) {
    Write-RvLog -Level RED -Message "PRECHECK missing: $precheckPath"
} else {
    try {
        $pre = Get-Content -LiteralPath $precheckPath -Raw | ConvertFrom-Json
        if ($pre.PREFLIGHT -eq 'PASS') { $precheckReady = $true } else { Write-RvLog -Level RED -Message "PREFLIGHT=$($pre.PREFLIGHT) != PASS" }
    } catch { Write-RvLog -Level RED -Message "PRECHECK malformed: $($_.Exception.Message)" }
}
if (-not $precheckReady) {
    Write-RvJson -Object ([ordered]@{ ENVIRONMENT_READY='BLOCKED'; BLOCKERS=@('PRECHECK_NOT_PASS'); RUN_DIR=$run.RunDir }) -Path (Join-Path $OUT 'PREPARE_RESULT.json')
    Write-RvSummaryText -Table ([ordered]@{ ENVIRONMENT_READY='BLOCKED'; BLOCKERS='PRECHECK_NOT_PASS' }) -Path (Join-Path $OUT 'PREPARE_RESULT.txt')
    Write-RvLog -Level RED -Message "00->01 gate FAILED. NO PREPARATION ACTIONS. Exiting nonzero."
    exit 2
}

# DEFECT-8: run-owned work root (mutating run state is run-specific, tied to exact evidence run).
$workRoot = Get-RvWorkRoot -RunRoot $RunRoot -RunDir $run.RunDir
$BIN = Join-Path $workRoot 'bin'
$CARGO_TARGET = Join-Path $workRoot 'cargo-target'
$FIXTURES = Join-Path $workRoot 'fixtures'
$LOGS = New-RvDirectory -Path (Join-Path $workRoot 'logs')
$TMP = Join-Path $workRoot 'tmp'
$cargoEnv = @{ CARGO_TARGET_DIR = $CARGO_TARGET }

# ---- 1. RS07 artifact validation (read-only; no copy/transfer) ----
$rs07Expected = '0e3d8a8b8a380829accc62ed8948eba97019802fe098644f8e1fa9a887f4c158'
$rs07Src = Join-Path (Split-Path -Parent $PSScriptRoot) 'RS07_budget_replay_verification.rs'
$rs07 = Get-RvFileIdentity -Path $rs07Src

$rs07Class = if (-not $rs07.Exists) {
    'ABSENT'
} elseif ($rs07.Sha256 -eq $rs07Expected) {
    'PRESENT_MATCHING'
} else {
    'PRESENT_DIFFERENT'
}

$rs07Relative = 'test/matrix/t_rs/RS07_budget_replay_verification.rs'
$rs07GitState = Get-RvGitPathState -RepoRoot $SidecarRepo -RelativePath $rs07Relative

$rs07Artifact = [ordered]@{
    PATH = $rs07Src
    REPOSITORY = 'hacp-sidecar'
    RELATIVE_PATH = $rs07Relative
    CLASSIFICATION = $rs07Class
    EXPECTED_SHA256 = $rs07Expected
    ACTUAL_SHA256 = $rs07.Sha256
    GIT_STATUS = $rs07GitState.GitStatus
    IGNORED = $rs07GitState.IsIgnored
}

Write-RvJson -Object $rs07Artifact -Path (Join-Path $OUT 'rs07-artifact.json')

# ---- 2. Repository identity gate (read-only; no checkout/detach) ----
$EXPECTED_BRANCH = 'admit/hacp-1.1.1-pre-release'

$pIdentity = Get-RvGitIdentity -RepoRoot $SpecRepo
$sIdentity = Get-RvGitIdentity -RepoRoot $SidecarRepo

$pBranch = (git -C $SpecRepo branch --show-current).Trim()
$sBranch = (git -C $SidecarRepo branch --show-current).Trim()

$repositoryIdentity = [ordered]@{
    EXPECTED_BRANCH = $EXPECTED_BRANCH
    P_BRANCH = $pBranch
    S_BRANCH = $sBranch
    P_HEAD = $pIdentity.Head
    S_HEAD = $sIdentity.Head
    P_STAGED = $pIdentity.StagedCount
    S_STAGED = $sIdentity.StagedCount
}

Write-RvJson -Object $repositoryIdentity -Path (Join-Path $OUT 'repository-identity.json')

# ---- 3. Dependencies (contract-approved only; record network; no upgrades) ----
$depSteps = @()
if (-not $SkipDependencyPrep) {
    $deps = @(
        @{ Id='dep-python-harness'; Exe='python'; Args=@('-m','pip','install','-r',(Join-Path $SpecRepo 'harness\requirements.txt')); Work=$SpecRepo; Net=$true }
        @{ Id='dep-cargo-locked';   Exe='cargo'; Args=@('+stable-x86_64-pc-windows-msvc','fetch','--locked','--manifest-path',(Join-Path $SpecRepo 'hacp-rs\Cargo.toml')); Work=(Join-Path $SpecRepo 'hacp-rs'); Net=$true }
    )
    foreach ($d in $deps) {
        $r = Invoke-RvNative -FilePath $d.Exe -ArgumentList $d.Args -WorkDir $d.Work -StepId $d.Id -SurfaceId 'ENV' -EvidenceDir $OUT -ExtraEnv $cargoEnv
        $depSteps += [pscustomobject]@{ ID=$d.Id; RC=$r.ProcessRc; NETWORK_REQUIRED=$d.Net; CLASSIFICATION=$(if ($r.ProcessRc -eq 0) { 'PASS' } else { 'ENVIRONMENT_RED' }) }
    }
} else { Write-RvLog -Level WARN -Message "SkipDependencyPrep set" }
Write-RvJson -Object $depSteps -Path (Join-Path $OUT 'dependency-steps.json')

# ---- 4. Builds (runner release + proxy fresh; external CARGO_TARGET_DIR) ----
$buildInventory = @()
if (-not $SkipBuilds) {
    $builds = @(
        @{ Id='build-rs-runner';  Bin='hacp-rs-runner';  Args=@('+stable-x86_64-pc-windows-msvc','build','--locked','--release','--bin','hacp-rs-runner','--manifest-path',(Join-Path $SpecRepo 'hacp-rs\Cargo.toml')); Profile='release' },
        @{ Id='build-rs-proxy';  Bin='hacp-rs-proxy';  Args=@('+stable-x86_64-pc-windows-msvc','build','--locked','--offline','--bin','hacp-rs-proxy','--manifest-path',(Join-Path $SpecRepo 'hacp-rs\Cargo.toml')); Profile='debug' }
    )
    foreach ($b in $builds) {
        $r = Invoke-RvNative -FilePath 'cargo' -ArgumentList $b.Args -WorkDir (Join-Path $SpecRepo 'hacp-rs') -StepId $b.Id -SurfaceId 'RS01' -EvidenceDir $OUT -ExtraEnv $cargoEnv
        $outPath = Join-Path (Join-Path $CARGO_TARGET $b.Profile) "$($b.Bin).exe"
        $fi = Get-RvFileIdentity -Path $outPath
        $buildInventory += [pscustomobject]@{
            BUILD_ID=$b.Id; BINARY=$outPath; SIZE_BYTES=$fi.SizeBytes; SHA256=$fi.Sha256
            BUILD_COMMAND=$r.Command; PROCESS_RC=$r.ProcessRc
            SOURCE_HEAD=(Get-RvGitIdentity -RepoRoot (Join-Path $SpecRepo 'hacp-rs')).Head
            TOOLCHAIN='rust stable-x86_64-pc-windows-msvc + protoc'
            CLASSIFICATION=$(if ($r.ProcessRc -eq 0 -and $fi.Exists) { 'PASS' } else { 'ENVIRONMENT_RED' })
        }
    }
} else { Write-RvLog -Level WARN -Message "SkipBuilds set" }
Write-RvJson -Object $buildInventory -Path (Join-Path $OUT 'build-inventory.json')

# ---- 5. RS04 fixtures (fresh; generator exact; TTL 7200s) ----
$genWork = Join-Path $SidecarRepo 'benchmarks\generate-tokens'
$fixtureInventory = @()
foreach ($f in @(
    @{ Id='fx-rs04-warmup';  Count=100;  Out='tokens-warmup-100.jsonl' },
    @{ Id='fx-rs04-measured';Count=5000; Out='tokens-5000.jsonl' }
)) {
    $outPath = Join-Path $FIXTURES $f.Out
    $r = Invoke-RvNative -FilePath 'go' -ArgumentList @('run','.\benchmarks\generate-tokens','-count',"$($f.Count)",'-out',$outPath,'-method','GET','-path','/api/test') -WorkDir $SidecarRepo -StepId $f.Id -SurfaceId 'RS04' -EvidenceDir $OUT
    $fi = Get-RvFileIdentity -Path $outPath
    $fixtureInventory += [pscustomobject]@{
        FIXTURE_ID=$f.Id; PATH=$outPath; GENERATOR_COMMAND=$r.Command; GENERATOR_IDENTITY='hacp-sidecar/benchmarks/generate-tokens'
        GENERATED_AT_UTC=$r.StartTimeUtc; RECORD_COUNT=$f.Count; SIZE_BYTES=$fi.SizeBytes; SHA256=$fi.Sha256
        TTL_SECONDS=7200; TOKEN_VALIDITY_AT_START=$(if ($r.ProcessRc -eq 0 -and $fi.Exists) { 'PASS' } else { 'ENVIRONMENT_RED' })
    }
}
Write-RvJson -Object $fixtureInventory -Path (Join-Path $OUT 'fixture-inventory.json')

# ---- 6. PREPARE_RESULT ----
$blockers = @()

if ($rs07Class -ne 'PRESENT_MATCHING') { $blockers += "RS07_ARTIFACT_$rs07Class" }
if ($pBranch -ne $EXPECTED_BRANCH) { $blockers += "P_BRANCH_MISMATCH expected=$EXPECTED_BRANCH actual=$pBranch" }
if ($sBranch -ne $EXPECTED_BRANCH) { $blockers += "S_BRANCH_MISMATCH expected=$EXPECTED_BRANCH actual=$sBranch" }
if (-not $SkipBuilds) {
    foreach ($b in $buildInventory) { if ($b.CLASSIFICATION -ne 'PASS') { $blockers += "BUILD_FAILED:$($b.BUILD_ID)" } }
}
foreach ($fx in $fixtureInventory) { if ($fx.TOKEN_VALIDITY_AT_START -ne 'PASS') { $blockers += "FIXTURE_FAILED:$($fx.FIXTURE_ID)" } }
foreach ($d in $depSteps) { if ($d.CLASSIFICATION -ne 'PASS') { $blockers += "DEP_FAILED:$($d.ID)" } }
$envReady = if ($blockers.Count -eq 0) { 'PASS' } else { 'BLOCKED' }

$prepare = [ordered]@{
    ENVIRONMENT_READY = $envReady
    BLOCKERS = $blockers
    RS07_ARTIFACT = $rs07Class
    RS07_ARTIFACT_PATH = $rs07Src
    P_BRANCH = $pBranch
    S_BRANCH = $sBranch
    EXPECTED_BRANCH = $EXPECTED_BRANCH

    RUN_ROOT = $RunRoot
    CARGO_TARGET_DIR = $CARGO_TARGET
    RUN_DIR = $run.RunDir
}
Write-RvJson -Object $prepare -Path (Join-Path $OUT 'PREPARE_RESULT.json')
Write-RvSummaryText -Table $prepare -Path (Join-Path $OUT 'PREPARE_RESULT.txt')
Write-RvLog -Message "ENVIRONMENT_READY=$envReady blockers=$($blockers -join '; '). If BLOCKED, do NOT run 02."
if ($envReady -eq 'PASS') { Set-RvRunState -RunDir $run.RunDir -State 'PREPARED' }
# NO tests, NO services left running, NO git add/commit/reset/clean/stash.
