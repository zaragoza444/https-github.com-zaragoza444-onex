# Finish Flash Coin project: build, compile contract, mirror, show status.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Write-Host "=== Flash Coin - finish ===" -ForegroundColor Cyan

Write-Host "`n[1/4] Building onex CLI..."
go build -o bin/onex.exe ./cmd/onex

Write-Host "[2/4] Compiling FlashCoin.sol..."
& "$PSScriptRoot\compile-flashcoin.ps1"

Write-Host "[3/4] Running tests..."
go test ./internal/bridge/chains/... ./cmd/onex/... ./bsc-launcher/...

$bridge = if ($env:ONEX_BRIDGE_URL) { $env:ONEX_BRIDGE_URL } else { "http://127.0.0.1:9338" }
$bridgeUp = $false
try {
    Invoke-RestMethod -Uri "$bridge/health" -TimeoutSec 2 | Out-Null
    $bridgeUp = $true
} catch {}

if ($bridgeUp) {
    Write-Host "[4/4] Bridge up - regenerating mirror manifest..."
    & "$PSScriptRoot\generate-flash-coin-mirror.ps1"
} else {
    Write-Host "[4/4] Bridge not running - skip mirror (start with run-onex-wallet.bat, then re-run)"
}

$livePath = "configs\flash-coin-live-addresses.json"
$mirrorPath = "configs\flash-coin-mirror-result.json"
$liveCount = 0
if (Test-Path $livePath) {
    $live = Get-Content $livePath -Raw | ConvertFrom-Json
    $liveCount = @($live.deployments | Where-Object { $_.status -eq "live" -or $_.verifiedOnChain }).Count
}
$predictedCount = 0
if (Test-Path $mirrorPath) {
    $mirror = Get-Content $mirrorPath -Raw | ConvertFrom-Json
    $predictedCount = @($mirror.steps | Where-Object { $_.chain -ne "onex-mainnet-1" }).Count
}

Write-Host ""
Write-Host "Status:" -ForegroundColor Green
Write-Host "  Predicted mirrors: $predictedCount (flash-coin-mirror-result.json)"
Write-Host "  Live on-chain:     $liveCount (flash-coin-live-addresses.json)"
Write-Host "  Dashboard:         http://127.0.0.1:9340/ (Token Lab)"
Write-Host ""
if ($liveCount -lt 7) {
    Write-Host "To complete live deploy:" -ForegroundColor Yellow
    Write-Host "  1. Set FLASH_DEPLOYER_PRIVATE_KEY in bsc-launcher\.env"
    Write-Host "  2. Fund deployer with gas on 7 EVM chains"
    Write-Host "  3. scripts\deploy-flash-coin-live.ps1"
    Write-Host ""
    Write-Host "See docs\FLASH-COIN.md for full guide."
}
