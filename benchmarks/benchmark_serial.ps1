# Serial benchmark: each request uses a unique token.
# Slower but avoids budget exhaustion and gives accurate overhead measurement.

$ErrorActionPreference = "Stop"
$sidecar = "http://127.0.0.1:8080"
$upstream = "http://127.0.0.1:8000"
$tokensFile = "benchmarks/tokens.jsonl"
$count = 1000

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Serial Benchmark: $count requests (unique token per request)" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

# Read tokens
$tokens = Get-Content $tokensFile -TotalCount $count | ConvertFrom-Json
if ($tokens.Count -lt $count) {
    Write-Host "Warning: only $($tokens.Count) tokens available, requested $count" -ForegroundColor Yellow
    $count = $tokens.Count
}

# ------------------------------------------------------------------
# Baseline (serial, no headers)
# ------------------------------------------------------------------
Write-Host "Baseline: upstream directly" -ForegroundColor Yellow
$sw = [System.Diagnostics.Stopwatch]::StartNew()
$baselineLatencies = New-Object System.Collections.ArrayList
$baselineSuccess = 0
$baselineFail = 0

for ($i = 0; $i -lt $count; $i++) {
    $reqSw = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        $resp = Invoke-WebRequest `
            -Uri "$upstream/api/test" `
            -Method GET `
            -UseBasicParsing `
            -DisableKeepAlive `
            -TimeoutSec 5

        if ($resp.StatusCode -eq 200) { 
            $baselineSuccess++ 
        } else { 
            $baselineFail++ 
        }
    } catch {
        $baselineFail++
    }

    $reqSw.Stop()
    [void]$baselineLatencies.Add($reqSw.Elapsed.TotalMilliseconds)
}
$sw.Stop()
$baselineTotal = $sw.Elapsed.TotalMilliseconds

# ------------------------------------------------------------------
# Sidecar (serial, with unique headers per request)
# ------------------------------------------------------------------
Write-Host ""
Write-Host "Sidecar: through HACP enforcement" -ForegroundColor Yellow
$sw = [System.Diagnostics.Stopwatch]::StartNew()
$sidecarLatencies = New-Object System.Collections.ArrayList
$sidecarSuccess = 0
$sidecarFail = 0

for ($i = 0; $i -lt $count; $i++) {
    $envHeader = $tokens[$i].env
    $tokHeader = $tokens[$i].tok

    $reqSw = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        $resp = Invoke-WebRequest `
            -Uri "$sidecar/api/test" `
            -Method GET `
            -UseBasicParsing `
            -DisableKeepAlive `
            -TimeoutSec 5 `
            -Headers @{
                "X-HACP-Intent-Envelope" = $envHeader
                "X-HACP-Decision-Token"  = $tokHeader
            }

        if ($resp.StatusCode -eq 200) {
            $sidecarSuccess++
        } else {
            $sidecarFail++
        }
    } catch {
        $sidecarFail++
    }

    $reqSw.Stop()
    [void]$sidecarLatencies.Add($reqSw.Elapsed.TotalMilliseconds)
}
$sw.Stop()
$sidecarTotal = $sw.Elapsed.TotalMilliseconds

# ------------------------------------------------------------------
# Percentile helper
# ------------------------------------------------------------------
function Get-Percentile($arr, $p) {
    $sorted = @($arr | Sort-Object)

    if ($sorted.Count -eq 0) {
        return 0
    }

    $rank = [math]::Ceiling(($p / 100.0) * $sorted.Count)
    $idx = [math]::Max(0, $rank - 1)

    return [math]::Round($sorted[$idx], 2)
}

# ------------------------------------------------------------------
# Results
# ------------------------------------------------------------------
Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Results" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "--- Baseline ---" -ForegroundColor Yellow
Write-Host "Total: $([math]::Round($baselineTotal, 2)) ms for $count requests"
Write-Host "Success: $baselineSuccess / $count"
Write-Host "Avg: $([math]::Round($baselineTotal / $count, 2)) ms"
Write-Host "p50: $(Get-Percentile $baselineLatencies 50) ms"
Write-Host "p95: $(Get-Percentile $baselineLatencies 95) ms"
Write-Host "p99: $(Get-Percentile $baselineLatencies 99) ms"
Write-Host "max: $([math]::Round(($baselineLatencies | Measure-Object -Maximum).Maximum, 2)) ms"
Write-Host ""

Write-Host "--- Sidecar ---" -ForegroundColor Yellow
Write-Host "Total: $([math]::Round($sidecarTotal, 2)) ms for $count requests"
Write-Host "Success: $sidecarSuccess / $count"
Write-Host "Avg: $([math]::Round($sidecarTotal / $count, 2)) ms"
Write-Host "p50: $(Get-Percentile $sidecarLatencies 50) ms"
Write-Host "p95: $(Get-Percentile $sidecarLatencies 95) ms"
Write-Host "p99: $(Get-Percentile $sidecarLatencies 99) ms"
Write-Host "max: $([math]::Round(($sidecarLatencies | Measure-Object -Maximum).Maximum, 2)) ms"
Write-Host ""

$overheadTotal = $sidecarTotal - $baselineTotal
$overheadAvg = $overheadTotal / $count
$p99Baseline = Get-Percentile $baselineLatencies 99
$p99Sidecar = Get-Percentile $sidecarLatencies 99
$overheadP99 = $p99Sidecar - $p99Baseline

Write-Host "=== Overhead ===" -ForegroundColor Green
Write-Host "Total overhead: $([math]::Round($overheadTotal, 2)) ms"
Write-Host "Avg overhead:   $([math]::Round($overheadAvg, 2)) ms/request"
Write-Host "p99 overhead:   $([math]::Round($overheadP99, 2)) ms"
Write-Host ""

# Gate D target: p99 overhead < 5ms (acceptable < 10ms)
if ($overheadP99 -lt 5) {
    Write-Host "GATE D PASS: p99 overhead $([math]::Round($overheadP99, 2)) ms < 5 ms target" -ForegroundColor Green
} elseif ($overheadP99 -lt 10) {
    Write-Host "GATE D ACCEPTABLE: p99 overhead $([math]::Round($overheadP99, 2)) ms < 10 ms acceptable" -ForegroundColor Yellow
} else {
    Write-Host "GATE D NEEDS OPTIMIZATION: p99 overhead $([math]::Round($overheadP99, 2)) ms > 10 ms" -ForegroundColor Red
}