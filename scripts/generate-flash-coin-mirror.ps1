# Deploy Flash Coin on OneX hub and mirror wrapped contracts on EVM chains (CREATE2 same address).
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$bridge = if ($env:ONEX_BRIDGE_URL) { $env:ONEX_BRIDGE_URL } else { "http://127.0.0.1:9338" }

Write-Host "Building onex + bridge..."
go build -o bin/onex.exe ./cmd/onex
go build -o bin/onex-bridge.exe ./cmd/onex-bridge

Write-Host "Bridge: $bridge"
try {
    Invoke-RestMethod -Uri "$bridge/health" -TimeoutSec 3 | Out-Null
} catch {
    Write-Host "Starting onex-bridge in background..."
    taskkill /IM onex-bridge.exe /F 2>$null | Out-Null
    Start-Process -FilePath "bin/onex-bridge.exe" -WorkingDirectory $root -WindowStyle Hidden
    Start-Sleep -Seconds 4
}

& "bin/onex.exe" flash-coin-mirror -config "configs/flash-coin-mirror.json" -bridge $bridge
