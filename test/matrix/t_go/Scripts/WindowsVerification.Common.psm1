Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# WindowsVerification.Common.psm1
# CLASS = VERIFICATION / EXECUTION TOOLING (not production semantics).
# Shared helpers for the HACP 1.1.1 Windows verification orchestration.
# Executable representation of the approved contracts (WINDOWS_EXECUTION_MANIFEST.md,
# CONTRACTS_S01_S12.md, Readme_map_file.md). NO new verification semantics, NO matrix
# scope expansion, NO admission automation, NO acceptance automation.

# ---- Classification model (§9) ----
New-Variable -Name WV_CLASS -Option ReadOnly -Force -Value @{
    PASS            = 'PASS'
    ENVIRONMENT_RED = 'ENVIRONMENT_RED'
    BEHAVIORAL_RED  = 'BEHAVIORAL_RED'
    IDENTITY_RED    = 'IDENTITY_RED'
    BLOCKED         = 'BLOCKED'
    NOT_RUN         = 'NOT_RUN'
    INFORMATIONAL   = 'INFORMATIONAL'
}

function Write-WvLog {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$Message,
        [ValidateSet('INFO','WARN','ERROR','STEP','OK','RED')][string]$Level = 'INFO'
    )
    $ts = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
    $line = "[$ts][$Level] $Message"
    Write-Host $line
    if (Get-Variable -Name WV_LOGFILE -Scope Global -ErrorAction SilentlyContinue) {
        if ($Global:WV_LOGFILE) { Add-Content -LiteralPath $Global:WV_LOGFILE -Value $line }
    }
}

function Set-WvLogFile {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$Path)
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path -LiteralPath $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    Set-Variable -Name WV_LOGFILE -Scope Global -Value $Path -Force
}

function New-WvDirectory {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
    return (Resolve-Path -LiteralPath $Path).Path
}

# ---- Machine-readable writers ----
function Write-WvJson {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)]$Object,
        [Parameter(Mandatory)][string]$Path
    )
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    $Object | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $Path -Encoding UTF8
}

function Write-WvJsonl {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][array]$Objects,
        [Parameter(Mandatory)][string]$Path
    )
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    $lines = @($Objects | ForEach-Object { $_ | ConvertTo-Json -Depth 8 -Compress })
    $lines | Set-Content -LiteralPath $Path -Encoding UTF8
}

function Write-WvCsv {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][array]$Objects,
        [Parameter(Mandatory)][string]$Path
    )
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    $Objects | Export-Csv -LiteralPath $Path -NoTypeInformation -Encoding UTF8
}

function Write-WvSummaryText {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][hashtable]$Table,
        [Parameter(Mandatory)][string]$Path
    )
    $lines = foreach ($k in ($Table.Keys | Sort-Object)) { "{0}={1}" -f $k, $Table[$k] }
    $lines | Set-Content -LiteralPath $Path -Encoding UTF8
}

# ---- Identity / hashing ----
function Get-WvFileIdentity {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$Path)
    $exists = Test-Path -LiteralPath $Path -PathType Leaf
    $sha = $null; $size = $null; $item = $null
    if ($exists) {
        $item = Get-Item -LiteralPath $Path
        $size = $item.Length
        $sha = (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
    }
    [pscustomobject]@{
        Path      = $Path
        Exists    = $exists
        SizeBytes = $size
        Sha256    = $sha
    }
}

function Get-WvGitIdentity {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$RepoRoot)
    if (-not (Test-Path -LiteralPath $RepoRoot -PathType Container)) {
        return [pscustomobject]@{ Root=$RepoRoot; Present=$false }
    }
    Push-Location -LiteralPath $RepoRoot
    try {
        $branch = (& git branch --show-current 2>$null | Out-String).Trim()
        $head   = (& git rev-parse HEAD 2>$null | Out-String).Trim()
        $tree   = (& git rev-parse 'HEAD^{tree}' 2>$null | Out-String).Trim()
        $status = @(& git status --short 2>$null | Out-String).Trim()
        $stagedRaw = @(& git diff --cached --name-only 2>$null)
        $staged = @($stagedRaw | Where-Object { $_ -and $_.Trim() })
        [pscustomobject]@{
            Root          = $RepoRoot
            Present       = $true
            Branch        = $branch
            Head          = $head
            Tree          = $tree
            StatusShort   = $status
            StagedCount   = $staged.Count
            StagedFiles   = $staged
        }
    } finally {
        Pop-Location
    }
}

