# HACP 1.1.1 — Go Verification Execution

## Windows PowerShell 7

The documented default layout uses sibling repository clones under an arbitrary workspace root:

```text
<workspace>\
  hacp-sidecar\
  hacp-spec\
  humanist-core\
```

Start PowerShell from anywhere inside the `hacp-sidecar` clone.

Repository paths are derived from the actual `hacp-sidecar` Git worktree. No machine-specific checkout path is required.

### 00 — Preflight

```powershell
$SidecarRoot = (git rev-parse --show-toplevel).Trim()
if ($LASTEXITCODE -ne 0 -or -not $SidecarRoot) {
    throw 'Unable to resolve hacp-sidecar repository root'
}

$WorkspaceRoot = Split-Path -Parent $SidecarRoot

$SpecRoot = Join-Path $WorkspaceRoot 'hacp-spec'
$CoreRoot = Join-Path $WorkspaceRoot 'humanist-core'
$Scripts = Join-Path $SidecarRoot 'test\matrix\t_go\Scripts'

$VerificationRoot = Join-Path ([System.IO.Path]::GetTempPath()) 'hacp-1.1.1-verification'
$RunRoot = Join-Path $VerificationRoot 'run'
$EvidenceRoot = Join-Path $VerificationRoot 'evidence'

& pwsh.exe -NoProfile -File (Join-Path $Scripts '00-Preflight.ps1') `
    -SidecarRoot $SidecarRoot `
    -SpecRoot $SpecRoot `
    -CoreRoot $CoreRoot `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 01 — Prepare Environment

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '01-Prepare-Environment.ps1') `
    -SidecarRoot $SidecarRoot `
    -SpecRoot $SpecRoot `
    -CoreRoot $CoreRoot `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 02 — Run All Verification Surfaces

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '02-Run-Verification.ps1') `
    -SidecarRoot $SidecarRoot `
    -SpecRoot $SpecRoot `
    -CoreRoot $CoreRoot `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 03 — Postflight

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '03-Postflight.ps1') `
    -SidecarRoot $SidecarRoot `
    -SpecRoot $SpecRoot `
    -CoreRoot $CoreRoot `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 04 — Reconcile Results

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '04-Reconcile-Results.ps1') `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

For a non-sibling repository layout, set `$SpecRoot` and `$CoreRoot` to the actual clone locations before running the commands above.
