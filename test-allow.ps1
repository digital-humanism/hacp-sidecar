# test-allow.ps1
$output = .\gen-token.exe 2>$null

$envLine = $output | Select-String "X-HACP-Intent-Envelope:"
$tokLine = $output | Select-String "X-HACP-Decision-Token:"

$envHeader = ($envLine -replace "X-HACP-Intent-Envelope: ", "").Trim()
$tokHeader = ($tokLine -replace "X-HACP-Decision-Token: ", "").Trim()

Write-Host "Sending request with headers..." -ForegroundColor Cyan

$response = curl.exe -i -s `
    -H "X-HACP-Intent-Envelope: $envHeader" `
    -H "X-HACP-Decision-Token: $tokHeader" `
    http://localhost:8080/api/test

Write-Host $response