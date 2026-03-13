<#
.SYNOPSIS
Starts the full local development stack for the marketplace project.

.DESCRIPTION
This script can:
- create missing .env files from examples
- start Docker Desktop if the daemon is not running
- start postgres, api, adminer, prometheus and grafana
- apply DB migrations through the container image
- install frontend dependencies if needed
- start the Vite dev server

.PARAMETER SkipFrontendInstall
Skips npm install / npm ci even if node_modules is missing.

.PARAMETER ForceFrontendInstall
Always reinstalls frontend dependencies before starting Vite.

.PARAMETER SkipMigrations
Skips automatic DB migrations.

.PARAMETER ForegroundFrontend
Runs npm run dev in the current terminal instead of a new PowerShell window.

.PARAMETER StopContainersOnExit
Stops docker compose services after the script finishes.

.PARAMETER OpenBrowser
Opens the frontend in the default browser after startup.

.PARAMETER SkipDockerDesktopStart
Fails immediately if Docker is not running instead of trying to launch Docker Desktop.
#>
param(
  [switch]$SkipFrontendInstall,
  [switch]$ForceFrontendInstall,
  [switch]$SkipMigrations,
  [switch]$ForegroundFrontend,
  [switch]$StopContainersOnExit,
  [switch]$OpenBrowser,
  [switch]$SkipDockerDesktopStart
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$frontendPath = Join-Path $root 'frontend'
$backendReadyUrl = 'http://localhost:8080/readyz'
$prometheusReadyUrl = 'http://localhost:9090/-/ready'
$grafanaHealthUrl = 'http://localhost:3000/api/health'
$frontendUrl = 'http://localhost:5173'
$frontendProcess = $null
$shouldStopContainers = $StopContainersOnExit.IsPresent

function Write-Step {
  param([string]$Message)
  Write-Host "==> $Message" -ForegroundColor Cyan
}

function Assert-Command {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Required command '$Name' was not found in PATH."
  }
}

function Copy-FileIfMissing {
  param(
    [string]$SourcePath,
    [string]$TargetPath
  )

  if (-not (Test-Path $TargetPath)) {
    Copy-Item $SourcePath $TargetPath
    Write-Host "Created $TargetPath from $SourcePath"
  }
}

function Test-HttpOk {
  param(
    [string]$Url,
    [int]$TimeoutSeconds = 3
  )

  try {
    $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec $TimeoutSeconds
    return $response.StatusCode -ge 200 -and $response.StatusCode -lt 400
  }
  catch {
    return $false
  }
}

function Wait-ForHttpOk {
  param(
    [string]$Url,
    [int]$TimeoutSeconds = 120,
    [int]$DelaySeconds = 2
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    if (Test-HttpOk -Url $Url) {
      return $true
    }
    Start-Sleep -Seconds $DelaySeconds
  }

  return $false
}

function Get-DockerDesktopPath {
  $roots = @($env:ProgramFiles, $env:ProgramW6432, $env:LocalAppData) | Where-Object { $_ }
  $candidates = @()
  foreach ($rootPath in $roots) {
    if ($rootPath -eq $env:LocalAppData) {
      $candidates += Join-Path $rootPath 'Programs\Docker\Docker\Docker Desktop.exe'
    }
    else {
      $candidates += Join-Path $rootPath 'Docker\Docker\Docker Desktop.exe'
    }
  }
  $candidates = $candidates | Where-Object { Test-Path $_ }

  if ($candidates.Count -gt 0) {
    return $candidates[0]
  }

  return $null
}

function Wait-ForDockerDaemon {
  param([int]$TimeoutSeconds = 180)

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    docker info *> $null
    if ($LASTEXITCODE -eq 0) {
      return $true
    }
    Start-Sleep -Seconds 3
  }

  return $false
}

function Ensure-DockerAvailable {
  docker info *> $null
  if ($LASTEXITCODE -eq 0) {
    return
  }

  if ($SkipDockerDesktopStart) {
    throw 'Docker daemon is not available. Start Docker Desktop and retry.'
  }

  $dockerDesktopPath = Get-DockerDesktopPath
  if (-not $dockerDesktopPath) {
    throw 'Docker daemon is not available and Docker Desktop.exe was not found automatically.'
  }

  Write-Step 'Starting Docker Desktop'
  Start-Process -FilePath $dockerDesktopPath | Out-Null

  if (-not (Wait-ForDockerDaemon -TimeoutSeconds 180)) {
    throw 'Docker Desktop did not become ready in time.'
  }
}

function Ensure-EnvFiles {
  Write-Step 'Ensuring local env files'
  Copy-FileIfMissing -SourcePath (Join-Path $root '.env.example') -TargetPath (Join-Path $root '.env')
  Copy-FileIfMissing -SourcePath (Join-Path $frontendPath '.env.example') -TargetPath (Join-Path $frontendPath '.env')
}

function Wait-ForContainerHealthy {
  param(
    [string]$ContainerName,
    [int]$TimeoutSeconds = 120
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    $status = docker inspect --format "{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}" $ContainerName 2>$null
    if ($LASTEXITCODE -eq 0 -and ($status -eq 'healthy' -or $status -eq 'running')) {
      return $true
    }
    Start-Sleep -Seconds 2
  }

  return $false
}

function Start-PostgresService {
  Write-Step 'Starting postgres'
  docker compose up -d --build postgres
  if ($LASTEXITCODE -ne 0) {
    throw 'docker compose up failed.'
  }
}

