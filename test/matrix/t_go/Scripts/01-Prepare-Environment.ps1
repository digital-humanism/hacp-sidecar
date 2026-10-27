#Requires -Version 7.0
<#
01-Prepare-Environment.ps1 — CONTROLLED MUTATING ENVIRONMENT PREPARATION (PHASE 1-6 prerequisites).
CLASS = VERIFICATION TOOLING. Only operations already approved by contracts. Does NOT run
verification tests. NO version substitution, NO "latest", NO toolchain upgrade. Build outputs
go to BUILD_ROOT (outside tracked repos) unless a contract requires otherwise. Does NOT leave
services running (services start in 02). Does NOT admit/stage/commit.
#>

[CmdletBinding()]
param(
    [string]$SidecarRoot = '',
    [string]$SpecRoot    = '',
    [string]$CoreRoot    = '',
    [string]$RunRoot     = '',
    [string]$EvidenceRoot= '',
    [switch]$SkipDependencyPrep,
    [switch]$SkipBuilds
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Portable default path resolution.
# Explicit caller-supplied paths take precedence.
if (-not $SidecarRoot) {
    $SidecarRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
}

$WorkspaceRoot = Split-Path -Parent $SidecarRoot

if (-not $SpecRoot) {
    $SpecRoot = Join-Path $WorkspaceRoot 'hacp-spec'
}

if (-not $CoreRoot) {
    $CoreRoot = Join-Path $WorkspaceRoot 'humanist-core'
}

if (-not $RunRoot -or -not $EvidenceRoot) {
    $VerificationRoot = Join-Path ([System.IO.Path]::GetTempPath()) 'hacp-1.1.1-verification'
}

if (-not $RunRoot) {
    $RunRoot = Join-Path $VerificationRoot 'run'
}

if (-not $EvidenceRoot) {
    $EvidenceRoot = Join-Path $VerificationRoot 'evidence'
}

Import-Module (Join-Path $PSScriptRoot 'WindowsVerification.Common.psm1') -Force

$BUILD_ROOT  = New-WvDirectory -Path (Join-Path $RunRoot 'build')
$FIXTURE_ROOT= New-WvDirectory -Path (Join-Path $RunRoot 'fixtures')
$LOG_ROOT    = New-WvDirectory -Path (Join-Path $RunRoot 'logs')
$EVID_ROOT   = New-WvDirectory -Path $EvidenceRoot
$PREP_EVID   = New-WvDirectory -Path (Join-Path $EVID_ROOT '01-Prepare')
Set-WvLogFile -Path (Join-Path $LOG_ROOT '01-prepare-console.log')
Write-WvLog -Message "01-Prepare-Environment start"

$paths = [ordered]@{ RUN_ROOT=$RunRoot; BUILD_ROOT=$BUILD_ROOT; FIXTURE_ROOT=$FIXTURE_ROOT; LOG_ROOT=$LOG_ROOT; EVIDENCE_ROOT=$EVID_ROOT }
Write-WvJson -Object $paths -Path (Join-Path $PREP_EVID 'external-directories.json')

# ---- 1. Dependency preparation (approved only; no version substitution) ----
$depSteps = @()
if (-not $SkipDependencyPrep) {
    $deps = @(
        @{ Id='dep-p-harness';   Name='P harness reqs';   Exe='pip'; Args=@('install','-r', (Join-Path $SpecRoot 'harness\requirements.txt')); Work=$SpecRoot },
        @{ Id='dep-c-dev';       Name='C dev (conformance)';Exe='pip';Args=@('install','-e','.[dev]'); Work=$CoreRoot },
        @{ Id='dep-ts-npm';      Name='hacp-ts npm install';Exe='npm.cmd';Args=@('install'); Work=(Join-Path $SpecRoot 'hacp-ts') },
        @{ Id='dep-s-gomod';     Name='S go mod download';  Exe='go'; Args=@('mod','download'); Work=$SidecarRoot }
    )
    foreach ($d in $deps) {
        if (-not (Test-Path -LiteralPath $d.Work)) {
            $depSteps += New-WvStepResult -StepId $d.Id -SurfaceId 'ENV' -Command "$($d.Exe) $($d.Args -join ' ')" `
                -StartTimeUtc (Get-Date).ToUniversalTime().ToString('o') -EndTimeUtc (Get-Date).ToUniversalTime().ToString('o') `
                -ProcessRc $null -Classification $WV_CLASS.ENVIRONMENT_RED -EvidencePath $PREP_EVID -Status "WORKDIR_MISSING:$($d.Work)"
            continue
        }
        $run = Invoke-WvNative -FilePath $d.Exe -ArgumentList $d.Args -WorkDir $d.Work -StepId $d.Id -SurfaceId 'ENV' -EvidenceDir $PREP_EVID
        $cls = if ($run.ProcessRc -eq 0) { $WV_CLASS.PASS } else { $WV_CLASS.ENVIRONMENT_RED }
        $depSteps += New-WvStepResult -StepId $d.Id -SurfaceId 'ENV' -Command $run.Command `
            -StartTimeUtc $run.StartTimeUtc -EndTimeUtc $run.EndTimeUtc -ProcessRc $run.ProcessRc `
            -Classification $cls -EvidencePath $PREP_EVID -Status $d.Name
    }
} else {
    Write-WvLog -Level WARN -Message "SkipDependencyPrep set: dependency prep NOT run"
}
Write-WvJson -Object $depSteps -Path (Join-Path $PREP_EVID 'dependency-steps.json')

# ---- 2. Builds (approved builds only; outputs to BUILD_ROOT) ----
$buildSteps = @(); $buildInventory = @()
if (-not $SkipBuilds) {
    $builds = @(
        @{ Id='build-sidecar';   Surface='S';   Exe='go'; Args=@('build','-o',(Join-Path $BUILD_ROOT 'hacp-sidecar.exe'),'.\cmd\sidecar'); Work=$SidecarRoot; Out='hacp-sidecar.exe' },
        @{ Id='build-conformance-runner'; Surface='S05'; Exe='go'; Args=@('build','-o',(Join-Path $BUILD_ROOT 'hacp-conformance-runner.exe'),'.\cmd\hacp-conformance-runner'); Work=$SidecarRoot; Out='hacp-conformance-runner.exe' },
        @{ Id='build-graceful-shutdown'; Surface='B'; Exe='go'; Args=@('build','-o',(Join-Path $BUILD_ROOT 'graceful-shutdown.exe'),'.'); Work=(Join-Path $SidecarRoot 'tools\reproduction\graceful-shutdown'); Out='graceful-shutdown.exe' },
        @{ Id='build-go-http-client'; Surface='D'; Exe='go'; Args=@('build','-o',(Join-Path $BUILD_ROOT 'go-http-reproduction.exe'),'.'); Work=(Join-Path $SidecarRoot 'benchmarks\go-http-reproduction'); Out='go-http-reproduction.exe' },
        @{ Id='build-hacp-go';   Surface='S01'; Exe='go'; Args=@('build','-o',(Join-Path $BUILD_ROOT 'hacp-go.exe'),'.'); Work=(Join-Path $SpecRoot 'hacp-go'); Out='hacp-go.exe' },
        @{ Id='build-hacp-ts';   Surface='S02'; Exe='npm.cmd'; Args=@('run','build'); Work=(Join-Path $SpecRoot 'hacp-ts'); Out='dist\src\cli.js'; OutFromWork=$true }
    )
    $rustBuilds = @(
        @{ Id='build-hacp-rs-runner'; Surface='RS01'; Bin='hacp-rs-runner' }
    )
    foreach ($b in $builds) {
        if (-not (Test-Path -LiteralPath $b.Work)) {
            $buildSteps += New-WvStepResult -StepId $b.Id -SurfaceId $b.Surface -Command "$($b.Exe) $($b.Args -join ' ')" `
                -StartTimeUtc (Get-Date).ToUniversalTime().ToString('o') -EndTimeUtc (Get-Date).ToUniversalTime().ToString('o') `
                -ProcessRc $null -Classification $WV_CLASS.ENVIRONMENT_RED -EvidencePath $PREP_EVID -Status "WORKDIR_MISSING:$($b.Work)"
            continue
        }
        $run = Invoke-WvNative -FilePath $b.Exe -ArgumentList $b.Args -WorkDir $b.Work -StepId $b.Id -SurfaceId $b.Surface -EvidenceDir $PREP_EVID
        # D-1: canonical output identity. TS builds emit into Work (npm run build -> Work\dist),
        # go builds emit into BUILD_ROOT (-o). Use the actual producer path, matching the 02 consumer.
        $outPath = if ($b.ContainsKey('OutFromWork')) { Join-Path $b.Work $b.Out } else { Join-Path $BUILD_ROOT $b.Out }
        $fi = Get-WvFileIdentity -Path $outPath
        $cls = if ($run.ProcessRc -eq 0 -and $fi.Exists) { $WV_CLASS.PASS } else { $WV_CLASS.ENVIRONMENT_RED }
        $buildSteps += New-WvStepResult -StepId $b.Id -SurfaceId $b.Surface -Command $run.Command `
            -StartTimeUtc $run.StartTimeUtc -EndTimeUtc $run.EndTimeUtc -ProcessRc $run.ProcessRc `
            -Classification $cls -EvidencePath $PREP_EVID -Status "OUT=$outPath"
        $buildInventory += [pscustomobject]@{ BUILD_ID=$b.Id; SURFACE_ID=$b.Surface; PATH=$outPath; SIZE_BYTES=$fi.SizeBytes; SHA256=$fi.Sha256; BUILD_COMMAND=$run.Command; PROCESS_RC=$run.ProcessRc; SOURCE_HEAD=(Get-WvGitIdentity -RepoRoot $b.Work).Head; TOOLCHAIN='see toolchain-inventory' }
    }
    # S07: prepare the verification-owned agent image before execution.
    # No production Compose changes and no verification assertions here.
    $agentImage = 'hacp-s07-test-agent:alpine-3.22-curl-v1'
    $agentContext = Join-Path $SidecarRoot 'test\deployments'
    $agentDockerfile = Join-Path $agentContext 'Dockerfile.agent'
    $agentDockerfileHash = (Get-FileHash -LiteralPath $agentDockerfile -Algorithm SHA256).Hash

    $agentBuild = Invoke-WvNative -FilePath 'docker' `
        -ArgumentList @('build','--file',$agentDockerfile,'--tag',$agentImage,$agentContext) `
        -WorkDir $SidecarRoot -StepId 'build-s07-agent' -SurfaceId 'S07' -EvidenceDir $PREP_EVID

    $agentImageId = $null
    $agentCurlVersion = $null
    $agentReady = $false

    if ($agentBuild.ProcessRc -eq 0) {
        $agentInspect = Invoke-WvNative -FilePath 'docker' `
            -ArgumentList @('image','inspect',$agentImage,'--format','{{.Id}}') `
            -WorkDir $SidecarRoot -StepId 'build-s07-agent-inspect' -SurfaceId 'S07' -EvidenceDir $PREP_EVID

        if ($agentInspect.ProcessRc -eq 0) {
            $agentImageId = (Get-Content -LiteralPath $agentInspect.StdOutPath -Raw).Trim()

            if ($agentImageId -match '^sha256:[0-9a-f]{64}$') {
                $agentCurl = Invoke-WvNative -FilePath 'docker' `
                    -ArgumentList @('run','--rm','--entrypoint','curl',$agentImage,'--version') `
                    -WorkDir $SidecarRoot -StepId 'build-s07-agent-curl' -SurfaceId 'S07' -EvidenceDir $PREP_EVID

                if ($agentCurl.ProcessRc -eq 0) {
                    $agentCurlVersion = (Get-Content -LiteralPath $agentCurl.StdOutPath -Raw).Trim()
                    $agentReady = ($agentCurlVersion -match '^curl\s+\S+')
                }
            }
        }
    }

    $agentCls = if ($agentReady) {
        $WV_CLASS.PASS
    } else {
        $WV_CLASS.ENVIRONMENT_RED
    }

    $buildSteps += New-WvStepResult `
        -StepId 'build-s07-agent' `
        -SurfaceId 'S07' `
        -Command $agentBuild.Command `
        -StartTimeUtc $agentBuild.StartTimeUtc `
        -EndTimeUtc (Get-Date).ToUniversalTime().ToString('o') `
        -ProcessRc $agentBuild.ProcessRc `
        -Classification $agentCls `
        -EvidencePath $PREP_EVID `
        -Status "IMAGE=$agentImage; IMAGE_ID=$agentImageId; CURL_READY=$agentReady"

    $buildInventory += [pscustomobject]@{
        BUILD_ID          = 'build-s07-agent'
        SURFACE_ID        = 'S07'
        IMAGE             = $agentImage
        IMAGE_ID          = $agentImageId
        DOCKERFILE        = $agentDockerfile
        DOCKERFILE_SHA256 = $agentDockerfileHash
        CURL_VERSION      = $agentCurlVersion
        BUILD_COMMAND     = $agentBuild.Command
        PROCESS_RC        = $agentBuild.ProcessRc
        CLASSIFICATION    = $agentCls
        SOURCE_HEAD       = (Get-WvGitIdentity -RepoRoot $SidecarRoot).Head
        TOOLCHAIN         = 'Docker'
    }
    # Windows cargo rejects CARGO_TARGET_DIR when it is explicitly empty.
    # Normalize empty -> absent before repository-local Rust builds.
    if ((Test-Path Env:CARGO_TARGET_DIR) -and [string]::IsNullOrEmpty($env:CARGO_TARGET_DIR)) {
        Remove-Item Env:CARGO_TARGET_DIR -ErrorAction SilentlyContinue
    }
    foreach ($rb in $rustBuilds) {
        $work = Join-Path $SpecRoot 'hacp-rs'
        $env:ARGP = $null
        $run = Invoke-WvNative -FilePath 'cargo' -ArgumentList @('+stable-x86_64-pc-windows-msvc','build','--locked','--release','--bin',$rb.Bin) -WorkDir $work -StepId $rb.Id -SurfaceId $rb.Surface -EvidenceDir $PREP_EVID
        $outPath = Join-Path (Join-Path $work 'target\release') "$($rb.Bin).exe"
        $fi = Get-WvFileIdentity -Path $outPath
        $cls = if ($run.ProcessRc -eq 0 -and $fi.Exists) { $WV_CLASS.PASS } else { $WV_CLASS.ENVIRONMENT_RED }
        $buildSteps += New-WvStepResult -StepId $rb.Id -SurfaceId $rb.Surface -Command $run.Command `
            -StartTimeUtc $run.StartTimeUtc -EndTimeUtc $run.EndTimeUtc -ProcessRc $run.ProcessRc `
            -Classification $cls -EvidencePath $PREP_EVID -Status "OUT=$outPath"
        $buildInventory += [pscustomobject]@{ BUILD_ID=$rb.Id; SURFACE_ID=$rb.Surface; PATH=$outPath; SIZE_BYTES=$fi.SizeBytes; SHA256=$fi.Sha256; BUILD_COMMAND=$run.Command; PROCESS_RC=$run.ProcessRc; SOURCE_HEAD=(Get-WvGitIdentity -RepoRoot $work).Head; TOOLCHAIN='rust stable-msvc' }
    }
    # DEFECT-6 / S08: exact R8.2 proxy build contract.
    # Runner remains on the existing release path; only S08 proxy uses this external target.
    $proxyWork = Join-Path $SpecRoot 'hacp-rs'
    $proxyTarget = New-WvDirectory -Path (Join-Path $RunRoot 'cargo-target-s08')
    $savedCargoTargetDir = [Environment]::GetEnvironmentVariable('CARGO_TARGET_DIR')
    try {
        [Environment]::SetEnvironmentVariable('CARGO_TARGET_DIR', $proxyTarget)
        $env:ARGP = $null
        $proxyRun = Invoke-WvNative -FilePath 'cargo' -ArgumentList @('+stable-x86_64-pc-windows-msvc','build','--locked','--offline','--bin','hacp-rs-proxy') -WorkDir $proxyWork -StepId 'build-hacp-rs-proxy' -SurfaceId 'RS04' -EvidenceDir $PREP_EVID
    } finally {
        if ([string]::IsNullOrEmpty($savedCargoTargetDir)) {
            Remove-Item Env:CARGO_TARGET_DIR -ErrorAction SilentlyContinue
        } else {
            [Environment]::SetEnvironmentVariable('CARGO_TARGET_DIR', $savedCargoTargetDir)
        }
    }

    $proxyOutPath = Join-Path (Join-Path $proxyTarget 'debug') 'hacp-rs-proxy.exe'
    $proxyFi = Get-WvFileIdentity -Path $proxyOutPath
    $proxyCls = if ($proxyRun.ProcessRc -eq 0 -and $proxyFi.Exists) { $WV_CLASS.PASS } else { $WV_CLASS.ENVIRONMENT_RED }

    $buildSteps += New-WvStepResult -StepId 'build-hacp-rs-proxy' -SurfaceId 'RS04' -Command $proxyRun.Command `
        -StartTimeUtc $proxyRun.StartTimeUtc -EndTimeUtc $proxyRun.EndTimeUtc -ProcessRc $proxyRun.ProcessRc `
        -Classification $proxyCls -EvidencePath $PREP_EVID -Status "OUT=$proxyOutPath"

    $buildInventory += [pscustomobject]@{
        BUILD_ID='build-hacp-rs-proxy'
        SURFACE_ID='RS04'
        PATH=$proxyOutPath
        SIZE_BYTES=$proxyFi.SizeBytes
        SHA256=$proxyFi.Sha256
        BUILD_COMMAND=$proxyRun.Command
        PROCESS_RC=$proxyRun.ProcessRc
        SOURCE_HEAD=(Get-WvGitIdentity -RepoRoot $proxyWork).Head
        TOOLCHAIN='rust stable-msvc'
    }
} else {
    Write-WvLog -Level WARN -Message "SkipBuilds set: builds NOT run"
}
Write-WvJson -Object $buildSteps -Path (Join-Path $PREP_EVID 'build-steps.json')
Write-WvJson -Object $buildInventory -Path (Join-Path $PREP_EVID 'build-inventory.json')

# ---- 3. Fixture generation (contract-defined only; fresh; no historical reuse) ----
$fixtureSpec = @(
    @{ Id='fx-b-h03';    Surface='B';   Count=1;    Out='tokens-h03.jsonl'; TtlSource='fixture TTL 7200s (generate-tokens/main.go)' },
    @{ Id='fx-d-warmup'; Surface='D';   Count=100;  Out='d-warmup.jsonl';   TtlSource='fixture TTL 7200s' },
    @{ Id='fx-d-measured';Surface='D';  Count=5000; Out='d-measured.jsonl'; TtlSource='fixture TTL 7200s' },
    @{ Id='fx-s08-warmup';Surface='S08';Count=100;  Out='s08-warmup.jsonl'; TtlSource='fixture TTL 7200s' },
    @{ Id='fx-s08-measured';Surface='S08';Count=5000;Out='s08-measured.jsonl';TtlSource='fixture TTL 7200s' }
    # D-2 follow-up: S10 fixture is orchestration-owned at the exact repo-local path
    # consumed by all three benchmark entrypoints: benchmarks\tokens.jsonl.
)
$fixtureInventory = @()
$genWork = Join-Path $SidecarRoot 'benchmarks\generate-tokens'
foreach ($f in $fixtureSpec) {
    $outPath = Join-Path $FIXTURE_ROOT $f.Out
    $run = Invoke-WvNative -FilePath 'go' -ArgumentList @('run','.','-count',"$($f.Count)",'-out',$outPath,'-method','GET','-path','/api/test') `
        -WorkDir $genWork -StepId $f.Id -SurfaceId $f.Surface -EvidenceDir $PREP_EVID
    $fi = Get-WvFileIdentity -Path $outPath
    $cls = if ($run.ProcessRc -eq 0 -and $fi.Exists) { $WV_CLASS.PASS } else { $WV_CLASS.ENVIRONMENT_RED }
    $fixtureInventory += [pscustomobject]@{
        FIXTURE_ID=$f.Id; SURFACE_ID=$f.Surface; PATH=$outPath
        GENERATOR_COMMAND=$run.Command; GENERATOR_IDENTITY='hacp-sidecar/benchmarks/generate-tokens'
        GENERATED_AT_UTC=$run.StartTimeUtc; RECORD_COUNT=$f.Count; SIZE_BYTES=$fi.SizeBytes; SHA256=$fi.Sha256
        TTL_SOURCE=$f.TtlSource; TOKEN_VALIDITY_AT_START=$(if ($cls -eq $WV_CLASS.PASS) { 'PASS' } else { 'ENVIRONMENT_RED' })
        CLASSIFICATION=$cls
    }
}
# ---- 3A. S10 repo-local fresh fixture (exact S10 contract path) ----
$s10FixturePath = Join-Path $SidecarRoot 'benchmarks\tokens.jsonl'
$s10FixtureRun = Invoke-WvNative -FilePath 'go' `
    -ArgumentList @('run','.\benchmarks\generate-tokens','-count','1000','-out','benchmarks\tokens.jsonl','-method','GET','-path','/api/test') `
    -WorkDir $SidecarRoot `
    -StepId 'fx-s10-tokens' `
    -SurfaceId 'S10' `
    -EvidenceDir $PREP_EVID

$s10FixtureFi = Get-WvFileIdentity -Path $s10FixturePath
$s10FixtureCls = if ($s10FixtureRun.ProcessRc -eq 0 -and $s10FixtureFi.Exists) {
    $WV_CLASS.PASS
} else {
    $WV_CLASS.ENVIRONMENT_RED
}

$fixtureInventory += [pscustomobject]@{
    FIXTURE_ID             = 'fx-s10-tokens'
    SURFACE_ID             = 'S10'
    PATH                   = $s10FixturePath
    GENERATOR_COMMAND      = $s10FixtureRun.Command
    GENERATOR_IDENTITY     = 'hacp-sidecar/benchmarks/generate-tokens'
    GENERATED_AT_UTC       = $s10FixtureRun.StartTimeUtc
    RECORD_COUNT           = 1000
    SIZE_BYTES             = $s10FixtureFi.SizeBytes
    SHA256                 = $s10FixtureFi.Sha256
    TTL_SOURCE             = 'fixture TTL 7200s (generate-tokens/main.go)'
    TOKEN_VALIDITY_AT_START= $(if ($s10FixtureCls -eq $WV_CLASS.PASS) { 'PASS' } else { 'ENVIRONMENT_RED' })
    CLASSIFICATION         = $s10FixtureCls
}
Write-WvJson -Object $fixtureInventory -Path (Join-Path $PREP_EVID 'fixture-inventory.json')

# ---- S06 existing local test key placement ----
$s06Source = Join-Path $SidecarRoot 'test\gitcheck-reproduction\hacp-test-key.pem'
$s06Target = Join-Path $RunRoot 'keys\hacp-test-key.pem'

if (-not (Test-Path -LiteralPath $s06Source -PathType Leaf)) {
    throw 'S06_ENVIRONMENT_RED: repository-local test fixture missing'
}
if ((Get-FileHash -LiteralPath $s06Source -Algorithm SHA256).Hash -ne '80C25A7B8450176CA940C19AA76D1136B4EA547A0FB37698AC535F0949457B80') {
    throw 'S06_ENVIRONMENT_RED: test fixture SHA-256 mismatch'
}

[void](New-Item -ItemType Directory -Path (Split-Path $s06Target -Parent) -Force)
Copy-Item -LiteralPath $s06Source -Destination $s06Target -Force

$env:HACP_TEST_PRIVATE_KEY = $s06Target
$env:HACP_TEST_SIGNER_KEY_ID = 'key-ed25519-test-001'

# ---- 4. S06 key material readiness (approved procedure; never log secret bytes) ----
# Key is the deterministic conformance Ed25519 identity (Integration doc §8). Preparation is
# presence + identity verification only; private bytes/seed are NEVER written to logs/summary.
$s06KeyPath = [Environment]::GetEnvironmentVariable('HACP_TEST_PRIVATE_KEY')
$s06KeyId  = [Environment]::GetEnvironmentVariable('HACP_TEST_SIGNER_KEY_ID')
$s06KeyReady = if ($s06KeyPath -and (Test-Path -LiteralPath $s06KeyPath) -and $s06KeyId) { 'READY' } else { 'ENVIRONMENT_RED' }
$s06Key = [ordered]@{
    HACP_TEST_PRIVATE_KEY_STATE = if ($s06KeyPath) { 'SET' } else { 'UNSET' }
    HACP_TEST_PRIVATE_KEY_FILE_EXISTS = if ($s06KeyPath) { (Test-Path -LiteralPath $s06KeyPath) } else { $false }
    HACP_TEST_SIGNER_KEY_ID_STATE = if ($s06KeyId) { 'SET' } else { 'UNSET' }
    KEY_READINESS = $s06KeyReady
    PROCEDURE_REF = 'humanist-core/docs/Integration with HACP Sidecar.md (S06 EXACT MODE; PKCS8 PEM; key id key-ed25519-test-001)'
    PRIVATE_BYTES_LOGGED = $false
}
Write-WvJson -Object $s06Key -Path (Join-Path $PREP_EVID 's06-key-readiness.json')

# ---- 5. Environment preparation status ----
$envPrepStatus = [ordered]@{
    ENVIRONMENT_PREPARATION = 'READY'
    EXTERNAL_DIRECTORIES_CREATED = 'YES'
    DEPENDENCY_STEPS = @($depSteps).Count
    DEPENDENCY_RED = @($depSteps | Where-Object { $_.CLASSIFICATION -ne $WV_CLASS.PASS }).Count
    BUILD_RED = @($buildSteps | Where-Object { $_.CLASSIFICATION -ne $WV_CLASS.PASS }).Count
    FIXTURE_RED = @($fixtureInventory | Where-Object { $_.CLASSIFICATION -ne $WV_CLASS.PASS }).Count
    S06_KEY_READINESS = $s06KeyReady
    NOTE = 'Per-surface BLOCKED/ENVIRONMENT_RED is evaluated at 02 execution; services start in 02, not here.'
}
if ($envPrepStatus.DEPENDENCY_RED -gt 0 -or $envPrepStatus.BUILD_RED -gt 0 -or $envPrepStatus.FIXTURE_RED -gt 0) {
    $envPrepStatus.ENVIRONMENT_PREPARATION = 'READY_WITH_REDS'
}
Write-WvJson -Object $envPrepStatus -Path (Join-Path $PREP_EVID 'environment-preparation.json')

Write-WvLog -Message "ENVIRONMENT_PREPARATION=$($envPrepStatus.ENVIRONMENT_PREPARATION). Review $PREP_EVID"
# NO verification tests, NO services left running, NO staging/commit.
