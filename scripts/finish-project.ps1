# Finish OneX Token Lab + Flash Coin project
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Write-Host "=== OneX Project Finish ===" -ForegroundColor Cyan

Write-Host "`n[1/6] Go build..."
go build -o bin/onex.exe ./cmd/onex
go build -o bin/onex-bridge.exe ./cmd/onex-bridge
go build -ldflags="-s -w" -o bin/bsc-launcher.exe ./bsc-launcher/server

Write-Host "[2/6] Go tests..."
go test ./internal/bridge/chains/... ./bsc-launcher/server/... -count=1

Write-Host "[3/6] Flash Coin contract..."
if (Test-Path "scripts\compile-flashcoin.ps1") {
  & "$PSScriptRoot\compile-flashcoin.ps1"
}

Write-Host "[4/6] BSCScan 1B USDT test quote..."
& "$PSScriptRoot\test-bscscan-1b-usdt.ps1"

$bridgeUp = $false
$bridge = if ($env:ONEX_BRIDGE_URL) { $env:ONEX_BRIDGE_URL } else { "http://127.0.0.1:9338" }
try {
  Invoke-RestMethod -Uri "$bridge/health" -TimeoutSec 2 | Out-Null
  $bridgeUp = $true
} catch {}

if ($bridgeUp) {
  Write-Host "[5/6] Regenerate mirror manifest..."
  & "$PSScriptRoot\generate-flash-coin-mirror.ps1"
} else {
  Write-Host "[5/6] Bridge offline — skip mirror regen (start run-onex-wallet.bat first)"
}

Write-Host "[6/6] Token Lab health..."
$labUp = $false
try {
  Invoke-RestMethod -Uri "http://127.0.0.1:9340/health" -TimeoutSec 2 | Out-Null
  $labUp = $true
} catch {}

Write-Host ""
Write-Host "=== DONE ===" -ForegroundColor Green
Write-Host "  Token Lab:  http://127.0.0.1:9340/"
Write-Host "  BSCScan test: configs\bscscan-1b-usdt-test.json"
Write-Host "  Liquidity UI: Liquidity -> BSCScan `$1B USDT test"
if (-not $labUp) {
  Write-Host "  Start: bsc-launcher\run-onex-token-lab.bat" -ForegroundColor Yellow
}
