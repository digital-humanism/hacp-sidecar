Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# WindowsVerification.Rust.Common.psm1
# CLASS = VERIFICATION / EXECUTION TOOLING (not production semantics).
# Self-contained helpers for the HACP 1.1.1 RUST Windows verification orchestration.
# Executable representation of the approved Rust contracts (WINDOWS_EXECUTION_MANIFEST_RS.md,
# RUST_VERIFICATION_CONTRACTS.md, RUST_MATRIX_RECONCILIATION.md). NO new semantics, NO matrix
# scope expansion, NO admission/acceptance automation.

# ---- Classification model (§6): ENVIRONMENT_RED / BEHAVIORAL_RED / PASS / BLOCKED ----
New-Variable -Name RV_CLASS -Option ReadOnly -Force -Value @{
    PASS            = 'PASS'
    ENVIRONMENT_RED = 'ENVIRONMENT_RED'
    BEHAVIORAL_RED  = 'BEHAVIORAL_RED'
BLOCKED         = 'BLOCKED'
    NOT_RUN         = 'NOT_RUN'
    INFORMATIONAL   = 'INFORMATIONAL'
}

function Write-RvLog {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$Message,
        [ValidateSet('INFO','WARN','ERROR','STEP','OK','RED')][string]$Level = 'INFO'
    )
    $ts = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
    $line = "[$ts][$Level] $Message"
    Write-Host $line
    if (Get-Variable -Name RV_LOGFILE -Scope Global -ErrorAction SilentlyContinue) {
        if ($Global:RV_LOGFILE) { Add-Content -LiteralPath $Global:RV_LOGFILE -Value $line }
    }
}

function Set-RvLogFile {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$Path)
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    Set-Variable -Name RV_LOGFILE -Scope Global -Value $Path -Force
}

function New-RvDirectory {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) { New-Item -ItemType Directory -Path $Path -Force | Out-Null }
    return (Resolve-Path -LiteralPath $Path).Path
}

function Get-RvRunContext {
    # R1: safe run-context reader. Returns Raw + RunState + Valid; $null when absent/unparseable.
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$RunDir)
    $ctx = Join-Path $RunDir 'metadata\run-context.json'
    if (-not (Test-Path -LiteralPath $ctx -PathType Leaf)) { return $null }
    try {
        $j = Get-Content -LiteralPath $ctx -Raw | ConvertFrom-Json
        $state = $null
        foreach ($pp in $j.PSObject.Properties) { if ($pp.Name -ieq 'RUN_STATE') { $state = $pp.Value } }
        return [pscustomobject]@{ Raw=$j; RunState=$state; Valid=[bool]($j.RunId -or $j.RunDir) }
    } catch { return $null }
}
function Set-RvRunState {
    # R1: advance the explicit run lifecycle marker (RUN_STATE) in run-context.json.
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$RunDir,[Parameter(Mandatory)][string]$State)
    $ctx = Join-Path $RunDir 'metadata\run-context.json'
    if (-not (Test-Path -LiteralPath $ctx -PathType Leaf)) { throw "RUN_CONTEXT_MISSING: $ctx" }
    $j = Get-Content -LiteralPath $ctx -Raw | ConvertFrom-Json
    $props = [ordered]@{}
    foreach ($pp in $j.PSObject.Properties) { $props[$pp.Name] = $pp.Value }
    $props['RUN_STATE'] = $State
    $props['RUN_STATE_UPDATED_UTC'] = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
    Write-RvJson -Object $props -Path $ctx
}
function Test-RvEligibleRun {
    # §8 active-run definition: valid run-context.json AND phase-compatible lifecycle state.
    # A completed (RECONCILED) run is NEVER phase-compatible, so it never blocks a new run (§8).
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$RunDir,[string]$RequireState = '')
    $c = Get-RvRunContext -RunDir $RunDir
    if ($null -eq $c -or -not $c.Valid) { return $false }
    if ($RequireState) { return ($c.RunState -eq $RequireState) }
    return $true
}

