$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$composeFile = Join-Path $repoRoot "deployments\docker-compose.yml"

Set-Location $repoRoot

$passed = 0
$failed = 0

function Write-TestPass {
    param([string]$Name, [string]$Message)

    $script:passed++
    Write-Host "[PASS] $Name - $Message" -ForegroundColor Green
}

function Write-TestFail {
    param([string]$Name, [string]$Message)

    $script:failed++
    Write-Host "[FAIL] $Name - $Message" -ForegroundColor Red
}

function Get-ServiceContainerId {
    param([string]$Service)

    $id = (& docker compose -f $composeFile ps -q $Service).Trim()
    if (-not $id) {
        throw "No running container found for service '$Service'."
    }

    return $id
}

function Get-ContainerNetworks {
    param([string]$ContainerId)

    $json = & docker inspect $ContainerId --format '{{json .NetworkSettings.Networks}}'
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to inspect container networks."
    }

    $networks = $json | ConvertFrom-Json
    return @($networks.PSObject.Properties.Name)
}

function Test-NetworkMembership {
    param(
        [string[]]$Networks,
        [string]$Suffix
    )

    return [bool]($Networks | Where-Object { $_ -like "*_$Suffix" })
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " PH-1B Anti-Bypass Deployment Verification" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

try {
    Write-Host "[SETUP] Building and starting reference deployment..."

    & docker compose -f $composeFile up -d --build
    if ($LASTEXITCODE -ne 0) {
        throw "Docker Compose startup failed."
    }

    Write-Host "[SETUP] Waiting for agent -> sidecar health connectivity..."

    $ready = $false

    for ($attempt = 1; $attempt -le 30; $attempt++) {
        & docker compose -f $composeFile exec -T agent `
            sh -c 'curl -fsS http://sidecar:8080/healthz >/dev/null 2>&1'

        if ($LASTEXITCODE -eq 0) {
            $ready = $true
            break
        }

        Start-Sleep -Seconds 1
    }

    if (-not $ready) {
        throw "Sidecar did not become reachable from the agent."
    }

    Write-Host ""

    # ---------------------------------------------------------------- AB-E2E-01
    & docker compose -f $composeFile exec -T agent `
        sh -c 'curl -fsS http://sidecar:8080/healthz >/dev/null'

    if ($LASTEXITCODE -eq 0) {
        Write-TestPass `
            "AB-E2E-01" `
            "agent can reach the sidecar enforcement point"
    }
    else {
        Write-TestFail `
            "AB-E2E-01" `
            "agent cannot reach the sidecar enforcement point"
    }

    # ---------------------------------------------------------------- AB-E2E-02
    & docker compose -f $composeFile exec -T agent `
        sh -c 'curl --connect-timeout 3 -sS http://upstream:8000/ >/dev/null 2>&1'

    if ($LASTEXITCODE -ne 0) {
        Write-TestPass `
            "AB-E2E-02" `
            "agent cannot directly reach the protected upstream"
    }
    else {
        Write-TestFail `
            "AB-E2E-02" `
            "agent directly reached the protected upstream"
    }

    # ---------------------------------------------------------------- AB-E2E-03
    $upstreamId = Get-ServiceContainerId "upstream"

    $portBindingsJson = & docker inspect $upstreamId `
        --format '{{json .HostConfig.PortBindings}}'

    if ($LASTEXITCODE -ne 0) {
        throw "Failed to inspect upstream port bindings."
    }

    $portBindings = $portBindingsJson | ConvertFrom-Json
    $bindingCount = @($portBindings.PSObject.Properties).Count

    if ($bindingCount -eq 0) {
        Write-TestPass `
            "AB-E2E-03" `
            "protected upstream has no host port bindings"
    }
    else {
        Write-TestFail `
            "AB-E2E-03" `
            "protected upstream exposes host port bindings"
    }

    # ---------------------------------------------------------------- AB-E2E-04
    $upstreamResponse = & docker compose -f $composeFile exec -T sidecar `
        wget -qO- http://upstream:8000/

    if (
        $LASTEXITCODE -eq 0 -and
        $upstreamResponse -match '"source"\s*:\s*"mock-upstream"'
    ) {
        Write-TestPass `
            "AB-E2E-04" `
            "sidecar can reach the protected upstream"
    }
    else {
        Write-TestFail `
            "AB-E2E-04" `
            "sidecar cannot reach the protected upstream"
    }

    # ---------------------------------------------------------------- AB-E2E-05
    $agentId = Get-ServiceContainerId "agent"
    $sidecarId = Get-ServiceContainerId "sidecar"

    $agentNetworks = Get-ContainerNetworks $agentId
    $sidecarNetworks = Get-ContainerNetworks $sidecarId
    $upstreamNetworks = Get-ContainerNetworks $upstreamId

    $agentUntrusted = Test-NetworkMembership $agentNetworks "untrusted-net"
    $agentProtected = Test-NetworkMembership $agentNetworks "protected-net"

    $sidecarUntrusted = Test-NetworkMembership $sidecarNetworks "untrusted-net"
    $sidecarProtected = Test-NetworkMembership $sidecarNetworks "protected-net"

    $upstreamUntrusted = Test-NetworkMembership $upstreamNetworks "untrusted-net"
    $upstreamProtected = Test-NetworkMembership $upstreamNetworks "protected-net"

    $topologyValid = (
        $agentUntrusted -and
        -not $agentProtected -and
        $sidecarUntrusted -and
        $sidecarProtected -and
        -not $upstreamUntrusted -and
        $upstreamProtected
    )

    if ($topologyValid) {
        Write-TestPass `
            "AB-E2E-05" `
            "network membership preserves the anti-bypass boundary"
    }
    else {
        Write-TestFail `
            "AB-E2E-05" `
            "network membership violates the anti-bypass boundary"

        Write-Host "       agent networks:    $($agentNetworks -join ', ')"
        Write-Host "       sidecar networks:  $($sidecarNetworks -join ', ')"
        Write-Host "       upstream networks: $($upstreamNetworks -join ', ')"
    }

    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Cyan
    Write-Host " Result: $passed passed, $failed failed" -ForegroundColor Cyan
    Write-Host "============================================================" -ForegroundColor Cyan
    Write-Host ""

    if ($failed -ne 0) {
        exit 1
    }

    exit 0
}
catch {
    Write-Host ""
    Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}