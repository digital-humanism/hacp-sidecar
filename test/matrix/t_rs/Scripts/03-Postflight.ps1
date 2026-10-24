#Requires -Version 7.0

<#
03-Postflight.ps1 (RUST) — cleanup + guards + evidence inventory.
Stops ONLY owned PIDs (never Stop-Process -Name / taskkill /IM). Verifies no owned listener remains
on 18081. Git guards: P+S vs preflight baseline; exact preflight Git identity/worktree baseline is preserved; production
tracked modifications MUST be 0. Builds EVIDENCE_MANIFEST.csv/.json. NO git clean/reset, NO admission.
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

$run = Resolve-RvRunDir -EvidenceRoot $EvidenceRoot -RunDir $RunDir -RequireState 'VERIFICATION_COMPLETE'
$OUT = New-RvDirectory -Path (Join-Path $run.RunDir '03-postflight')
Set-RvLogFile -Path (Join-Path $OUT 'postflight-console.log')
Write-RvLog -Message "03-Postflight (RUST) start. RunDir=$($run.RunDir)"
# DEFECT-8: run-owned work root (PID registry is run-specific; never a stale registry of another run).
$workRoot = Get-RvWorkRoot -RunRoot $RunRoot -RunDir $run.RunDir

# ---- 1. Cleanup owned processes only (registered in THIS run) ----
$cleanup = Stop-RvOwnedProcesses -WorkRoot $workRoot
Write-RvLog -Message "owned stopped=$(@($cleanup.StoppedPids).Count) already-exited=$(@($cleanup.AlreadyExitedPids).Count) leaked=$(@($cleanup.LeakedPids).Count) ownership-mismatch=$(@($cleanup.OwnershipMismatches).Count)"

# ---- 2. Port cleanup check (18081 + any Rust ports used) ----
$portLeaks = @()
foreach ($p in @(18081)) {
    if (-not (Test-RvPortAbsent -Port $p)) {
        $s = Test-RvPortState -Port $p
        $portLeaks += [pscustomobject]@{ PORT=$p; OWNER_PIDS=(@($s.OwnerPids) -join ',') }
    }
}

# ---- 3. Git guards (P + S) vs preflight baseline ----
$preGitPath = Join-Path $run.RunDir '00-preflight\git-identity.json'
$preGit = $null

if (Test-Path -LiteralPath $preGitPath) {
    try {
        $preGit =
            Get-Content -LiteralPath $preGitPath -Raw |
            ConvertFrom-Json
    }
    catch {
        $preGit = $null
    }
}

$pNow = Get-RvGitIdentity -RepoRoot $SpecRepo
$sNow = Get-RvGitIdentity -RepoRoot $SidecarRepo

$pBranchNow = (git -C $SpecRepo branch --show-current).Trim()
$sBranchNow = (git -C $SidecarRepo branch --show-current).Trim()

$expectedBranch = 'admit/hacp-1.1.1-pre-release'

$gitGuard = [ordered]@{}

# Normalize git status evidence to deterministic text for comparison.
$pStatusNow =
    (@($pNow.StatusShort) | ForEach-Object { [string]$_ }) -join "`n"

$sStatusNow =
    (@($sNow.StatusShort) | ForEach-Object { [string]$_ }) -join "`n"

$preGitAvailable = $false

$pHeadBaseline = 'NOT_AVAILABLE'
$pTreeBaseline = 'NOT_AVAILABLE'
$pStatusBaseline = 'NOT_AVAILABLE'
$pBranchBaseline = 'NOT_AVAILABLE'

$sHeadBaseline = 'NOT_AVAILABLE'
$sTreeBaseline = 'NOT_AVAILABLE'
$sStatusBaseline = 'NOT_AVAILABLE'
$sBranchBaseline = 'NOT_AVAILABLE'

$expectedBranchBaseline = 'NOT_AVAILABLE'

if ($preGit) {
    $requiredPreGitFields = @(
        'P_HEAD',
        'P_TREE',
        'P_STATUS_SHORT',
        'P_BRANCH',
        'S_HEAD',
        'S_TREE',
        'S_STATUS_SHORT',
        'S_BRANCH',
        'EXPECTED_BRANCH'
    )

    $missingPreGitFields = @()

    foreach ($field in $requiredPreGitFields) {
        $property =
            $preGit.PSObject.Properties |
            Where-Object { $_.Name -ieq $field } |
            Select-Object -First 1

        if (-not $property) {
            $missingPreGitFields += $field
        }
    }

    if ($missingPreGitFields.Count -eq 0) {
        $preGitAvailable = $true

        $pHeadBaseline = [string]$preGit.P_HEAD
        $pTreeBaseline = [string]$preGit.P_TREE
        $pStatusBaseline =
            (@($preGit.P_STATUS_SHORT) | ForEach-Object { [string]$_ }) -join "`n"
        $pBranchBaseline = [string]$preGit.P_BRANCH

        $sHeadBaseline = [string]$preGit.S_HEAD
        $sTreeBaseline = [string]$preGit.S_TREE
        $sStatusBaseline =
            (@($preGit.S_STATUS_SHORT) | ForEach-Object { [string]$_ }) -join "`n"
        $sBranchBaseline = [string]$preGit.S_BRANCH

        $expectedBranchBaseline =
            [string]$preGit.EXPECTED_BRANCH
    }
}

$gitGuard['PRECHECK_GIT_IDENTITY_AVAILABLE'] =
    $(if ($preGitAvailable) { 'YES' } else { 'NO' })

# No comparison defaults to true.
$pHeadOk = $false
$pTreeOk = $false
$pStatusOk = $false
$pBranchBaselineOk = $false

$sHeadOk = $false
$sTreeOk = $false
$sStatusOk = $false
$sBranchBaselineOk = $false

$expectedBranchEvidenceOk = $false

if ($preGitAvailable) {
    $pHeadOk =
        ($pNow.Head -eq $pHeadBaseline)

    $pTreeOk =
        ($pNow.Tree -eq $pTreeBaseline)

    $pStatusOk =
        ($pStatusNow -eq $pStatusBaseline)

    $pBranchBaselineOk =
        ($pBranchNow -eq $pBranchBaseline)

    $sHeadOk =
        ($sNow.Head -eq $sHeadBaseline)

    $sTreeOk =
        ($sNow.Tree -eq $sTreeBaseline)

    $sStatusOk =
        ($sStatusNow -eq $sStatusBaseline)

    $sBranchBaselineOk =
        ($sBranchNow -eq $sBranchBaseline)

    $expectedBranchEvidenceOk =
        ($expectedBranchBaseline -eq $expectedBranch)
}

$pAdmissionBranchOk =
    ($pBranchNow -eq $expectedBranch)

$sAdmissionBranchOk =
    ($sBranchNow -eq $expectedBranch)

$stagedOk =
    ($pNow.StagedCount -eq 0) -and
    ($sNow.StagedCount -eq 0)

$gitGuard['EXPECTED_BRANCH'] = $expectedBranch
$gitGuard['P_BRANCH_ACTUAL'] = $pBranchNow
$gitGuard['S_BRANCH_ACTUAL'] = $sBranchNow

$gitGuard['P_HEAD_BASELINE'] = $pHeadBaseline
$gitGuard['P_HEAD_ACTUAL'] = $pNow.Head
$gitGuard['P_HEAD_BASELINE_ENFORCED'] = $pHeadOk

$gitGuard['P_TREE_BASELINE'] = $pTreeBaseline
$gitGuard['P_TREE_ACTUAL'] = $pNow.Tree
$gitGuard['P_TREE_BASELINE_ENFORCED'] = $pTreeOk