function Start-RemainingServices {
  Write-Step 'Starting api, adminer, prometheus and grafana'
  docker compose up -d --build api adminer prometheus grafana
  if ($LASTEXITCODE -ne 0) {
    throw 'docker compose up failed.'
  }
}

function Run-DatabaseMigrations {
  if ($SkipMigrations) {
    Write-Host 'Skipping DB migrations.'
    return
  }

  Write-Step 'Applying database migrations'
  docker compose run --rm api /app/migrate up
  if ($LASTEXITCODE -ne 0) {
    throw 'DB migrations failed.'
  }
}

function Ensure-FrontendDependencies {
  if ($SkipFrontendInstall) {
    Write-Host 'Skipping frontend dependency installation.'
    return
  }

  Push-Location $frontendPath
  try {
    $needsInstall = $ForceFrontendInstall -or -not (Test-Path (Join-Path $frontendPath 'node_modules'))
    if (-not $needsInstall) {
      Write-Host 'Frontend dependencies already present.'
      return
    }

    Write-Step 'Installing frontend dependencies'
    if (Test-Path (Join-Path $frontendPath 'package-lock.json')) {
      npm ci
    }
    else {
      npm install
    }

    if ($LASTEXITCODE -ne 0) {
      throw 'Frontend dependency installation failed.'
    }
  }
  finally {
    Pop-Location
  }
}

function Get-ChildPowerShell {
  if (Get-Command pwsh.exe -ErrorAction SilentlyContinue) {
    return 'pwsh.exe'
  }

  return 'powershell.exe'
}

function Start-FrontendServer {
  if (Test-HttpOk -Url $frontendUrl -TimeoutSeconds 2) {
    Write-Host "Frontend already responds on $frontendUrl"
    return $null
  }

  if ($ForegroundFrontend) {
    Write-Step 'Starting frontend dev server in current terminal'
    Push-Location $frontendPath
    try {
      npm run dev
    }
    finally {
      Pop-Location
    }
    return $null
  }

  Write-Step 'Starting frontend dev server in a new PowerShell window'
  $shell = Get-ChildPowerShell
  $command = "Set-Location -LiteralPath '$frontendPath'; npm run dev"
  $process = Start-Process -FilePath $shell -ArgumentList @('-NoExit', '-ExecutionPolicy', 'Bypass', '-Command', $command) -PassThru

  if (Wait-ForHttpOk -Url $frontendUrl -TimeoutSeconds 90 -DelaySeconds 2) {
    Write-Host "Frontend is ready on $frontendUrl"
  }
  else {
    Write-Warning "Frontend did not become reachable in time. Check the new window or run: cd frontend; npm run dev"
  }

  return $process
}

function Show-ServiceSummary {
  Write-Host ''
  Write-Host 'Local services:'
  Write-Host '  Frontend:    http://localhost:5173'
  Write-Host '  Backend API: http://localhost:8080'
  Write-Host '  Swagger UI:  http://localhost:8080/docs/'
  Write-Host '  Adminer:     http://localhost:8081'
  Write-Host '  Prometheus:  http://localhost:9090'
  Write-Host '  Grafana:     http://localhost:3000'
  Write-Host ''
  Write-Host 'Useful commands:'
  Write-Host '  docker compose logs -f api'
  Write-Host '  docker compose down'
}

Push-Location $root
try {
  Assert-Command -Name 'docker'
  Assert-Command -Name 'npm'

  Ensure-EnvFiles
  Ensure-DockerAvailable
  Start-PostgresService
  if (-not (Wait-ForContainerHealthy -ContainerName 'marketplace-postgres' -TimeoutSeconds 120)) {
    Write-Warning 'Postgres health check timed out. Migrations may fail. Inspect: docker compose logs -f postgres'
  }

  Run-DatabaseMigrations
  Start-RemainingServices

  if (Wait-ForHttpOk -Url $backendReadyUrl -TimeoutSeconds 180 -DelaySeconds 2) {
    Write-Host 'Backend is ready.'
  }
  else {
    Write-Warning 'Backend readiness check timed out. Inspect: docker compose logs -f api'
  }

  if (Wait-ForHttpOk -Url $prometheusReadyUrl -TimeoutSeconds 60 -DelaySeconds 2) {
    Write-Host 'Prometheus is ready.'
  }

  if (Wait-ForHttpOk -Url $grafanaHealthUrl -TimeoutSeconds 60 -DelaySeconds 2) {
    Write-Host 'Grafana is ready.'
  }

  Ensure-FrontendDependencies
  $frontendProcess = Start-FrontendServer

  if ($StopContainersOnExit -and -not $ForegroundFrontend -and -not $frontendProcess) {
    Write-Warning 'StopContainersOnExit was ignored because frontend was already running or was not started by this script.'
    $shouldStopContainers = $false
  }

  if ($OpenBrowser -and (Test-HttpOk -Url $frontendUrl -TimeoutSeconds 2)) {
    Start-Process $frontendUrl | Out-Null
  }

  Show-ServiceSummary

  if ($StopContainersOnExit -and -not $ForegroundFrontend -and $frontendProcess) {
    Write-Host 'Waiting for the frontend window to exit before stopping containers...'
    Wait-Process -Id $frontendProcess.Id
  }
}
finally {
  Pop-Location

  if ($shouldStopContainers) {
    Push-Location $root
    try {
      Write-Step 'Stopping docker compose services'
      docker compose down
    }
    finally {
      Pop-Location
    }
  }
}
