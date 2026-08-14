# HACP Sidecar demo — 3 scenarios against a running sidecar on :8080.
# Prerequisites: sidecar, upstream, control-plane all running.

$ErrorActionPreference = "Stop"
$sidecar = "http://localhost:8080"

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " HACP Enforcement Demo — 3 scenarios" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

# ---------------------------------------------------------------- Scenario 1: ALLOW
Write-Host "[1] ALLOW — GET /api/test with scope=[read]" -ForegroundColor Green
$tok = go run ./cmd/gen-token `
    -method GET -path /api/test `
    -verbs read -audience external -reversibility reversible -externality internal -data-class internal
$env = ($tok | Select-String "X-HACP-Intent-Envelope: (.*)").Matches[0].Groups[1].Value
$dec = ($tok | Select-String "X-HACP-Decision-Token: (.*)").Matches[0].Groups[1].Value

curl.exe -i "$sidecar/api/test" `
    -H "X-HACP-Intent-Envelope: $env" `
    -H "X-HACP-Decision-Token: $dec"
Write-Host ""

# ---------------------------------------------------------------- Scenario 2: REAUTHORIZE
Write-Host "[2] REAUTHORIZE — DELETE /api/test with scope=[read]" -ForegroundColor Yellow
Write-Host "     verb=delete is outside granted scope [read] -> boundary matrix -> REAUTHORIZE" -ForegroundColor DarkYellow
$tok = go run ./cmd/gen-token `
    -method DELETE -path /api/test `
    -verbs read -audience external -reversibility reversible -externality internal -data-class internal
$env = ($tok | Select-String "X-HACP-Intent-Envelope: (.*)").Matches[0].Groups[1].Value
$dec = ($tok | Select-String "X-HACP-Decision-Token: (.*)").Matches[0].Groups[1].Value

curl.exe -i "$sidecar/api/test" -X DELETE `
    -H "X-HACP-Intent-Envelope: $env" `
    -H "X-HACP-Decision-Token: $dec"
Write-Host ""

# ---------------------------------------------------------------- Scenario 3: DENY
Write-Host "[3] DENY — GET /api/test with INVALID signature" -ForegroundColor Red
$tok = go run ./cmd/gen-token `
    -method GET -path /api/test `
    -verbs read -invalidate-signature
$env = ($tok | Select-String "X-HACP-Intent-Envelope: (.*)").Matches[0].Groups[1].Value
$dec = ($tok | Select-String "X-HACP-Decision-Token: (.*)").Matches[0].Groups[1].Value

curl.exe -i "$sidecar/api/test" `
    -H "X-HACP-Intent-Envelope: $env" `
    -H "X-HACP-Decision-Token: $dec"
Write-Host ""

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Demo complete" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan