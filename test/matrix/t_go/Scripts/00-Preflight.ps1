#Requires -Version 7.0

<#
00-Preflight.ps1 — READ-ONLY PRE-EXECUTION VERIFICATION (PHASE 0B + global gates).
CLASS = VERIFICATION TOOLING. Does NOT install, build, generate fixtures, or run
tests/runtime. Does NOT auto-invoke the next script. Does NOT admit anything.

Repo identity (S/P/C) + transferred-file check (A/B/C/D/S11/S12) + toolchain
inventory + port state + env-var capture -> preflight-summary.json / .txt.
#>

[CmdletBinding()]
param(
    [string]$SidecarRoot = 'C:\Personal\GitHub\Dev\hacp-sidecar',
    [string]$SpecRoot    = 'C:\Personal\GitHub\Dev\hacp-spec',
    [string]$CoreRoot    = 'C:\Personal\GitHub\Dev\humanist-core',
    [string]$PackageRoot = (Split-Path -Parent $PSScriptRoot),
    [string]$RunRoot     = 'C:\Temp\hacp-1.1.1-verification\run',
    [string]$EvidenceRoot= 'C:\Temp\hacp-1.1.1-verification\evidence',
    [string]$OutputDir   = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Import-Module (Join-Path $PSScriptRoot 'WindowsVerification.Common.psm1') -Force

if (-not $OutputDir) { $OutputDir = Join-Path $EvidenceRoot '00-Preflight' }
New-WvDirectory -Path $OutputDir | Out-Null
Set-WvLogFile -Path (Join-Path $OutputDir 'preflight-console.log')

Write-WvLog -Message "00-Preflight start (READ-ONLY)"

# ---- 1. Repository identity (S/P/C) ----
$repos = [ordered]@{ S = $SidecarRoot; P = $SpecRoot; C = $CoreRoot }
$identity = [ordered]@{}
$stagedTotal = 0
foreach ($k in $repos.Keys) {
    $gi = Get-WvGitIdentity -RepoRoot $repos[$k]
    $identity[$k] = $gi
    if ($gi.Present) {
        $stagedTotal += $gi.StagedCount
        Write-WvLog -Message "$k HEAD=$($gi.Head) tree=$($gi.Tree) staged=$($gi.StagedCount)"
    } else {
        Write-WvLog -Level WARN -Message "$k repo missing at $($repos[$k])"
    }
}
$stagedGate = if ($stagedTotal -eq 0) { 'PASS' } else { 'IDENTITY_RED' }
Write-WvLog -Message "STAGED_FILES_TOTAL=$stagedTotal gate=$stagedGate"

# ---- 2. Transferred file check (A/B/C/D/S11/S12) ----
# expectedSha filled from package record where established (S11/S12); null = record-only.
$transferSpec = @(
    @{ Surface='A';   Repo='S'; Rel='internal/proxy/r4_provenance_verification_test.go';        Expected=$null },
    @{ Surface='B';   Repo='S'; Rel='tools/reproduction/graceful-shutdown/main.go';              Expected=$null },
    @{ Surface='C';   Repo='S'; Rel='docs/verification/rust-core-composition-reproduction.md';   Expected=$null },
    @{ Surface='D';   Repo='S'; Rel='benchmarks/go-http-reproduction/main.go';                   Expected=$null },
    @{ Surface='D';   Repo='S'; Rel='docs/verification/go-http-local-reproduction.md';           Expected=$null },
    @{ Surface='S11'; Repo='S'; Rel='test/matrix/t_go/S11_signature_canonicalization_test.go.overlay'; Expected='3046aecc5d1efa7542d71c4c68eb776e657ff551292a9305847155a367a22364' },
    @{ Surface='S12'; Repo='S'; Rel='test/matrix/t_go/S12_budget_ledger_test.go.overlay'; Expected='f83829ae75add2724f3f2fc2d6cf2f4e3d18c1b252b94fad3bfbbf4995966463' },
    @{ Surface='S12'; Repo='S'; Rel='test/matrix/t_go/S12_budget_replay_verification_test.go.overlay'; Expected='09601447a1c933e152ec2e810dc820d51ffa5bde7c0b57e94211dea418ea6bfa' }
)
$repoRootMap = @{ S = $SidecarRoot; P = $SpecRoot; C = $CoreRoot }
$transferResults = @()
foreach ($t in $transferSpec) {
    $abs = Join-Path $repoRootMap[$t.Repo] ($t.Rel -replace '/', '\')
    $fi = Get-WvFileIdentity -Path $abs
    $gp = Get-WvGitPathState -RepoRoot $repoRootMap[$t.Repo] -RelativePath $t.Rel
    $shaMatch = if ($t.Expected) { ($fi.Sha256 -eq $t.Expected) } else { $null }
    $transferResults += [pscustomobject]@{
        SURFACE_ID      = $t.Surface
        TARGET_REPO     = $t.Repo
        TARGET_PATH     = $t.Rel
        PRESENT         = $fi.Exists
        SIZE_BYTES      = $fi.SizeBytes
        SHA256          = $fi.Sha256
        EXPECTED_SHA256 = $t.Expected
        SHA_MATCH       = $shaMatch
        GIT_STATUS      = $gp.GitStatus
        IGNORED         = $gp.IsIgnored
        CLASSIFICATION  = if (-not $fi.Exists) { 'NOT_RUN' }
                          elseif ($t.Expected -and -not $shaMatch) { 'IDENTITY_RED' }
                          else { 'PASS' }
    }
}
Write-WvJson -Object $transferResults -Path (Join-Path $OutputDir 'transferred-file-check.json')

# ---- 3. Toolchain inventory (presence + version; install nothing) ----
$toolchainSpec = @(
    @{ Name='PowerShell'; Cmd='pwsh';        Args=@('--version') },
    @{ Name='PowerShell5';Cmd='powershell';  Args=@('-NoProfile','-Command','$PSVersionTable.PSVersion.ToString()') },
    @{ Name='git';        Cmd='git';         Args=@('--version') },
    @{ Name='Go';         Cmd='go';          Args=@('version') },
    @{ Name='Python';     Cmd='python';      Args=@('--version') },
    @{ Name='pip';        Cmd='pip';         Args=@('--version') },
    @{ Name='pytest';     Cmd='pytest';      Args=@('--version') },
    @{ Name='Node';       Cmd='node';        Args=@('--version') },
    @{ Name='npm';        Cmd='npm';         Args=@('--version') },
    @{ Name='rustc';      Cmd='rustc';       Args=@('--version') },
    @{ Name='cargo';      Cmd='cargo';       Args=@('--version') },
    @{ Name='rustup';     Cmd='rustup';      Args=@('--version') },
    @{ Name='protoc';     Cmd='protoc';      Args=@('--version') },
    @{ Name='Docker';     Cmd='docker';      Args=@('--version') },
    @{ Name='docker-compose'; Cmd='docker'; Args=@('compose','version') }
)
$toolchain = @()
foreach ($t in $toolchainSpec) {
    $cmd = Get-Command $t.Cmd -ErrorAction SilentlyContinue
    $present = [bool]$cmd
    $version = $null
    if ($present) {
        try {
            $out = & $t.Cmd @($t.Args) 2>&1 | Out-String
            $version = ($out -split "`r?`n")[0].Trim()
        } catch { $version = 'VERSION_QUERY_FAILED' }
    }
    $toolchain += [pscustomobject]@{
        TOOL=$t.Name; COMMAND=$t.Cmd; PRESENT=$present
        SOURCE=$(if ($present) { $cmd.Source } else { $null }); VERSION=$version
    }
}
Write-WvJson -Object $toolchain -Path (Join-Path $OutputDir 'toolchain-inventory.json')

# ---- 4. Port state (never kill) ----
$ports = @(18080,18081,18082,18083,18084,8080,8000)
$portStates = @()
foreach ($p in $ports) {
    $s = Test-WvPortState -Port $p
    $portStates += [pscustomobject]@{
        PORT=$p
        STATE=$(if ($s.State -eq 'FREE') { 'FREE' } else { 'OCCUPIED_UNEXPECTED' })
        OWNER_PIDS=(@($s.OwnerPids) -join ',')
    }
}
Write-WvJson -Object $portStates -Path (Join-Path $OutputDir 'port-state.json')

# ---- 5. Env snapshot (secret-safe) ----
$envSnap = Get-WvEnvSnapshot
Write-WvJson -Object $envSnap -Path (Join-Path $OutputDir 'environment-snapshot.json')

# ---- 6. Summary + readiness ----
$identityRed = ($stagedGate -ne 'PASS')
$transferRed = @($transferResults | Where-Object { $_.CLASSIFICATION -eq 'IDENTITY_RED' }).Count
$missingCriticalTools = @($toolchain | Where-Object { $_.TOOL -in @('git','Go') -and -not $_.PRESENT }).Count
$preflightReady = if ($identityRed -or $transferRed -gt 0 -or $missingCriticalTools -gt 0) { 'NO' } else { 'YES' }

$summary = [ordered]@{
    PREFLIGHT_READY            = $preflightReady
    STAGED_FILES_TOTAL         = $stagedTotal
    STAGED_GATE                = $stagedGate
    TRANSFER_IDENTITY_RED_ROWS = $transferRed
    MISSING_CRITICAL_TOOLS     = $missingCriticalTools
    S_HEAD                     = $identity['S'].Head
    P_HEAD                     = $identity['P'].Head
    C_HEAD                     = $identity['C'].Head
    OUTPUT_DIR                 = $OutputDir
}
Write-WvJson -Object $summary -Path (Join-Path $OutputDir 'preflight-summary.json')
Write-WvSummaryText -Table $summary -Path (Join-Path $OutputDir 'preflight-summary.txt')

Write-WvLog -Message "PREFLIGHT_READY=$preflightReady (script does NOT auto-run 01; human triggers next step)"
Write-WvLog -Message "00-Preflight complete. Review $OutputDir"
# NO admission, NO staging, NO commit.
