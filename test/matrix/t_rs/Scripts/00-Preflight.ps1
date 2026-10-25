#Requires -Version 7.0

<#
00-Preflight.ps1 (RUST) — READ-ONLY pre-execution verification.
NO build/install/copy/fixture generation/git mutation/process start. Records OS, PowerShell/toolchains,
P+S git identity and branch, RS07 artifact identity, port 18081, trust/env inventory, SHA inventory.
Outputs 00-preflight\PRECHECK.json / PRECHECK.txt with PREFLIGHT=PASS|BLOCKED.
#>

[CmdletBinding()]
param(
    [string]$SpecRepo    = '',
    [string]$SidecarRepo = '',
    [string]$EvidenceRoot= '',
    [string]$RunRoot     = '',
    [string]$RunDir      = ''
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

# DEFECT-1: canonical package-root based RS07 source resolution (single implementation).
$RustPackageRoot = Split-Path -Parent $PSScriptRoot
$Rs07Source = Join-Path $RustPackageRoot 'RS07_budget_replay_verification.rs'

# R1: 00 WITHOUT -RunDir ALWAYS creates a NEW run (never reuses an existing/latest run). Explicit -RunDir wins exactly.
try {
    $run = Resolve-RvRunDir -EvidenceRoot $EvidenceRoot -RunDir $RunDir -Mode CREATE_NEW
} catch {
    Write-RvLog -Level RED -Message "00 run selection failed: $($_.Exception.Message)"
    $fallback = New-RvDirectory -Path $EvidenceRoot
    Write-RvJson -Object ([ordered]@{ PREFLIGHT='BLOCKED'; BLOCKERS=@("RUN_SELECTION_FAILED:$($_.Exception.Message)") }) -Path (Join-Path $fallback 'preflight-blocked.json')
    Write-RvLog -Level RED -Message "PREFLIGHT=BLOCKED (run selection). Exiting nonzero."
    exit 2
}
$OUT = Join-Path $run.RunDir '00-preflight'
New-RvDirectory -Path $OUT | Out-Null
New-RvDirectory -Path (Join-Path $run.RunDir 'metadata') | Out-Null
Set-RvLogFile -Path (Join-Path $OUT 'preflight-console.log')
Write-RvLog -Message "00-Preflight (RUST) start — READ-ONLY. RunDir=$($run.RunDir) RunSelectionMode=$($run.RunSelectionMode)"
# R1 evidence (§10): full run identity + selection mode.
$workRootPath = Join-Path $RunRoot (Split-Path -Leaf $run.RunDir)
$runContext = [ordered]@{
    RunId = $run.RunId
    CreatedAtUtc = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
    RunDir = $run.RunDir
    WorkRoot = $workRootPath
    EvidenceRoot = $EvidenceRoot
    CreatedByScript = '00-Preflight.ps1'
    RUN_SELECTION_MODE = $run.RunSelectionMode
    RUN_STATE = 'CREATED'
}
Write-RvJson -Object $runContext -Path (Join-Path $run.RunDir 'metadata\run-context.json')

# Publish the newly created timestamped run as the current Rust verification
# contour. Historical evidence directories are never deleted or rewritten.
$currentRunPointer = Join-Path (Split-Path -Parent $EvidenceRoot) 'current-run.txt'
Set-Content -LiteralPath $currentRunPointer -Value $run.RunDir -Encoding utf8

# ---- 1. OS ----
$os = [ordered]@{
    OS = [System.Environment]::OSVersion.VersionString
    ARCH = [System.Environment]::GetEnvironmentVariable('PROCESSOR_ARCHITECTURE')
    POWERSHELL_VERSION = $PSVersionTable.PSVersion.ToString()
    POWERSHELL_EDITION = $PSVersionTable.PSEdition
}
Write-RvJson -Object $os -Path (Join-Path $OUT 'os.json')

# ---- 2. Toolchains (path + version + availability; install nothing) ----
$tools = @(
    @{ N='git';     C='git';     A=@('--version') },
    @{ N='rustc';   C='rustc';   A=@('--version') },
    @{ N='cargo';   C='cargo';   A=@('--version') },
    @{ N='rustup';  C='rustup';  A=@('--version') },
    @{ N='protoc';  C='protoc';  A=@('--version') },
    @{ N='python';  C='python';  A=@('--version') },
    @{ N='go';      C='go';      A=@('version') }
)
$toolchain = @()
foreach ($t in $tools) {
    $cmd = Get-Command $t.C -ErrorAction SilentlyContinue
    $present = [bool]$cmd; $ver = $null
    if ($present) {
        try { $ver = ((& $t.C @($t.A) 2>&1 | Out-String) -split "`r?`n")[0].Trim() } catch { $ver = 'VERSION_QUERY_FAILED' }
    }
    $toolchain += [pscustomobject]@{ TOOL=$t.N; PATH=$(if ($present) { $cmd.Source } else { $null }); PRESENT=$present; VERSION=$ver }
}
Write-RvJson -Object $toolchain -Path (Join-Path $OUT 'toolchain.json')

# ---- 3. Git identity (P + S) ----
$pIdentity = Get-RvGitIdentity -RepoRoot $SpecRepo
$sIdentity = Get-RvGitIdentity -RepoRoot $SidecarRepo
$EXPECTED_BRANCH = 'admit/hacp-1.1.1-pre-release'
$pBranch = (git -C $SpecRepo branch --show-current).Trim()
$sBranch = (git -C $SidecarRepo branch --show-current).Trim()
$git = [ordered]@{
    P_ROOT=$SpecRepo; P_HEAD=$pIdentity.Head; P_TREE=$pIdentity.Tree; P_STAGED=$pIdentity.StagedCount
    P_STATUS_SHORT=$pIdentity.StatusShort
    S_ROOT=$SidecarRepo; S_HEAD=$sIdentity.Head; S_TREE=$sIdentity.Tree; S_STAGED=$sIdentity.StagedCount
    S_STATUS_SHORT=$sIdentity.StatusShort
    P_BRANCH=$pBranch; S_BRANCH=$sBranch; EXPECTED_BRANCH=$EXPECTED_BRANCH
}
Write-RvJson -Object $git -Path (Join-Path $OUT 'git-identity.json')
$stagedOk = ($pIdentity.StagedCount -eq 0 -and $sIdentity.StagedCount -eq 0)

# ---- 4. RS07 artifact identity check (read-only; no copy/transfer) ----
$rs07Expected = 'e97e732546f6d1688f0eeb464259f4f0b7b53c82be3f37b97eec776c94859fb2'
$rs07 = Get-RvCanonicalTextIdentity -Path $Rs07Source
$rs07Class = if (-not $rs07.Exists) { 'ABSENT' } elseif ($rs07.Sha256 -eq $rs07Expected) { 'PRESENT_MATCHING' } else { 'PRESENT_DIFFERENT' }

$rs07Relative = 'test/matrix/t_rs/RS07_budget_replay_verification.rs'
$rs07State = Get-RvGitPathState -RepoRoot $SidecarRepo -RelativePath $rs07Relative

$rs07Info = [ordered]@{
    PATH=$Rs07Source
    REPOSITORY='hacp-sidecar'
    RELATIVE_PATH=$rs07Relative
    CLASSIFICATION=$rs07Class
    EXPECTED_SHA256=$rs07Expected
    ACTUAL_SHA256=$rs07.Sha256
    GIT_STATUS=$rs07State.GitStatus
    IGNORED=$rs07State.IsIgnored
}

$rs07ArtifactState = if ($rs07Class -eq 'PRESENT_MATCHING') {
    'PRESENT_MATCHING'
} else {
    'BLOCKED'
}

Write-RvJson -Object $rs07Info -Path (Join-Path $OUT 'rs07-artifact-check.json')

# ---- 5. Port inventory (RS04: 18081; UPSTREAM_REQUIRED=NO) ----
$portStates = @()
foreach ($p in @(18081)) {
    $s = Test-RvPortState -Port $p
    $portStates += [pscustomobject]@{ PORT=$p; STATE=$s.State; OWNER_PIDS=(@($s.OwnerPids) -join ','); UPSTREAM_REQUIRED='NO' }
}
Write-RvJson -Object $portStates -Path (Join-Path $OUT 'port-inventory.json')
# PORT GATE: 18081 must be free before Rust execution (never auto-kill a foreign listener).
$port18081Free = (Test-RvPortState -Port 18081).State -eq 'FREE'

# ---- 6. Trust / env inventory (read-only; secret-safe) ----
$envSnap = Get-RvEnvSnapshot
Write-RvJson -Object $envSnap -Path (Join-Path $OUT 'env-inventory.json')

# ---- 7. SHA inventory (authoritative inputs only) ----
$shaTargets = @(
    @{ Id='RS07-source';          Path=$Rs07Source },
    @{ Id='manifest-rs';          Path=(Join-Path (Split-Path -Parent $PSScriptRoot) 'WINDOWS_EXECUTION_MANIFEST_RS.md') },
    @{ Id='contracts-rs';         Path=(Join-Path (Split-Path -Parent $PSScriptRoot) 'RUST_VERIFICATION_CONTRACTS.md') },
    @{ Id='matrix-recon-rs';      Path=(Join-Path (Split-Path -Parent $PSScriptRoot) 'RUST_MATRIX_RECONCILIATION.md') },
    @{ Id='runner-source';        Path=(Join-Path $SpecRepo 'hacp-rs\src\bin\hacp-rs-runner.rs') },
    @{ Id='proxy-source';         Path=(Join-Path $SpecRepo 'hacp-rs\src\bin\hacp-rs-proxy.rs') },
    @{ Id='evaluate-source';      Path=(Join-Path $SpecRepo 'hacp-rs\src\evaluate.rs') },
    @{ Id='cargo-lock';           Path=(Join-Path $SpecRepo 'hacp-rs\Cargo.lock') }
)
$shaInventory = @()
foreach ($t in $shaTargets) {
    $fi = Get-RvFileIdentity -Path $t.Path
    $shaInventory += [pscustomobject]@{ ID=$t.Id; PATH=$t.Path; EXISTS=$fi.Exists; SIZE_BYTES=$fi.SizeBytes; SHA256=$fi.Sha256 }
}
Write-RvJson -Object $shaInventory -Path (Join-Path $OUT 'sha-inventory.json')

# ---- 7A. PowerShell version gate ----
$psVersion = $PSVersionTable.PSVersion
$ps7Ok = ($psVersion.Major -ge 7)

$psInfo = [ordered]@{
    VERSION = $psVersion.ToString()
    MAJOR = $psVersion.Major
    MINIMUM = '7.0'
    PASS = $ps7Ok
}

Write-RvJson -Object $psInfo -Path (Join-Path $OUT 'powershell-version.json')

# ---- 8. PRECHECK verdict (hard gates) ----
$blockers = @()
if (-not $stagedOk) { $blockers += "STAGED_FILES_NOT_ZERO (P=$($pIdentity.StagedCount) S=$($sIdentity.StagedCount))" }
if ($pBranch -ne $EXPECTED_BRANCH) { $blockers += "P_BRANCH_MISMATCH expected=$EXPECTED_BRANCH actual=$pBranch" }
if ($sBranch -ne $EXPECTED_BRANCH) { $blockers += "S_BRANCH_MISMATCH expected=$EXPECTED_BRANCH actual=$sBranch" }
if (-not $ps7Ok) { $blockers += "POWERSHELL_VERSION_UNSUPPORTED actual=$($psVersion.ToString()) required=>=7.0" }
# DEFECT-2: ALL mandatory toolchains are hard gates.
foreach ($need in @('git','rustc','cargo','rustup','protoc','python','go')) {
    if (-not ($toolchain | Where-Object { $_.TOOL -eq $need -and $_.PRESENT })) { $blockers += "TOOL_MISSING:$need" }
}
# PORT GATE (18081 free).
if (-not $port18081Free) { $blockers += 'PORT_18081_OCCUPIED' }
# RS07 authoritative t_rs artifact must be PRESENT_MATCHING.
if ($rs07ArtifactState -ne 'PRESENT_MATCHING') { $blockers += "RS07_ARTIFACT_$rs07ArtifactState" }
$preflight = if ($blockers.Count -eq 0) { 'PASS' } else { 'BLOCKED' }

$precheck = [ordered]@{
    PREFLIGHT = $preflight
    BLOCKERS = $blockers
    P_BRANCH = $pBranch
    S_BRANCH = $sBranch
    EXPECTED_BRANCH = $EXPECTED_BRANCH
    P_BRANCH_OK = ($pBranch -eq $EXPECTED_BRANCH)
    S_BRANCH_OK = ($sBranch -eq $EXPECTED_BRANCH)
    POWERSHELL_VERSION = $psVersion.ToString()
    POWERSHELL_7_PLUS = $ps7Ok
    STAGED_FILES_OK = $stagedOk
    RS07_CLASSIFICATION = $rs07Class
    RS07_ARTIFACT = $rs07ArtifactState
    RS07_ARTIFACT_PATH = $Rs07Source
    PORT_18081 = $(if ($port18081Free) { 'FREE' } else { 'OCCUPIED' })
    RUN_DIR = $run.RunDir
}
Write-RvJson -Object $precheck -Path (Join-Path $OUT 'PRECHECK.json')
Write-RvSummaryText -Table $precheck -Path (Join-Path $OUT 'PRECHECK.txt')
Write-RvLog -Message "PREFLIGHT=$preflight blockers=$($blockers -join '; '). If BLOCKED, do NOT run 01."
if ($preflight -eq 'PASS') { Set-RvRunState -RunDir $run.RunDir -State 'PREFLIGHT_COMPLETE' }
# READ-ONLY: no build/install/copy/git mutation/process start.
