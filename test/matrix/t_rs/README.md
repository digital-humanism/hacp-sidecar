# HACP 1.1.1 — Rust Verification Execution

## Windows PowerShell 7

### 00 — Preflight

```powershell
$Scripts = 'C:\Personal\GitHub\Dev\hacp-sidecar\test\matrix\t_rs\Scripts'
$RunRoot = 'C:\Temp\hacp-rust-verification\run'
$EvidenceRoot = 'C:\Temp\hacp-rust-verification\evidence'

& pwsh.exe -NoProfile -File (Join-Path $Scripts '00-Preflight.ps1') `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 01 — Prepare Environment

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '01-Prepare-Environment.ps1') `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 02 — Run All Verification Surfaces

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '02-Run-Verification.ps1') `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 03 — Postflight

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '03-Postflight.ps1') `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```

### 04 — Reconcile Results

```powershell
& pwsh.exe -NoProfile -File (Join-Path $Scripts '04-Reconcile-Results.ps1') `
    -RunRoot $RunRoot `
    -EvidenceRoot $EvidenceRoot
```
