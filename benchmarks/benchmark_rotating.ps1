# Concurrent rotating-token benchmark.
# Each request uses its own unique DecisionToken.
# Purpose: compare concurrent enforcement with unique tokens
# against direct-upstream baseline, avoiding shared-token contention.

$ErrorActionPreference = "Stop"

$sidecar = "http://127.0.0.1:8080"
$upstream = "http://127.0.0.1:8000"
$tokensFile = "benchmarks/tokens.jsonl"

$count = 1000
$concurrency = 5

# ============================================================
# Helpers
# ============================================================

function Get-Percentile($arr, $p) {
    $sorted = @($arr | Sort-Object)

    if ($sorted.Count -eq 0) {
        return 0
    }

    $rank = [math]::Ceiling(($p / 100.0) * $sorted.Count)
    $idx = [math]::Max(0, $rank - 1)

    return [math]::Round($sorted[$idx], 2)
}

function Show-Stats(
    $name,
    $latencies,
    $success,
    $failed,
    $totalMs
) {
    $countTotal = $success + $failed

    $avg = if ($latencies.Count -gt 0) {
        [math]::Round(
            ($latencies | Measure-Object -Average).Average,
            2
        )
    }
    else {
        0
    }

    $max = if ($latencies.Count -gt 0) {
        [math]::Round(
            ($latencies | Measure-Object -Maximum).Maximum,
            2
        )
    }
    else {
        0
    }

    $rps = if ($totalMs -gt 0) {
        [math]::Round(
            $countTotal / ($totalMs / 1000.0),
            2
        )
    }
    else {
        0
    }

    Write-Host ""
    Write-Host "--- $name ---" -ForegroundColor Yellow
    Write-Host "Total:        $([math]::Round($totalMs, 2)) ms"
    Write-Host "Success:      $success / $countTotal"
    Write-Host "Failed:       $failed"
    Write-Host "Average:      $avg ms"
    Write-Host "p50:          $(Get-Percentile $latencies 50) ms"
    Write-Host "p95:          $(Get-Percentile $latencies 95) ms"
    Write-Host "p99:          $(Get-Percentile $latencies 99) ms"
    Write-Host "max:          $max ms"
    Write-Host "Requests/sec: $rps"
}

# ============================================================
# Validate environment
# ============================================================

if (-not (Test-Path $tokensFile)) {
    throw "Tokens file not found: $tokensFile"
}

$tokens = @(
    Get-Content $tokensFile -TotalCount $count |
        ConvertFrom-Json
)

