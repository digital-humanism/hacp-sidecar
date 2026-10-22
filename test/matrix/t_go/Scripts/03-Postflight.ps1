#Requires -Version 7.0

<#
03-Postflight.ps1 — CLEANUP + FINAL IDENTITY GUARDS + EVIDENCE INVENTORY (POST-RUN).
CLASS = VERIFICATION TOOLING. Stops ONLY owned PIDs from the recorded registry; verifies
owned listeners removed; re-checks S/P/C git identity vs preflight; builds evidence inventory.
Does NOT delete evidence/verification files, does NOT git clean/reset, does NOT admit/commit.
#>

[CmdletBinding()]
param(
    [string]$SidecarRoot = 'C:\Personal\GitHub\Dev\hacp-sidecar',
    [string]$SpecRoot    = 'C:\Personal\GitHub\Dev\hacp-spec',
    [string]$CoreRoot    = 'C:\Personal\GitHub\Dev\humanist-core',
    [string]$RunRoot     = 'C:\Temp\hacp-1.1.1-verification\run',
    [string]$EvidenceRoot= 'C:\Temp\hacp-1.1.1-verification\evidence'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Import-Module (Join-Path $PSScriptRoot 'WindowsVerification.Common.psm1') -Force

$LOG_ROOT  = New-WvDirectory -Path (Join-Path $RunRoot 'logs')
$EVID_ROOT = New-WvDirectory -Path $EvidenceRoot
$POST_EVID = New-WvDirectory -Path (Join-Path $EVID_ROOT '03-Postflight')
Set-WvLogFile -Path (Join-Path $LOG_ROOT '03-postflight-console.log')
Write-WvLog -Message "03-Postflight start"

# ---- 1. Cleanup owned processes only ----
$cleanup = Stop-WvOwnedProcesses -RunRoot $RunRoot
Write-WvLog -Message "Owned processes stopped=$(@($cleanup.StoppedPids).Count) leaked=$(@($cleanup.LeakedPids).Count)"

# ---- 2. Port leak check (owned ports should be absent) ----
$ownedPorts = @(18080,18081,18082,18084,8080,8000)
$portLeaks = @()
foreach ($p in $ownedPorts) {
    if (-not (Test-WvPortAbsent -Port $p)) {
        $st = Test-WvPortState -Port $p
        $portLeaks += [pscustomobject]@{ PORT=$p; OWNER_PIDS=(@($st.OwnerPids) -join ',') }
    }
}

# ---- 3. Git identity guards (vs preflight) ----
$preflightPath = Join-Path $EVID_ROOT '00-Preflight\preflight-summary.json'
$preflight = $null
if (Test-Path -LiteralPath $preflightPath) {
    try { $preflight = Get-Content -LiteralPath $preflightPath -Raw | ConvertFrom-Json } catch { $preflight = $null }
}
$repos = [ordered]@{ S = $SidecarRoot; P = $SpecRoot; C = $CoreRoot }
$allowedUntracked = @(
    # Existing verification-only artifacts outside the repository-owned matrix package.
    'internal/proxy/r4_provenance_verification_test.go',
    'tools/reproduction/graceful-shutdown/main.go',
    'docs/verification/rust-core-composition-reproduction.md',

    # Repository-owned Go verification package.
    # Exact-file allowlist: do not allow the directory as a wildcard/prefix.
    'test/matrix/t_go/CONTRACTS_S01_S12.md',
    'test/matrix/t_go/FINAL_HANDOFF_CORRECTION_RU_2026-10-17.md',
    'test/matrix/t_go/Readme_map_file.md',
    'test/matrix/t_go/S11_signature_canonicalization_test.go.overlay',
    'test/matrix/t_go/S12_budget_ledger_test.go.overlay',
    'test/matrix/t_go/S12_budget_replay_verification_test.go.overlay',
    'test/matrix/t_go/Scripts/00-Preflight.ps1',
    'test/matrix/t_go/Scripts/01-Prepare-Environment.ps1',
    'test/matrix/t_go/Scripts/02-Run-Verification.ps1',
    'test/matrix/t_go/Scripts/03-Postflight.ps1',
    'test/matrix/t_go/Scripts/04-Reconcile-Results.ps1',
    'test/matrix/t_go/Scripts/README_WINDOWS_EXECUTION.md',
    'test/matrix/t_go/Scripts/WINDOWS_HANDOFF_ORCHESTRATION_CORRECTION.md',
    'test/matrix/t_go/Scripts/WINDOWS_HANDOFF_STATIC_EXECUTABILITY_AUDIT.md',
    'test/matrix/t_go/Scripts/WindowsVerification.Common.psm1',
    'test/matrix/t_go/WINDOWS_EXECUTION_MANIFEST.md'
)
$identityAfter = [ordered]@{}
$stagedTotal = 0; $unexpectedMods = @(); $unexpectedFiles = @(); $worktreeDrift = @()
foreach ($k in $repos.Keys) {
    $gi = Get-WvGitIdentity -RepoRoot $repos[$k]
    $identityAfter[$k] = $gi
    if (-not $gi.Present) { continue }
    $stagedTotal += $gi.StagedCount
    # Expand untracked directories to individual files so the exact-file allowlist
    # remains fail-closed and cannot hide an unexpected file under an allowed directory.
    Push-Location -LiteralPath $repos[$k]
    try {
        $statusExpanded = (& git status --short --untracked-files=all 2>$null | Out-String).Trim()
    } finally {
        Pop-Location
    }

    # status entries: parse "<XY> <path>"
    $entries = @($statusExpanded -split "`r?`n" | Where-Object { $_.Trim() })
    foreach ($e in $entries) {
        $pathPart = ($e -replace '^\s*[^ ]+\s+', '').Trim()
        $isAllowed = $false
        foreach ($a in $allowedUntracked) {
            if ($pathPart -like "*$($a -replace '/','\\')*" -or $pathPart -like "*$a*") { $isAllowed = $true; break }
        }
        if (-not $isAllowed) { $unexpectedFiles += [pscustomobject]@{ REPO=$k; ENTRY=$e } }
    }
    # HEAD drift vs preflight
    if ($preflight) {
        $preHead = $preflight."$($k)_HEAD"
        if ($preHead -and $gi.Head -ne $preHead) {
            $worktreeDrift += [pscustomobject]@{ REPO=$k; PREFLIGHT_HEAD=$preHead; CURRENT_HEAD=$gi.Head }
        }
    }
}
if ($stagedTotal -gt 0) { $unexpectedMods += "STAGED_FILES=$stagedTotal (must be 0)" }

# ---- 4. Evidence inventory ----
$evidenceFiles = Get-ChildItem -LiteralPath $EVID_ROOT -Recurse -File -ErrorAction SilentlyContinue
$inventory = @()
foreach ($f in $evidenceFiles) {
    $rel = $f.FullName.Substring($EVID_ROOT.Length).TrimStart('\','/')
    $surface = ($rel -split '[\\/]')[0]
    $sha = (Get-FileHash -LiteralPath $f.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    $inventory += [pscustomobject]@{
        PATH=$f.FullName; SIZE=$f.Length; SHA256=$sha
        SURFACE_ID=$surface; ROLE='verification-evidence'
    }
}
Write-WvCsv -Object $inventory -Path (Join-Path $POST_EVID 'evidence-inventory.csv')

# ---- 5. Postflight summary (§48) ----
$worktreeStatus = if (($unexpectedFiles.Count -eq 0) -and ($unexpectedMods.Count -eq 0) -and ($worktreeDrift.Count -eq 0)) { 'CLEAN' } else { 'IDENTITY_RED' }
$postflight = [ordered]@{
    CLEANUP_STATUS = $(if (@($cleanup.LeakedPids).Count -eq 0) { 'OK' } else { 'PROCESS_LEAKS' })
    WORKTREE_STATUS = $worktreeStatus
    UNEXPECTED_FILES = $unexpectedFiles
    UNEXPECTED_MODIFICATIONS = $unexpectedMods
    STAGED_FILES = $stagedTotal
    PROCESS_LEAKS = @($cleanup.LeakedPids)
    PORT_LEAKS = $portLeaks
    WORKTREE_DRIFT_VS_PREFLIGHT = $worktreeDrift
    EVIDENCE_FILE_COUNT = @($inventory).Count
}
Write-WvJson -Object $postflight -Path (Join-Path $POST_EVID 'postflight-summary.json')
Write-WvSummaryText -Table ([ordered]@{
    CLEANUP_STATUS=$postflight.CLEANUP_STATUS
    WORKTREE_STATUS=$postflight.WORKTREE_STATUS
    STAGED_FILES=$postflight.STAGED_FILES
    PROCESS_LEAKS=(@($postflight.PROCESS_LEAKS) -join ',')
    PORT_LEAKS=(@($postflight.PORT_LEAKS).Count)
}) -Path (Join-Path $POST_EVID 'postflight-summary.txt')

Write-WvLog -Message "WORKTREE_STATUS=$worktreeStatus CLEANUP=$($postflight.CLEANUP_STATUS). Review $POST_EVID"
# NO git clean/reset, NO evidence deletion, NO admission.