function Resolve-RvRunDir {
    # R1 run-selection semantics. NO latest-run guessing.
    #   CREATE_NEW     (00 without -RunDir): ALWAYS create a NEW run identity.
    #   AUTO_RESOLVE   (01-04 without -RunDir): require EXACTLY ONE eligible active run.
    #   Explicit -RunDir always wins (exact path; NO auto-substitution / NO latest replacement).
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$EvidenceRoot,
        [string]$RunDir = '',
        [ValidateSet('CREATE_NEW','AUTO_RESOLVE')][string]$Mode = 'AUTO_RESOLVE',
        [string]$RequireState = ''   # R1: phase-compatible lifecycle state for 01-04
    )
    $subs = @('00-preflight','01-prepare','02-verification','03-postflight','04-reconciliation','metadata')
    if ($RunDir) {
        if (-not (Test-Path -LiteralPath $RunDir -PathType Container)) { throw "EXPLICIT_RUNDIR_INVALID: $RunDir" }
        # R1 §9: even an explicit -RunDir must be phase-compatible.
        if ($RequireState) {
            $c = Get-RvRunContext -RunDir $RunDir
            $st = if ($c) { $c.RunState } else { $null }
            if ($st -ne $RequireState) { throw "RUN_STATE_INCOMPATIBLE: run state '$st' != required '$RequireState'; REQUIRE a phase-compatible run" }
        }
        foreach ($sub in $subs) { New-RvDirectory -Path (Join-Path $RunDir $sub) | Out-Null }
        return [pscustomobject]@{ RunDir=$RunDir; RunId=(Split-Path -Leaf $RunDir); RunSelectionMode='EXPLICIT_RUNDIR' }
    }
    if ($Mode -eq 'CREATE_NEW') {
        $new = New-RvRunDirectory -EvidenceRoot $EvidenceRoot
        return [pscustomobject]@{ RunDir=$new.RunDir; RunId=$new.RunId; RunSelectionMode='NEW_EXPLICIT_CREATION' }
    }
    # AUTO_RESOLVE: first bind to the run explicitly published by the latest
    # 00-Preflight invocation. Historical unfinished evidence runs remain
    # preserved but do not compete with the current execution contour.
    $currentRunPointer = Join-Path (Split-Path -Parent $EvidenceRoot) 'current-run.txt'

    if (Test-Path -LiteralPath $currentRunPointer -PathType Leaf) {
        $currentRunDir = (Get-Content -LiteralPath $currentRunPointer -Raw).Trim()

        if ($currentRunDir) {
            $expectedEvidenceRoot =
                [System.IO.Path]::GetFullPath($EvidenceRoot).TrimEnd('\')

            $resolvedCurrentRun =
                [System.IO.Path]::GetFullPath($currentRunDir).TrimEnd('\')

            $underEvidenceRoot =
                $resolvedCurrentRun.StartsWith(
                    $expectedEvidenceRoot + '\',
                    [System.StringComparison]::OrdinalIgnoreCase
                )

            if (-not $underEvidenceRoot) {
                throw "CURRENT_RUN_POINTER_OUTSIDE_EVIDENCE_ROOT: $resolvedCurrentRun"
            }

            if (-not (Test-Path -LiteralPath $resolvedCurrentRun -PathType Container)) {
                throw "CURRENT_RUN_POINTER_INVALID: $resolvedCurrentRun"
            }

            $currentContext = Get-RvRunContext -RunDir $resolvedCurrentRun

            if ($null -eq $currentContext -or -not $currentContext.Valid) {
                throw "CURRENT_RUN_CONTEXT_INVALID: $resolvedCurrentRun"
            }

            if ($RequireState -and $currentContext.RunState -ne $RequireState) {
                throw "CURRENT_RUN_STATE_INCOMPATIBLE: run state '$($currentContext.RunState)' != required '$RequireState'; run=$resolvedCurrentRun"
            }

            foreach ($sub in $subs) {
                New-RvDirectory -Path (Join-Path $resolvedCurrentRun $sub) | Out-Null
            }

            return [pscustomobject]@{
                RunDir = $resolvedCurrentRun
                RunId = (Split-Path -Leaf $resolvedCurrentRun)
                RunSelectionMode = 'CURRENT_RUN_POINTER'
            }
        }
    }

    # Compatibility fallback for evidence roots created before current-run.txt.
    # EXACTLY ONE phase-compatible run is still required in that case.
    $eligible = @(Get-ChildItem -LiteralPath $EvidenceRoot -Directory -ErrorAction SilentlyContinue |
        Where-Object { Test-RvEligibleRun -RunDir $_.FullName -RequireState $RequireState })
    if ($eligible.Count -eq 0) { throw "ZERO_ACTIVE_RUNS: 0 phase-compatible active runs (require state '$RequireState'); REQUIRE explicit -RunDir" }
    if ($eligible.Count -gt 1) { throw "MULTIPLE_ACTIVE_RUNS($($eligible.Count)): REQUIRE explicit -RunDir" }
    $one = $eligible[0]
    foreach ($sub in $subs) { New-RvDirectory -Path (Join-Path $one.FullName $sub) | Out-Null }
    return [pscustomobject]@{ RunDir=$one.FullName; RunId=$one.Name; RunSelectionMode='UNIQUE_ACTIVE_AUTO_RESOLVE' }
}

function New-RvRunDirectory {
    # <EvidenceRoot>\<timestamp>_<short-run-id>\ with the required subfolders (§8).
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$EvidenceRoot,[string]$RunId = '')
    if (-not $RunId) { $RunId = ([guid]::NewGuid().ToString('N').Substring(0,8)) }
    $stamp = (Get-Date).ToUniversalTime().ToString('yyyyMMdd-HHmmss')
    $runDir = New-RvDirectory -Path (Join-Path $EvidenceRoot "$stamp`_$RunId")
    foreach ($sub in @('00-preflight','01-prepare','02-verification','03-postflight','04-reconciliation','metadata')) {
        New-RvDirectory -Path (Join-Path $runDir $sub) | Out-Null
    }
    [pscustomobject]@{ RunDir=$runDir; RunId=$RunId; Timestamp=$stamp }
}

