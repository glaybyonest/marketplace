param(
  [switch]$SkipFrontendInstall,
  [switch]$StopContainersOnExit
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$frontendPath = Join-Path $root 'frontend'

Push-Location $root
try {
  if (-not (Test-Path '.env')) {
    Copy-Item '.env.example' '.env'
    Write-Host 'Created .env from .env.example'
    Write-Host 'Update JWT_SECRET in .env before production use.'
  }

  Write-Host 'Starting Docker services (postgres, api, adminer)...'
  docker compose up -d --build

  $readyURL = 'http://localhost:8080/readyz'
  $deadline = (Get-Date).AddMinutes(2)
  $ready = $false

  while ((Get-Date) -lt $deadline) {
    try {
      $response = Invoke-WebRequest -Uri $readyURL -UseBasicParsing -TimeoutSec 3
      if ($response.StatusCode -eq 200) {
        $ready = $true
        break
      }
    }
    catch {
      Start-Sleep -Seconds 2
    }
  }

  if ($ready) {
    Write-Host 'Backend is ready on http://localhost:8080'
  }
  else {
    Write-Warning 'Backend readiness check timed out. Use: docker compose logs -f api'
  }

  Push-Location $frontendPath
  try {
    if (-not $SkipFrontendInstall -and -not (Test-Path 'node_modules')) {
      Write-Host 'Installing frontend dependencies...'
      npm install
    }

    Write-Host 'Starting frontend dev server on http://localhost:5173'
    npm run dev
  }
  finally {
    Pop-Location
  }
}
finally {
  Pop-Location

  if ($StopContainersOnExit) {
    Push-Location $root
    try {
      Write-Host 'Stopping Docker services...'
      docker compose down
    }
    finally {
      Pop-Location
    }
  }
}