function Get-WvGitPathState {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$RepoRoot,
        [Parameter(Mandatory)][string]$RelativePath
    )
    Push-Location -LiteralPath $RepoRoot
    try {
        $status = (& git status --short -- $RelativePath 2>$null | Out-String).Trim()
        $ignored = (& git check-ignore -v -- $RelativePath 2>$null | Out-String).Trim()
        [pscustomobject]@{
            RelativePath = $RelativePath
            GitStatus    = $status
            CheckIgnore  = $ignored
            IsIgnored    = [bool]$ignored
        }
    } finally {
        Pop-Location
    }
}

# ---- Ports ----
function Test-WvPortState {
    [CmdletBinding()]
    param([Parameter(Mandatory)][int]$Port)
    $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    if ($conn) {
        $pids = @($conn | Select-Object -ExpandProperty OwningProcess -Unique)
        return [pscustomobject]@{ Port=$Port; State='OCCUPIED'; OwnerPids=$pids }
    }
    [pscustomobject]@{ Port=$Port; State='FREE'; OwnerPids=@() }
}

function Test-WvPortOwned {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][int]$Port,
        [Parameter(Mandatory)][int]$OwningPid
    )
    $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    if (-not $conn) { return $false }
    $pids = @($conn | Select-Object -ExpandProperty OwningProcess -Unique)
    return ($pids -contains $OwningPid)
}

function Test-WvPortAbsent {
    [CmdletBinding()]
    param([Parameter(Mandatory)][int]$Port)
    $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    return (-not $conn)
}

# ---- Environment snapshot (secret-safe) ----
function Get-WvEnvSnapshot {
    [CmdletBinding()]
    param()
    $publicVars = @('HTTP_PROXY','HTTPS_PROXY','ALL_PROXY','NO_PROXY','HACP_TRUST_MODE','HACP_CONTROL_MODE','HACP_SIDECAR_PORT','HACP_UPSTREAM','HACP_SIDECAR_REPO','HACP_SIDECAR_EXTERNAL','HACP_SIDECAR_URL','HACP_TIME_TOLERANCE')
    $secretPathVars = @('HACP_TRUST_KEYS_FILE','HACP_SIDECAR_BIN','HACP_TEST_PRIVATE_KEY')
    $out = [ordered]@{}
    foreach ($v in $publicVars) {
        $val = [Environment]::GetEnvironmentVariable($v)
        $out[$v] = if ($null -eq $val) { 'UNSET' } elseif ($val -eq '') { 'EMPTY' } else { $val }
    }
    foreach ($v in $secretPathVars) {
        $val = [Environment]::GetEnvironmentVariable($v)
        if ($null -eq $val -or $val -eq '') {
            $out[$v] = 'UNSET'
        } else {
            $out["$v`_SET"] = 'SET'
            $out["$v`_FILE_EXISTS"] = (Test-Path -LiteralPath $val)
        }
    }
    $kId = [Environment]::GetEnvironmentVariable('HACP_TEST_SIGNER_KEY_ID')
    $out['HACP_TEST_SIGNER_KEY_ID'] = if ($null -eq $kId -or $kId -eq '') { 'UNSET' } else { 'SET' }
    [pscustomobject]$out
}

