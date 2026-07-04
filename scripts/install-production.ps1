# Build and start production Docker stack locally
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

if (-not (Test-Path .env)) {
    Copy-Item .env.example .env
    Write-Host "Created .env from .env.example — review before public deploy."
}

docker compose -f docker-compose.prod.yml up -d --build
Write-Host ""
Write-Host "Node:   http://127.0.0.1:8545/health"
Write-Host "Wallet: http://127.0.0.1:9338/wallet/"