# ---- Machine-readable writers ----
function Write-RvJson {
    [CmdletBinding()] param([Parameter(Mandatory)]$Object,[Parameter(Mandatory)][string]$Path)
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    $Object | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $Path -Encoding UTF8
}
function Write-RvJsonl {
    [CmdletBinding()] param([Parameter(Mandatory)][array]$Objects,[Parameter(Mandatory)][string]$Path)
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    @($Objects | ForEach-Object { $_ | ConvertTo-Json -Depth 10 -Compress }) | Set-Content -LiteralPath $Path -Encoding UTF8
}
function Write-RvCsv {
    [CmdletBinding()] param([Parameter(Mandatory)][array]$Objects,[Parameter(Mandatory)][string]$Path)
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    $Objects | Export-Csv -LiteralPath $Path -NoTypeInformation -Encoding UTF8
}
function Write-RvSummaryText {
    [CmdletBinding()] param([Parameter(Mandatory)][hashtable]$Table,[Parameter(Mandatory)][string]$Path)
    ($Table.Keys | Sort-Object | ForEach-Object { "{0}={1}" -f $_, $Table[$_] }) | Set-Content -LiteralPath $Path -Encoding UTF8
}

# ---- Identity / hashing ----
function Get-RvFileIdentity {
    [CmdletBinding()] param([Parameter(Mandatory)][string]$Path)
    $exists = Test-Path -LiteralPath $Path -PathType Leaf
    $sha = $null; $size = $null
    if ($exists) {
        $size = (Get-Item -LiteralPath $Path).Length
        $sha = (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
    }
    [pscustomobject]@{ Path=$Path; Exists=$exists; SizeBytes=$size; Sha256=$sha }
}

function Get-RvCanonicalTextIdentity {
    [CmdletBinding()] param([Parameter(Mandatory)][string]$Path)

    $exists = Test-Path -LiteralPath $Path -PathType Leaf

    if (-not $exists) {
        return [pscustomobject]@{
            Path=$Path
            Exists=$false
            SizeBytes=$null
            CanonicalSizeBytes=$null
            Sha256=$null
            Canonicalization='UTF8_BOM_REMOVED_CRLF_CR_TO_LF'
        }
    }

    $raw = [System.IO.File]::ReadAllBytes($Path)
    $start = 0

    if (
        $raw.Length -ge 3 -and
        $raw[0] -eq 0xEF -and
        $raw[1] -eq 0xBB -and
        $raw[2] -eq 0xBF
    ) {
        $start = 3
    }

    $stream = [System.IO.MemoryStream]::new()

    try {
        for ($i = $start; $i -lt $raw.Length; $i++) {
            $b = $raw[$i]

            if ($b -eq 0x0D) {
                if (
                    ($i + 1) -lt $raw.Length -and
                    $raw[$i + 1] -eq 0x0A
                ) {
                    $i++
                }

                $stream.WriteByte(0x0A)
            }
            else {
                $stream.WriteByte($b)
            }
        }

        $canonical = $stream.ToArray()
    }
    finally {
        $stream.Dispose()
    }

    $hasher = [System.Security.Cryptography.SHA256]::Create()

    try {
        $digest = $hasher.ComputeHash($canonical)
        $sha = [System.Convert]::ToHexString($digest).ToLowerInvariant()
    }
    finally {
        $hasher.Dispose()
    }

    [pscustomobject]@{
        Path=$Path
        Exists=$true
        SizeBytes=$raw.Length
        CanonicalSizeBytes=$canonical.Length
        Sha256=$sha
        Canonicalization='UTF8_BOM_REMOVED_CRLF_CR_TO_LF'
    }
}

function Get-RvGitIdentity {
    [CmdletBinding()] param([Parameter(Mandatory)][string]$RepoRoot)
    if (-not (Test-Path -LiteralPath $RepoRoot -PathType Container)) {
        return [pscustomobject]@{ Root=$RepoRoot; Present=$false }
    }
    Push-Location -LiteralPath $RepoRoot
    try {
        $branch = (& git branch --show-current 2>$null | Out-String).Trim()
        $head   = (& git rev-parse HEAD 2>$null | Out-String).Trim()
        $tree   = (& git rev-parse 'HEAD^{tree}' 2>$null | Out-String).Trim()
        $status = @(& git status --short 2>$null | Out-String).Trim()
        $staged = @(& git diff --cached --name-only 2>$null | Where-Object { $_ -and $_.Trim() })
        # tracked-modified entries (not untracked)
        $trackedMod = @(& git status --short 2>$null | Where-Object { $_ -match '^\s*[MADR]' })
        [pscustomobject]@{
            Root=$RepoRoot; Present=$true; Branch=$branch; Head=$head; Tree=$tree
            StatusShort=$status; StagedCount=$staged.Count; StagedFiles=$staged
            TrackedModifiedCount=$trackedMod.Count
        }
    } finally { Pop-Location }
}

function Get-RvGitPathState {
    [CmdletBinding()] param([Parameter(Mandatory)][string]$RepoRoot,[Parameter(Mandatory)][string]$RelativePath)
    Push-Location -LiteralPath $RepoRoot
    try {
        $status = (& git status --short -- $RelativePath 2>$null | Out-String).Trim()
        $ignored = (& git check-ignore -v -- $RelativePath 2>$null | Out-String).Trim()
        [pscustomobject]@{ RelativePath=$RelativePath; GitStatus=$status; CheckIgnore=$ignored; IsIgnored=[bool]$ignored }
    } finally { Pop-Location }
}

# ---- Ports ----
function Test-RvPortState {
    [CmdletBinding()] param([Parameter(Mandatory)][int]$Port)
    $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    if ($conn) {
        [pscustomobject]@{ Port=$Port; State='OCCUPIED'; OwnerPids=@($conn | Select-Object -ExpandProperty OwningProcess -Unique) }
    } else {
        [pscustomobject]@{ Port=$Port; State='FREE'; OwnerPids=@() }
    }
}
function Test-RvPortOwned {
    [CmdletBinding()] param([Parameter(Mandatory)][int]$Port,[Parameter(Mandatory)][int]$OwningPid)
    $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    if (-not $conn) { return $false }
    return (@($conn | Select-Object -ExpandProperty OwningProcess -Unique) -contains $OwningPid)
}
function Test-RvPortAbsent {
    [CmdletBinding()] param([Parameter(Mandatory)][int]$Port)
    return (-not (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue))
}

# ---- Env snapshot (secret-safe) ----
function Get-RvEnvSnapshot {
    [CmdletBinding()] param()
    $public = @('HACP_TRUST_MODE','HTTP_PROXY','HTTPS_PROXY','ALL_PROXY','NO_PROXY','CARGO_TARGET_DIR')
    $secretPath = @('HACP_TRUST_KEYS_FILE')
    $out = [ordered]@{}
    foreach ($v in $public) {
        $val = [Environment]::GetEnvironmentVariable($v)
        $out[$v] = if ($null -eq $val) { 'UNSET' } elseif ($val -eq '') { 'EMPTY' } else { $val }
    }
    foreach ($v in $secretPath) {
        $val = [Environment]::GetEnvironmentVariable($v)
        if ($null -eq $val -or $val -eq '') { $out[$v] = 'UNSET' }
        else { $out["$v`_SET"]='SET'; $out["$v`_FILE_EXISTS"]=(Test-Path -LiteralPath $val) }
    }
    [pscustomobject]$out
}

# ---- Native command execution (RC always captured; never silently lost) ----
function Invoke-RvNative {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$FilePath,
        [string[]]$ArgumentList = @(),
        [Parameter(Mandatory)][string]$WorkDir,
        [Parameter(Mandatory)][string]$StepId,
        [Parameter(Mandatory)][string]$SurfaceId,
        [Parameter(Mandatory)][string]$EvidenceDir,
        [hashtable]$ExtraEnv = @{},
        [string]$DisplayName = ''
    )

    $stdoutPath = Join-Path $EvidenceDir "$StepId.stdout.log"
    $stderrPath = Join-Path $EvidenceDir "$StepId.stderr.log"
    $cmdPath    = Join-Path $EvidenceDir "$StepId.command.txt"
    $rcPath     = Join-Path $EvidenceDir "$StepId.exit_code.txt"

    $cmdDisplay =
        if ($DisplayName) {
            $DisplayName
        }
        else {
            "$FilePath $($ArgumentList -join ' ')"
        }

    Set-Content `
        -LiteralPath $cmdPath `
        -Value $cmdDisplay `
        -Encoding UTF8

    # Scoped environment for this invocation only.
    # Restoration is guaranteed even if launch/wait/result handling fails.
    $saved = @{}

    foreach ($k in $ExtraEnv.Keys) {
        $saved[$k] =
            [Environment]::GetEnvironmentVariable($k)

        [Environment]::SetEnvironmentVariable(
            $k,
            $ExtraEnv[$k]
        )
    }

    $start = (Get-Date).ToUniversalTime()
    $end   = $null
    $rc    = $null
    $p     = $null

    Write-RvLog `
        -Level STEP `
        -Message "RUN $SurfaceId/$StepId : $cmdDisplay"

    try {
        try {
            # Do not use Start-Process -Wait here.
            # On the controlled Windows execution path it was observed
            # remaining blocked after cargo and its descendants had exited.
            #
            # Wait only for the exact process handle we started.
            $p = Start-Process `
                -FilePath $FilePath `
                -ArgumentList $ArgumentList `
                -WorkingDirectory $WorkDir `
                -RedirectStandardOutput $stdoutPath `
                -RedirectStandardError $stderrPath `
                -NoNewWindow `
                -PassThru

            $p.WaitForExit()
            $p.Refresh()

            $rc = $p.ExitCode
        }
        catch {
            $rc = -1

            Add-Content `
                -LiteralPath $stderrPath `
                -Value ("LAUNCH_OR_WAIT_ERROR: " + $_.Exception.Message)
        }

        $end = (Get-Date).ToUniversalTime()

        Set-Content `
            -LiteralPath $rcPath `
            -Value "$rc" `
            -Encoding UTF8
    }
    finally {
        foreach ($k in $saved.Keys) {
            [Environment]::SetEnvironmentVariable(
                $k,
                $saved[$k]
            )
        }
    }

    [pscustomobject]@{
        StepId      = $StepId
        SurfaceId   = $SurfaceId
        Command     = $cmdDisplay
        StartTimeUtc = $start.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
        EndTimeUtc   = $end.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
        ProcessRc    = $rc
        StdOutPath   = $stdoutPath
        StdErrPath   = $stderrPath
        EvidencePath = $EvidenceDir
    }
}
function Get-RvStdout {
    [CmdletBinding()] param([Parameter(Mandatory)]$Run)
    if ($Run.StdOutPath -and (Test-Path -LiteralPath $Run.StdOutPath)) { (Get-Content -LiteralPath $Run.StdOutPath -Raw) } else { '' }
}

# ---- Localhost HTTP probe (runtime-only; negative smoke). Never external. ----
# PS5.1 + PS7 compatible: does NOT rely on -SkipHttpErrorCheck (PS7-only). An expected
# 403 + X-HACP-Decision: DENY error response is read from the exception response in BOTH hosts.
function Get-RvHeaderValue {
    [CmdletBinding()] param($Response,[string]$Name)
    if (-not $Response) { return $null }
    try {
        $h = $Response.Headers
        if (-not $h) { return $null }
        try { $v = $h[$Name]; if ($v) { return ($v | Select-Object -First 1) } } catch {}
        try { if ($h.GetValues) { $v = $h.GetValues($Name); if ($v) { return ($v | Select-Object -First 1) } } } catch {}
        try { foreach ($e in $h.GetEnumerator()) { if ($e.Key -eq $Name) { return ($e.Value | Select-Object -First 1) } } } catch {}
    } catch {}
    return $null
}
function Invoke-RvLocalProbe {
    [CmdletBinding()] param([Parameter(Mandatory)][string]$Uri,[string]$Method = 'GET')
    $status = $null; $decision = $null; $reason = $null; $err = $null
    try {
        $r = Invoke-WebRequest -Uri $Uri -Method $Method -TimeoutSec 10 -ErrorAction Stop
        $status = [int]$r.StatusCode
        $decision = Get-RvHeaderValue -Response $r -Name 'X-HACP-Decision'
        $reason   = Get-RvHeaderValue -Response $r -Name 'X-HACP-Reason'
    } catch {
        $err = $_.Exception.Message
        $resp = $_.Exception.Response
        if ($resp) {
            try { $status = [int]$resp.StatusCode } catch { try { $status = [int]$resp.StatusCode.value__ } catch {} }
            $decision = Get-RvHeaderValue -Response $resp -Name 'X-HACP-Decision'
            $reason   = Get-RvHeaderValue -Response $resp -Name 'X-HACP-Reason'
        }
    }
    [pscustomobject]@{ StatusCode=$status; Decision=$decision; Reason=$reason; Error=$err }
}

# ---- Run-owned work root (DEFECT-8: mutating run state is run-specific) ----
function Get-RvWorkRoot {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$RunRoot,[Parameter(Mandatory)][string]$RunDir)
    $key = Split-Path -Leaf $RunDir
    if (-not $key) { $key = 'default' }
    $root = New-RvDirectory -Path (Join-Path $RunRoot $key)
    foreach ($sub in @('bin','cargo-target','fixtures','tmp')) { New-RvDirectory -Path (Join-Path $root $sub) | Out-Null }
    return $root
}

# ---- Process ownership (only owned PIDs ever touched; run-specific registry) ----
function Get-RvPidRegistryPath { param([Parameter(Mandatory)][string]$WorkRoot); (Join-Path $WorkRoot 'pid-registry.jsonl') }
function Register-RvOwnedProcess {
    # R2: record ACTUAL process identity (StartTime + ExecutablePath + RunId) for PID-reuse protection.
    [CmdletBinding()] param([Parameter(Mandatory)][string]$WorkRoot,[Parameter(Mandatory)][int]$Pid,[Parameter(Mandatory)][string]$Command,[Parameter(Mandatory)][string]$SurfaceOwner,[int]$Port = 0)
    $procStart = $null; $exePath = 'NOT_AVAILABLE'
    $proc = Get-Process -Id $Pid -ErrorAction SilentlyContinue
    if ($proc) {
        try { $procStart = $proc.StartTime.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ') } catch { $procStart = $null }
        try { if ($proc.Path) { $exePath = $proc.Path } } catch { $exePath = 'NOT_AVAILABLE' }
    }
    # R2 §10/§11: ProcessStartTimeUtc is MANDATORY. A registration without a reliable StartTime FAILS
    # (PROCESS_OWNERSHIP_REGISTRATION=FAIL) — the caller must treat this as ENVIRONMENT_RED.
    if (-not $procStart) {
        throw "PROCESS_OWNERSHIP_REGISTRATION=FAIL: no reliable ProcessStartTimeUtc for PID $Pid"
    }
    $runId = Split-Path -Leaf $WorkRoot   # WorkRoot = <RunRoot>\<RunId>
    $rec = [pscustomobject]@{
        Pid = $Pid
        ProcessStartTimeUtc = $procStart
        ExecutablePath = $exePath
        Command = $Command
        SurfaceOwner = $SurfaceOwner
        Port = $Port
        RunId = $runId
    }
    Add-Content -LiteralPath (Get-RvPidRegistryPath -WorkRoot $WorkRoot) -Value ($rec | ConvertTo-Json -Depth 4 -Compress) -Encoding UTF8
    $rec
}
function Get-RvOwnedProcesses {
    [CmdletBinding()] param([Parameter(Mandatory)][string]$WorkRoot)
    $path = Get-RvPidRegistryPath -WorkRoot $WorkRoot
    if (-not (Test-Path -LiteralPath $path)) { return @() }
    @(Get-Content -LiteralPath $path | Where-Object { $_.Trim() } | ForEach-Object { $_ | ConvertFrom-Json })
}
function Stop-RvOwnedProcesses {
    # R2: verify ownership (PID + StartTime [+ ExecutablePath]) BEFORE Stop-Process. PID-reuse safe.
    [CmdletBinding()] param([Parameter(Mandatory)][string]$WorkRoot)
    $stopped = @(); $alreadyExited = @(); $leaked = @(); $mismatches = @()
    foreach ($rec in (Get-RvOwnedProcesses -WorkRoot $WorkRoot)) {
        $proc = Get-Process -Id $rec.Pid -ErrorAction SilentlyContinue
        if (-not $proc) { $alreadyExited += $rec.Pid; continue }   # §17 PROCESS_ALREADY_EXITED (not RED)
        $obsStart = $null; $obsPath = $null
        try { $obsStart = $proc.StartTime.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ') } catch {}
        try { $obsPath = $proc.Path } catch {}
        $expectedStart = $null
        if ($rec.ProcessStartTimeUtc -is [datetime]) {
            $expectedStart = $rec.ProcessStartTimeUtc.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
        } elseif ($rec.ProcessStartTimeUtc) {
            $expectedStart = [string]$rec.ProcessStartTimeUtc
        }
        $startMatch = [bool]($expectedStart -and $obsStart -and ($obsStart -eq $expectedStart))
        $pathMatch = $true
        if ($rec.ExecutablePath -and $rec.ExecutablePath -ne 'NOT_AVAILABLE' -and $obsPath) {
            $pathMatch = ($obsPath -eq $rec.ExecutablePath)
        }
        if (-not ($startMatch -and $pathMatch)) {
            # §16 PID reuse / ownership mismatch -> DO NOT KILL; record + cleanup failure.
            $mismatches += [pscustomobject]@{
                PID=$rec.Pid; EXPECTED_START_TIME=$rec.ProcessStartTimeUtc; OBSERVED_START_TIME=$obsStart
                EXPECTED_PATH=$rec.ExecutablePath; OBSERVED_PATH=$obsPath; OWNERSHIP_MISMATCH=$true
            }
            continue
        }
        try { Stop-Process -Id $rec.Pid -Force -ErrorAction Stop; $stopped += $rec.Pid } catch { $leaked += $rec.Pid }
    }
    [pscustomobject]@{
        StoppedPids=$stopped; AlreadyExitedPids=$alreadyExited; LeakedPids=$leaked
        OwnershipMismatches=$mismatches
        CleanupComplete=($mismatches.Count -eq 0 -and $leaked.Count -eq 0)
    }
}

function Save-RvEnvironment {
    # D3 §8: snapshot exact previous state (EXISTED/NOT_EXISTED + VALUE) for the given names.
    # Values are captured for restoration only and NEVER logged (secret-safe).
    [CmdletBinding()] param([Parameter(Mandatory)][string[]]$Names)
    $saved = @{}
    foreach ($n in $Names) {
        $v = [Environment]::GetEnvironmentVariable($n)   # $null if NOT_EXISTED; value (incl '') if EXISTED
        $saved[$n] = @{ Set = ($null -ne $v); Value = $v }
    }
    return $saved
}
function Restore-RvEnvironment {
    # D3 §9: restore a Save-RvEnvironment snapshot. EXISTED -> exact original value; NOT_EXISTED -> remove.
    [CmdletBinding()] param([Parameter(Mandatory)]$Saved)
    foreach ($n in @($Saved.Keys)) {
        if ($Saved[$n].Set) { [Environment]::SetEnvironmentVariable($n, $Saved[$n].Value) }
        else { [Environment]::SetEnvironmentVariable($n, $null) }
    }
}

# ---- Result object (§38 schema) ----
function New-RvResult {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$Surface,
        [Parameter(Mandatory)][string[]]$MatrixRows,
        [Parameter(Mandatory)][hashtable]$ProcessExitCodes,
        [Parameter(Mandatory)][string]$Classification,
        [Parameter(Mandatory)][string]$ExpectedResult,
        [Parameter(Mandatory)][string]$ObservedResult,
        [string[]]$EvidenceFiles = @(),
        [bool]$EnvironmentRed = $false,
        [bool]$BehavioralRed = $false
    )
    [pscustomobject]@{
        surface=$Surface; matrix_rows=$MatrixRows; process_exit_codes=$ProcessExitCodes
        classification=$Classification; expected_result=$ExpectedResult; observed_result=$ObservedResult
        evidence_files=$EvidenceFiles; environment_red=$EnvironmentRed; behavioral_red=$BehavioralRed
        human_review_required=$true
    }
}

Export-ModuleMember -Function @(
    'Write-RvLog','Set-RvLogFile','New-RvDirectory','New-RvRunDirectory','Resolve-RvRunDir',
    'Write-RvJson','Write-RvJsonl','Write-RvCsv','Write-RvSummaryText',
    'Get-RvFileIdentity','Get-RvCanonicalTextIdentity','Get-RvGitIdentity','Get-RvGitPathState',
    'Test-RvPortState','Test-RvPortOwned','Test-RvPortAbsent',
    'Get-RvEnvSnapshot','Invoke-RvNative','Get-RvStdout','Invoke-RvLocalProbe',
    'Register-RvOwnedProcess','Get-RvOwnedProcesses','Stop-RvOwnedProcesses','New-RvResult','Get-RvWorkRoot','Get-RvHeaderValue',
    'Get-RvRunContext','Set-RvRunState','Save-RvEnvironment','Restore-RvEnvironment'
) -Variable @('RV_CLASS')
