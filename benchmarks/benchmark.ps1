# Benchmark: upstream directly vs through sidecar.
# High-rate shared-token load test using hey.exe when available.

$ErrorActionPreference = "Stop"

$sidecar = "http://127.0.0.1:8080"
$upstream = "http://127.0.0.1:8000"

$count = 1000
$concurrency = 5

$tokensFile = "benchmarks/tokens.jsonl"
$resultsDir = "benchmarks/results"
$generator = ".\benchmarks\generate-tokens.exe"

New-Item `
    -ItemType Directory `
    -Force `
    -Path $resultsDir |
    Out-Null

# ------------------------------------------------------------------
# Step 1: Validate token generator
# ------------------------------------------------------------------

if (-not (Test-Path $generator)) {
    throw "Token generator not found: $generator"
}

# ------------------------------------------------------------------
# Step 2: Generate fresh tokens
# ------------------------------------------------------------------

Write-Host ""
Write-Host "============================================================"
Write-Host " Generating $count pre-signed tokens..."
Write-Host "============================================================"
Write-Host ""

if (Test-Path $tokensFile) {
    Remove-Item $tokensFile -Force
}

& $generator `
    -count $count `
    -out $tokensFile `
    -method GET `
    -path "/api/test"

if ($LASTEXITCODE -ne 0) {
    throw "Token generation failed with exit code $LASTEXITCODE"
}

if (-not (Test-Path $tokensFile)) {
    throw "Token file was not created: $tokensFile"
}

$tokens = @(
    Get-Content $tokensFile |
        ConvertFrom-Json
)

if ($tokens.Count -lt $count) {
    throw "Not enough generated tokens: expected=$count actual=$($tokens.Count)"
}

Write-Host "Generated tokens: $($tokens.Count)"

# ------------------------------------------------------------------
# Step 3: Read first token
#
# This benchmark intentionally uses one shared token for all sidecar
# requests. max_uses is large enough to allow the full run.
# ------------------------------------------------------------------

$first = $tokens[0]

$envHeader = $first.env
$tokHeader = $first.tok

Write-Host "First token envelope (first 40 chars): $($envHeader.Substring(0, 40))..."
Write-Host "First token decision (first 40 chars): $($tokHeader.Substring(0, 40))..."
Write-Host ""

# ------------------------------------------------------------------
# Step 4: Detect hey.exe
# ------------------------------------------------------------------

$heyPath = $null

$candidates = @(
    (Join-Path $PWD "hey.exe"),
    "$env:USERPROFILE\go\bin\hey.exe"
)

foreach ($candidate in $candidates) {
    if (Test-Path $candidate) {
        $heyPath = $candidate
        break
    }
}

if ($null -eq $heyPath) {
    $cmd = Get-Command hey -ErrorAction SilentlyContinue

    if ($cmd) {
        $heyPath = $cmd.Source
    }
}

$useHey = ($null -ne $heyPath)

if ($useHey) {
    Write-Host "Using hey.exe for load testing" -ForegroundColor Green
}
else {
    Write-Host "hey.exe not found, using PowerShell Invoke-WebRequest fallback" -ForegroundColor Yellow
    Write-Host "(slower and not suitable for authoritative concurrent p99)" -ForegroundColor Yellow
}

# ------------------------------------------------------------------
# Step 5: Baseline
# ------------------------------------------------------------------

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Baseline: upstream directly ($count requests)" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

$baselineOut = Join-Path $resultsDir "baseline.txt"

if ($useHey) {

    $baselineArgs = @(
        "-n", "$count",
        "-c", "$concurrency",
        "$upstream/api/test"
    )

    & $heyPath @baselineArgs |
        Tee-Object -FilePath $baselineOut
}
else {

    $sw = [System.Diagnostics.Stopwatch]::StartNew()

    $success = 0
    $failed = 0

    for ($i = 0; $i -lt $count; $i++) {

        try {
            $resp = Invoke-WebRequest `
                -Uri "$upstream/api/test" `
                -Method GET `
                -UseBasicParsing

            if ($resp.StatusCode -eq 200) {
                $success++
            }
            else {
                $failed++
            }
        }
        catch {
            $failed++
        }
    }

    $sw.Stop()

    @"
Total: $($sw.ElapsedMilliseconds) ms for $count requests (serial fallback)
Success: $success / $count
Failed: $failed
"@ |
        Tee-Object -FilePath $baselineOut
}

# ------------------------------------------------------------------
# Step 6: Sidecar
# ------------------------------------------------------------------

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Sidecar: through HACP enforcement ($count requests)" -ForegroundColor Cyan
Write-Host " Shared DecisionToken workload" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

$sidecarOut = Join-Path $resultsDir "sidecar.txt"

if ($useHey) {

    $sidecarArgs = @(
        "-n", "$count",
        "-c", "$concurrency",
        "-H", "X-HACP-Intent-Envelope: $envHeader",
        "-H", "X-HACP-Decision-Token: $tokHeader",
        "$sidecar/api/test"
    )

    & $heyPath @sidecarArgs |
        Tee-Object -FilePath $sidecarOut
}
else {

    $sw = [System.Diagnostics.Stopwatch]::StartNew()

    $success = 0
    $failed = 0

    for ($i = 0; $i -lt $count; $i++) {

        try {

            $resp = Invoke-WebRequest `
                -Uri "$sidecar/api/test" `
                -Method GET `
                -UseBasicParsing `
                -Headers @{
                    "X-HACP-Intent-Envelope" = $envHeader
                    "X-HACP-Decision-Token"  = $tokHeader
                }

            if ($resp.StatusCode -eq 200) {
                $success++
            }
            else {
                $failed++
            }
        }
        catch {
            $failed++
        }
    }

    $sw.Stop()

    @"
Total: $($sw.ElapsedMilliseconds) ms for $count requests (serial fallback)
Success: $success / $count
Failed: $failed
"@ |
        Tee-Object -FilePath $sidecarOut
}

# ------------------------------------------------------------------
# Step 7: Summary
# ------------------------------------------------------------------

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Results" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

if ($useHey) {

    Write-Host "--- Baseline ---" -ForegroundColor Yellow

    Get-Content $baselineOut |
        Select-String -Pattern "Slowest|Fastest|Average|99%|Total|Requests/sec" |
        ForEach-Object {
            $_.Line.Trim()
        }

    Write-Host ""

    Write-Host "--- Sidecar ---" -ForegroundColor Yellow

    Get-Content $sidecarOut |
        Select-String -Pattern "Slowest|Fastest|Average|99%|Total|Requests/sec" |
        ForEach-Object {
            $_.Line.Trim()
        }
}
else {

    Write-Host "Serial fallback only." -ForegroundColor Yellow
    Write-Host "Install hey.exe for concurrent latency distribution." -ForegroundColor Yellow
    Write-Host ""

    Get-Content $baselineOut
    Get-Content $sidecarOut
}

Write-Host ""