# ---- Native command execution (captures stdout/stderr/RC; never silently loses RC) ----
function Invoke-WvNative {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$FilePath,
        [string[]]$ArgumentList = @(),
        [Parameter(Mandatory)][string]$WorkDir,
        [Parameter(Mandatory)][string]$StepId,
        [Parameter(Mandatory)][string]$SurfaceId,
        [Parameter(Mandatory)][string]$EvidenceDir,
        [string]$DisplayName = ''
    )
    $stdoutPath = Join-Path $EvidenceDir "$StepId.stdout.log"
    $stderrPath = Join-Path $EvidenceDir "$StepId.stderr.log"
    $cmdPath    = Join-Path $EvidenceDir "$StepId.command.txt"
    $rcPath     = Join-Path $EvidenceDir "$StepId.exit-code.txt"
    $cmdDisplay = if ($DisplayName) { $DisplayName } else { "$FilePath $($ArgumentList -join ' ')" }

    # A previous interrupted execution must never leave a numeric RC
    # that can be mistaken for the current invocation.
    Remove-Item -LiteralPath $rcPath -Force -ErrorAction SilentlyContinue

    Set-Content -LiteralPath $cmdPath -Value $cmdDisplay -Encoding UTF8

    $start = (Get-Date).ToUniversalTime()
    Write-WvLog -Level STEP -Message "RUN $SurfaceId/$StepId : $cmdDisplay"
    $rc = $null
    try {
        $p = Start-Process -FilePath $FilePath -ArgumentList $ArgumentList -WorkingDirectory $WorkDir `
                -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath -NoNewWindow -PassThru
        $p.WaitForExit()
        $rc = $p.ExitCode
    } catch {
        $rc = -1
        Add-Content -LiteralPath $stderrPath -Value ("LAUNCH_ERROR: " + $_.Exception.Message)
    }
    $end = (Get-Date).ToUniversalTime()
    Set-Content -LiteralPath $rcPath -Value "$rc" -Encoding UTF8

    [pscustomobject]@{
        StepId        = $StepId
        SurfaceId     = $SurfaceId
        Command       = $cmdDisplay
        StartTimeUtc  = $start.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
        EndTimeUtc    = $end.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
        ProcessRc     = $rc
        StdOutPath    = $stdoutPath
        StdErrPath    = $stderrPath
        EvidencePath  = $EvidenceDir
    }
}

# ---- Process ownership (only owned PIDs ever touched) ----
function Get-WvPidRegistryPath {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$RunRoot)
    return (Join-Path $RunRoot 'pid-registry.jsonl')
}

function Register-WvOwnedProcess {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$RunRoot,
        [Parameter(Mandatory)][int]$Pid,
        [Parameter(Mandatory)][string]$Command,
        [Parameter(Mandatory)][string]$SurfaceOwner,
        [int]$Port = 0
    )
    $path = Get-WvPidRegistryPath -RunRoot $RunRoot
    $rec = [pscustomobject]@{
        Pid          = $Pid
        Command      = $Command
        StartTimeUtc = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
        SurfaceOwner = $SurfaceOwner
        Port         = $Port
    }
    Add-Content -LiteralPath $path -Value ($rec | ConvertTo-Json -Depth 4 -Compress) -Encoding UTF8
    return $rec
}

function Get-WvOwnedProcesses {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$RunRoot)
    $path = Get-WvPidRegistryPath -RunRoot $RunRoot
    if (-not (Test-Path -LiteralPath $path)) { return @() }
    @(Get-Content -LiteralPath $path | Where-Object { $_.Trim() } | ForEach-Object { $_ | ConvertFrom-Json })
}

function Stop-WvOwnedProcesses {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$RunRoot)
    $owned = Get-WvOwnedProcesses -RunRoot $RunRoot
    $stopped = @(); $leaked = @()
    foreach ($rec in $owned) {
        $proc = Get-Process -Id $rec.Pid -ErrorAction SilentlyContinue
        if ($proc) {
            try { Stop-Process -Id $rec.Pid -Force -ErrorAction Stop; $stopped += $rec.Pid }
            catch { $leaked += $rec.Pid }
        }
    }
    [pscustomobject]@{ StoppedPids = $stopped; LeakedPids = $leaked }
}

# ---- Surface evidence + result model ----
function New-WvSurfaceEvidenceDir {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$EvidenceRoot,
        [Parameter(Mandatory)][string]$SurfaceId
    )
    $d = Join-Path $EvidenceRoot $SurfaceId
    return (New-WvDirectory -Path $d)
}

function New-WvStepResult {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$StepId,
        [Parameter(Mandatory)][string]$SurfaceId,
        [Parameter(Mandatory)][string]$Command,
        [Parameter(Mandatory)][string]$StartTimeUtc,
        [Parameter(Mandatory)][string]$EndTimeUtc,
        [Parameter(Mandatory)][AllowNull()]$ProcessRc,
        [Parameter(Mandatory)][string]$Classification,
        [Parameter(Mandatory)][string]$EvidencePath,
        [string]$Status = ''
    )
    [pscustomobject]@{
        STEP_ID       = $StepId
        SURFACE_ID    = $SurfaceId
        COMMAND       = $Command
        START_TIME_UTC= $StartTimeUtc
        END_TIME_UTC  = $EndTimeUtc
        PROCESS_RC    = $ProcessRc
        CLASSIFICATION= $Classification
        EVIDENCE_PATH = $EvidencePath
        STATUS        = $Status
    }
}

function Resolve-WvStepClassification {
    # Map a native run + optional expected-marker check to a classification.
    # NOTE: this does NOT invent new acceptance criteria; it only maps RC/output
    # to the approved classification model (§9). Expectations come from contracts.
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)]$Run,
        [string[]]$RequiredMarkers = @(),
        [string[]]$ForbiddenMarkers = @(),
        [switch]$ExpectedNonZero   # O-1: scoped to C — a sanctioned expected-nonzero RC is not RED
    )
    if ($Run.ProcessRc -eq $null) { return $WV_CLASS.ENVIRONMENT_RED }
    if ($Run.ProcessRc -ne 0) {
        # RC!=0 → inspect output: compile/toolchain/setup => ENVIRONMENT_RED; an
        # executed assertion failure => BEHAVIORAL_RED. Without deeper parsing we
        # surface it as ENVIRONMENT_RED unless a forbidden behavior marker is seen.
        foreach ($m in $ForbiddenMarkers) {
            if (Select-String -LiteralPath $Run.StdOutPath -Pattern ([regex]::Escape($m)) -Quiet) {
                return $WV_CLASS.BEHAVIORAL_RED
            }
        }
        if ($ExpectedNonZero) {
            # O-1 (scoped to C only): a sanctioned expected-nonzero RC. The approved state requires the
            # strict witness (required marker) present. Witness present + nonzero = approved expected-nonzero
            # C state (36/38 direct + 2 adjudicated; NOT 38/38). Witness missing = unexpected nonzero => RED.
            $witness = $true
            foreach ($m in $RequiredMarkers) {
                if (-not (Select-String -LiteralPath $Run.StdOutPath -Pattern ([regex]::Escape($m)) -Quiet)) { $witness = $false }
            }
            if ($witness) { return $WV_CLASS.PASS }
        }
        return $WV_CLASS.ENVIRONMENT_RED
    }
    foreach ($m in $ForbiddenMarkers) {
        if (Select-String -LiteralPath $Run.StdOutPath -Pattern ([regex]::Escape($m)) -Quiet) {
            return $WV_CLASS.BEHAVIORAL_RED
        }
    }
    foreach ($m in $RequiredMarkers) {
        if (-not (Select-String -LiteralPath $Run.StdOutPath -Pattern ([regex]::Escape($m)) -Quiet)) {
            return $WV_CLASS.ENVIRONMENT_RED
        }
    }
    return $WV_CLASS.PASS
}

function Save-WvEnvironment {
    # D-3: snapshot process env for the given names. Records SET vs UNSET (UNSET restored as UNSET).
    # Values are captured for restoration only and are NEVER logged (§27 secret safety).
    [CmdletBinding()] param([Parameter(Mandatory)][string[]]$Names)
    $saved = @{}
    foreach ($n in $Names) {
        $v = [Environment]::GetEnvironmentVariable($n)   # $null if UNSET; value (incl '') if SET
        $saved[$n] = @{ Set = ($null -ne $v); Value = $v }
    }
    return $saved
}
function Restore-WvEnvironment {
    # D-3: restore a Save-WvEnvironment snapshot. SET values restored as-is; UNSET removed (not '').
    [CmdletBinding()] param([Parameter(Mandatory)]$Saved)
    foreach ($n in @($Saved.Keys)) {
        if ($Saved[$n].Set) { [Environment]::SetEnvironmentVariable($n, $Saved[$n].Value) }
        else { [Environment]::SetEnvironmentVariable($n, $null) }
    }
}

Export-ModuleMember -Function @(
    'Write-WvLog','Set-WvLogFile','New-WvDirectory',
    'Write-WvJson','Write-WvJsonl','Write-WvCsv','Write-WvSummaryText',
    'Get-WvFileIdentity','Get-WvGitIdentity','Get-WvGitPathState',
    'Test-WvPortState','Test-WvPortOwned','Test-WvPortAbsent',
    'Get-WvEnvSnapshot',
    'Invoke-WvNative',
    'Register-WvOwnedProcess','Get-WvOwnedProcesses','Stop-WvOwnedProcesses',
    'New-WvSurfaceEvidenceDir','New-WvStepResult','Resolve-WvStepClassification',
    'Save-WvEnvironment','Restore-WvEnvironment'
) -Variable @('WV_CLASS')
