# Production deploy — BSC Token Launcher (Docker)
param(
  [switch]$Proxy,
  [switch]$BuildOnly
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

if (-not (Test-Path "bsc-launcher\.env")) {
  Copy-Item "bsc-launcher\.env.production.example" "bsc-launcher\.env"
  Write-Host "Created bsc-launcher\.env — edit API keys before going live."
}

$args = @("-f", "docker-compose.bsc-launcher.yml")
if ($Proxy) { $args += "--profile", "proxy" }
$args += "up", "-d", "--build"
if ($BuildOnly) {
  docker compose -f docker-compose.bsc-launcher.yml build
  exit $LASTEXITCODE
}

docker compose @args
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Start-Sleep -Seconds 3
Invoke-RestMethod "http://127.0.0.1:9340/health" | ConvertTo-Json
Write-Host "BSC Token Launcher running on http://127.0.0.1:9340"