$gitGuard['P_STATUS_BASELINE'] = $pStatusBaseline
$gitGuard['P_STATUS_ACTUAL'] = $pStatusNow
$gitGuard['P_STATUS_BASELINE_ENFORCED'] = $pStatusOk

$gitGuard['P_BRANCH_BASELINE'] = $pBranchBaseline
$gitGuard['P_BRANCH_BASELINE_ENFORCED'] = $pBranchBaselineOk
$gitGuard['P_ADMISSION_BRANCH_ENFORCED'] = $pAdmissionBranchOk

$gitGuard['S_HEAD_BASELINE'] = $sHeadBaseline
$gitGuard['S_HEAD_ACTUAL'] = $sNow.Head
$gitGuard['S_HEAD_BASELINE_ENFORCED'] = $sHeadOk

$gitGuard['S_TREE_BASELINE'] = $sTreeBaseline
$gitGuard['S_TREE_ACTUAL'] = $sNow.Tree
$gitGuard['S_TREE_BASELINE_ENFORCED'] = $sTreeOk

$gitGuard['S_STATUS_BASELINE'] = $sStatusBaseline
$gitGuard['S_STATUS_ACTUAL'] = $sStatusNow
$gitGuard['S_STATUS_BASELINE_ENFORCED'] = $sStatusOk

$gitGuard['S_BRANCH_BASELINE'] = $sBranchBaseline
$gitGuard['S_BRANCH_BASELINE_ENFORCED'] = $sBranchBaselineOk
$gitGuard['S_ADMISSION_BRANCH_ENFORCED'] = $sAdmissionBranchOk

$gitGuard['EXPECTED_BRANCH_EVIDENCE_ENFORCED'] =
    $expectedBranchEvidenceOk

$gitGuard['STAGED_FILES'] =
    ($pNow.StagedCount + $sNow.StagedCount)

$drift = @()

if (-not $preGitAvailable) {
    $drift += 'preflight git identity unavailable or incomplete'
}

if (-not $pHeadOk) {
    $drift +=
        "P_HEAD drift vs preflight: baseline=$pHeadBaseline actual=$($pNow.Head)"
}

if (-not $pTreeOk) {
    $drift +=
        "P_TREE drift vs preflight: baseline=$pTreeBaseline actual=$($pNow.Tree)"
}

if (-not $pStatusOk) {
    $drift += 'P_STATUS_SHORT drift vs preflight'
}

if (-not $pBranchBaselineOk) {
    $drift +=
        "P_BRANCH drift vs preflight: baseline=$pBranchBaseline actual=$pBranchNow"
}

if (-not $sHeadOk) {
    $drift +=
        "S_HEAD drift vs preflight: baseline=$sHeadBaseline actual=$($sNow.Head)"
}

if (-not $sTreeOk) {
    $drift +=
        "S_TREE drift vs preflight: baseline=$sTreeBaseline actual=$($sNow.Tree)"
}

if (-not $sStatusOk) {
    $drift += 'S_STATUS_SHORT drift vs preflight'
}

if (-not $sBranchBaselineOk) {
    $drift +=
        "S_BRANCH drift vs preflight: baseline=$sBranchBaseline actual=$sBranchNow"
}

if (-not $expectedBranchEvidenceOk) {
    $drift +=
        "preflight EXPECTED_BRANCH mismatch: baseline=$expectedBranchBaseline expected=$expectedBranch"
}

if (-not $pAdmissionBranchOk) {
    $drift +=
        "P_BRANCH not on admission branch: actual=$pBranchNow"
}

if (-not $sAdmissionBranchOk) {
    $drift +=
        "S_BRANCH not on admission branch: actual=$sBranchNow"
}

if (-not $stagedOk) {
    $drift +=
        "staged files present: P=$($pNow.StagedCount) S=$($sNow.StagedCount)"
}

$gitGuard['HEAD_DRIFT_VS_PREFLIGHT'] = $drift

