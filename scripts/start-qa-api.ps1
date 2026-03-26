param(
  [string]$HttpPort = "18080",
  [string]$DatabaseUrl = "postgres://postgres:postgres@localhost:5433/marketplace?sslmode=disable",
  [string]$JwtSecret = "local-dev-jwt-secret-not-for-production",
  [string]$AppBaseUrl = "http://localhost:5173",
  [string]$MailFrom = "no-reply@marketplace.local",
  [string]$LogLevel = "info"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
  $env:APP_ENV = "development"
  $env:HTTP_PORT = $HttpPort
  $env:DATABASE_URL = $DatabaseUrl
  $env:JWT_SECRET = $JwtSecret
  $env:APP_BASE_URL = $AppBaseUrl
  $env:MAIL_FROM = $MailFrom
  $env:LOG_LEVEL = $LogLevel
  $env:AUTH_COOKIE_MODE = "false"
  $env:AUTH_CSRF_ENABLED = "true"
  $env:JOBS_ENABLED = "true"

  go run ./cmd/api
}
finally {
  Pop-Location
}
