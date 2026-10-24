# HACP 1.1.1 — Rust Verification Execution

## Windows PowerShell 7

### 00 — Preflight

```powershell
$SidecarRepo = (git rev-parse --show-toplevel).Trim()
if ($LASTEXITCODE -ne 0 -or -not $SidecarRepo) {
    throw 'Unable to resolve hacp-sidecar repository root'
}

$WorkspaceRoot = Split-Path -Parent $SidecarRepo
$SpecRepo = Join-Path $WorkspaceRoot 'hacp-spec'
$Scripts = Join-Path $SidecarRepo 'test\matrix\t_rs\Scripts'

$VerificationRoot = Join-Path ([System.IO.Path]::GetTempPath()) 'hacp-rust-verification'
$RunRoot = Join-Path $VerificationRoot 'run'
$EvidenceRoot = Join-Path $VerificationRoot 'evidence'

& pwsh.exe -NoProfile -File (Join-Path $Scripts '00-Preflight.ps1') `
    -SpecRepo $SpecRepo `
    -SidecarRepo $SidecarRepo `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 01 — Prepare Environment

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '01-Prepare-Environment.ps1') `
    -SpecRepo $SpecRepo `
    -SidecarRepo $SidecarRepo `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 02 — Run All Verification Surfaces

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '02-Run-Verification.ps1') `
    -SpecRepo $SpecRepo `
    -SidecarRepo $SidecarRepo `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 03 — Postflight

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '03-Postflight.ps1') `
    -SpecRepo $SpecRepo `
    -SidecarRepo $SidecarRepo `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 04 — Reconcile Results

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '04-Reconcile-Results.ps1') `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```