if ($tokens.Count -lt $count) {
    throw "Not enough tokens: required=$count available=$($tokens.Count)"
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Rotating Token Benchmark" -ForegroundColor Cyan
Write-Host " Requests: $count | Concurrency: $concurrency" -ForegroundColor Cyan
Write-Host " Unique DecisionToken per request" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

# ============================================================
# Baseline
# ============================================================

Write-Host ""
Write-Host "Baseline: upstream directly"

$baselineLatencies =
    [System.Collections.Concurrent.ConcurrentBag[double]]::new()

$baselineSuccess =
    [System.Collections.Concurrent.ConcurrentBag[int]]::new()

$baselineFailed =
    [System.Collections.Concurrent.ConcurrentBag[int]]::new()

$baselineSw = [System.Diagnostics.Stopwatch]::StartNew()

0..($count - 1) |
    ForEach-Object -Parallel {

        $latencies = $using:baselineLatencies
        $successBag = $using:baselineSuccess
        $failedBag = $using:baselineFailed
        $upstreamUrl = $using:upstream

        $sw = [System.Diagnostics.Stopwatch]::StartNew()

        try {
            $resp = Invoke-WebRequest `
                -Uri "$upstreamUrl/api/test" `
                -Method GET `
                -UseBasicParsing `
                -DisableKeepAlive `
                -TimeoutSec 5

            if ($resp.StatusCode -eq 200) {
                $successBag.Add(1)
            }
            else {
                $failedBag.Add(1)
            }
        }
        catch {
            $failedBag.Add(1)
        }

        $sw.Stop()
        $latencies.Add(
            $sw.Elapsed.TotalMilliseconds
        )

    } -ThrottleLimit $concurrency

$baselineSw.Stop()

# ============================================================
# Sidecar — unique token per request
# ============================================================

Write-Host ""
Write-Host "Sidecar: rotating unique DecisionToken per request"

$sidecarLatencies =
    [System.Collections.Concurrent.ConcurrentBag[double]]::new()

$sidecarSuccess =
    [System.Collections.Concurrent.ConcurrentBag[int]]::new()

$sidecarFailed =
    [System.Collections.Concurrent.ConcurrentBag[int]]::new()

$sidecarSw = [System.Diagnostics.Stopwatch]::StartNew()

0..($count - 1) |
    ForEach-Object -Parallel {

        $i = $_

        $tokensLocal = $using:tokens
        $latencies = $using:sidecarLatencies
        $successBag = $using:sidecarSuccess
        $failedBag = $using:sidecarFailed
        $sidecarUrl = $using:sidecar

        $token = $tokensLocal[$i]

        $sw = [System.Diagnostics.Stopwatch]::StartNew()

        try {
            $resp = Invoke-WebRequest `
                -Uri "$sidecarUrl/api/test" `
                -Method GET `
                -UseBasicParsing `
                -DisableKeepAlive `
                -TimeoutSec 5 `
                -Headers @{
                    "X-HACP-Intent-Envelope" = $token.env
                    "X-HACP-Decision-Token"  = $token.tok
                }

            if ($resp.StatusCode -eq 200) {
                $successBag.Add(1)
            }
            else {
                $failedBag.Add(1)
            }
        }
        catch {
            $failedBag.Add(1)
        }

        $sw.Stop()
        $latencies.Add(
            $sw.Elapsed.TotalMilliseconds
        )

    } -ThrottleLimit $concurrency

$sidecarSw.Stop()

# ============================================================
# Results
# ============================================================

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Results" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

Show-Stats `
    "Baseline" `
    $baselineLatencies `
    $baselineSuccess.Count `
    $baselineFailed.Count `
    $baselineSw.Elapsed.TotalMilliseconds

Show-Stats `
    "Sidecar rotating tokens" `
    $sidecarLatencies `
    $sidecarSuccess.Count `
    $sidecarFailed.Count `
    $sidecarSw.Elapsed.TotalMilliseconds

# ============================================================
# Overhead
# ============================================================

$baselineAvg =
    if ($baselineLatencies.Count -gt 0) {
        ($baselineLatencies | Measure-Object -Average).Average
    }
    else {
        0
    }

$sidecarAvg =
    if ($sidecarLatencies.Count -gt 0) {
        ($sidecarLatencies | Measure-Object -Average).Average
    }
    else {
        0
    }

$baselineP50 = Get-Percentile $baselineLatencies 50
$sidecarP50 = Get-Percentile $sidecarLatencies 50

$baselineP95 = Get-Percentile $baselineLatencies 95
$sidecarP95 = Get-Percentile $sidecarLatencies 95

$baselineP99 = Get-Percentile $baselineLatencies 99
$sidecarP99 = Get-Percentile $sidecarLatencies 99

$avgOverhead = [math]::Round(
    $sidecarAvg - $baselineAvg,
    2
)

$p50Overhead = [math]::Round(
    $sidecarP50 - $baselineP50,
    2
)

$p95Overhead = [math]::Round(
    $sidecarP95 - $baselineP95,
    2
)

$p99Overhead = [math]::Round(
    $sidecarP99 - $baselineP99,
    2
)

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Overhead" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Avg overhead: $avgOverhead ms/request"
Write-Host "p50 overhead: $p50Overhead ms"
Write-Host "p95 overhead: $p95Overhead ms"
Write-Host "p99 overhead: $p99Overhead ms"

# ============================================================
# Verdict
# ============================================================

Write-Host ""

if ($baselineFailed.Count -gt 0) {

    Write-Host `
        "TEST INVALID: baseline failures detected ($($baselineFailed.Count))." `
        -ForegroundColor Red

}
elseif ($sidecarFailed.Count -gt 0) {

    Write-Host `
        "TEST INVALID: sidecar failures detected ($($sidecarFailed.Count))." `
        -ForegroundColor Red

}
elseif ($p99Overhead -lt 5) {

    Write-Host `
        "PASS: rotating-token p99 overhead $p99Overhead ms < 5 ms target" `
        -ForegroundColor Green

}
elseif ($p99Overhead -lt 10) {

    Write-Host `
        "ACCEPTABLE: rotating-token p99 overhead $p99Overhead ms < 10 ms" `
        -ForegroundColor Yellow

}
else {

    Write-Host `
        "TAIL LATENCY HIGH: rotating-token p99 overhead $p99Overhead ms" `
        -ForegroundColor Red

}

Write-Host ""