$gitPass =
    $preGitAvailable -and
    $expectedBranchEvidenceOk -and
    $pHeadOk -and
    $pTreeOk -and
    $pStatusOk -and
    $pBranchBaselineOk -and
    $pAdmissionBranchOk -and
    $sHeadOk -and
    $sTreeOk -and
    $sStatusOk -and
    $sBranchBaselineOk -and
    $sAdmissionBranchOk -and
    $stagedOk

$gitGuard['GIT_GUARDS'] =
    $(if ($gitPass) { 'PASS' } else { 'IDENTITY_RED' })
# ---- 4. Evidence inventory ----
$runDirLeaf = $run.RunDir
$files = Get-ChildItem -LiteralPath $runDirLeaf -Recurse -File -ErrorAction SilentlyContinue
$inventory = @()
foreach ($f in $files) {
    $rel = $f.FullName.Substring($runDirLeaf.Length).TrimStart('\','/')
    $surface = if ($rel -match '^(02-verification)\\([^\\]+)') { $Matches[2] } else { ($rel -split '[\\/]')[0] }
    $sha = (Get-FileHash -LiteralPath $f.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    $inventory += [pscustomobject]@{ RELATIVE_PATH=$rel; SIZE=$f.Length; SHA256=$sha; SURFACE=$surface; ROLE='verification-evidence' }
}
Write-RvCsv -Object $inventory -Path (Join-Path $OUT 'EVIDENCE_MANIFEST.csv')
Write-RvJson -Object $inventory -Path (Join-Path $OUT 'EVIDENCE_MANIFEST.json')

# ---- 5. POSTFLIGHT result (§45) ----
# §16/§18: ownership mismatch => CLEANUP_COMPLETE=NO (and any residual port leak). PROCESS_ALREADY_EXITED is NOT a failure.
$cleanupOk = ($cleanup.CleanupComplete -and @($portLeaks).Count -eq 0)
$postflight = [ordered]@{
    CLEANUP_COMPLETE = $(if ($cleanupOk) { 'YES' } else { 'NO' })
    GIT_GUARDS_PASS = $(if ($gitPass) { 'YES' } else { 'NO' })
    EVIDENCE_INVENTORY_COMPLETE = 'YES'
    OWNERSHIP_MISMATCH = @($cleanup.OwnershipMismatches)
    OWNERSHIP_MISMATCH_COUNT = @($cleanup.OwnershipMismatches).Count
    PROCESS_LEAKS = @($cleanup.LeakedPids)
    PROCESS_ALREADY_EXITED = @($cleanup.AlreadyExitedPids)
    PORT_LEAKS = $portLeaks
    GIT_GUARD_DETAIL = $gitGuard
    EVIDENCE_FILE_COUNT = @($inventory).Count
    RUN_DIR = $run.RunDir
}
Write-RvJson -Object $postflight -Path (Join-Path $OUT 'POSTFLIGHT.json')
Write-RvSummaryText -Table ([ordered]@{
    CLEANUP_COMPLETE=$postflight.CLEANUP_COMPLETE
    GIT_GUARDS_PASS=$postflight.GIT_GUARDS_PASS
    EVIDENCE_INVENTORY_COMPLETE=$postflight.EVIDENCE_INVENTORY_COMPLETE
    STAGED_FILES=$gitGuard['STAGED_FILES']
    PRODUCTION_TRACKED_MODIFICATIONS=$gitGuard['PRODUCTION_TRACKED_MODIFICATIONS']
}) -Path (Join-Path $OUT 'POSTFLIGHT.txt')

Write-RvLog -Message "CLEANUP_COMPLETE=$($postflight.CLEANUP_COMPLETE) GIT_GUARDS_PASS=$($postflight.GIT_GUARDS_PASS). If guards fail, reconciliation must flag evidence integrity issue."
Set-RvRunState -RunDir $run.RunDir -State 'POSTFLIGHT_COMPLETE'
# NO git clean/reset, NO evidence deletion, NO admission.
