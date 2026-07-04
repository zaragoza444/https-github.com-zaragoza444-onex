# Deploy wFLASH to real mainnet contract addresses (one per mirror chain).
# Requires funded deployer wallet with gas on each chain.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$envFile = Join-Path $root "bsc-launcher\.env"
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq "" -or $line.StartsWith("#")) { return }
        $idx = $line.IndexOf("=")
        if ($idx -lt 1) { return }
        $key = $line.Substring(0, $idx).Trim()
        $val = $line.Substring($idx + 1).Trim().Trim('"').Trim("'")
        if ($key -and -not [Environment]::GetEnvironmentVariable($key)) {
            [Environment]::SetEnvironmentVariable($key, $val, "Process")
        }
    }
}

if (-not $env:FLASH_DEPLOYER_PRIVATE_KEY -and -not $env:BSC_DEPLOYER_PRIVATE_KEY) {
    Write-Host ""
    Write-Host "Live deploy needs a funded EVM wallet."
    Write-Host "1. Copy bsc-launcher\.env.example to bsc-launcher\.env (if missing)"
    Write-Host "2. Set FLASH_DEPLOYER_PRIVATE_KEY=0x... (or BSC_DEPLOYER_PRIVATE_KEY)"
    Write-Host "3. Fund that address with native gas on: Ethereum, BSC, Polygon, Arbitrum, Optimism, Avalanche, Base"
    Write-Host "4. Re-run: scripts\deploy-flash-coin-live.ps1"
    exit 1
}

if (-not (Test-Path "bin/onex.exe")) {
    go build -o bin/onex.exe ./cmd/onex
}

& bin/onex.exe flash-coin-deploy-live -config configs/flash-coin-mirror.json -out configs/flash-coin-live-addresses.json

Write-Host ""
Write-Host "Verify on-chain bytecode:"
& bin/onex.exe flash-coin-deploy-live -verify -out configs/flash-coin-live-addresses.json
