# Deploy Flash Coin on OneX hub and mirror wrapped ERC-20 contracts on EVM chains.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$bridge = if ($env:ONEX_BRIDGE_URL) { $env:ONEX_BRIDGE_URL } else { "http://127.0.0.1:9338" }

if (-not (Test-Path "bin/onex.exe")) {
    Write-Host "Building onex CLI..."
    go build -o bin/onex.exe ./cmd/onex
}

Write-Host "Bridge: $bridge"
try {
    Invoke-RestMethod -Uri "$bridge/health" -TimeoutSec 3 | Out-Null
} catch {
    Write-Host "Starting onex-bridge in background..."
    if (-not (Test-Path "bin/onex-bridge.exe")) {
        go build -o bin/onex-bridge.exe ./cmd/onex-bridge
    }
    Start-Process -FilePath "bin/onex-bridge.exe" -WorkingDirectory $root -WindowStyle Hidden
    Start-Sleep -Seconds 3
}

& "bin/onex.exe" flash-coin-mirror -config "configs/flash-coin-mirror.json" -bridge $bridge
